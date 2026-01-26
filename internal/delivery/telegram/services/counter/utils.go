// internal/delivery/telegram/services/counter/utils.go
package counter

import periodPkg "crypto-exchange-screener-bot/pkg/period"

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
	minutes, err := periodPkg.StringToMinutes(period)
	if err != nil {
		return periodPkg.DefaultMinutes
	}
	return minutes
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
func ConvertPeriodToInt(period string) (int, error) {
	return periodPkg.StringToMinutes(period)
}
