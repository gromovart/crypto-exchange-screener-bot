// internal/analysis/analyzers/counter_analyzer.go
package analyzers

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/storage"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"sync"
	"time"
)

// CounterAnalyzer - анализатор счетчика сигналов
type CounterAnalyzer struct {
	config              AnalyzerConfig
	stats               AnalyzerStats
	storage             storage.PriceStorage
	telegramBot         *telegram.TelegramBot
	counters            map[string]*internalCounter // Используем внутреннюю структуру
	mu                  sync.RWMutex
	notificationEnabled bool
	chartProvider       string
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(config AnalyzerConfig, storage storage.PriceStorage, tgBot *telegram.TelegramBot) *CounterAnalyzer {
	return &CounterAnalyzer{
		config:              config,
		storage:             storage,
		telegramBot:         tgBot,
		counters:            make(map[string]*internalCounter),
		notificationEnabled: true,
		chartProvider:       "coinglass",
	}
}

// Name возвращает имя анализатора
func (a *CounterAnalyzer) Name() string {
	return "counter_analyzer"
}

// Version возвращает версию
func (a *CounterAnalyzer) Version() string {
	return "1.0.0"
}

// Supports проверяет поддержку символа
func (a *CounterAnalyzer) Supports(symbol string) bool {
	return true
}

// Analyze анализирует данные и обновляет счетчики
func (a *CounterAnalyzer) Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) < 2 {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("insufficient data points")
	}

	symbol := data[0].Symbol

	// Получаем или создаем счетчик для символа
	counter := a.getOrCreateCounter(symbol)

	// Рассчитываем изменение за базовый период
	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price
	change := ((endPrice - startPrice) / startPrice) * 100

	// Получаем пороги из конфигурации
	growthThreshold := a.getGrowthThreshold()
	fallThreshold := a.getFallThreshold()

	var signals []analysis.Signal
	var counterUpdated bool

	// Проверяем рост
	if change > growthThreshold && a.shouldTrackGrowth() {
		counter.Lock()
		counter.GrowthCount++
		counter.LastGrowthTime = time.Now()
		counterUpdated = true
		counter.Unlock()

		// Создаем сигнал анализа
		signal := a.createAnalysisSignal(symbol, "growth", change, counter.GrowthCount)
		signals = append(signals, signal)

		// Отправляем уведомление если нужно
		a.sendNotificationIfNeeded(symbol, types.CounterTypeGrowth, counter)
	}

	// Проверяем падение
	if change < -fallThreshold && a.shouldTrackFall() {
		counter.Lock()
		counter.FallCount++
		counter.LastFallTime = time.Now()
		counterUpdated = true
		counter.Unlock()

		// Создаем сигнал анализа
		signal := a.createAnalysisSignal(symbol, "fall", change, counter.FallCount)
		signals = append(signals, signal)

		// Отправляем уведомление если нужно
		a.sendNotificationIfNeeded(symbol, types.CounterTypeFall, counter)
	}

	// Проверяем сброс периода
	if counterUpdated {
		a.checkPeriodReset(counter)
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

// getOrCreateCounter получает или создает счетчик для символа
func (a *CounterAnalyzer) getOrCreateCounter(symbol string) *internalCounter {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		a.mu.Lock()
		counter = &internalCounter{
			SignalCounter: types.SignalCounter{
				Symbol:          symbol,
				GrowthCount:     0,
				FallCount:       0,
				Period:          a.getCurrentPeriod(),
				PeriodStartTime: time.Now(),
				LastGrowthTime:  time.Time{},
				LastFallTime:    time.Time{},
			},
		}
		a.counters[symbol] = counter
		a.mu.Unlock()
	}

	return counter
}

// createAnalysisSignal создает сигнал анализа
func (a *CounterAnalyzer) createAnalysisSignal(symbol, direction string, change float64, count int) analysis.Signal {
	return analysis.Signal{
		Symbol:        symbol,
		Type:          "counter_" + direction,
		Direction:     direction,
		ChangePercent: change,
		Confidence:    a.calculateConfidence(count),
		DataPoints:    2,
		StartPrice:    0,
		EndPrice:      0,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "counter_analyzer",
			Tags:     []string{"counter", direction, fmt.Sprintf("count_%d", count)},
			Indicators: map[string]float64{
				"count":  float64(count),
				"change": change,
				"period": a.getPeriodMinutes(),
			},
		},
	}
}

// sendNotificationIfNeeded отправляет уведомление если достигнут порог
func (a *CounterAnalyzer) sendNotificationIfNeeded(symbol string, signalType types.CounterSignalType, counter *internalCounter) {
	if !a.notificationEnabled || a.telegramBot == nil {
		return
	}

	counter.RLock()
	var count int
	var lastTime time.Time
	var periodStart = counter.PeriodStartTime

	if signalType == types.CounterTypeGrowth {
		count = counter.GrowthCount
		lastTime = counter.LastGrowthTime
	} else {
		count = counter.FallCount
		lastTime = counter.LastFallTime
	}
	counter.RUnlock()

	// Проверяем порог уведомления
	if count%a.getNotificationThreshold() == 0 {
		notification := types.CounterNotification{
			Symbol:          symbol,
			SignalType:      signalType,
			CurrentCount:    count,
			Period:          counter.Period,
			PeriodStartTime: periodStart,
			Timestamp:       lastTime,
			MaxSignals:      a.getMaxSignalsForPeriod(counter.Period),
			Percentage:      float64(count) / float64(a.getMaxSignalsForPeriod(counter.Period)) * 100,
		}

		a.sendTelegramNotification(notification)
	}
}

// sendTelegramNotification отправляет уведомление в Telegram
func (a *CounterAnalyzer) sendTelegramNotification(notification types.CounterNotification) {
	// Форматируем сообщение
	message := a.formatNotificationMessage(notification)

	// Создаем клавиатуру с кнопками
	keyboard := a.createNotificationKeyboard(notification)

	// Отправляем сообщение
	if err := a.telegramBot.SendMessageWithKeyboard(message, keyboard); err != nil {
		log.Printf("❌ Ошибка отправки уведомления счетчика: %v", err)
	}
}

// formatNotificationMessage форматирует сообщение уведомления
func (a *CounterAnalyzer) formatNotificationMessage(notification types.CounterNotification) string {
	icon := "🟢"
	directionStr := "РОСТ"
	if notification.SignalType == types.CounterTypeFall {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
	}

	periodStr := a.periodToString(notification.Period)
	timeStr := notification.Timestamp.Format("2006/01/02 15:04:05")

	return fmt.Sprintf(
		"📊 *Счетчик сигналов*\n"+
			"⚫ Символ: %s\n"+
			"🕐 Время: %s\n"+
			"⏱️  Период: %s\n"+
			"%s Направление: %s\n"+
			"📈 Счетчик: %d/%d (%.0f%%)\n"+
			"📊 Базовый период: %d мин",
		notification.Symbol,
		timeStr,
		periodStr,
		icon, directionStr,
		notification.CurrentCount, notification.MaxSignals, notification.Percentage,
		a.getBasePeriodMinutes(),
	)
}

// createNotificationKeyboard создает клавиатуру для уведомления
func (a *CounterAnalyzer) createNotificationKeyboard(notification types.CounterNotification) *telegram.InlineKeyboardMarkup {
	chartURL := a.getChartURL(notification.Symbol)
	symbolURL := fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", notification.Symbol)

	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text: "📊 График",
					URL:  chartURL,
				},
				{
					Text: "💱 Торговать",
					URL:  symbolURL,
				},
			},
			{
				{
					Text:         "🔕 Отключить уведомления",
					CallbackData: fmt.Sprintf("counter_notify_%s_off", notification.Symbol),
				},
				{
					Text:         "⚙️ Настройки счетчика",
					CallbackData: "counter_settings",
				},
			},
		},
	}
}

// getChartURL возвращает URL графика в зависимости от провайдера
func (a *CounterAnalyzer) getChartURL(symbol string) string {
	switch a.chartProvider {
	case "tradingview":
		return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", symbol)
	default: // coinglass
		return fmt.Sprintf("https://www.coinglass.com/tv/%s", symbol)
	}
}

// periodToString конвертирует период в строку
func (a *CounterAnalyzer) periodToString(period types.CounterPeriod) string {
	switch period {
	case types.Period5Min:
		return "5 минут"
	case types.Period15Min:
		return "15 минут"
	case types.Period30Min:
		return "30 минут"
	case types.Period1Hour:
		return "1 час"
	case types.Period4Hours:
		return "4 часа"
	case types.Period1Day:
		return "1 день"
	default:
		return "15 минут"
	}
}

// checkPeriodReset проверяет и сбрасывает счетчик если период истек
func (a *CounterAnalyzer) checkPeriodReset(counter *internalCounter) {
	now := time.Now()
	periodDuration := a.getPeriodDuration(counter.Period)

	if now.Sub(counter.PeriodStartTime) >= periodDuration {
		counter.Lock()
		counter.GrowthCount = 0
		counter.FallCount = 0
		counter.PeriodStartTime = now
		counter.Period = a.getCurrentPeriod()
		counter.Unlock()

		log.Printf("🔄 Счетчик для %s сброшен, новый период: %s", counter.Symbol, counter.Period)
	}
}

// Вспомогательные методы для получения значений из конфигурации
func (a *CounterAnalyzer) getGrowthThreshold() float64 {
	if val, ok := a.config.CustomSettings["growth_threshold"].(float64); ok {
		return val
	}
	return 0.1 // 0.1% по умолчанию
}

func (a *CounterAnalyzer) getFallThreshold() float64 {
	if val, ok := a.config.CustomSettings["fall_threshold"].(float64); ok {
		return val
	}
	return 0.1 // 0.1% по умолчанию
}

func (a *CounterAnalyzer) getBasePeriodMinutes() int {
	if val, ok := a.config.CustomSettings["base_period_minutes"].(int); ok {
		return val
	}
	return 1
}

func (a *CounterAnalyzer) getNotificationThreshold() int {
	if val, ok := a.config.CustomSettings["notification_threshold"].(int); ok {
		return val
	}
	return 1
}

func (a *CounterAnalyzer) shouldTrackGrowth() bool {
	if val, ok := a.config.CustomSettings["track_growth"].(bool); ok {
		return val
	}
	return true
}

func (a *CounterAnalyzer) shouldTrackFall() bool {
	if val, ok := a.config.CustomSettings["track_fall"].(bool); ok {
		return val
	}
	return true
}

func (a *CounterAnalyzer) getCurrentPeriod() types.CounterPeriod {
	if val, ok := a.config.CustomSettings["analysis_period"].(string); ok {
		return types.CounterPeriod(val)
	}
	return types.Period15Min
}

func (a *CounterAnalyzer) getPeriodMinutes() float64 {
	switch a.getCurrentPeriod() {
	case types.Period5Min:
		return 5
	case types.Period15Min:
		return 15
	case types.Period30Min:
		return 30
	case types.Period1Hour:
		return 60
	case types.Period4Hours:
		return 240
	case types.Period1Day:
		return 1440
	default:
		return 15
	}
}

func (a *CounterAnalyzer) getPeriodDuration(period types.CounterPeriod) time.Duration {
	switch period {
	case types.Period5Min:
		return 5 * time.Minute
	case types.Period15Min:
		return 15 * time.Minute
	case types.Period30Min:
		return 30 * time.Minute
	case types.Period1Hour:
		return time.Hour
	case types.Period4Hours:
		return 4 * time.Hour
	case types.Period1Day:
		return 24 * time.Hour
	default:
		return 15 * time.Minute
	}
}

func (a *CounterAnalyzer) getMaxSignalsForPeriod(period types.CounterPeriod) int {
	// Создаем ключ для поиска в настройках
	key := ""
	switch period {
	case types.Period5Min:
		key = "max_signals_5m"
	case types.Period15Min:
		key = "max_signals_15m"
	case types.Period30Min:
		key = "max_signals_30m"
	case types.Period1Hour:
		key = "max_signals_1h"
	case types.Period4Hours:
		key = "max_signals_4h"
	case types.Period1Day:
		key = "max_signals_1d"
	default:
		key = "max_signals_15m"
	}

	// Пробуем получить из настроек
	if maxSignals, ok := a.config.CustomSettings[key]; ok {
		if intVal, ok := maxSignals.(int); ok {
			return intVal
		}
		if floatVal, ok := maxSignals.(float64); ok {
			return int(floatVal)
		}
	}

	// Значения по умолчанию
	switch period {
	case types.Period5Min:
		return 5
	case types.Period15Min:
		return 8
	case types.Period30Min:
		return 10
	case types.Period1Hour:
		return 12
	case types.Period4Hours:
		return 15
	case types.Period1Day:
		return 20
	default:
		return 8
	}
}

func (a *CounterAnalyzer) calculateConfidence(count int) float64 {
	maxSignals := a.getMaxSignalsForPeriod(a.getCurrentPeriod())
	if maxSignals == 0 {
		return 0.0
	}
	return float64(count) / float64(maxSignals) * 100
}

// SetNotificationEnabled включает/выключает уведомления
func (a *CounterAnalyzer) SetNotificationEnabled(enabled bool) {
	a.notificationEnabled = enabled
}

// SetChartProvider устанавливает провайдера графиков
func (a *CounterAnalyzer) SetChartProvider(provider string) {
	a.chartProvider = provider
}

// SetAnalysisPeriod устанавливает период анализа
func (a *CounterAnalyzer) SetAnalysisPeriod(period types.CounterPeriod) {
	// Создаем новую мапу настроек
	newSettings := make(map[string]interface{})
	for k, v := range a.config.CustomSettings {
		newSettings[k] = v
	}
	newSettings["analysis_period"] = string(period)
	a.config.CustomSettings = newSettings

	// Сбрасываем все счетчики при смене периода
	a.resetAllCounters()
}

// resetAllCounters сбрасывает все счетчики
func (a *CounterAnalyzer) resetAllCounters() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, counter := range a.counters {
		counter.Lock()
		counter.GrowthCount = 0
		counter.FallCount = 0
		counter.PeriodStartTime = time.Now()
		counter.Period = a.getCurrentPeriod()
		counter.Unlock()
	}
}

// GetCounterStats возвращает статистику счетчика для символа (ИСПРАВЛЕННАЯ)
func (a *CounterAnalyzer) GetCounterStats(symbol string) (types.SignalCounter, bool) {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		return types.SignalCounter{}, false
	}

	counter.RLock()
	defer counter.RUnlock()

	// Возвращаем копию данных без мьютекса
	return types.SignalCounter{
		Symbol:          counter.Symbol,
		GrowthCount:     counter.GrowthCount,
		FallCount:       counter.FallCount,
		Period:          counter.Period,
		PeriodStartTime: counter.PeriodStartTime,
		LastGrowthTime:  counter.LastGrowthTime,
		LastFallTime:    counter.LastFallTime,
	}, true
}

// GetAllCounters возвращает все счетчики (ИСПРАВЛЕННАЯ)
func (a *CounterAnalyzer) GetAllCounters() map[string]types.SignalCounter {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]types.SignalCounter)
	for symbol, counter := range a.counters {
		counter.RLock()

		// Создаем копию без мьютекса
		result[symbol] = types.SignalCounter{
			Symbol:          counter.Symbol,
			GrowthCount:     counter.GrowthCount,
			FallCount:       counter.FallCount,
			Period:          counter.Period,
			PeriodStartTime: counter.PeriodStartTime,
			LastGrowthTime:  counter.LastGrowthTime,
			LastFallTime:    counter.LastFallTime,
		}

		counter.RUnlock()
	}

	return result
}

// GetConfig возвращает конфигурацию
func (a *CounterAnalyzer) GetConfig() AnalyzerConfig {
	return a.config
}

// GetStats возвращает статистику
func (a *CounterAnalyzer) GetStats() AnalyzerStats {
	return a.stats
}

// updateStats обновляет статистику
func (a *CounterAnalyzer) updateStats(duration time.Duration, success bool) {
	a.stats.TotalCalls++
	a.stats.TotalTime += duration
	a.stats.LastCallTime = time.Now()

	if success {
		a.stats.SuccessCount++
	} else {
		a.stats.ErrorCount++
	}

	if a.stats.TotalCalls > 0 {
		a.stats.AverageTime = time.Duration(
			int64(a.stats.TotalTime) / int64(a.stats.TotalCalls),
		)
	}
}

// DefaultCounterConfig - конфигурация по умолчанию
var DefaultCounterConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        0.7,
	MinConfidence: 10.0,
	MinDataPoints: 2,
	CustomSettings: map[string]interface{}{
		"base_period_minutes":    1,
		"analysis_period":        "15m",
		"growth_threshold":       0.1,
		"fall_threshold":         0.1,
		"track_growth":           true,
		"track_fall":             true,
		"notification_threshold": 1,
		"max_signals_5m":         5,
		"max_signals_15m":        8,
		"max_signals_30m":        10,
		"max_signals_1h":         12,
		"max_signals_4h":         15,
		"max_signals_1d":         20,
		"chart_provider":         "coinglass",
	},
}
