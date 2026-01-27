// internal/core/domain/signals/detectors/counter/utils.go
package counter

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/pkg/logger"
	periodPkg "crypto-exchange-screener-bot/pkg/period"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SafeGetFloat безопасно получает float64 из конфига
func SafeGetFloat(config map[string]interface{}, key string, defaultValue float64) float64 {
	if config == nil {
		return defaultValue
	}

	value, exists := config[key]
	if !exists {
		return defaultValue
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
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			return floatVal
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// SafeGetInt безопасно получает int из конфига
func SafeGetInt(config map[string]interface{}, key string, defaultValue int) int {
	if config == nil {
		return defaultValue
	}

	value, exists := config[key]
	if !exists {
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
		if intVal, err := strconv.Atoi(v); err == nil {
			return intVal
		}
		return defaultValue
	default:
		return defaultValue
	}
}

// SafeGetBool безопасно получает bool из конфига
func SafeGetBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if config == nil {
		return defaultValue
	}

	value, exists := config[key]
	if !exists {
		return defaultValue
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		lower := strings.ToLower(v)
		return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
	case int:
		return v == 1
	case int64:
		return v == 1
	case float64:
		return v == 1.0
	case float32:
		return v == 1.0
	default:
		return defaultValue
	}
}

// SafeGetString безопасно получает string из конфига
func SafeGetString(config map[string]interface{}, key string, defaultValue string) string {
	if config == nil {
		return defaultValue
	}

	value, exists := config[key]
	if !exists {
		return defaultValue
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int64, float32, float64:
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	default:
		return defaultValue
	}
}

// GetGrowthThreshold получает порог роста
func GetGrowthThreshold(config common.AnalyzerConfig) float64 {
	return SafeGetFloat(config.CustomSettings, "growth_threshold", 0.1)
}

// GetFallThreshold получает порог падения
func GetFallThreshold(config common.AnalyzerConfig) float64 {
	return SafeGetFloat(config.CustomSettings, "fall_threshold", 0.1)
}

// GetBasePeriodMinutes получает базовый период в минутах
func GetBasePeriodMinutes(config common.AnalyzerConfig) int {
	return SafeGetInt(config.CustomSettings, "base_period_minutes", 1)
}

// GetAnalysisPeriod получает период анализа
func GetAnalysisPeriod(config common.AnalyzerConfig) string {
	return SafeGetString(config.CustomSettings, "analysis_period", "15m")
}

// ShouldTrackGrowth проверяет нужно ли отслеживать рост
func ShouldTrackGrowth(config common.AnalyzerConfig) bool {
	return SafeGetBool(config.CustomSettings, "track_growth", true)
}

// ShouldTrackFall проверяет нужно ли отслеживать падение
func ShouldTrackFall(config common.AnalyzerConfig) bool {
	return SafeGetBool(config.CustomSettings, "track_fall", true)
}

// ShouldNotifyOnSignal проверяет нужно ли отправлять уведомления
func ShouldNotifyOnSignal(config common.AnalyzerConfig) bool {
	return SafeGetBool(config.CustomSettings, "notify_on_signal", true)
}

// GetChartProvider получает провайдера графиков
func GetChartProvider(config common.AnalyzerConfig) string {
	return SafeGetString(config.CustomSettings, "chart_provider", "coinglass")
}

// FormatDuration форматирует длительность
func FormatDuration(d time.Duration) string {
	if d.Hours() >= 1 {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dч %dм", hours, minutes)
		}
		return fmt.Sprintf("%dч", hours)
	}

	minutes := int(d.Minutes())
	if minutes <= 0 {
		return "менее минуты"
	}
	return fmt.Sprintf("%dм", minutes)
}

// FormatPercentage форматирует процент
func FormatPercentage(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}

// FormatPrice форматирует цену
func FormatPrice(price float64) string {
	if price >= 1000 {
		return fmt.Sprintf("%.0f", price)
	} else if price >= 100 {
		return fmt.Sprintf("%.1f", price)
	} else if price >= 10 {
		return fmt.Sprintf("%.2f", price)
	} else if price >= 1 {
		return fmt.Sprintf("%.3f", price)
	} else if price >= 0.1 {
		return fmt.Sprintf("%.4f", price)
	}
	return fmt.Sprintf("%.5f", price)
}

// GetDirectionEmoji возвращает эмодзи для направления
func GetDirectionEmoji(direction string) string {
	if direction == "growth" {
		return "🟢"
	}
	return "🔴"
}

// GetDirectionText возвращает текст для направления
func GetDirectionText(direction string) string {
	if direction == "growth" {
		return "РОСТ"
	}
	return "ПАДЕНИЕ"
}

// ========== ДОПОЛНИТЕЛЬНЫЕ УТИЛИТЫ ДЛЯ PERIOD MANAGER ==========

// GetRequiredPointsForPeriod возвращает необходимое количество точек для периода
func GetRequiredPointsForPeriod(period string) int {
	switch period {
	case "5m":
		return 6
	case "15m":
		return 10
	case "30m":
		return 15
	case "1h":
		return 20
	case "4h":
		return 25
	case "1d":
		return 30
	default:
		return 15
	}
}

// IsValidPeriod проверяет валидность периода
func IsValidPeriod(period string) bool {
	switch period {
	case "5m", "15m", "30m", "1h", "4h", "1d":
		return true
	default:
		return false
	}
}

// PeriodToMinutes конвертирует период в минуты
func PeriodToMinutes(period string) int {
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
		return 15
	}
}

// PeriodToDuration конвертирует период в time.Duration
func PeriodToDuration(period string) time.Duration {
	return time.Duration(PeriodToMinutes(period)) * time.Minute
}

// FormatPeriod форматирует период для отображения
func FormatPeriod(period string) string {
	switch period {
	case "5m":
		return "5 минут"
	case "15m":
		return "15 минут"
	case "30m":
		return "30 минут"
	case "1h":
		return "1 час"
	case "4h":
		return "4 часа"
	case "1d":
		return "1 день"
	default:
		return period
	}
}

// GetPeriodMinutes конвертирует строковый период в минуты
func GetPeriodMinutes(period string) int {
	minutes, err := periodPkg.StringToMinutes(period)
	if err != nil {
		logger.Warn("⚠️ Невалидный период '%s', используем дефолтный %s",
			period, periodPkg.DefaultPeriod)
		return periodPkg.DefaultMinutes
	}
	return minutes
}
