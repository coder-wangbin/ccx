//go:build linux

package editor

import (
	"os"
	"os/exec"
)

// Detect 返回 Linux 上可用的编辑器列表。
func Detect() []Editor {
	var editors []Editor

	if envEditor := editorFromEnv(); envEditor != nil {
		editors = append(editors, *envEditor)
	}

	candidates := []struct {
		id, name, bin string
	}{
		{"vscode", "Visual Studio Code", "code"},
		{"cursor", "Cursor", "cursor"},
		{"sublime", "Sublime Text", "subl"},
		{"gedit", "GNOME Text Editor", "gedit"},
		{"kate", "Kate", "kate"},
		{"vim", "Vim", "vim"},
		{"nvim", "Neovim", "nvim"},
		{"nano", "Nano", "nano"},
	}

	for _, c := range candidates {
		if resolved, err := exec.LookPath(c.bin); err == nil {
			editors = append(editors, Editor{ID: c.id, Name: c.name, Path: resolved})
		}
	}

	return editors
}
