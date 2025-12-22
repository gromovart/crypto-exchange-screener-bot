// internal/analysis/analyzers/counter_analyzer.go
package analyzers

import (
	tgbot "crypto_exchange_screener_bot/internal/telegram" // Импортируем пакет с реализацией бота
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/common"
	"crypto_exchange_screener_bot/internal/types/storage"
	"crypto_exchange_screener_bot/internal/types/telegram" // Импортируем пакет с типами
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CounterAnalyzer - анализатор счетчика сигналов
type CounterAnalyzer struct {
	config              analysis.AnalyzerConfig
	stats               analysis.AnalyzerStats
	storage             storage.PriceStorage
	telegramBot         *tgbot.TelegramBot // Бот из пакета telegram
	counters            map[string]*analysis.InternalCounter
	mu                  sync.RWMutex
	notificationEnabled bool
	chartProvider       string
	lastPriceCache      map[string]float64
	priceCacheMu        sync.RWMutex
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(config analysis.AnalyzerConfig, storage storage.PriceStorage, tgBot *tgbot.TelegramBot) *CounterAnalyzer {
	return &CounterAnalyzer{
		config:              config,
		storage:             storage,
		telegramBot:         tgBot,
		counters:            make(map[string]*analysis.InternalCounter),
		notificationEnabled: true,
		chartProvider:       "coinglass",
		lastPriceCache:      make(map[string]float64),
	}
}

// Name возвращает имя анализатора
func (a *CounterAnalyzer) Name() string {
	return "counter_analyzer"
}

// Version возвращает версию
func (a *CounterAnalyzer) Version() string {
	return "2.0.0"
}

// Supports проверяет поддержку символа
func (a *CounterAnalyzer) Supports(symbol string) bool {
	return true
}

// Analyze анализирует данные и обновляет счетчики
func (a *CounterAnalyzer) Analyze(data []common.PriceData, config analysis.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) < 2 {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("insufficient data points")
	}

	symbol := string(data[0].Symbol)

	// Получаем или создаем счетчик для символа
	counter := a.getOrCreateCounter(symbol)

	// Получаем базовый период (по умолчанию 1 минута)
	basePeriodMinutes := a.getBasePeriodMinutes()

	// Получаем текущий выбранный период анализа
	selectedPeriod := a.getCurrentPeriod()

	// Рассчитываем максимальное количество сигналов для выбранного периода
	maxSignals := a.calculateMaxSignals(selectedPeriod, basePeriodMinutes)

	// Проверяем истечение периода
	a.checkAndResetPeriod(counter, selectedPeriod, maxSignals)

	// Рассчитываем изменение цены за базовый период
	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price
	change := ((endPrice - startPrice) / startPrice) * 100

	// Получаем пороги из конфигурации
	growthThreshold := a.getGrowthThreshold()
	fallThreshold := a.getFallThreshold()

	var signals []analysis.Signal
	var signalDetected bool
	var signalType analysis.CounterSignalType

	counter.Lock()

	// Увеличиваем счетчик обработанных базовых периодов
	counter.BasePeriodCount++

	// Проверяем рост
	if change > growthThreshold && counter.Settings.TrackGrowth {
		counter.GrowthCount++
		counter.SignalCount++
		counter.LastSignalTime = time.Now()
		signalDetected = true
		signalType = analysis.CounterTypeGrowth

		// Создаем сигнал анализа
		signal := a.createAnalysisSignal(symbol, analysis.TrendBullish, change, counter.SignalCount, maxSignals, startPrice, endPrice)
		signals = append(signals, signal)
	}

	// Проверяем падение
	if change < -fallThreshold && counter.Settings.TrackFall {
		counter.FallCount++
		counter.SignalCount++
		counter.LastSignalTime = time.Now()
		signalDetected = true
		signalType = analysis.CounterTypeFall

		// Создаем сигнал анализа
		signal := a.createAnalysisSignal(symbol, analysis.TrendBearish, math.Abs(change), counter.SignalCount, maxSignals, startPrice, endPrice)
		signals = append(signals, signal)
	}

	counter.Unlock()

	// Отправляем уведомление если нужно
	if signalDetected {
		a.sendNotificationIfNeeded(symbol, signalType, counter, maxSignals, change)
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

// getOrCreateCounter получает или создает счетчик для символа
func (a *CounterAnalyzer) getOrCreateCounter(symbol string) *analysis.InternalCounter {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		a.mu.Lock()
		// Создаем счетчик с настройками по умолчанию
		counter = &analysis.InternalCounter{
			SignalCounter: analysis.SignalCounter{
				Symbol:          common.Symbol(symbol),
				SelectedPeriod:  a.getCurrentPeriod(),
				BasePeriodCount: 0,
				SignalCount:     0,
				GrowthCount:     0,
				FallCount:       0,
				PeriodStartTime: time.Now(),
				PeriodEndTime:   time.Now().Add(a.getPeriodDuration(a.getCurrentPeriod())),
				LastSignalTime:  time.Time{},
				Settings: analysis.CounterSettings{
					BasePeriodMinutes: a.getBasePeriodMinutes(),
					SelectedPeriod:    a.getCurrentPeriod(),
					TrackGrowth:       a.shouldTrackGrowth(),
					TrackFall:         a.shouldTrackFall(),
					ChartProvider:     a.getChartProvider(),
					NotifyOnSignal:    a.shouldNotifyOnSignal(),
				},
			},
		}
		a.counters[symbol] = counter
		a.mu.Unlock()
	}

	return counter
}

// createAnalysisSignal создает сигнал анализа
func (a *CounterAnalyzer) createAnalysisSignal(
	symbol string,
	direction analysis.TrendDirection,
	change float64,
	count, maxSignals int,
	startPrice, endPrice float64,
) analysis.Signal {

	confidence := a.calculateConfidence(count, maxSignals)
	selectedPeriod := a.getCurrentPeriod()

	// Определяем тип сигнала на основе направления
	var signalType analysis.SignalType
	if direction == analysis.TrendBullish {
		signalType = analysis.SignalType("counter_growth")
	} else {
		signalType = analysis.SignalType("counter_fall")
	}

	return analysis.Signal{
		ID:            uuid.New().String(),
		Symbol:        common.Symbol(symbol),
		Type:          signalType,
		Direction:     direction,
		ChangePercent: change,
		Confidence:    confidence,
		Strength:      confidence / 100.0,
		DataPoints:    2,
		StartPrice:    startPrice,
		EndPrice:      endPrice,
		Timestamp:     time.Now(),
		Metadata: analysis.SignalMetadata{ // Используем SignalMetadata вместо map[string]interface{}
			Strategy:       "counter_analyzer_v2",
			Tags:           []string{"counter", string(direction), fmt.Sprintf("count_%d", count), string(selectedPeriod)},
			IsContinuous:   false, // Для счетчика это false
			ContinuousFrom: 0,
			ContinuousTo:   0,
			Indicators: map[string]float64{
				"count":           float64(count),
				"max_signals":     float64(maxSignals),
				"current_count":   float64(count),
				"total_max":       float64(maxSignals),
				"change":          change,
				"period_minutes":  float64(selectedPeriod.GetMinutes()),
				"base_period":     float64(a.getBasePeriodMinutes()),
				"period_progress": float64(count) / float64(maxSignals) * 100,
				"percentage":      float64(count) / float64(maxSignals) * 100,
			},
		},
	}
}

// sendNotificationIfNeeded отправляет уведомление если достигнут порог
func (a *CounterAnalyzer) sendNotificationIfNeeded(symbol string, signalType analysis.CounterSignalType, counter *analysis.InternalCounter, maxSignals int, change float64) {
	if !a.notificationEnabled || a.telegramBot == nil {
		return
	}

	if !counter.Settings.NotifyOnSignal {
		return
	}

	counter.RLock()
	var count int
	if signalType == analysis.CounterTypeGrowth {
		count = counter.GrowthCount
	} else {
		count = counter.FallCount
	}

	notification := analysis.CounterNotification{
		Symbol:          counter.Symbol,
		SignalType:      signalType,
		CurrentCount:    count,
		TotalCount:      counter.SignalCount,
		Period:          counter.SelectedPeriod,
		PeriodStartTime: counter.PeriodStartTime,
		PeriodEndTime:   counter.PeriodEndTime,
		Timestamp:       time.Now(),
		MaxSignals:      maxSignals,
		Percentage:      float64(counter.SignalCount) / float64(maxSignals) * 100,
		ChangePercent:   math.Abs(change),
	}
	counter.RUnlock()

	// Проверяем, не превышен ли лимит уведомлений
	if a.canSendNotification(symbol, signalType) {
		a.sendTelegramNotification(notification)
		a.updateNotificationSent(symbol, signalType)
	}
}

// canSendNotification проверяет лимит уведомлений
func (a *CounterAnalyzer) canSendNotification(symbol string, signalType analysis.CounterSignalType) bool {
	// Здесь можно добавить логику ограничения частоты уведомлений
	// если требуется (например, не чаще 1 раза в 30 секунд)
	return true
}

// updateNotificationSent обновляет время последнего уведомления
func (a *CounterAnalyzer) updateNotificationSent(symbol string, signalType analysis.CounterSignalType) {
	// Можно добавить кэш времени последнего уведомления
	// для ограничения частоты
}

// sendTelegramNotification отправляет уведомление в Telegram
func (a *CounterAnalyzer) sendTelegramNotification(notification analysis.CounterNotification) {
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
func (a *CounterAnalyzer) formatNotificationMessage(notification analysis.CounterNotification) string {
	icon := "🟢"
	directionStr := "РОСТ"
	changeStr := fmt.Sprintf("+%.2f%%", notification.ChangePercent)

	if notification.SignalType == analysis.CounterTypeFall {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
		changeStr = fmt.Sprintf("-%.2f%%", notification.ChangePercent)
	}

	timeStr := notification.Timestamp.Format("2006/01/02 15:04:05")

	return fmt.Sprintf(
		"⚫ Bybit - 1мин - %s\n"+
			"🕐 %s\n"+
			"%s %s: %s\n"+
			"📡 Сигнал: %d\n"+
			"⏱️  Период: %s",
		notification.Symbol,
		timeStr,
		icon, directionStr, changeStr,
		notification.CurrentCount,
		notification.Period.ToString(),
	)
}

// createNotificationKeyboard создает клавиатуру для уведомления
func (a *CounterAnalyzer) createNotificationKeyboard(notification analysis.CounterNotification) *telegram.InlineKeyboardMarkup {
	// Используем провайдера из настроек счетчика
	chartProvider := notification.SignalType.GetChartProvider()
	if chartProvider == "" {
		chartProvider = a.chartProvider
	}

	chartURL := a.getChartURL(string(notification.Symbol), chartProvider)

	// Получаем период из настроек счетчика
	periodMinutes := notification.Period.GetMinutes()
	symbolURL := a.getTradingURL(string(notification.Symbol), periodMinutes)

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
		},
	}
}

// getTradingURL формирует URL для торговли с учетом периода
func (a *CounterAnalyzer) getTradingURL(symbol string, periodMinutes int) string {
	// Определяем интервал для графика на основе периода анализа
	interval := a.getTradingInterval(periodMinutes)

	// Формируем URL для Bybit с параметром интервала
	return fmt.Sprintf(
		"https://www.bybit.com/trade/usdt/%s?interval=%s",
		symbol,
		interval,
	)
}

// getTradingInterval преобразует минуты в интервал торгового терминала
func (a *CounterAnalyzer) getTradingInterval(periodMinutes int) string {
	switch periodMinutes {
	case 1, 5:
		return "5"
	case 15:
		return "15"
	case 30:
		return "30"
	case 60:
		return "60"
	case 240: // 4 часа
		return "240"
	case 1440: // 1 день
		return "1D"
	default:
		return "15" // по умолчанию 15 минут
	}
}

// getChartURL возвращает URL графика в зависимости от провайдера
func (a *CounterAnalyzer) getChartURL(symbol, provider string) string {
	switch provider {
	case "tradingview":
		return fmt.Sprintf("https://www.tradingview.com/chart/?symbol=BYBIT:%s", symbol)
	default: // coinglass
		return fmt.Sprintf("https://www.coinglass.com/tv/%s", symbol)
	}
}

// checkAndResetPeriod проверяет и сбрасывает счетчик если период истек или превышен лимит
func (a *CounterAnalyzer) checkAndResetPeriod(counter *analysis.InternalCounter, period analysis.CounterPeriod, maxSignals int) {
	counter.Lock()
	defer counter.Unlock()

	now := time.Now()
	periodDuration := period.GetDuration()

	// Проверяем условия для сброса:
	// 1. Истек период времени
	// 2. Достигнут максимум сигналов
	// 3. Изменился выбранный период
	if now.Sub(counter.PeriodStartTime) >= periodDuration ||
		counter.SignalCount >= maxSignals ||
		counter.SelectedPeriod != period {

		log.Printf("🔄 Счетчик для %s сброшен. Причина: ", counter.Symbol)
		if now.Sub(counter.PeriodStartTime) >= periodDuration {
			log.Printf("истек период")
		} else if counter.SignalCount >= maxSignals {
			log.Printf("достигнут максимум сигналов (%d/%d)", counter.SignalCount, maxSignals)
		} else {
			log.Printf("изменился период на %s", period)
		}

		// Сбрасываем счетчик
		counter.BasePeriodCount = 0
		counter.SignalCount = 0
		counter.GrowthCount = 0
		counter.FallCount = 0
		counter.PeriodStartTime = now
		counter.PeriodEndTime = now.Add(periodDuration)
		counter.SelectedPeriod = period
		counter.Settings.SelectedPeriod = period
	}
}

// calculateMaxSignals вычисляет максимальное количество сигналов
func (a *CounterAnalyzer) calculateMaxSignals(period analysis.CounterPeriod, basePeriodMinutes int) int {
	// Согласно требованию: выбранный период / базовый период = сигнал
	totalPossibleSignals := period.GetMinutes() / basePeriodMinutes

	// Согласно требованию 4: ограничиваем 5-15 сигналами
	if totalPossibleSignals < 5 {
		return 5
	}
	if totalPossibleSignals > 15 {
		return 15
	}
	return totalPossibleSignals
}

// Вспомогательные методы для получения значений из конфигурации
func (a *CounterAnalyzer) getGrowthThreshold() float64 {
	return SafeGetFloat(a.config.CustomSettings["growth_threshold"], 0.1)
}

func (a *CounterAnalyzer) getFallThreshold() float64 {
	return SafeGetFloat(a.config.CustomSettings["fall_threshold"], 0.1)
}

func (a *CounterAnalyzer) getBasePeriodMinutes() int {
	return SafeGetInt(a.config.CustomSettings["base_period_minutes"], 1)
}

func (a *CounterAnalyzer) getNotificationThreshold() int {
	return SafeGetInt(a.config.CustomSettings["notification_threshold"], 1)
}

func (a *CounterAnalyzer) shouldTrackGrowth() bool {
	return SafeGetBool(a.config.CustomSettings["track_growth"], true)
}

func (a *CounterAnalyzer) shouldTrackFall() bool {
	return SafeGetBool(a.config.CustomSettings["track_fall"], true)
}

func (a *CounterAnalyzer) shouldNotifyOnSignal() bool {
	return SafeGetBool(a.config.CustomSettings["notify_on_signal"], true)
}

func (a *CounterAnalyzer) getCurrentPeriod() analysis.CounterPeriod {
	periodStr := SafeGetString(a.config.CustomSettings["analysis_period"], "15m")
	return analysis.CounterPeriod(periodStr)
}

func (a *CounterAnalyzer) getChartProvider() string {
	return SafeGetString(a.config.CustomSettings["chart_provider"], "coinglass")
}

func (a *CounterAnalyzer) calculateConfidence(count, maxSignals int) float64 {
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
func (a *CounterAnalyzer) SetAnalysisPeriod(period analysis.CounterPeriod) {
	// Обновляем настройки
	newSettings := make(map[string]interface{})
	for k, v := range a.config.CustomSettings {
		newSettings[k] = v
	}
	newSettings["analysis_period"] = string(period)
	a.config.CustomSettings = newSettings

	// Сбрасываем все счетчики при смене периода
	a.resetAllCountersForPeriod(period)
}

// resetAllCountersForPeriod сбрасывает все счетчики для нового периода
func (a *CounterAnalyzer) resetAllCountersForPeriod(newPeriod analysis.CounterPeriod) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, counter := range a.counters {
		counter.Lock()
		counter.BasePeriodCount = 0
		counter.SignalCount = 0
		counter.GrowthCount = 0
		counter.FallCount = 0
		counter.PeriodStartTime = time.Now()
		counter.PeriodEndTime = time.Now().Add(newPeriod.GetDuration())
		counter.SelectedPeriod = newPeriod
		counter.Settings.SelectedPeriod = newPeriod
		counter.Unlock()
	}

	log.Printf("🔄 Все счетчики сброшены для нового периода: %s", newPeriod)
}

// SetTrackingOptions устанавливает опции отслеживания
func (a *CounterAnalyzer) SetTrackingOptions(symbol string, trackGrowth, trackFall bool) error {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("counter for symbol %s not found", symbol)
	}

	counter.Lock()
	counter.Settings.TrackGrowth = trackGrowth
	counter.Settings.TrackFall = trackFall
	counter.Unlock()

	return nil
}

// GetCounterStats возвращает статистику счетчика для символа
func (a *CounterAnalyzer) GetCounterStats(symbol string) (analysis.SignalCounter, bool) {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		return analysis.SignalCounter{}, false
	}

	counter.RLock()
	defer counter.RUnlock()

	// Возвращаем копию данных без мьютекса
	return analysis.SignalCounter{
		Symbol:          counter.Symbol,
		SelectedPeriod:  counter.SelectedPeriod,
		BasePeriodCount: counter.BasePeriodCount,
		SignalCount:     counter.SignalCount,
		GrowthCount:     counter.GrowthCount,
		FallCount:       counter.FallCount,
		PeriodStartTime: counter.PeriodStartTime,
		PeriodEndTime:   counter.PeriodEndTime,
		LastSignalTime:  counter.LastSignalTime,
		Settings:        counter.Settings,
	}, true
}

// GetAllCounters возвращает все счетчики
func (a *CounterAnalyzer) GetAllCounters() map[string]analysis.SignalCounter {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]analysis.SignalCounter)
	for symbol, counter := range a.counters {
		counter.RLock()

		// Создаем копию без мьютекса
		result[symbol] = analysis.SignalCounter{
			Symbol:          counter.Symbol,
			SelectedPeriod:  counter.SelectedPeriod,
			BasePeriodCount: counter.BasePeriodCount,
			SignalCount:     counter.SignalCount,
			GrowthCount:     counter.GrowthCount,
			FallCount:       counter.FallCount,
			PeriodStartTime: counter.PeriodStartTime,
			PeriodEndTime:   counter.PeriodEndTime,
			LastSignalTime:  counter.LastSignalTime,
			Settings:        counter.Settings,
		}

		counter.RUnlock()
	}

	return result
}

// GetConfig возвращает конфигурацию
func (a *CounterAnalyzer) GetConfig() analysis.AnalyzerConfig {
	return a.config
}

// GetStats возвращает статистику
func (a *CounterAnalyzer) GetStats() analysis.AnalyzerStats {
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

// getPeriodDuration возвращает длительность периода
func (a *CounterAnalyzer) getPeriodDuration(period analysis.CounterPeriod) time.Duration {
	return period.GetDuration()
}

// DefaultCounterConfig - конфигурация по умолчанию
var DefaultCounterConfig = analysis.AnalyzerConfig{
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
		"notify_on_signal":       true,
		"notification_threshold": 1,
		"chart_provider":         "coinglass",
	},
}
