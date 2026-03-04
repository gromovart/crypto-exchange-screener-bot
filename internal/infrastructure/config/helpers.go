// internal/infrastructure/config/helpers.go
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// ============================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ДЛЯ ЧТЕНИЯ ENV
// ============================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// ============================================
// ПАРСИНГ ЗНАЧЕНИЙ
// ============================================

func parseIntList(value string) []int {
	var result []int
	if value == "" {
		return result
	}

	parts := strings.Split(value, ",")
	for _, part := range parts {
		if part == "" {
			continue
		}
		if intValue, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			result = append(result, intValue)
		}
	}
	return result
}

func parsePatterns(value string) []string {
	if value == "" {
		return []string{}
	}

	patterns := strings.Split(value, ",")
	var result []string
	for _, pattern := range patterns {
		if trimmed := strings.TrimSpace(pattern); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func isValidPeriod(period string) bool {
	validPeriods := map[string]bool{
		"1m":  true,
		"5m":  true,
		"15m": true,
		"30m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
	}
	return validPeriods[period]
}
