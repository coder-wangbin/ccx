//go:build windows

package editor

import (
	"os/exec"
)

// Detect 返回 Windows 上可用的编辑器列表。
func Detect() []Editor {
	var editors []Editor

	if envEditor := editorFromEnv(); envEditor != nil {
		editors = append(editors, *envEditor)
	}

	candidates := []struct {
		id, name, bin string
	}{
		{"vscode", "Visual Studio Code", "code.cmd"},
		{"cursor", "Cursor", "cursor.cmd"},
		{"notepad++", "Notepad++", "notepad++.exe"},
		{"sublime", "Sublime Text", "subl.exe"},
		{"vim", "Vim", "vim.exe"},
		{"nano", "Nano", "nano.exe"},
	}

	for _, c := range candidates {
		if resolved, err := exec.LookPath(c.bin); err == nil {
			editors = append(editors, Editor{ID: c.id, Name: c.name, Path: resolved})
		}
	}

	return editors
}
