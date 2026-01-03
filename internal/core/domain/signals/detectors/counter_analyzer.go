// internal/core/domain/signals/detectors/counter_analyzer.go
package analyzers

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CounterAnalyzer - анализатор счетчика сигналов
type CounterAnalyzer struct {
	config              AnalyzerConfig
	stats               AnalyzerStats
	storage             storage.PriceStorage
	telegramBot         *telegram.TelegramBot
	counters            map[string]*internalCounter
	mu                  sync.RWMutex
	notificationEnabled bool
	chartProvider       string
	lastPriceCache      map[string]float64
	priceCacheMu        sync.RWMutex
	buttonBuilder       *telegram.ButtonURLBuilder
	messageFormatter    *telegram.MarketMessageFormatter
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(config AnalyzerConfig, storage storage.PriceStorage, tgBot *telegram.TelegramBot) *CounterAnalyzer {
	// Получаем провайдер графиков из конфигурации
	chartProvider := SafeGetString(config.CustomSettings["chart_provider"], "coinglass")
	exchange := SafeGetString(config.CustomSettings["exchange"], "bybit")

	// Создаем buttonBuilder с правильным провайдером
	buttonBuilder := telegram.NewButtonURLBuilderWithProvider(exchange, chartProvider)

	return &CounterAnalyzer{
		config:              config,
		storage:             storage,
		telegramBot:         tgBot,
		counters:            make(map[string]*internalCounter),
		notificationEnabled: true,
		chartProvider:       chartProvider,
		lastPriceCache:      make(map[string]float64),
		buttonBuilder:       buttonBuilder,
		messageFormatter:    telegram.NewMarketMessageFormatter(exchange),
	}
}

// Name возвращает имя анализатора
func (a *CounterAnalyzer) Name() string {
	return "counter_analyzer"
}

// Version возвращает версию
func (a *CounterAnalyzer) Version() string {
	return "2.1.0" // Увеличиваем версию из-за добавления новых данных
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
	var signalType CounterSignalType

	counter.Lock()

	// Увеличиваем счетчик обработанных базовых периодов
	counter.BasePeriodCount++

	// Проверяем рост
	if change > growthThreshold && counter.Settings.TrackGrowth {
		counter.GrowthCount++
		counter.SignalCount++
		counter.LastSignalTime = time.Now()
		signalDetected = true
		signalType = CounterTypeGrowth

		// Создаем сигнал анализа
		signal := a.createAnalysisSignal(symbol, "growth", change, counter.SignalCount, maxSignals, data)
		signals = append(signals, signal)
	}

	// Проверяем падение
	if change < -fallThreshold && counter.Settings.TrackFall {
		counter.FallCount++
		counter.SignalCount++
		counter.LastSignalTime = time.Now()
		signalDetected = true
		signalType = CounterTypeFall

		// Создаем сигнал анализа
		signal := a.createAnalysisSignal(symbol, "fall", math.Abs(change), counter.SignalCount, maxSignals, data)
		signals = append(signals, signal)
	}

	counter.Unlock()

	log.Printf("🔍 CounterAnalyzer.Analyze для %s:", symbol)
	for i, d := range data {
		log.Printf("   data[%d].OpenInterest = %f", i, d.OpenInterest)
	}

	// Отправляем улучшенное уведомление если нужно
	if signalDetected {
		a.sendEnhancedNotification(symbol, signalType, counter, maxSignals, change, data)
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

// createAnalysisSignal создает сигнал анализа
func (a *CounterAnalyzer) createAnalysisSignal(symbol, direction string,
	change float64, count, maxSignals int, data []types.PriceData) analysis.Signal {

	confidence := a.calculateConfidence(count, maxSignals)
	selectedPeriod := a.getCurrentPeriod()

	// Получаем последние данные для дополнительных метрик
	latestData := data[len(data)-1]
	oiChange24h := a.calculateOIChange24h(data)

	return analysis.Signal{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          "counter_" + direction,
		Direction:     direction,
		ChangePercent: change,
		Confidence:    confidence,
		DataPoints:    2,
		StartPrice:    data[0].Price,
		EndPrice:      latestData.Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "counter_analyzer_v2",
			Tags:     []string{"counter", direction, fmt.Sprintf("count_%d", count), string(selectedPeriod), "no_duplicate"},
			Indicators: map[string]float64{
				"count":            float64(count),
				"max_signals":      float64(maxSignals),
				"current_count":    float64(count),
				"total_max":        float64(maxSignals),
				"change":           change,
				"period_minutes":   float64(selectedPeriod.GetMinutes()),
				"base_period":      float64(a.getBasePeriodMinutes()),
				"period_progress":  float64(count) / float64(maxSignals) * 100,
				"percentage":       float64(count) / float64(maxSignals) * 100,
				"volume_24h":       latestData.Volume24h,
				"open_interest":    latestData.OpenInterest,
				"oi_change_24h":    oiChange24h,
				"funding_rate":     latestData.FundingRate,
				"current_price":    latestData.Price,
				"price_change_24h": latestData.Change24h,
				"high_24h":         latestData.High24h,
				"low_24h":          latestData.Low24h,
			},
		},
	}
}

// sendEnhancedNotification отправляет улучшенное уведомление
func (a *CounterAnalyzer) sendEnhancedNotification(
	symbol string,
	signalType CounterSignalType,
	counter *internalCounter,
	maxSignals int,
	change float64,
	priceData []types.PriceData,
) {
	if !a.notificationEnabled || a.telegramBot == nil {
		return
	}

	if !counter.Settings.NotifyOnSignal {
		return
	}

	counter.RLock()

	var count int
	if signalType == CounterTypeGrowth {
		count = counter.GrowthCount
	} else {
		count = counter.FallCount
	}

	// Создаем уведомление
	notification := CounterNotification{
		Symbol:          symbol,
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

	// Проверяем лимит уведомлений
	if a.canSendNotification(symbol, signalType) {
		// Форматируем сообщение с полной информацией
		message := a.formatEnhancedNotificationMessage(notification, priceData)

		// Создаем клавиатуру
		keyboard := a.createNotificationKeyboard(notification)

		// Отправляем сообщение
		if err := a.telegramBot.SendMessageWithKeyboard(message, keyboard); err != nil {
			log.Printf("❌ Ошибка отправки улучшенного уведомления: %v", err)
		} else {
			log.Printf("✅ Отправлено улучшенное уведомление для %s", symbol)
		}

		a.updateNotificationSent(symbol, signalType)
	}
}

// formatEnhancedNotificationMessage форматирует сообщение с полной информацией
func (a *CounterAnalyzer) formatEnhancedNotificationMessage(
	notification CounterNotification,
	priceData []types.PriceData,
) string {
	if len(priceData) == 0 {
		return a.formatNotificationMessage(notification) // fallback
	}

	// Получаем последние данные
	latestData := priceData[len(priceData)-1]

	// Отладочная информация
	log.Printf("DEBUG CounterAnalyzer: Symbol=%s, OI=%f, VolumeUSD=%f, Price=%f",
		notification.Symbol,
		latestData.OpenInterest,
		latestData.VolumeUSD,
		latestData.Price)
	// Детальный лог для отладки OI
	log.Printf("🔍 CounterAnalyzer.formatEnhancedNotificationMessage для %s:", notification.Symbol)
	log.Printf("   latestData.OpenInterest = %f", latestData.OpenInterest)
	log.Printf("   latestData.VolumeUSD = %f", latestData.VolumeUSD)
	log.Printf("   latestData.Price = %f", latestData.Price)
	log.Printf("   latestData.FundingRate = %f", latestData.FundingRate)
	log.Printf("   len(priceData) = %d", len(priceData))

	// Проверяем все точки данных
	for i, data := range priceData {
		if data.OpenInterest > 0 {
			log.Printf("   priceData[%d].OpenInterest = %f", i, data.OpenInterest)
		}
	}

	// Рассчитываем изменение OI за 24 часа
	oiChange24h := a.calculateOIChange24h(priceData)

	// Рассчитываем время следующего фандинга
	nextFundingTime := a.calculateNextFundingTime()

	// Рассчитываем среднюю ставку фандинга
	averageFunding := a.calculateAverageFunding(priceData)

	// Используем форматтер сообщений
	return a.messageFormatter.FormatCounterMessage(
		notification.Symbol,
		a.getDirectionFromSignalType(notification.SignalType),
		notification.ChangePercent,
		notification.CurrentCount,
		notification.MaxSignals,
		latestData.Price,
		latestData.Volume24h,
		latestData.OpenInterest,
		oiChange24h,
		latestData.FundingRate,
		averageFunding,
		nextFundingTime,
		notification.Period.ToString(),
	)
}

// getOrCreateCounter получает или создает счетчик для символа
func (a *CounterAnalyzer) getOrCreateCounter(symbol string) *internalCounter {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		a.mu.Lock()
		// Создаем счетчик с настройками по умолчанию
		counter = &internalCounter{
			SignalCounter: SignalCounter{
				Symbol:          symbol,
				SelectedPeriod:  a.getCurrentPeriod(),
				BasePeriodCount: 0,
				SignalCount:     0,
				GrowthCount:     0,
				FallCount:       0,
				PeriodStartTime: time.Now(),
				PeriodEndTime:   time.Now().Add(a.getPeriodDuration(a.getCurrentPeriod())),
				LastSignalTime:  time.Time{},
				Settings: CounterSettings{
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

// formatNotificationMessage форматирует сообщение уведомления (старая версия)
func (a *CounterAnalyzer) formatNotificationMessage(notification CounterNotification) string {
	icon := "🟢"
	directionStr := "РОСТ"
	changeStr := fmt.Sprintf("+%.2f%%", notification.ChangePercent)

	if notification.SignalType == CounterTypeFall {
		icon = "🔴"
		directionStr = "ПАДЕНИЕ"
		changeStr = fmt.Sprintf("-%.2f%%", notification.ChangePercent)
	}

	timeStr := notification.Timestamp.Format("2006/01/02 15:04:05")

	// Компактный формат
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
func (a *CounterAnalyzer) createNotificationKeyboard(notification CounterNotification) *telegram.InlineKeyboardMarkup {
	// Используем строитель который уже знает о провайдере графиков
	periodMinutes := notification.Period.GetMinutes()

	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				a.buttonBuilder.GetChartButton(notification.Symbol),
				a.buttonBuilder.GetTradeButton(notification.Symbol, periodMinutes),
			},
		},
	}
}

// calculateOIChange24h рассчитывает изменение OI за 24 часа
func (a *CounterAnalyzer) calculateOIChange24h(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	now := time.Now()
	twentyFourHoursAgo := now.Add(-24 * time.Hour)
	latestData := data[len(data)-1]

	// Если текущий OI = 0, не можем рассчитать изменение
	if latestData.OpenInterest <= 0 {
		return 0
	}

	// Находим OI 24 часа назад (ближайшее значение)
	var oldOI float64
	var minDiff time.Duration = 24 * time.Hour
	var found bool

	for _, point := range data {
		diff := point.Timestamp.Sub(twentyFourHoursAgo)
		if diff.Abs() < minDiff.Abs() && point.OpenInterest > 0 {
			minDiff = diff
			oldOI = point.OpenInterest
			found = true
		}
	}

	if !found || oldOI == 0 || latestData.OpenInterest == 0 {
		return 0
	}

	return ((latestData.OpenInterest - oldOI) / oldOI) * 100
}

// calculateAverageFunding рассчитывает среднюю ставку фандинга
func (a *CounterAnalyzer) calculateAverageFunding(data []types.PriceData) float64 {
	var totalFunding float64
	var count int

	for _, point := range data {
		if point.FundingRate != 0 {
			totalFunding += point.FundingRate
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return totalFunding / float64(count)
}

// calculateNextFundingTime рассчитывает время следующего фандинга
func (a *CounterAnalyzer) calculateNextFundingTime() time.Time {
	now := time.Now().UTC()

	// Фандинг в 00:00, 08:00, 16:00 UTC
	hour := now.Hour()
	var nextHour int

	switch {
	case hour < 8:
		nextHour = 8
	case hour < 16:
		nextHour = 16
	default:
		// Завтра в 00:00
		nextHour = 0
		now = now.Add(24 * time.Hour)
	}

	// Создаем время следующего фандинга
	nextTime := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		nextHour,
		0, 0, 0,
		time.UTC,
	)

	return nextTime
}

// checkAndResetPeriod проверяет и сбрасывает счетчик если период истек или превышен лимит
func (a *CounterAnalyzer) checkAndResetPeriod(counter *internalCounter, period CounterPeriod, maxSignals int) {
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
func (a *CounterAnalyzer) calculateMaxSignals(period CounterPeriod, basePeriodMinutes int) int {
	// Выбранный период / базовый период = сигнал
	totalPossibleSignals := period.GetMinutes() / basePeriodMinutes

	// Ограничиваем 5-15 сигналами
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

func (a *CounterAnalyzer) getCurrentPeriod() CounterPeriod {
	periodStr := SafeGetString(a.config.CustomSettings["analysis_period"], "15m")
	return CounterPeriod(periodStr)
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

// canSendNotification проверяет лимит уведомлений
func (a *CounterAnalyzer) canSendNotification(symbol string, signalType CounterSignalType) bool {
	// Можно добавить логику ограничения частоты уведомлений
	// если требуется (например, не чаще 1 раза в 30 секунд)
	return true
}

// updateNotificationSent обновляет время последнего уведомления
func (a *CounterAnalyzer) updateNotificationSent(symbol string, signalType CounterSignalType) {
	// Можно добавить кэш времени последнего уведомления
	// для ограничения частоты
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
func (a *CounterAnalyzer) SetAnalysisPeriod(period CounterPeriod) {
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
func (a *CounterAnalyzer) resetAllCountersForPeriod(newPeriod CounterPeriod) {
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
func (a *CounterAnalyzer) GetCounterStats(symbol string) (SignalCounter, bool) {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		return SignalCounter{}, false
	}

	counter.RLock()
	defer counter.RUnlock()

	// Возвращаем копию данных без мьютекса
	return SignalCounter{
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
func (a *CounterAnalyzer) GetAllCounters() map[string]SignalCounter {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]SignalCounter)
	for symbol, counter := range a.counters {
		counter.RLock()

		// Создаем копию без мьютекса
		result[symbol] = SignalCounter{
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

// getPeriodDuration возвращает длительность периода
func (a *CounterAnalyzer) getPeriodDuration(period CounterPeriod) time.Duration {
	return period.GetDuration()
}

// getDirectionFromSignalType преобразует тип сигнала в направление
func (a *CounterAnalyzer) getDirectionFromSignalType(signalType CounterSignalType) string {
	switch signalType {
	case CounterTypeGrowth:
		return "growth"
	case CounterTypeFall:
		return "fall"
	default:
		return "neutral"
	}
}

// DefaultCounterConfig - конфигурация по умолчанию
var DefaultCounterConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        0.7,
	MinConfidence: 10.0,
	MinDataPoints: 2,
	CustomSettings: map[string]interface{}{
		"base_period_minutes":    1,           // Базовый период 1 минута
		"analysis_period":        "15m",       // По умолчанию 15 минут
		"growth_threshold":       0.1,         // Порог роста 0.1%
		"fall_threshold":         0.1,         // Порог падения 0.1%
		"track_growth":           true,        // Отслеживать рост
		"track_fall":             true,        // Отслеживать падение
		"notify_on_signal":       true,        // Уведомлять при каждом сигнале
		"notification_threshold": 1,           // Уведомлять на каждый сигнал
		"chart_provider":         "coinglass", // Основная система - coinglass
		"exchange":               "bybit",     // Биржа по умолчанию
		"include_oi":             true,        // Включать открытый интерес
		"include_volume":         true,        // Включать объем
		"include_funding":        true,        // Включать фандинг
	},
}

// ============== Методы CounterPeriod ==============

// GetMinutes возвращает количество минут для периода
func (cp CounterPeriod) GetMinutes() int {
	switch cp {
	case Period5Min:
		return 5
	case Period15Min:
		return 15
	case Period30Min:
		return 30
	case Period1Hour:
		return 60
	case Period4Hours:
		return 240
	case Period1Day:
		return 1440
	default:
		return 15 // По умолчанию 15 минут
	}
}

// GetDuration возвращает длительность периода как time.Duration
func (cp CounterPeriod) GetDuration() time.Duration {
	return time.Duration(cp.GetMinutes()) * time.Minute
}

// ToString возвращает строковое представление периода
func (cp CounterPeriod) ToString() string {
	switch cp {
	case Period5Min:
		return "5 минут"
	case Period15Min:
		return "15 минут"
	case Period30Min:
		return "30 минут"
	case Period1Hour:
		return "1 час"
	case Period4Hours:
		return "4 часа"
	case Period1Day:
		return "1 день"
	default:
		return "15 минут"
	}
}

// ============== Методы internalCounter ==============

// Lock блокирует счетчик для записи
func (c *internalCounter) Lock() {
	c.mu.Lock()
}

// Unlock разблокирует счетчик для записи
func (c *internalCounter) Unlock() {
	c.mu.Unlock()
}

// RLock блокирует счетчик для чтения
func (c *internalCounter) RLock() {
	c.mu.RLock()
}

// RUnlock разблокирует счетчика для чтения
func (c *internalCounter) RUnlock() {
	c.mu.RUnlock()
}
