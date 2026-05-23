//go:build darwin

package editor

import (
	"os"
	"os/exec"
)

// Detect 返回 macOS 上可用的编辑器列表。
func Detect() []Editor {
	var editors []Editor

	// 优先检查 $EDITOR / $VISUAL 环境变量
	if envEditor := editorFromEnv(); envEditor != nil {
		editors = append(editors, *envEditor)
	}

	// 常见 macOS 编辑器（按推荐顺序）
	candidates := []struct {
		id, name, appPath string
	}{
		{"vscode", "Visual Studio Code", "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"},
		{"cursor", "Cursor", "/Applications/Cursor.app/Contents/Resources/app/bin/code"},
		{"sublime", "Sublime Text", "/Applications/Sublime Text.app/Contents/SharedSupport/bin/subl"},
		{"textmate", "TextMate", "/Applications/TextMate.app/Contents/Resources/mate"},
		{"bbedit", "BBEdit", "/Applications/BBEdit.app/Contents/Helpers/bbedit_tool"},
		{"nova", "Nova", "/Applications/Nova.app/Contents/Helpers/nova"},
		{"vim", "Vim", ""},
		{"nano", "Nano", ""},
	}

	for _, c := range candidates {
		path := c.appPath
		if path == "" {
			if resolved, err := exec.LookPath(c.id); err == nil {
				path = resolved
			} else {
				continue
			}
		}
		if _, err := os.Stat(path); err == nil {
			editors = append(editors, Editor{ID: c.id, Name: c.name, Path: path})
		}
	}

	return editors
}
