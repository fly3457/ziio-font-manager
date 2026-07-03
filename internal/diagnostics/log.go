package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var logMu sync.Mutex

func Append(logDir, fileName, message string) {
	if strings.TrimSpace(logDir) == "" || strings.TrimSpace(fileName) == "" {
		return
	}
	line := fmt.Sprintf("%s %s\n", time.Now().Format(time.RFC3339), message)
	logMu.Lock()
	defer logMu.Unlock()
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return
	}
	path := filepath.Join(logDir, filepath.Base(fileName))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

func Appendf(logDir, fileName, format string, args ...any) {
	Append(logDir, fileName, fmt.Sprintf(format, args...))
}
