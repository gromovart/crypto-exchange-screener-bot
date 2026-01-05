// internal/core/domain/signals/detectors/utils.go
package analyzers

import (
	"log"
	"strconv"
	"strings"
	"time"
)

// ==================== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ДЛЯ ЗНАЧЕНИЙ ПО УМОЛЧАНИЮ ====================

// getDefaultFloat возвращает значение по умолчанию для float
func getDefaultFloat(defaultValue []float64) float64 {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0.0
}

// getDefaultInt возвращает значение по умолчанию для int
func getDefaultInt(defaultValue []int) int {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// getDefaultBool возвращает значение по умолчанию для bool
func getDefaultBool(defaultValue []bool) bool {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// getDefaultString возвращает значение по умолчанию для string
func getDefaultString(defaultValue []string) string {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// ==================== SAFE GET ФУНКЦИИ ====================

// SafeGetFloat безопасно получает float64 из конфига
func SafeGetFloat(config map[string]interface{}, key string, defaultValue ...float64) float64 {
	if config == nil {
		return getDefaultFloat(defaultValue)
	}

	value, exists := config[key]
	if !exists {
		return getDefaultFloat(defaultValue)
	}

	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		// Пытаемся распарсить строку
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			return floatVal
		}
		return getDefaultFloat(defaultValue)
	default:
		return getDefaultFloat(defaultValue)
	}
}

// SafeGetInt безопасно получает int из интерфейса
func SafeGetInt(value interface{}, defaultValue int) int {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int64:
		return int(v)
	case string:
		// Пытаемся распарсить строку
		if intVal, err := strconv.Atoi(v); err == nil {
			return intVal
		}
		return defaultValue
	default:
		log.Printf("⚠️  Warning: cannot convert %T to int, using default: %v", value, defaultValue)
		return defaultValue
	}
}

// SafeGetBool безопасно получает bool из конфига
func SafeGetBool(config map[string]interface{}, key string, defaultValue ...bool) bool {
	if config == nil {
		return getDefaultBool(defaultValue)
	}

	value, exists := config[key]
	if !exists {
		return getDefaultBool(defaultValue)
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		lower := strings.ToLower(v)
		return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
	case int:
		// 1 = true, 0 = false
		return v == 1
	case int64:
		return v == 1
	case float64:
		// 1.0 = true, 0.0 = false
		return v == 1.0
	case float32:
		return v == 1.0
	default:
		log.Printf("⚠️  Warning: cannot convert %T to bool, using default: %v", value, defaultValue)
		return getDefaultBool(defaultValue)
	}
}

// SafeGetString безопасно получает string из интерфейса
func SafeGetString(value interface{}, defaultValue string) string {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int64, float32, float64:
		// Конвертируем числа в строку
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	default:
		log.Printf("⚠️  Warning: cannot convert %T to string, using default: %v", value, defaultValue)
		return defaultValue
	}
}

// ==================== ДОПОЛНИТЕЛЬНЫЕ ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ====================

// SafeGetStringFromConfig безопасно получает строку из конфига
func SafeGetStringFromConfig(config map[string]interface{}, key string, defaultValue ...string) string {
	if config == nil {
		return getDefaultString(defaultValue)
	}

	value, exists := config[key]
	if !exists {
		return getDefaultString(defaultValue)
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int64, float32, float64:
		// Конвертируем числа в строку
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	default:
		return getDefaultString(defaultValue)
	}
}

// SafeGetIntFromConfig безопасно получает int из конфига
func SafeGetIntFromConfig(config map[string]interface{}, key string, defaultValue ...int) int {
	if config == nil {
		return getDefaultInt(defaultValue)
	}

	value, exists := config[key]
	if !exists {
		return getDefaultInt(defaultValue)
	}

	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int64:
		return int(v)
	case string:
		if intVal, err := strconv.Atoi(v); err == nil {
			return intVal
		}
		return getDefaultInt(defaultValue)
	default:
		return getDefaultInt(defaultValue)
	}
}

// SafeGetDuration безопасно получает time.Duration из конфига
func SafeGetDuration(config map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	value := SafeGetFloat(config, key, float64(defaultValue.Seconds()))
	return time.Duration(value * float64(time.Second))
}

// SafeGetStringSlice безопасно получает []string из конфига
func SafeGetStringSlice(config map[string]interface{}, key string, defaultValue []string) []string {
	if config == nil {
		return defaultValue
	}

	value, exists := config[key]
	if !exists {
		return defaultValue
	}

	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		var result []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case string:
		// Разделяем строку по запятым
		return strings.Split(v, ",")
	default:
		return defaultValue
	}
}
