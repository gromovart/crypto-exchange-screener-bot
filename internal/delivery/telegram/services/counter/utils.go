// internal/delivery/telegram/services/counter/utils.go
package counter

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
