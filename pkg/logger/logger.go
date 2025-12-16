// pkg/logger/logger.go
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type Logger struct {
	logFile   *os.File
	console   io.Writer
	debugMode bool
}

func NewLogger(logPath string, debug bool) (*Logger, error) {
	// Создаем директорию для логов если нужно
	os.MkdirAll("logs", 0755)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	multiWriter := io.MultiWriter(os.Stdout, file)

	return &Logger{
		logFile:   file,
		console:   multiWriter,
		debugMode: debug,
	}, nil
}

func (l *Logger) Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[INFO] %s %s", timestamp, msg)
}

func (l *Logger) Status(stats map[string]string) {
	fmt.Fprintln(l.console, strings.Repeat("─", 50))
	fmt.Fprintln(l.console, "📊 СТАТУС СИСТЕМЫ")
	for key, value := range stats {
		fmt.Fprintf(l.console, "   %-20s: %s\n", key, value)
	}
	fmt.Fprintln(l.console, strings.Repeat("─", 50))
}

func (l *Logger) Close() {
	l.logFile.Close()
}
