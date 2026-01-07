// internal/delivery/telegram/menu_utils.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"strings"
	"time"
)

// MenuUtils - утилиты для работы с меню
type MenuUtils struct {
	buttonBuilder *ButtonURLBuilder
	exchange      string
}

// NewMenuUtils создает новые утилиты меню
func NewMenuUtils(exchange string) *MenuUtils {
	return &MenuUtils{
		buttonBuilder: NewButtonURLBuilder(exchange),
		exchange:      exchange,
	}
}

// NewDefaultMenuUtils создает утилиты с биржей по умолчанию (Bybit)
func NewDefaultMenuUtils() *MenuUtils {
	return NewMenuUtils("bybit")
}

// FormatCompactMenu создает компактное меню
func (mu *MenuUtils) FormatCompactMenu() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
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
func (mu *MenuUtils) FormatSettingsMenu() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
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
func (mu *MenuUtils) FormatNotificationKeyboard(signal types.GrowthSignal) *InlineKeyboardMarkup {
	// Используем строитель для создания кнопок
	return mu.buttonBuilder.StandardNotificationKeyboard(signal.Symbol, signal.PeriodMinutes)
}

// FormatEnhancedNotificationKeyboard создает расширенную клавиатуру для уведомлений
func (mu *MenuUtils) FormatEnhancedNotificationKeyboard(signal types.GrowthSignal) *InlineKeyboardMarkup {
	// Используем строитель для создания расширенных кнопок
	return mu.buttonBuilder.EnhancedNotificationKeyboard(signal.Symbol, signal.PeriodMinutes)
}

// FormatCounterNotificationKeyboard создает клавиатуру для уведомлений счетчика
func (mu *MenuUtils) FormatCounterNotificationKeyboard(symbol string, periodMinutes int) *InlineKeyboardMarkup {
	// Используем строитель для создания кнопок счетчика
	return mu.buttonBuilder.CounterNotificationKeyboard(symbol, periodMinutes)
}

// FormatCounterMessage форматирует сообщение счетчика в компактном формате
func (mu *MenuUtils) FormatCounterMessage(symbol string, signalType string, count int, maxSignals int, period string) string {
	icon := "🟢"
	directionStr := "РОСТ"
	if signalType == "fall" {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
	}

	// УДАЛЕНА неиспользуемая переменная percentage
	// percentage := float64(count) / float64(maxSignals) * 100
	timeStr := time.Now().Format("2006/01/02 15:04:05")

	return fmt.Sprintf(
		"⚫ %s - 1мин - %s\n"+
			"🕐 %s\n"+
			"%s %s\n"+
			"📡 Сигнал: %d",
		strings.ToUpper(mu.exchange), symbol,
		timeStr,
		icon, directionStr,
		count,
	)
}

// FormatCounterMessageFull форматирует полное сообщение счетчика
func (mu *MenuUtils) FormatCounterMessageFull(symbol string, signalType string, count int, maxSignals int, period string) string {
	icon := "🟢"
	directionStr := "РОСТ"
	if signalType == "fall" {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
	}

	percentage := float64(count) / float64(maxSignals) * 100
	timeStr := time.Now().Format("2006/01/02 15:04:05")

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
func (mu *MenuUtils) CreateNotificationMenu() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
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
func (mu *MenuUtils) CreatePeriodMenu() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
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

// GetChartButton возвращает кнопку "График"
func (mu *MenuUtils) GetChartButton(symbol string) InlineKeyboardButton {
	return mu.buttonBuilder.GetChartButton(symbol)
}

// GetTradeButton возвращает кнопку "Торговать"
func (mu *MenuUtils) GetTradeButton(symbol string, periodMinutes int) InlineKeyboardButton {
	return mu.buttonBuilder.GetTradeButton(symbol, periodMinutes)
}
