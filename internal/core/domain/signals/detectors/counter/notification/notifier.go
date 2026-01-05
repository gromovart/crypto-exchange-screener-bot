// internal/core/domain/signals/detectors/counter/notification/notifier.go
package notification

import (
	"log"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/types"
)

// CounterNotifier - отправитель уведомлений для счетчика
type CounterNotifier struct {
	telegramBot         *telegram.TelegramBot
	marketMetrics       *calculator.MarketMetricsCalculator
	technicalCalculator *calculator.TechnicalCalculator
	volumeCalculator    *calculator.VolumeDeltaCalculator
	enabled             bool

	lastNotificationTime map[string]time.Time
	notificationMu       sync.RWMutex
	minNotificationDelay time.Duration
}

// NewCounterNotifier создает новый отправитель уведомлений
func NewCounterNotifier(
	telegramBot *telegram.TelegramBot,
	marketMetrics *calculator.MarketMetricsCalculator,
	technicalCalculator *calculator.TechnicalCalculator,
	volumeCalculator *calculator.VolumeDeltaCalculator,
) *CounterNotifier {
	return &CounterNotifier{
		telegramBot:          telegramBot,
		marketMetrics:        marketMetrics,
		technicalCalculator:  technicalCalculator,
		volumeCalculator:     volumeCalculator,
		enabled:              true,
		lastNotificationTime: make(map[string]time.Time),
		minNotificationDelay: 30 * time.Second,
	}
}

// SendNotification отправляет уведомление о сигнале счетчика
func (n *CounterNotifier) SendNotification(
	symbol string,
	direction string,
	change float64,
	signalCount int,
	maxSignals int,
	priceData []types.PriceData,
) error {
	if !n.enabled || n.telegramBot == nil {
		return nil
	}

	// Проверяем частоту уведомлений
	if !n.canSendNotification(symbol, direction) {
		log.Printf("⚠️ Пропускаем уведомление для %s: слишком часто", symbol)
		return nil
	}

	// Получаем рыночные метрики
	currentPrice := priceData[len(priceData)-1].Price
	volume24h := priceData[len(priceData)-1].Volume24h
	openInterest := priceData[len(priceData)-1].OpenInterest
	fundingRate := priceData[len(priceData)-1].FundingRate

	// Рассчитываем дополнительные метрики
	oiChange24h := n.marketMetrics.CalculateOIChange24h(symbol)
	averageFunding := n.marketMetrics.CalculateAverageFunding(getFundingRates(priceData))
	nextFundingTime := n.marketMetrics.CalculateNextFundingTime()
	liquidationVolume, longLiqVolume, shortLiqVolume := n.marketMetrics.GetLiquidationData(symbol)

	// Технические индикаторы
	rsi := n.technicalCalculator.CalculateRSI(priceData)
	macdSignal := n.technicalCalculator.CalculateMACD(priceData)

	// Получаем дельту объемов с источником
	var volumeDelta, volumeDeltaPercent float64
	var deltaSource string

	if n.volumeCalculator != nil {
		deltaData := n.volumeCalculator.CalculateWithFallback(symbol, direction)
		if deltaData != nil {
			volumeDelta = deltaData.Delta
			volumeDeltaPercent = deltaData.DeltaPercent
			deltaSource = string(deltaData.Source)
			log.Printf("📊 Дельта для %s: $%.0f (%.1f%%, источник: %s)",
				symbol, volumeDelta, volumeDeltaPercent, deltaSource)
		}
	}

	// Определяем период
	period := n.getPeriodFromSignalCount(signalCount, maxSignals)

	// Форматируем сообщение
	message := n.formatMessage(
		symbol,
		direction,
		change,
		signalCount,
		maxSignals,
		currentPrice,
		volume24h,
		openInterest,
		oiChange24h,
		fundingRate,
		averageFunding,
		nextFundingTime,
		period,
		liquidationVolume,
		longLiqVolume,
		shortLiqVolume,
		volumeDelta,
		volumeDeltaPercent,
		rsi,
		macdSignal,
		deltaSource,
	)

	// Отправляем сообщение
	if err := n.telegramBot.SendMessage(message); err != nil {
		log.Printf("❌ Ошибка отправки уведомления для %s: %v", symbol, err)
		return err
	}

	// Обновляем время последнего уведомления
	n.updateLastNotificationTime(symbol, direction)
	log.Printf("✅ Отправлено уведомление для %s", symbol)
	return nil
}

// canSendNotification проверяет можно ли отправить уведомление
func (n *CounterNotifier) canSendNotification(symbol, direction string) bool {
	n.notificationMu.RLock()
	defer n.notificationMu.RUnlock()

	key := symbol + "_" + direction
	lastTime, exists := n.lastNotificationTime[key]

	if !exists {
		return true
	}

	return time.Since(lastTime) >= n.minNotificationDelay
}

// updateLastNotificationTime обновляет время последнего уведомления
func (n *CounterNotifier) updateLastNotificationTime(symbol, direction string) {
	n.notificationMu.Lock()
	defer n.notificationMu.Unlock()

	key := symbol + "_" + direction
	n.lastNotificationTime[key] = time.Now()
}

// formatMessage форматирует сообщение
func (n *CounterNotifier) formatMessage(
	symbol, direction string,
	change float64,
	signalCount, maxSignals int,
	currentPrice, volume24h, openInterest, oiChange24h float64,
	fundingRate, averageFunding float64,
	nextFundingTime time.Time,
	period string,
	liquidationVolume, longLiqVolume, shortLiqVolume float64,
	volumeDelta, volumeDeltaPercent float64,
	rsi, macdSignal float64,
	deltaSource string,
) string {
	// Используем существующий MarketMessageFormatter
	formatter := telegram.NewMarketMessageFormatter("bybit")

	return formatter.FormatMessage(
		symbol,
		direction,
		change,
		signalCount,
		maxSignals,
		currentPrice,
		volume24h,
		openInterest,
		oiChange24h,
		fundingRate,
		averageFunding,
		nextFundingTime,
		period,
		liquidationVolume,
		longLiqVolume,
		shortLiqVolume,
		volumeDelta,
		volumeDeltaPercent,
		rsi,
		macdSignal,
		deltaSource,
	)
}

// getPeriodFromSignalCount определяет период на основе количества сигналов
func (n *CounterNotifier) getPeriodFromSignalCount(signalCount, maxSignals int) string {
	percentage := float64(signalCount) / float64(maxSignals) * 100

	switch {
	case percentage < 20:
		return "5 минут"
	case percentage < 40:
		return "15 минут"
	case percentage < 60:
		return "30 минут"
	case percentage < 80:
		return "1 час"
	default:
		return "4 часа"
	}
}

// SetEnabled включает/выключает уведомления
func (n *CounterNotifier) SetEnabled(enabled bool) {
	n.enabled = enabled
}

// IsEnabled возвращает статус
func (n *CounterNotifier) IsEnabled() bool {
	return n.enabled
}

// SetMinNotificationDelay устанавливает минимальную задержку между уведомлениями
func (n *CounterNotifier) SetMinNotificationDelay(delay time.Duration) {
	n.minNotificationDelay = delay
}

// ClearNotificationHistory очищает историю уведомлений
func (n *CounterNotifier) ClearNotificationHistory() {
	n.notificationMu.Lock()
	defer n.notificationMu.Unlock()

	n.lastNotificationTime = make(map[string]time.Time)
}

// GetNotificationStats возвращает статистику уведомлений
func (n *CounterNotifier) GetNotificationStats() map[string]interface{} {
	n.notificationMu.RLock()
	defer n.notificationMu.RUnlock()

	stats := make(map[string]interface{})
	stats["enabled"] = n.enabled
	stats["min_delay"] = n.minNotificationDelay.String()
	stats["total_notifications_tracked"] = len(n.lastNotificationTime)

	// Считаем по символам
	symbolCount := make(map[string]int)
	for key := range n.lastNotificationTime {
		if len(key) > 7 && key[len(key)-7:] == "_growth" {
			symbol := key[:len(key)-7]
			symbolCount[symbol]++
		} else if len(key) > 5 && key[len(key)-5:] == "_fall" {
			symbol := key[:len(key)-5]
			symbolCount[symbol]++
		}
	}
	stats["symbols_with_notifications"] = len(symbolCount)

	return stats
}

// SendTestNotification отправляет тестовое уведомление
func (n *CounterNotifier) SendTestNotification(symbol string) error {
	if !n.enabled || n.telegramBot == nil {
		return nil
	}

	testData := []types.PriceData{
		{
			Symbol:       symbol,
			Price:        100.0,
			Volume24h:    1000000.0,
			OpenInterest: 500000.0,
			FundingRate:  0.0005,
			Timestamp:    time.Now(),
		},
	}

	return n.SendNotification(
		symbol,
		"growth",
		2.5,
		1,
		5,
		testData,
	)
}

// Вспомогательные функции
func getFundingRates(priceData []types.PriceData) []float64 {
	var rates []float64
	for _, data := range priceData {
		if data.FundingRate != 0 {
			rates = append(rates, data.FundingRate)
		}
	}
	return rates
}
