// internal/delivery/telegram/services/counter/utils.go
package counter

import (
	"fmt"
	"strings"
)

// GetRequiredConfirmations возвращает количество требуемых подтверждений для периода
func GetRequiredConfirmations(period string) int {
	if period == "" {
		return 3 // дефолт
	}

	switch period {
	case "5m":
		return 3
	case "15m":
		return 3
	case "30m":
		return 4
	case "1h":
		return 6
	case "4h":
		return 8
	case "1d":
		return 12
	default:
		return 3
	}
}

// periodToMinutes конвертирует период строки в минуты
func (s *serviceImpl) periodToMinutes(period string) int {
	switch period {
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	default:
		return 15 // дефолт
	}
}

// containsString проверяет наличие подстроки в строке (вспомогательная функция)
func ContainsString(str, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(str) == 0 {
		return false
	}
	// Простая проверка на вхождение
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// convertPeriodToInt преобразует период из string в int
func ConvertPeriodToInt(periodStr string) (int, error) {
	// Нормализуем строку
	periodStr = strings.ToLower(strings.TrimSpace(periodStr))

	// ТОЛЬКО реальные форматы из CounterAnalyzer
	switch periodStr {
	case "5m":
		return 5, nil
	case "15m":
		return 15, nil
	case "30m":
		return 30, nil
	case "1h":
		return 60, nil
	case "4h":
		return 240, nil
	case "1d":
		return 1440, nil
	default:
		return 0, fmt.Errorf("неизвестный формат периода: %s", periodStr)
	}
}
