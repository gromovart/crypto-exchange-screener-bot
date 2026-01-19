// internal/delivery/telegram/services/signal_settings/utils.go
package signal_settings

import (
	"fmt"
	"strconv"
)

// getToggleText возвращает текст для переключателя
func getToggleText(enabled bool) string {
	if enabled {
		return "включены ✅"
	}
	return "выключены ❌"
}

// convertToFloat преобразует значение в float64
func convertToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("неподдерживаемый тип значения: %T", value)
	}
}
