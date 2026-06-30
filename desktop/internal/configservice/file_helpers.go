package configservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func readJSONMap(path string) (map[string]any, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, false, nil
		}
		return nil, false, err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return map[string]any{}, true, nil
	}
	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, true, err
	}
	if data == nil {
		data = map[string]any{}
	}
	return data, true, nil
}

func readJSONFile(path string, dest any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, dest)
}

func readTextFile(path string) (string, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(content), true, nil
}

func mergeJSONFile(path string, patch map[string]any) error {
	data, _, err := readJSONMap(path)
	if err != nil {
		return err
	}
	for key, value := range patch {
		data[key] = value
	}
	return writeJSONAtomic(path, data)
}

func writeJSONAtomic(path string, data any) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return writeBytesAtomic(path, content)
}

func writeTextAtomic(path string, content string) error {
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return writeBytesAtomic(path, []byte(content))
}

func writeBytesAtomic(path string, content []byte) error {
	return writeBytesAtomicWithMode(path, content, 0o600)
}

func writeBytesAtomicWithMode(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if mode != 0 {
		if err := tmp.Chmod(mode); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func getNestedString(data map[string]any, keys ...string) (string, bool) {
	var current any = data
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = m[key]
		if !ok {
			return "", false
		}
	}
	value, ok := current.(string)
	return value, ok
}

func optionalString(value string, ok bool) *string {
	if !ok {
		return nil
	}
	return &value
}

func restoreStringField(data map[string]any, key string, value *string) {
	if value == nil {
		delete(data, key)
		return
	}
	data[key] = *value
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isLocalBaseURL(value string) bool {
	return strings.Contains(value, "127.0.0.1") || strings.Contains(value, "localhost")
}

func readEnvValueFromFile(path, key string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "export "); ok {
			line = strings.TrimSpace(rest)
		}
		k, value, _ := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if k != key {
			continue
		}
		value = strings.TrimSpace(value)
		return strings.Trim(value, `"'`)
	}
	return ""
}

// findNamedTomlBlock 返回 [table] 块的起止偏移量。
// 注意：仅支持标准 [table] 格式，不支持带引号的 key（如 [providers."ccx"]）或 inline table。
// 当前仅用于 Codex config.toml 的 [model_providers.ccx] 块。
