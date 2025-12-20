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

// Уровни логирования
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"
)

type Logger struct {
	logFile   *os.File
	console   io.Writer
	logLevel  string // Уровень логирования
	debugMode bool
}

func NewLogger(logPath string, logLevel string, debug bool) (*Logger, error) {
	os.MkdirAll("logs", 0755)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	multiWriter := io.MultiWriter(os.Stdout, file)

	return &Logger{
		logFile:   file,
		console:   multiWriter,
		logLevel:  strings.ToUpper(logLevel),
		debugMode: debug,
	}, nil
}

// shouldLog проверяет, нужно ли логировать сообщение на данном уровне
func (l *Logger) shouldLog(level string) bool {
	levelPriority := map[string]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
		LevelFatal: 4,
	}

	currentPriority, ok1 := levelPriority[l.logLevel]
	msgPriority, ok2 := levelPriority[level]

	if !ok1 || !ok2 {
		return true // Если неизвестный уровень, логируем всё
	}

	return msgPriority >= currentPriority
}

func (l *Logger) log(level string, format string, v ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	msg := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Цвета для консоли
	color := ""
	reset := ""
	if l.debugMode {
		switch level {
		case LevelDebug:
			color = "\033[36m" // Cyan
		case LevelInfo:
			color = "\033[32m" // Green
		case LevelWarn:
			color = "\033[33m" // Yellow
		case LevelError:
			color = "\033[31m" // Red
		case LevelFatal:
			color = "\033[35m" // Magenta
		}
		reset = "\033[0m"
	}

	log.Printf("%s[%s] %s %s%s", color, level, timestamp, msg, reset)
}

// Методы для разных уровней
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(LevelDebug, format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.log(LevelInfo, format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(LevelWarn, format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.log(LevelError, format, v...)
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(LevelFatal, format, v...)
	log.Fatalf(format, v...)
}

func (l *Logger) Status(stats map[string]string) {
	fmt.Fprintln(l.console, strings.Repeat("─", 50))
	fmt.Fprintln(l.console, "📊 СТАТУС СИСТЕМЫ")
	for key, value := range stats {
		fmt.Fprintf(l.console, "   %-20s: %s\n", key, value)
	}
	fmt.Fprintln(l.console, strings.Repeat("─", 50))
}

func (l *Logger) Signal(symbol, direction string, change, confidence float64, period int) {
	icon := "📈"
	if direction == "down" {
		icon = "📉"
	}

	arrow := "↑"
	if direction == "down" {
		arrow = "↓"
	}

	l.Info("%s СИГНАЛ: %s %s%.2f%% за %d минут (уверенность: %.0f%%)",
		icon, symbol, arrow, change, period, confidence)
}

func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
