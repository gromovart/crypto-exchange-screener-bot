// internal/core/domain/signals/detectors/counter/progress/progress.go
package progress

import (
	"crypto-exchange-screener-bot/pkg/logger"
	"math"
	"time"
)

// ProgressData содержит данные для отображения прогресса подтверждений
type ProgressData struct {
	Confirmations         int     `json:"confirmations"`          // текущее количество подтверждений
	RequiredConfirmations int     `json:"required_confirmations"` // сколько нужно подтверждений
	FilledGroups          int     `json:"filled_groups"`          // заполненные группы (кружки)
	TotalGroups           int     `json:"total_groups"`           // всего групп (кружков)
	Percentage            float64 `json:"percentage"`             // процент заполнения
	NextAnalysisMinutes   int     `json:"next_analysis_minutes"`  // минут до следующего анализа
	NextSignalMinutes     int     `json:"next_signal_minutes"`    // минут до следующего сигнала
}

// Константы для новой логики
const (
	VisualTargetConfirmations = 6 // Визуальная цель = 6 подтверждений (100%)
	SignalInterval            = 3 // Сигнал каждые 3 подтверждения
)

// NewProgressData создает новые данные прогресса для сигнала
func NewProgressData(confirmations, requiredConfirmations int, period string, timestamp time.Time) ProgressData {
	if requiredConfirmations == 0 {
		return ProgressData{}
	}

	// Получаем общее количество групп для периода
	totalGroups := getTotalGroups(period)

	// НОВАЯ ЛОГИКА: Рассчитываем заполненные группы по нормализации к 6 подтверждениям
	filledGroups := calculateNormalizedGroups(confirmations, totalGroups)

	// Рассчитываем процент относительно визуальной цели (6)
	percentage := calculateNormalizedPercentage(confirmations)

	// Рассчитываем время до следующего анализа и сигнала
	nextAnalysis, nextSignal := calculateNextTimes(confirmations, period, timestamp)

	logger.Warn("🧮 Progress расчет для %s: подтверждений %d, групп %d/%d, процент %.0f%%",
		period, confirmations, filledGroups, totalGroups, percentage)

	return ProgressData{
		Confirmations:         confirmations,
		RequiredConfirmations: requiredConfirmations,
		FilledGroups:          filledGroups,
		TotalGroups:           totalGroups,
		Percentage:            percentage,
		NextAnalysisMinutes:   nextAnalysis,
		NextSignalMinutes:     nextSignal,
	}
}

// getTotalGroups возвращает общее количество групп для периода
func getTotalGroups(period string) int {
	switch period {
	case "5m", "15m":
		return 5
	case "30m", "1h":
		return 6
	case "4h":
		return 8
	case "1d":
		return 12
	default:
		return 5
	}
}

// calculateNormalizedGroups рассчитывает заполненные группы по нормализации к 6 подтверждениям
func calculateNormalizedGroups(confirmations, totalGroups int) int {
	if confirmations <= 0 {
		return 0
	}

	// Ограничиваем подтверждения визуальной целью (6)
	normalizedConfirmations := float64(minInt(confirmations, VisualTargetConfirmations))

	// Рассчитываем прогресс: подтверждения / 6
	progressRatio := normalizedConfirmations / float64(VisualTargetConfirmations)

	// Количество заполненных групп = progressRatio * totalGroups (МАТЕМАТИЧЕСКОЕ ОКРУГЛЕНИЕ)
	filledGroups := int(math.Round(progressRatio * float64(totalGroups)))

	// Минимум 1 кружок, если есть подтверждения (кроме 0)
	if filledGroups == 0 && confirmations > 0 {
		filledGroups = 1
	}

	// Ограничиваем максимумом групп
	if filledGroups > totalGroups {
		filledGroups = totalGroups
	}

	// Не может быть отрицательным
	if filledGroups < 0 {
		filledGroups = 0
	}

	return filledGroups
}

// calculateNormalizedPercentage рассчитывает процент относительно визуальной цели (6)
func calculateNormalizedPercentage(confirmations int) float64 {
	// Процент = min(подтверждения, 6) / 6 * 100
	normalizedConfirmations := float64(minInt(confirmations, VisualTargetConfirmations))
	percentage := (normalizedConfirmations / float64(VisualTargetConfirmations)) * 100.0

	// Ограничиваем 100%
	if percentage > 100.0 {
		percentage = 100.0
	}

	return math.Round(percentage*10) / 10 // Округляем до 1 десятичного знака
}

// calculateNextTimes рассчитывает время до следующего анализа и сигнала
func calculateNextTimes(confirmations int, period string, timestamp time.Time) (nextAnalysis, nextSignal int) {
	// Следующий анализ всегда через 1 минуту
	nextAnalysis = 1

	// Следующий сигнал: считаем до следующего кратного 3
	if confirmations <= 0 {
		nextSignal = SignalInterval
	} else {
		// До следующего сигнала осталось: (3 - (confirmations % 3)) % 3
		// Но если confirmations кратно 3, то следующий через 3
		remainingToNextSignal := SignalInterval - (confirmations % SignalInterval)
		if remainingToNextSignal == 0 {
			remainingToNextSignal = SignalInterval
		}
		nextSignal = remainingToNextSignal
	}

	return nextAnalysis, nextSignal
}

// getPeriodMinutes возвращает период в минутах
func getPeriodMinutes(period string) int {
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

// GetProgressBar создает строку прогресс-бара
func (p *ProgressData) GetProgressBar() string {
	if p.TotalGroups == 0 {
		return ""
	}

	var bar string
	for i := 0; i < p.TotalGroups; i++ {
		if i < p.FilledGroups {
			// Цвет зависит от процента заполнения (силы тренда)
			if p.Percentage >= 80 {
				bar += "🟢" // Сильный тренд (80-100%)
			} else if p.Percentage >= 50 {
				bar += "🟡" // Подтвержденный тренд (50-79%)
			} else {
				bar += "🔴" // Формирующийся тренд (0-49%)
			}
		} else {
			bar += "▫️"
		}
	}
	return bar
}

// ShouldSendSignal проверяет, нужно ли отправлять сигнал (каждое 3-е подтверждение)
func ShouldSendSignal(confirmations int) bool {
	return confirmations > 0 && confirmations%SignalInterval == 0
}

// minInt возвращает минимальное из двух целых чисел
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
