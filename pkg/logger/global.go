// pkg/logger/global.go
package logger

import (
	"log"
	"os"
)

var globalLogger *Logger

func InitGlobal(logPath, logLevel string, debug bool) error {
	var err error
	globalLogger, err = NewLogger(logPath, logLevel, debug)
	return err
}

func GetLogger() *Logger {
	if globalLogger == nil {
		// Fallback к простому логгеру
		log.SetOutput(os.Stdout)
	}
	return globalLogger
}

// Глобальные методы для удобства
func Debug(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(format, v...)
	}
}

func Info(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(format, v...)
	}
}

func Warn(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warn(format, v...)
	}
}

func Error(format string, v ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(format, v...)
	}
}

func Signal(symbol, direction string, change, confidence float64, period int) {
	if globalLogger != nil {
		globalLogger.Signal(symbol, direction, change, confidence, period)
	}
}
