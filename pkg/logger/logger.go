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
	appLogFile   *os.File // для всех логов
	errorLogFile *os.File // только для ошибок
	console      io.Writer
	logLevel     string
	debugMode    bool
}

func NewLogger(logPath string, logLevel string, debug bool) (*Logger, error) {
	// Всегда создаем директорию logs
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию logs: %w", err)
	}

	// Игнорируем переданный logPath, используем только стандартные имена
	appLogPath := "logs/app.log"
	errorLogPath := "logs/error.log"

	// Основной лог-файл (app.log)
	appFile, err := os.OpenFile(appLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть app.log: %w", err)
	}

	// Файл для ошибок (error.log)
	errorFile, err := os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		appFile.Close()
		return nil, fmt.Errorf("не удалось открыть error.log: %w", err)
	}

	multiWriter := io.MultiWriter(os.Stdout, appFile)

	return &Logger{
		appLogFile:   appFile,
		errorLogFile: errorFile,
		console:      multiWriter,
		logLevel:     strings.ToUpper(logLevel),
		debugMode:    debug,
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
		return true
	}

	return msgPriority >= currentPriority
}

// isErrorLevel проверяет, является ли уровень ошибкой
func isErrorLevel(level string) bool {
	return level == LevelError || level == LevelFatal
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

	// Формируем строку лога
	logLine := fmt.Sprintf("%s[%s] %s %s%s\n", color, level, timestamp, msg, reset)

	// Пишем в консоль и app.log
	fmt.Fprint(l.console, logLine)

	// Если это ошибка - пишем также в error.log
	if isErrorLevel(level) && l.errorLogFile != nil {
		errorLine := fmt.Sprintf("[%s] %s %s\n", level, timestamp, msg)
		fmt.Fprint(l.errorLogFile, errorLine)
	}
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
	if l.appLogFile != nil {
		l.appLogFile.Close()
	}
	if l.errorLogFile != nil {
		l.errorLogFile.Close()
	}
}
