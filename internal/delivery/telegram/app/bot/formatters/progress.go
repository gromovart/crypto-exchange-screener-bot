// internal/delivery/telegram/app/bot/formatters/progress.go
package formatters

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ProgressFormatter отвечает ТОЛЬКО за ОТОБРАЖЕНИЕ прогресса
// ВСЯ логика расчета находится в counter/progress/progress.go
type ProgressFormatter struct{}

// NewProgressFormatter создает новый форматтер прогресса
func NewProgressFormatter() *ProgressFormatter {
	return &ProgressFormatter{}
}

// FormatConfirmationProgressWithGroups форматирует прогресс с готовыми данными групп
// ТОЛЬКО ОТОБРАЖЕНИЕ, БЕЗ РАСЧЕТОВ - все расчеты в counter/progress/progress.go
func (f *ProgressFormatter) FormatConfirmationProgressWithGroups(
	confirmations int,
	required int,
	filledGroups int,
	totalGroups int,
	period string,
	nextAnalysis, nextSignal time.Time,
) string {
	// Валидация входных данных
	if totalGroups <= 0 {
		totalGroups = 5 // дефолтное значение
	}
	if filledGroups < 0 {
		filledGroups = 0
	}
	if filledGroups > totalGroups {
		filledGroups = totalGroups
	}
	// Валидация
	if required <= 0 {
		required = 6
	}
	// Процент заполнения групп
	percentage := math.Min(float64(confirmations)/float64(required), 1.0) * 100

	// Создаем прогресс-бар
	progressBar := f.createGroupedProgressBar(filledGroups, totalGroups, percentage)

	// Форматируем время
	timeUntilNextAnalysis := f.formatTimeUntil(nextAnalysis)
	timeUntilNextSignal := f.formatTimeUntil(nextSignal)

	// Информация о группировке (только для длинных периодов)
	groupInfo := f.formatGroupInfo(period, filledGroups, totalGroups)

	return fmt.Sprintf("📡 Подтверждений: %d/%d %s (%.0f%%)\n%s🕐 Следующий анализ: %s\n⏰ Следующий сигнал: %s",
		confirmations, required, progressBar, percentage,
		groupInfo, timeUntilNextAnalysis, timeUntilNextSignal)
}

// createGroupedProgressBar создает прогресс-бар с группами (ТОЛЬКО ОТОБРАЖЕНИЕ)
func (f *ProgressFormatter) createGroupedProgressBar(filledGroups, totalGroups int, percentage float64) string {
	var progressBar strings.Builder

	for i := 0; i < totalGroups; i++ {
		if i < filledGroups {
			// Цвет заполненной группы зависит от процента заполнения
			if percentage >= 80 {
				progressBar.WriteString("🟢") // Большинство групп заполнено - зеленый
			} else if percentage >= 50 {
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

// formatGroupInfo форматирует информацию о группах (только отображение)
func (f *ProgressFormatter) formatGroupInfo(period string, filledGroups, totalGroups int) string {
	if totalGroups <= 5 {
		// Для коротких периодов не показываем детали
		return ""
	}

	// Справочная информация о минутах на группу
	minutesPerGroup := 0
	switch period {
	case "30m":
		minutesPerGroup = 5
	case "1h":
		minutesPerGroup = 10
	case "4h":
		minutesPerGroup = 30
	case "1d":
		minutesPerGroup = 120
	}

	if minutesPerGroup > 0 {
		return fmt.Sprintf("🏷️ Группы: %d/%d (по %d минут)\n", filledGroups, totalGroups, minutesPerGroup)
	}

	return fmt.Sprintf("🏷️ Группы: %d/%d\n", filledGroups, totalGroups)
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
	minutes := int(duration.Minutes())

	// Округляем минуты вверх
	if duration.Seconds() > float64(minutes*60) {
		minutes++
	}

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
