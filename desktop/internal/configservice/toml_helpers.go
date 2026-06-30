package configservice

import (
	"fmt"
	"regexp"
	"strings"
)

func extractCodexProviderBlock(content string) string {
	block, _ := extractNamedTomlBlock(content, "model_providers.ccx")
	return block
}

// extractTopLevelTomlString 从 TOML 内容中提取顶层字符串值。
// 注意：仅适用于简单格式（key = "value"），不支持多行字符串、转义引号或 inline table。
// 当前仅用于 Codex config.toml 的 model_provider 字段，该字段始终为简单字符串。

func extractTopLevelTomlString(content string, key string) (string, bool) {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]*)"\s*(?:#.*)?$`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", false
	}
	return match[1], true
}

func extractTomlStringField(content string, key string) (string, bool) {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]*)"\s*(?:#.*)?$`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", false
	}
	return match[1], true
}

func extractTomlBoolField(content string, key string) (bool, bool) {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*(true|false)\s*(?:#.*)?$`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return false, false
	}
	return strings.EqualFold(match[1], "true"), true
}

func findNamedTomlBlock(content string, table string) (int, int, bool) {
	header := "[" + table + "]"
	for lineStart := 0; lineStart < len(content); {
		lineEnd := strings.IndexByte(content[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(content)
		} else {
			lineEnd += lineStart
		}
		if strings.TrimSpace(content[lineStart:lineEnd]) == header {
			for nextLineStart := lineEnd + 1; nextLineStart < len(content); {
				nextLineEnd := strings.IndexByte(content[nextLineStart:], '\n')
				if nextLineEnd < 0 {
					nextLineEnd = len(content)
				} else {
					nextLineEnd += nextLineStart
				}
				nextLine := strings.TrimSpace(content[nextLineStart:nextLineEnd])
				if strings.HasPrefix(nextLine, "[") && strings.Contains(nextLine, "]") {
					return lineStart, nextLineStart, true
				}
				if nextLineEnd == len(content) {
					break
				}
				nextLineStart = nextLineEnd + 1
			}
			return lineStart, len(content), true
		}
		if lineEnd == len(content) {
			break
		}
		lineStart = lineEnd + 1
	}
	return 0, 0, false
}

func extractNamedTomlBlock(content string, table string) (string, bool) {
	start, end, ok := findNamedTomlBlock(content, table)
	if !ok {
		return "", false
	}
	return strings.TrimRight(content[start:end], "\n"), true
}

func upsertTopLevelTomlString(content string, key string, value string) string {
	line := fmt.Sprintf("%s = %q", key, value)
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=.*$`)
	if re.MatchString(content) {
		return re.ReplaceAllString(content, line)
	}
	if strings.TrimSpace(content) == "" {
		return line + "\n"
	}
	return line + "\n" + content
}

func restoreTopLevelTomlString(content string, key string, original *string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=.*(?:\n|$)`)
	if original == nil {
		return re.ReplaceAllString(content, "")
	}
	line := fmt.Sprintf("%s = %q", key, *original)
	if re.MatchString(content) {
		return re.ReplaceAllString(content, line+"\n")
	}
	return line + "\n" + content
}

func upsertNamedTomlBlock(content string, table string, block string) string {
	block = strings.TrimRight(block, "\n")
	if start, end, ok := findNamedTomlBlock(content, table); ok {
		return content[:start] + block + "\n" + content[end:]
	}
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return block + "\n"
	}
	return content + "\n\n" + block + "\n"
}

func restoreNamedTomlBlock(content string, table string, original *string) string {
	start, end, ok := findNamedTomlBlock(content, table)
	if original == nil {
		if !ok {
			return strings.TrimRight(content, "\n") + "\n"
		}
		return strings.TrimRight(content[:start]+content[end:], "\n") + "\n"
	}
	block := strings.TrimRight(*original, "\n") + "\n"
	if ok {
		return content[:start] + block + content[end:]
	}
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return block
	}
	return content + "\n\n" + block
}

// PreviewApply 预览 Apply 操作的变更，不实际写入文件。
