package logfile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var logPath string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	logPath = filepath.Join(home, ".cache", "lazysvn", "log")
}

func Append(msg string) {
	if logPath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format(time.RFC3339), msg)
}
