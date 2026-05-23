package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// editorFromEnv 从 $EDITOR / $VISUAL 环境变量提取编辑器。
func editorFromEnv() *Editor {
	for _, key := range []string{"EDITOR", "VISUAL"} {
		val := strings.TrimSpace(os.Getenv(key))
		if val == "" {
			continue
		}
		bin := strings.Fields(val)[0]
		resolved, err := exec.LookPath(bin)
		if err != nil {
			continue
		}
		name := bin
		if strings.Contains(bin, "code") {
			name = "VS Code (EDITOR)"
		} else if strings.Contains(bin, "vim") {
			name = "Vim (EDITOR)"
		} else if strings.Contains(bin, "nvim") {
			name = "Neovim (EDITOR)"
		} else {
			name = bin + " (EDITOR)"
		}
		return &Editor{ID: "env:" + bin, Name: name, Path: resolved}
	}
	return nil
}

// Open 使用指定编辑器打开文件。
func Open(editorPath, filePath string) error {
	if editorPath == "" {
		return fmt.Errorf("编辑器路径不能为空")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// 终端编辑器（vim/nano 等）不能用 open 启动，需要在终端模拟器中运行。
		// 对于 GUI 编辑器，直接调用可执行文件。
		cmd = exec.Command(editorPath, filePath)
	case "linux":
		cmd = exec.Command(editorPath, filePath)
	case "windows":
		cmd = exec.Command(editorPath, filePath)
	default:
		return fmt.Errorf("不支持的平台: %s", runtime.GOOS)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
