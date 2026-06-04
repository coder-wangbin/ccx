package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupLogDirNoneConsoleTrue(t *testing.T) {
	cfg := &Config{
		LogDir:  "none",
		Console: true,
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// 不应创建目录
	if _, err := os.Stat("none"); !os.IsNotExist(err) {
		t.Fatal("Setup() created directory 'none', should not")
	}

	// rawFileLog 应写入 io.Discard
	if rawFileLog == nil {
		t.Fatal("rawFileLog is nil")
	}
	if rawFileLog.Writer() != io.Discard {
		t.Fatalf("rawFileLog writer = %v, want io.Discard", rawFileLog.Writer())
	}

	// consoleLog 应写入 os.Stdout
	if consoleLog == nil {
		t.Fatal("consoleLog is nil")
	}
	if consoleLog.Writer() != os.Stdout {
		t.Fatalf("consoleLog writer = %v, want os.Stdout", consoleLog.Writer())
	}

	// 全局 log 应写入 os.Stdout
	if log.Default().Writer() != os.Stdout {
		t.Fatalf("global log writer = %v, want os.Stdout", log.Default().Writer())
	}
}

func TestSetupLogDirNoneConsoleFalse(t *testing.T) {
	cfg := &Config{
		LogDir:  "none",
		Console: false,
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// 不应创建目录
	if _, err := os.Stat("none"); !os.IsNotExist(err) {
		t.Fatal("Setup() created directory 'none', should not")
	}

	// rawFileLog 应写入 io.Discard
	if rawFileLog == nil {
		t.Fatal("rawFileLog is nil")
	}
	if rawFileLog.Writer() != io.Discard {
		t.Fatalf("rawFileLog writer = %v, want io.Discard", rawFileLog.Writer())
	}

	// consoleLog 应写入 io.Discard
	if consoleLog == nil {
		t.Fatal("consoleLog is nil")
	}
	if consoleLog.Writer() != io.Discard {
		t.Fatalf("consoleLog writer = %v, want io.Discard", consoleLog.Writer())
	}

	// 全局 log 应写入 io.Discard
	if log.Default().Writer() != io.Discard {
		t.Fatalf("global log writer = %v, want io.Discard", log.Default().Writer())
	}
}

func TestSetupLogDirNullConsoleTrue(t *testing.T) {
	cfg := &Config{
		LogDir:  "null",
		Console: true,
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// 不应创建目录
	if _, err := os.Stat("null"); !os.IsNotExist(err) {
		t.Fatal("Setup() created directory 'null', should not")
	}

	// rawFileLog 应写入 io.Discard
	if rawFileLog.Writer() != io.Discard {
		t.Fatalf("rawFileLog writer = %v, want io.Discard", rawFileLog.Writer())
	}
}

func TestSetupNormalLogDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		LogDir:     tmpDir,
		LogFile:    "test.log",
		MaxSize:    10,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
		Console:    true,
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// 应创建日志文件
	logPath := filepath.Join(tmpDir, "test.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}

	// rawFileLog 不应是 io.Discard
	if rawFileLog.Writer() == io.Discard {
		t.Fatal("rawFileLog should not be io.Discard for normal log dir")
	}
}

func TestIsLogDisabled(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"none", true},
		{"null", true},
		{"NONE", true},
		{"NULL", true},
		{"None", true},
		{"Null", true},
		{" none ", true},
		{" null ", true},
		{"logs", false},
		{"/var/log/ccx", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsLogDisabled(tt.input)
			if got != tt.want {
				t.Fatalf("IsLogDisabled(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
