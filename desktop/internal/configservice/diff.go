package configservice

import (
	"encoding/json"
	"fmt"
	"strings"
)

// computeTextDiff 逐行对比 before/after，生成 git-style diff 行。
func computeTextDiff(path, before, after string) FileDiff {
	oldLines := splitLines(before)
	newLines := splitLines(after)

	action := "modify"
	if before == "" && after != "" {
		action = "create"
	} else if before != "" && after == "" {
		action = "delete"
	} else if before == after {
		action = "modify"
	}

	lines := lcsDiff(oldLines, newLines)
	return FileDiff{Path: path, Action: action, Lines: lines}
}

// computeJSONDiff 将两个 map 格式化为 JSON 后逐行对比。
func computeJSONDiff(path string, before, after map[string]any) FileDiff {
	oldContent := ""
	if before != nil {
		oldContent = formatJSON(before)
	}
	newContent := ""
	if after != nil {
		newContent = formatJSON(after)
	}
	return computeTextDiff(path, oldContent, newContent)
}

// formatJSON 将 map 格式化为缩进 JSON 字符串。
func formatJSON(data map[string]any) string {
	if data == nil {
		return ""
	}
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(content) + "\n"
}

// maskSensitiveValue 对单个值进行脱敏。
// 短于 12 字符显示为 "***"；否则保留前 3 后 4，中间 "***"。
func maskSensitiveValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) < 12 {
		return "***"
	}
	prefix := string(runes[:3])
	suffix := string(runes[len(runes)-4:])
	return prefix + "***" + suffix
}

// sensitiveFieldKeys 需要脱敏的配置字段名。
var sensitiveFieldKeys = []string{
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"OPENAI_API_KEY",
}

// maskMapSensitiveKeys 对 map 中指定 key 的值进行脱敏（返回新 map，不修改原 map）。
func maskMapSensitiveKeys(data map[string]any, keys ...string) map[string]any {
	if data == nil {
		return nil
	}
	result := make(map[string]any, len(data))
	for k, v := range data {
		result[k] = v
	}
	for _, key := range keys {
		if val, ok := result[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				result[key] = maskSensitiveValue(s)
			}
		}
	}
	return result
}

// maskJSONSensitiveKeys 对 JSON map 中嵌套的 env map 内的敏感字段进行脱敏。
func maskJSONSensitiveKeys(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	result := make(map[string]any, len(data))
	for k, v := range data {
		result[k] = v
	}
	if env, ok := result["env"].(map[string]any); ok {
		result["env"] = maskMapSensitiveKeys(env, sensitiveFieldKeys...)
	}
	return result
}

// maskTextSensitiveValues 对文本内容中出现的敏感值进行行内脱敏。
// 用于 TOML / JSON 文本 diff 的 before/after 内容。
func maskTextSensitiveValues(content string, keyValues map[string]string) string {
	if len(keyValues) == 0 {
		return content
	}
	result := content
	for _, value := range keyValues {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		masked := maskSensitiveValue(value)
		result = strings.ReplaceAll(result, value, masked)
	}
	return result
}

// computeJSONDiffWithMask 在原始数据上计算 diff（正确识别变更），
// 再对展示内容进行脱敏。敏感字段值虽均掩码为 "***"，但 diff 类型
// （removed/added）基于原始值判定，用户仍能感知到变更。
func computeJSONDiffWithMask(path string, oldData, newData map[string]any, keys ...string) FileDiff {
	if len(keys) == 0 {
		return computeTextDiff(path, formatJSON(oldData), formatJSON(newData))
	}
	// 在数据层按字段掩码敏感值（含嵌套 map），再序列化做普通文本 diff，
	// 避免文本级 ReplaceAll 对短值（如 "key"）造成的子串误伤（如 auth_mode="apikey"）。
	oldMasked := maskDataSensitiveKeys(oldData, keys)
	newMasked := maskDataSensitiveKeys(newData, keys)
	return computeTextDiff(path, formatJSON(oldMasked), formatJSON(newMasked))
}

// maskDataSensitiveKeys 递归深拷贝 map，并对任意层级中命中 keys 的字符串值脱敏。
// 不修改原始 map。
func maskDataSensitiveKeys(data map[string]any, keys []string) map[string]any {
	if data == nil {
		return nil
	}
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}
	result := make(map[string]any, len(data))
	for k, v := range data {
		switch val := v.(type) {
		case string:
			if keySet[k] && val != "" {
				result[k] = maskSensitiveValue(val)
			} else {
				result[k] = val
			}
		case map[string]any:
			result[k] = maskDataSensitiveKeys(val, keys)
		default:
			result[k] = v
		}
	}
	return result
}

// computeTextDiffWithMask 在原始文本上计算 diff，再对展示内容进行脱敏。
// 用于两侧敏感值相同的场景（如 restore）。
func computeTextDiffWithMask(path, before, after string, keyValues map[string]string) FileDiff {
	return computeTextDiffWithSeparateMasks(path, before, after, keyValues, keyValues)
}

// computeTextDiffWithSeparateMasks 在原始文本上计算 diff，再对展示内容进行脱敏。
// oldKeys/newKeys 分别用于脱敏 before/after 内容，避免密钥变更时单侧泄露。
func computeTextDiffWithSeparateMasks(path, before, after string, oldKeys, newKeys map[string]string) FileDiff {
	// 用值集合去重，避免同键名在 map 合并时旧值被新值覆盖
	seen := make(map[string]bool, len(oldKeys)+len(newKeys))
	for _, v := range oldKeys {
		if v = strings.TrimSpace(v); v != "" {
			seen[v] = true
		}
	}
	for _, v := range newKeys {
		if v = strings.TrimSpace(v); v != "" {
			seen[v] = true
		}
	}
	oldMasked, newMasked := before, after
	for v := range seen {
		masked := maskSensitiveValue(v)
		oldMasked = strings.ReplaceAll(oldMasked, v, masked)
		newMasked = strings.ReplaceAll(newMasked, v, masked)
	}
	return computeTextDiffFromMasked(path, before, after, oldMasked, newMasked)
}

// computeTextDiffFromMasked 在原始内容上做 LCS diff 确定变更类型，
// 再将掩码后的内容填充到对应 diff 行。
func computeTextDiffFromMasked(path, oldRaw, newRaw, oldMasked, newMasked string) FileDiff {
	oldRawLines := splitLines(oldRaw)
	newRawLines := splitLines(newRaw)
	oldMaskedLines := splitLines(oldMasked)
	newMaskedLines := splitLines(newMasked)

	action := "modify"
	if oldRaw == "" && newRaw != "" {
		action = "create"
	} else if oldRaw != "" && newRaw == "" {
		action = "delete"
	}

	rawDiff := lcsDiff(oldRawLines, newRawLines)

	// 将掩码后的内容映射到 diff 行
	oldIdx, newIdx := 0, 0
	lines := make([]DiffLine, len(rawDiff))
	for i, d := range rawDiff {
		switch d.Type {
		case "context":
			lines[i] = DiffLine{Type: "context", Content: oldMaskedLines[oldIdx]}
			oldIdx++
			newIdx++
		case "removed":
			lines[i] = DiffLine{Type: "removed", Content: oldMaskedLines[oldIdx]}
			oldIdx++
		case "added":
			lines[i] = DiffLine{Type: "added", Content: newMaskedLines[newIdx]}
			newIdx++
		}
	}

	return FileDiff{Path: path, Action: action, Lines: lines}
}

// extractNestedStringValues 从 map 中提取指定 key 的字符串值，
// 支持嵌套 map（如 env 子 map）的递归查找。
func extractNestedStringValues(data map[string]any, keys []string) map[string]string {
	result := make(map[string]string, len(keys))
	if data == nil {
		return result
	}
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}
	// 顶层直接匹配
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if s, ok := val.(string); ok {
				result[key] = s
			}
		}
	}
	// 递归查找嵌套 map（如 env 子 map）
	for _, v := range data {
		if sub, ok := v.(map[string]any); ok {
			for key := range keySet {
				if _, exists := result[key]; exists {
					continue
				}
				if val, ok := sub[key]; ok {
					if s, ok := val.(string); ok {
						result[key] = s
					}
				}
			}
		}
	}
	return result
}

// splitLines 将文本按换行符分割为行切片。空文本返回空切片。
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	// 去除末尾空行（由末尾换行符产生）
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// lcsDiff 使用 LCS 算法生成 diff 行序列。
func lcsDiff(oldLines, newLines []string) []DiffLine {
	m, n := len(oldLines), len(newLines)
	if m == 0 && n == 0 {
		return nil
	}

	// 特殊情况优化
	if m == 0 {
		lines := make([]DiffLine, n)
		for i, l := range newLines {
			lines[i] = DiffLine{Type: "added", Content: l}
		}
		return lines
	}
	if n == 0 {
		lines := make([]DiffLine, m)
		for i, l := range oldLines {
			lines[i] = DiffLine{Type: "removed", Content: l}
		}
		return lines
	}

	// LCS DP 表
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	// 回溯生成 diff（逆序收集，最后反转）
	var reversed []DiffLine
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			reversed = append(reversed, DiffLine{Type: "context", Content: oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			reversed = append(reversed, DiffLine{Type: "added", Content: newLines[j-1]})
			j--
		} else {
			reversed = append(reversed, DiffLine{Type: "removed", Content: oldLines[i-1]})
			i--
		}
	}

	// 反转得到正确顺序
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed
}
