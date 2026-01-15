// internal/delivery/telegram/controllers/counter/utils.go
package counter

// getString безопасно извлекает строку из map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getFloat64 безопасно извлекает float64 из map
func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0.0
}

// getFloat64FromMap безопасно извлекает float64 из вложенной map[string]interface{}
func getFloat64FromMap(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0.0
}

// getFloat64FromFloatMap безопасно извлекает float64 из вложенной map[string]float64
func getFloat64FromFloatMap(m map[string]float64, key string) float64 {
	if val, ok := m[key]; ok {
		return val
	}
	return 0.0
}
