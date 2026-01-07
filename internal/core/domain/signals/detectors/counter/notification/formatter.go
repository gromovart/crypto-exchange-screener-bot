// internal/core/domain/signals/detectors/counter/notification/formatter.go
package notification

import (
	"fmt"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram"
)

// NotificationFormatter - форматирование уведомлений для счетчика
type NotificationFormatter struct {
	exchange string
}

// NewNotificationFormatter создает новый форматтер уведомлений
func NewNotificationFormatter(exchange string) *NotificationFormatter {
	return &NotificationFormatter{
		exchange: strings.ToUpper(exchange),
	}
}

// CounterNotificationData - данные для уведомления счетчика
type CounterNotificationData struct {
	Symbol             string
	Direction          string
	Change             float64
	SignalCount        int
	MaxSignals         int
	CurrentPrice       float64
	Volume24h          float64
	OpenInterest       float64
	OIChange24h        float64
	FundingRate        float64
	AverageFunding     float64
	NextFundingTime    time.Time
	Period             string
	LiquidationVolume  float64
	LongLiqVolume      float64
	ShortLiqVolume     float64
	VolumeDelta        float64
	VolumeDeltaPercent float64
	RSI                float64
	MACDSignal         float64
	DeltaSource        string
}

// FormatCounterNotification форматирует уведомление счетчика
func (f *NotificationFormatter) FormatCounterNotification(data CounterNotificationData) string {
	formatter := telegram.NewMarketMessageFormatter(f.exchange)

	params := &telegram.MessageParams{
		Symbol:             data.Symbol,
		Direction:          data.Direction,
		Change:             data.Change,
		SignalCount:        data.SignalCount,
		MaxSignals:         data.MaxSignals,
		CurrentPrice:       data.CurrentPrice,
		Volume24h:          data.Volume24h,
		OpenInterest:       data.OpenInterest,
		OIChange24h:        data.OIChange24h,
		FundingRate:        data.FundingRate,
		AverageFunding:     data.AverageFunding,
		NextFundingTime:    data.NextFundingTime,
		Period:             data.Period,
		LiquidationVolume:  data.LiquidationVolume,
		LongLiqVolume:      data.LongLiqVolume,
		ShortLiqVolume:     data.ShortLiqVolume,
		VolumeDelta:        data.VolumeDelta,
		VolumeDeltaPercent: data.VolumeDeltaPercent,
		RSI:                data.RSI,
		MACDSignal:         data.MACDSignal,
		DeltaSource:        data.DeltaSource,
	}

	return formatter.FormatMessage(params)
}

// FormatCompactNotification форматирует компактное уведомление
func (f *NotificationFormatter) FormatCompactNotification(data CounterNotificationData) string {
	directionIcon := "🟢"
	directionText := "РОСТ"
	changePrefix := "+"

	if data.Direction == "fall" {
		directionIcon = "🔴"
		directionText = "ПАДЕНИЕ"
		changePrefix = "-"
	}

	timeframe := f.extractTimeframe(data.Period)
	percentage := float64(data.SignalCount) / float64(data.MaxSignals) * 100

	return fmt.Sprintf(
		"%s %s • %s\n"+
			"%s %s: %s%.2f%%\n"+
			"💰 $%.2f | 📊 %d/%d (%.0f%%)",
		f.exchange, timeframe, data.Symbol,
		directionIcon, directionText, changePrefix, data.Change,
		data.CurrentPrice, data.SignalCount, data.MaxSignals, percentage,
	)
}

// FormatWithKeyboard форматирует уведомление с клавиатурой
func (f *NotificationFormatter) FormatWithKeyboard(
	data CounterNotificationData,
	chartProvider string,
) (string, *telegram.InlineKeyboardMarkup) {
	message := f.FormatCounterNotification(data)
	keyboard := f.createNotificationKeyboard(data.Symbol, chartProvider, data.Period)

	return message, keyboard
}

// FormatTestMessage форматирует тестовое сообщение
func (f *NotificationFormatter) FormatTestMessage(symbol string) string {
	return fmt.Sprintf(
		"🧪 Тестовое уведомление счетчика\n"+
			"🏷️  %s • 15мин\n"+
			"📛 %s\n"+
			"🕐 %s\n\n"+
			"✅ Счетчик работает корректно\n"+
			"📊 Уведомления настроены",
		f.exchange, symbol, time.Now().Format("15:04:05"),
	)
}

// CreateNotificationKeyboard создает клавиатуру для уведомления
func (f *NotificationFormatter) createNotificationKeyboard(
	symbol string,
	chartProvider string,
	period string,
) *telegram.InlineKeyboardMarkup {
	periodMinutes := f.extractMinutesFromPeriod(period)
	buttonBuilder := telegram.NewButtonURLBuilderWithProvider(f.exchange, chartProvider)

	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				buttonBuilder.GetChartButton(symbol),
				buttonBuilder.GetTradeButton(symbol, periodMinutes),
			},
		},
	}
}

// FormatErrorNotification форматирует уведомление об ошибке
func (f *NotificationFormatter) FormatErrorNotification(symbol, errorMsg string) string {
	return fmt.Sprintf(
		"❌ Ошибка счетчика\n"+
			"🏷️  %s\n"+
			"📛 %s\n"+
			"🕐 %s\n\n"+
			"Ошибка: %s\n"+
			"Проверьте настройки счетчика",
		f.exchange, symbol, time.Now().Format("15:04:05"), errorMsg,
	)
}

// FormatStatsNotification форматирует уведомление со статистикой
func (f *NotificationFormatter) FormatStatsNotification(
	symbol string,
	growthCount, fallCount, totalSignals int,
	periodStart, periodEnd time.Time,
) string {
	remainingTime := time.Until(periodEnd).Round(time.Minute)
	periodProgress := time.Since(periodStart).Seconds() / periodEnd.Sub(periodStart).Seconds() * 100

	return fmt.Sprintf(
		"📊 Статистика счетчика\n"+
			"🏷️  %s\n"+
			"📛 %s\n"+
			"🕐 %s\n\n"+
			"📈 Рост: %d\n"+
			"📉 Падение: %d\n"+
			"📡 Всего: %d\n\n"+
			"⏳ Прогресс периода: %.0f%%\n"+
			"⏰ До сброса: %v",
		f.exchange, symbol, time.Now().Format("15:04:05"),
		growthCount, fallCount, totalSignals,
		periodProgress, remainingTime,
	)
}

// extractTimeframe извлекает таймфрейм из периода
func (f *NotificationFormatter) extractTimeframe(period string) string {
	switch {
	case strings.Contains(period, "5"):
		return "5мин"
	case strings.Contains(period, "15"):
		return "15мин"
	case strings.Contains(period, "30"):
		return "30мин"
	case strings.Contains(period, "1 час"):
		return "1ч"
	case strings.Contains(period, "4"):
		return "4ч"
	case strings.Contains(period, "1 день"):
		return "1д"
	default:
		return "1мин"
	}
}

// extractMinutesFromPeriod извлекает минуты из периода
func (f *NotificationFormatter) extractMinutesFromPeriod(period string) int {
	switch {
	case strings.Contains(period, "5"):
		return 5
	case strings.Contains(period, "15"):
		return 15
	case strings.Contains(period, "30"):
		return 30
	case strings.Contains(period, "1 час"):
		return 60
	case strings.Contains(period, "4"):
		return 240
	case strings.Contains(period, "1 день"):
		return 1440
	default:
		return 15
	}
}
