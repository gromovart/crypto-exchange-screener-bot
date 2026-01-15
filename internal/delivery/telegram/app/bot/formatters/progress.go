// internal/delivery/telegram/app/bot/formatters/progress.go
package formatters

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ProgressFormatter отвечает за форматирование прогресс-баров с группировкой
type ProgressFormatter struct{}

// NewProgressFormatter создает новый форматтер прогресса
func NewProgressFormatter() *ProgressFormatter {
	return &ProgressFormatter{}
}

// FormatProgressBlock форматирует блок прогресса сигналов (старый формат, для обратной совместимости)
func (f *ProgressFormatter) FormatProgressBlock(
	signalCount int,
	maxSignals int,
	period string,
) string {
	percentage := float64(signalCount) / float64(maxSignals) * 100
	progressBar := f.formatOldProgressBar(percentage)

	return fmt.Sprintf("📡 %d/%d %s (%.0f%%)\n🕐 Период: %s\n\n",
		signalCount, maxSignals, progressBar, percentage, period)
}

// FormatConfirmationProgress форматирует прогресс подтверждений с группировкой
func (f *ProgressFormatter) FormatConfirmationProgress(
	confirmations int,
	required int,
	period string,
	nextAnalysis, nextSignal time.Time,
) string {
	// Получаем информацию о группировке для периода
	totalGroups, minutesPerGroup := f.getGroupingInfo(period)
	filledGroups := f.calculateFilledGroups(confirmations, required, totalGroups)

	// ИЗМЕНЕНИЕ: Процент = процент заполненных групп, а не подтверждений
	groupPercentage := float64(filledGroups) / float64(totalGroups) * 100

	// Создаем прогресс-бар с группами
	progressBar := f.createGroupedProgressBar(filledGroups, totalGroups, groupPercentage)

	// Форматируем время
	timeUntilNextAnalysis := f.formatTimeUntil(nextAnalysis)
	timeUntilNextSignal := f.formatTimeUntil(nextSignal)

	// Добавляем информацию о группировке
	groupInfo := f.formatGroupInfo(period, filledGroups, totalGroups, minutesPerGroup)

	return fmt.Sprintf("📡 Подтверждений: %d/%d %s (%.0f%%)\n%s🕐 Следующий анализ: %s\n⏰ Следующий сигнал: %s",
		confirmations, required, progressBar, groupPercentage,
		groupInfo, timeUntilNextAnalysis, timeUntilNextSignal)
}

// getGroupingInfo возвращает информацию о группировке для периода
func (f *ProgressFormatter) getGroupingInfo(period string) (totalGroups int, minutesPerGroup int) {
	switch period {
	case "5m":
		return 5, 1 // 5 групп по 1 минуте
	case "15m":
		return 5, 3 // 5 групп по 3 минуты
	case "30m":
		return 6, 5 // 6 групп по 5 минут
	case "1h":
		return 6, 10 // 6 групп по 10 минут
	case "4h":
		return 8, 30 // 8 групп по 30 минут
	case "1d":
		return 12, 120 // 12 групп по 2 часа
	default:
		return 5, 1
	}
}

// calculateFilledGroups рассчитывает заполненные группы для прогресс-бара
func (f *ProgressFormatter) calculateFilledGroups(confirmations, required, totalGroups int) int {
	if required == 0 {
		return 0
	}

	// Если все требуемые подтверждения получены - все группы заполнены
	if confirmations >= required {
		return totalGroups
	}

	// Если не все подтверждения, заполняем пропорционально
	ratio := float64(confirmations) / float64(required)
	filledGroups := int(float64(totalGroups) * ratio)

	// Минимум 1 группа если есть хотя бы 1 подтверждение
	if filledGroups == 0 && confirmations > 0 {
		filledGroups = 1
	}

	if filledGroups > totalGroups {
		filledGroups = totalGroups
	}

	return filledGroups
}

// createGroupedProgressBar создает прогресс-бар с группами
func (f *ProgressFormatter) createGroupedProgressBar(filledGroups, totalGroups int, groupPercentage float64) string {
	var progressBar strings.Builder

	for i := 0; i < totalGroups; i++ {
		if i < filledGroups {
			// Цвет заполненной группы зависит от процента заполнения групп
			if groupPercentage >= 80 {
				progressBar.WriteString("🟢") // Большинство групп заполнено - зеленый
			} else if groupPercentage >= 50 {
				progressBar.WriteString("🟡") // Половина групп заполнена - желтый
			} else {
				progressBar.WriteString("🔴") // Меньшинство групп заполнено - красный
			}
		} else {
			progressBar.WriteString("▫️")
		}
	}

	return progressBar.String()
}

// formatGroupInfo форматирует информацию о группах
func (f *ProgressFormatter) formatGroupInfo(period string, filledGroups, totalGroups, minutesPerGroup int) string {
	if totalGroups <= 5 {
		// Для коротких периодов не показываем детали
		return ""
	}

	groupInfo := fmt.Sprintf("🏷️ Группы: %d/%d (по %d минут)\n", filledGroups, totalGroups, minutesPerGroup)

	// Добавляем детали для каждого периода
	switch period {
	case "1h":
		groupInfo += "⏱️ Каждая группа = 10 минут анализа\n"
	case "4h":
		groupInfo += "⏱️ Каждая группа = 30 минут анализа\n"
	case "1d":
		groupInfo += "⏱️ Каждая группа = 2 часа анализа\n"
	}

	return groupInfo
}

// formatTimeUntil форматирует время до события
func (f *ProgressFormatter) formatTimeUntil(t time.Time) string {
	if t.IsZero() {
		return "─"
	}

	now := time.Now()
	if t.Before(now) {
		return "сейчас"
	}

	duration := t.Sub(now)

	// Округляем минуты вверх
	minutes := int(math.Ceil(duration.Minutes()))

	// Минимум 1 минута
	if minutes <= 0 {
		minutes = 1
	}

	return fmt.Sprintf("%dм", minutes)
}

// formatOldProgressBar создает компактный прогресс-бар для старого формата
func (f *ProgressFormatter) formatOldProgressBar(percentage float64) string {
	// Рассчитываем количество заполненных баров
	bars := int(percentage / 20) // 5 баров по 20% каждый
	if bars > 5 {
		bars = 5
	}
	if bars < 0 {
		bars = 0
	}

	// Строим прогресс-бар с цветами в зависимости от заполнения
	var result string
	for i := 0; i < 5; i++ {
		if i < bars {
			// Цвет баров меняется в зависимости от уровня заполнения
			switch {
			case percentage >= 80:
				result += "🔴" // Высокое заполнение - красный
			case percentage >= 50:
				result += "🟡" // Среднее заполнение - желтый
			default:
				result += "🟢" // Низкое заполнение - зеленый
			}
		} else {
			result += "▫️"
		}
	}
	return result
}
