// internal/telegram/menu_utils.go
package telegram

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/telegram"
	"fmt"
	"strings"
	"time"
)

// MenuUtils - утилиты для работы с меню
type MenuUtils struct{}

// NewMenuUtils создает новые утилиты меню
func NewMenuUtils() *MenuUtils {
	return &MenuUtils{}
}

// FormatCompactMenu создает компактное меню
func (mu *MenuUtils) FormatCompactMenu() telegram.ReplyKeyboardMarkup {
	return telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.ReplyKeyboardButton{
			{
				{Text: "⚙️ Настройки"},
				{Text: "📊 Статус"},
			},
			{
				{Text: "🔔 Уведомления"},
				{Text: "📋 Помощь"},
			},
			{
				{Text: "📈 Рост/Падение"},
				{Text: "⏱️ Период"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// FormatSettingsMenu создает меню настроек
func (mu *MenuUtils) FormatSettingsMenu() telegram.ReplyKeyboardMarkup {
	return telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.ReplyKeyboardButton{
			{
				{Text: "🔔 Вкл/Выкл"},
				{Text: "📈 Тип сигналов"},
			},
			{
				{Text: "⏱️ Период"},
				{Text: "🔄 Сбросить"},
			},
			{
				{Text: "📊 Статус"},
				{Text: "🔙 Главное меню"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
		Selective:       false,
		IsPersistent:    true,
	}
}

// FormatNotificationKeyboard создает клавиатуру для уведомлений
func (mu *MenuUtils) FormatNotificationKeyboard(signal analysis.GrowthSignal) *telegram.InlineKeyboardMarkup {
	chartURL := fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", signal.Symbol)
	tradeURL := fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", signal.Symbol)

	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text: "📈 График",
					URL:  chartURL,
				},
				{
					Text: "💱 Торговать",
					URL:  tradeURL,
				},
			},
		},
	}
}

// FormatSignalMessage форматирует сообщение сигнала для компактного отображения
func (mu *MenuUtils) FormatSignalMessage(signal analysis.GrowthSignal, format string) string {
	var icon, directionStr, changeStr string
	changePercent := signal.GrowthPercent + signal.FallPercent

	if signal.Direction == "growth" {
		icon = "🟢"
		directionStr = "📈 РОСТ"
		changeStr = fmt.Sprintf("+%.2f%%", changePercent)
	} else {
		icon = "🔴"
		directionStr = "📉 ПАДЕНИЕ"
		changeStr = fmt.Sprintf("-%.2f%%", -changePercent)
	}

	timeStr := signal.Timestamp.Format("15:04:05")

	switch format {
	case "compact":
		return fmt.Sprintf(
			"%s *%s*\n"+
				"%s %s: %s\n"+
				"🕐 %s",
			icon, signal.Symbol,
			directionStr, changeStr,
			timeStr,
		)
	case "full":
		return fmt.Sprintf(
			"%s *%s*\n"+
				"%s %s\n"+
				"🕐 %s\n"+
				"⏱️ %d мин\n"+
				"📊 Объем: $%.0f",
			icon, signal.Symbol,
			directionStr, changeStr,
			timeStr,
			signal.PeriodMinutes,
			signal.Volume24h,
		)
	default:
		return fmt.Sprintf(
			"%s *%s*\n"+
				"%s: %s\n"+
				"🕐 %s",
			icon, signal.Symbol,
			directionStr, changeStr,
			timeStr,
		)
	}
}

// FormatCounterMessage форматирует сообщение счетчика
func (mu *MenuUtils) FormatCounterMessage(symbol string, signalType string, count int, maxSignals int, period string) string {
	icon := "🟢"
	directionStr := "РОСТ"
	if signalType == "fall" {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
	}

	percentage := float64(count) / float64(maxSignals) * 100
	timeStr := time.Now().Format("15:04:05")

	return fmt.Sprintf(
		"📊 *Счетчик сигналов*\n"+
			"%s %s\n"+
			"Символ: %s\n"+
			"Текущее: %d/%d (%.0f%%)\n"+
			"Период: %s\n"+
			"🕐 %s",
		icon, directionStr,
		symbol,
		count, maxSignals, percentage,
		period,
		timeStr,
	)
}

// GetPeriodName возвращает человекочитаемое название периода
func (mu *MenuUtils) GetPeriodName(period string) string {
	periodMap := map[string]string{
		"5m":  "5 минут",
		"15m": "15 минут",
		"30m": "30 минут",
		"1h":  "1 час",
		"4h":  "4 часа",
		"1d":  "1 день",
	}

	if name, exists := periodMap[period]; exists {
		return name
	}
	return "15 минут"
}

// ParseCallbackData парсит callback данные
func (mu *MenuUtils) ParseCallbackData(callbackData string) (action string, params []string) {
	parts := strings.Split(callbackData, "_")
	if len(parts) > 0 {
		action = parts[0]
		if len(parts) > 1 {
			params = parts[1:]
		}
	}
	return action, params
}

// IsValidPeriod проверяет валидность периода
func (mu *MenuUtils) IsValidPeriod(period string) bool {
	validPeriods := map[string]bool{
		"5m":  true,
		"15m": true,
		"30m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
	}
	return validPeriods[period]
}

// CalculateMaxButtons рассчитывает максимальное количество кнопок для меню
func (mu *MenuUtils) CalculateMaxButtons(screenWidth int) (int, int) {
	// Стандартные настройки для Telegram
	// 2 колонки обычно хорошо вписываются без скролла
	maxColumns := 2
	maxRows := 4 // 8 кнопок всего

	// Если ширина экрана большая, можно больше колонок
	if screenWidth > 400 {
		maxColumns = 3
		maxRows = 3 // 9 кнопок всего
	}

	return maxColumns, maxRows
}

// CreateNotificationMenu создает меню для уведомлений
func (mu *MenuUtils) CreateNotificationMenu() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "✅ Включить", CallbackData: "notify_on"},
				{Text: "❌ Выключить", CallbackData: "notify_off"},
			},
			{
				{Text: "📈 Только рост", CallbackData: "notify_growth"},
				{Text: "📉 Только падение", CallbackData: "notify_fall"},
			},
		},
	}
}

// CreatePeriodMenu создает меню периодов
func (mu *MenuUtils) CreatePeriodMenu() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "5 мин", CallbackData: "period_5m"},
				{Text: "15 мин", CallbackData: "period_15m"},
			},
			{
				{Text: "30 мин", CallbackData: "period_30m"},
				{Text: "1 час", CallbackData: "period_1h"},
			},
			{
				{Text: "4 часа", CallbackData: "period_4h"},
				{Text: "🔙 Назад", CallbackData: "back_to_menu"},
			},
		},
	}
}
