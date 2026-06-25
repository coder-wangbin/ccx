package utils

import (
	"sort"
	"strings"

	"github.com/tidwall/gjson"
)

const imagePlaceholderJSON = `"<image>"`

// imageBase64Marker 是所有受支持图片 schema 的共同特征子串：
//   - OpenAI Chat / Responses 的 data URL 形如 "data:image/...;base64,..."
//   - Anthropic 的 source.type == "base64"
//
// 三者都必然包含 "base64"，故不含该子串（大小写不敏感）的 body 一定没有内联图片，
// 可跳过 gjson 全量解析直接返回。data URL 的 ";base64" 段大小写不限，故用大小写不敏感匹配，
// 保证短路只是"必要条件"过滤、绝不漏判任何真实图片请求。
const imageBase64Marker = "base64"

// containsBase64Fold 在 body 中做 ASCII 大小写不敏感的子串查找，不分配额外内存
// （避免 bytes.ToLower 复制整个 body 抵消短路收益）。marker 必须为全小写。
func containsBase64Fold(body []byte, marker string) bool {
	n, m := len(body), len(marker)
	if m == 0 {
		return true
	}
	for i := 0; i+m <= n; i++ {
		match := true
		for j := 0; j < m; j++ {
			c := body[i+j]
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A' // 统一转小写后比较
			}
			if c != marker[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// imagePlaceholder 是占位符的解析后值（不含 JSON 引号），
// 用于识别已被本函数剥离过的字段，保证重复调用的幂等性。
const imagePlaceholder = "<image>"

type imageReplacement struct {
	start  int
	end    int
	tokens int
}

// extractImageTokensAndStripBytes 用 gjson 遍历请求体，按 type 字段区分图片 schema：
//   - OpenAI Chat: content[i].type=="image_url"，base64 在 content[i].image_url.url
//   - Responses:   content[i].type=="input_image"，base64 在 content[i].image_url（字符串）
//   - Anthropic:   content[i].type=="image"，base64 在 content[i].source.data
//
// PDF / audio 等非图 base64 会被忽略，不算图片 token。
// 同时用 gjson Result.Index/Raw 精确定位目标 JSON string literal，并把它替换成
// "<image>"，避免 EstimateTokens 把 base64 长字段按文本字符数高估。
func extractImageTokensAndStripBytes(body []byte) ([]byte, int) {
	// 性能短路：受支持的图片 schema 必含 "base64" 子串，不含则一定无内联图片，
	// 直接返回原 body，省掉一次 gjson 全量解析（高频小包如流式 usage 修补尤其受益）。
	if !containsBase64Fold(body, imageBase64Marker) {
		return body, 0
	}

	var replacements []imageReplacement
	imageTokens := 0

	// 支持根本身就是消息数组（EstimateMessagesTokens 的输入）
	root := gjson.ParseBytes(body)
	if root.IsArray() {
		replacements, imageTokens = collectImageReplacementsFromMessageArray(body, root)
		return applyImageReplacements(body, replacements, imageTokens)
	}

	// messages（Chat/Anthropic）与 input（Responses）是互斥的请求格式，
	// 命中其一即返回，避免畸形请求两者并存时被双重计数。
	for _, rootPath := range []string{"messages", "input"} {
		arr := gjson.GetBytes(body, rootPath)
		if !arr.IsArray() {
			continue
		}
		replacements, imageTokens = collectImageReplacementsFromMessageArray(body, arr)
		return applyImageReplacements(body, replacements, imageTokens)
	}

	return applyImageReplacements(body, replacements, imageTokens)
}

func collectImageReplacementsFromMessageArray(body []byte, arr gjson.Result) ([]imageReplacement, int) {
	var replacements []imageReplacement
	imageTokens := 0

	arr.ForEach(func(_, msg gjson.Result) bool {
		content := msg.Get("content")
		if !content.IsArray() {
			return true
		}
		content.ForEach(func(_, block gjson.Result) bool {
			b64, field := imagePayloadFromBlock(block)
			if b64 == "" {
				return true
			}
			start, end, ok := stringLiteralRange(body, field)
			if !ok {
				// 定位失败时不剥离也不计 token：若只计 token 而 base64 仍留在 body 中，
				// EstimateTokens 会把它按字符数再算一遍，反而退回到本次修复要解决的高估问题。
				return true
			}
			tokens := estimateImageTokensFromBase64(b64)
			imageTokens += tokens
			replacements = append(replacements, imageReplacement{start: start, end: end, tokens: tokens})
			return true
		})
		return true
	})

	return replacements, imageTokens
}

func imagePayloadFromBlock(block gjson.Result) (b64 string, field gjson.Result) {
	switch block.Get("type").String() {
	case "image_url":
		// OpenAI Chat: image_url.url 是 data:image/...;base64,...
		// 已剥离的 "<image>" 占位符不是合法 data URL，dataURLPayload 返回空，天然跳过（幂等）。
		if url := block.Get("image_url.url"); url.Type == gjson.String {
			if b := dataURLPayload(url.String()); b != "" {
				return b, url
			}
		}
	case "input_image":
		// Responses: image_url 直接是 data:image/...;base64,... 字符串
		// 同上，占位符经 dataURLPayload 返回空，重复调用幂等。
		if url := block.Get("image_url"); url.Type == gjson.String {
			if b := dataURLPayload(url.String()); b != "" {
				return b, url
			}
		}
	case "image":
		// Anthropic: 仅 media_type=image/* 的 base64 source 才按图片估算
		if src := block.Get("source"); src.Exists() {
			mediaType := strings.ToLower(src.Get("media_type").String())
			if src.Get("type").String() == "base64" && strings.HasPrefix(mediaType, "image/") {
				if data := src.Get("data"); data.Type == gjson.String {
					// 跳过已被本函数剥离过的占位符，保证对已处理 body 的幂等性
					if b := data.String(); b != "" && b != imagePlaceholder {
						return b, data
					}
				}
			}
		}
	}
	return "", gjson.Result{}
}

func stringLiteralRange(body []byte, r gjson.Result) (start, end int, ok bool) {
	if r.Raw == "" || r.Index < 0 {
		return 0, 0, false
	}

	start = r.Index
	end = start + len(r.Raw)
	if start < 0 || start >= len(body) || end > len(body) || start >= end {
		return 0, 0, false
	}
	if len(r.Raw) < 2 || r.Raw[0] != '"' || r.Raw[len(r.Raw)-1] != '"' {
		return 0, 0, false
	}
	if !rawMatches(body[start:end], r.Raw) {
		return 0, 0, false
	}
	return start, end, true
}

func rawMatches(body []byte, raw string) bool {
	if len(body) != len(raw) {
		return false
	}
	for i := range body {
		if body[i] != raw[i] {
			return false
		}
	}
	return true
}

func applyImageReplacements(body []byte, replacements []imageReplacement, imageTokens int) ([]byte, int) {
	if len(replacements) == 0 {
		return body, imageTokens
	}

	kept := normalizeImageReplacements(replacements)
	if len(kept) == 0 {
		return body, imageTokens
	}

	// 从后往前替换，避免前面的替换改变后续 byte range 的 offset。
	sort.Slice(kept, func(i, j int) bool {
		return kept[i].start > kept[j].start
	})

	out := body
	for _, repl := range kept {
		next := make([]byte, 0, len(out)-(repl.end-repl.start)+len(imagePlaceholderJSON))
		next = append(next, out[:repl.start]...)
		next = append(next, imagePlaceholderJSON...)
		next = append(next, out[repl.end:]...)
		out = next
	}
	return out, imageTokens
}

func normalizeImageReplacements(replacements []imageReplacement) []imageReplacement {
	valid := replacements[:0]
	for _, repl := range replacements {
		if repl.start < repl.end {
			valid = append(valid, repl)
		}
	}
	if len(valid) == 0 {
		return nil
	}

	sort.Slice(valid, func(i, j int) bool {
		if valid[i].start == valid[j].start {
			return valid[i].end < valid[j].end
		}
		return valid[i].start < valid[j].start
	})

	kept := valid[:0]
	lastEnd := -1
	for _, repl := range valid {
		// 理论上不同 image 字段不应重叠；遇到异常/重复 range 时保守跳过，避免 panic 或错替。
		if repl.start < lastEnd {
			continue
		}
		kept = append(kept, repl)
		lastEnd = repl.end
	}
	return kept
}

// dataURLPayload 从 "data:image/...;base64,xxx" 提取 base64 主体；不是图片 data URL 返回空串。
// 按 RFC 2397，";base64" 必须是逗号前的最后一个分号段，故用 HasSuffix 而非 Contains，
// 避免把 "data:image/x;base64xyz,..." 这类畸形 header 误判为图片。
func dataURLPayload(url string) string {
	comma := strings.IndexByte(url, ',')
	if comma < 0 {
		return ""
	}
	header := strings.ToLower(url[:comma])
	if !strings.HasPrefix(header, "data:image/") || !strings.HasSuffix(header, ";base64") {
		return ""
	}
	return url[comma+1:]
}
