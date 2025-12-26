package analyzers

import (
	"log"
)

// SafeGetFloat безопасно получает float из интерфейса
func SafeGetFloat(value interface{}, defaultValue float64) float64 {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case float32:
		return float64(v)
	default:
		log.Printf("⚠️  Warning: cannot convert %T to float64, using default: %v", value, defaultValue)
		return defaultValue
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
	default:
		log.Printf("⚠️  Warning: cannot convert %T to int, using default: %v", value, defaultValue)
		return defaultValue
	}
}

// SafeGetBool безопасно получает bool из интерфейса
func SafeGetBool(value interface{}, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case bool:
		return v
	default:
		log.Printf("⚠️  Warning: cannot convert %T to bool, using default: %v", value, defaultValue)
		return defaultValue
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
	default:
		log.Printf("⚠️  Warning: cannot convert %T to string, using default: %v", value, defaultValue)
		return defaultValue
	}
}
