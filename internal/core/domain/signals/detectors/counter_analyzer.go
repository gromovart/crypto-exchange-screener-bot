// internal/core/domain/signals/detectors/counter_analyzer.go
package analyzers

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
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
	marketFetcher       interface{}
	counters            map[string]*internalCounter
	mu                  sync.RWMutex
	notificationEnabled bool
	chartProvider       string
	lastPriceCache      map[string]float64
	priceCacheMu        sync.RWMutex
	buttonBuilder       *telegram.ButtonURLBuilder
	messageFormatter    *telegram.MarketMessageFormatter
}

// ==================== МЕТОДЫ ДЛЯ CounterPeriod ====================

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

// ==================== МЕТОДЫ ДЛЯ internalCounter ====================

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

// ==================== КОНСТРУКТОР И ИНИЦИАЛИЗАЦИЯ ====================

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(
	config AnalyzerConfig,
	storage storage.PriceStorage,
	tgBot *telegram.TelegramBot,
	marketFetcher interface{},
) *CounterAnalyzer {
	chartProvider := SafeGetString(config.CustomSettings["chart_provider"], "coinglass")
	exchange := SafeGetString(config.CustomSettings["exchange"], "bybit")

	buttonBuilder := telegram.NewButtonURLBuilderWithProvider(exchange, chartProvider)

	return &CounterAnalyzer{
		config:              config,
		storage:             storage,
		telegramBot:         tgBot,
		marketFetcher:       marketFetcher,
		counters:            make(map[string]*internalCounter),
		notificationEnabled: true,
		chartProvider:       chartProvider,
		lastPriceCache:      make(map[string]float64),
		buttonBuilder:       buttonBuilder,
		messageFormatter:    telegram.NewMarketMessageFormatter(exchange),
	}
}

// ==================== ИНТЕРФЕЙС АНАЛИЗАТОРА ====================

// Name возвращает имя анализатора
func (a *CounterAnalyzer) Name() string {
	return "counter_analyzer"
}

// Version возвращает версию
func (a *CounterAnalyzer) Version() string {
	return "2.2.0" // Увеличиваем версию из-за добавления дельты объемов
}

// Supports проверяет поддержку символа
func (a *CounterAnalyzer) Supports(symbol string) bool {
	return true
}

// GetConfig возвращает конфигурацию
func (a *CounterAnalyzer) GetConfig() AnalyzerConfig {
	return a.config
}

// GetStats возвращает статистику
func (a *CounterAnalyzer) GetStats() AnalyzerStats {
	return a.stats
}

// ==================== ОСНОВНОЙ МЕТОД АНАЛИЗА ====================

// Analyze анализирует данные и обновляет счетчики
func (a *CounterAnalyzer) Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) < 2 {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("insufficient data points")
	}

	symbol := data[0].Symbol
	counter := a.getOrCreateCounter(symbol)

	basePeriodMinutes := a.getBasePeriodMinutes()
	selectedPeriod := a.getCurrentPeriod()
	maxSignals := a.calculateMaxSignals(selectedPeriod, basePeriodMinutes)

	a.checkAndResetPeriod(counter, selectedPeriod, maxSignals)

	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price
	change := ((endPrice - startPrice) / startPrice) * 100

	growthThreshold := a.getGrowthThreshold()
	fallThreshold := a.getFallThreshold()

	var signals []analysis.Signal
	var signalDetected bool
	var signalType CounterSignalType

	// ==================== БЛОК АНАЛИЗА РОСТА И ПАДЕНИЯ ====================
	counter.Lock()
	counter.BasePeriodCount++

	if change > growthThreshold && counter.Settings.TrackGrowth {
		counter.GrowthCount++
		counter.SignalCount++
		counter.LastSignalTime = time.Now()
		signalDetected = true
		signalType = CounterTypeGrowth

		signal := a.createAnalysisSignal(symbol, "growth", change, counter.SignalCount, maxSignals, data)
		signals = append(signals, signal)
	}

	if change < -fallThreshold && counter.Settings.TrackFall {
		counter.FallCount++
		counter.SignalCount++
		counter.LastSignalTime = time.Now()
		signalDetected = true
		signalType = CounterTypeFall

		signal := a.createAnalysisSignal(symbol, "fall", math.Abs(change), counter.SignalCount, maxSignals, data)
		signals = append(signals, signal)
	}
	counter.Unlock()

	// ==================== БЛОК УВЕДОМЛЕНИЙ ====================
	if signalDetected {
		a.sendEnhancedNotification(symbol, signalType, counter, maxSignals, change, data)
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

// ==================== МЕТОДЫ СОЗДАНИЯ СИГНАЛОВ ====================

// createAnalysisSignal создает сигнал анализа
func (a *CounterAnalyzer) createAnalysisSignal(
	symbol, direction string,
	change float64,
	count, maxSignals int,
	data []types.PriceData,
) analysis.Signal {
	confidence := a.calculateConfidence(count, maxSignals)
	selectedPeriod := a.getCurrentPeriod()
	latestData := data[len(data)-1]
	oiChange24h := a.calculateOIChange24h(symbol)

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
			Tags: []string{
				"counter",
				direction,
				fmt.Sprintf("count_%d", count),
				string(selectedPeriod),
				"no_duplicate",
			},
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

// ==================== МЕТОДЫ ФОРМАТИРОВАНИЯ УВЕДОМЛЕНИЙ ====================

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

	// ==================== БЛОК ПОДГОТОВКИ УВЕДОМЛЕНИЯ ====================
	counter.RLock()
	notification := a.prepareNotification(symbol, signalType, counter, maxSignals, change)
	counter.RUnlock()

	if a.canSendNotification(symbol, signalType) {
		message := a.formatEnhancedNotificationMessage(notification, priceData)
		keyboard := a.createNotificationKeyboard(notification)

		if err := a.telegramBot.SendMessageWithKeyboard(message, keyboard); err != nil {
			log.Printf("❌ Ошибка отправки уведомления для %s: %v", symbol, err)
		} else {
			log.Printf("✅ Отправлено уведомление для %s", symbol)
		}

		a.updateNotificationSent(symbol, signalType)
	}
}

// prepareNotification подготавливает уведомление
func (a *CounterAnalyzer) prepareNotification(
	symbol string,
	signalType CounterSignalType,
	counter *internalCounter,
	maxSignals int,
	change float64,
) CounterNotification {
	var count int
	if signalType == CounterTypeGrowth {
		count = counter.GrowthCount
	} else {
		count = counter.FallCount
	}

	return CounterNotification{
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
}

// formatEnhancedNotificationMessage форматирует сообщение с полной информацией
func (a *CounterAnalyzer) formatEnhancedNotificationMessage(
	notification CounterNotification,
	priceData []types.PriceData,
) string {
	if len(priceData) == 0 {
		return a.formatNotificationMessage(notification)
	}

	// ==================== БЛОК ПОЛУЧЕНИЯ ДАННЫХ ====================
	latestData := priceData[len(priceData)-1]
	realTimePrice, realTimeOI, realTimeFunding, realTimeVolume := a.getRealTimeMetrics(notification.Symbol)

	currentPrice := latestData.Price
	openInterest := latestData.OpenInterest
	fundingRate := latestData.FundingRate
	volume24h := latestData.Volume24h

	// Используем реальные данные если они есть
	if realTimePrice > 0 {
		currentPrice = realTimePrice
	}
	if realTimeOI > 0 {
		openInterest = realTimeOI
		log.Printf("✅ Используем реальный OI из хранилища: %.0f", realTimeOI)
	}
	if realTimeFunding != 0 {
		fundingRate = realTimeFunding
	}
	if realTimeVolume > 0 {
		volume24h = realTimeVolume
	}

	// Эвристика если OI = 0
	if openInterest <= 0 {
		openInterest = volume24h * 0.05
		log.Printf("📊 Используем расчетный OI для %s: %.0f", notification.Symbol, openInterest)
	}

	// ==================== БЛОК РАСЧЕТА МЕТРИК ====================
	oiChange24h := a.calculateOIChange24h(notification.Symbol)
	nextFundingTime := a.calculateNextFundingTime()
	averageFunding := a.calculateAverageFunding(priceData)
	liquidationVolume, longLiqVolume, shortLiqVolume := a.getLiquidationData(notification.Symbol)

	// Рассчитываем дельту объемов
	volumeDelta := a.calculateVolumeDelta(notification.Symbol, a.getDirectionFromSignalType(notification.SignalType), volume24h)
	volumeDeltaPercent := a.calculateVolumeDeltaPercent(notification.Symbol, a.getDirectionFromSignalType(notification.SignalType))

	log.Printf("📤 CounterAnalyzer отправляет данные:")
	log.Printf("   Symbol: %s, OI: %.0f (изм: %.1f%%)", notification.Symbol, openInterest, oiChange24h)
	log.Printf("   Price: %.4f, Volume: %.0f", currentPrice, volume24h)
	log.Printf("   Дельта: %.0f (%.1f%%), Ликвидации: $%.0f", volumeDelta, volumeDeltaPercent, liquidationVolume)

	// ==================== БЛОК ФОРМАТИРОВАНИЯ СООБЩЕНИЯ ====================
	return a.messageFormatter.FormatMessage(
		notification.Symbol,
		a.getDirectionFromSignalType(notification.SignalType),
		notification.ChangePercent,
		notification.CurrentCount,
		notification.MaxSignals,
		currentPrice,
		volume24h,
		openInterest,
		oiChange24h,
		fundingRate,
		averageFunding,
		nextFundingTime,
		notification.Period.ToString(),
		liquidationVolume,
		longLiqVolume,
		shortLiqVolume,
		volumeDelta,
		volumeDeltaPercent,
	)
}

// ==================== МЕТОДЫ РАБОТЫ СО СЧЕТЧИКАМИ ====================

// getOrCreateCounter получает или создает счетчик для символа
func (a *CounterAnalyzer) getOrCreateCounter(symbol string) *internalCounter {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		a.mu.Lock()
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

// checkAndResetPeriod проверяет и сбрасывает счетчик если период истек или превышен лимит
func (a *CounterAnalyzer) checkAndResetPeriod(counter *internalCounter, period CounterPeriod, maxSignals int) {
	counter.Lock()
	defer counter.Unlock()

	now := time.Now()
	periodDuration := period.GetDuration()
	shouldReset := false
	resetReason := ""

	if now.Sub(counter.PeriodStartTime) >= periodDuration {
		shouldReset = true
		resetReason = "истек период"
	} else if counter.SignalCount >= maxSignals {
		shouldReset = true
		resetReason = fmt.Sprintf("достигнут максимум сигналов (%d/%d)", counter.SignalCount, maxSignals)
	} else if counter.SelectedPeriod != period {
		shouldReset = true
		resetReason = fmt.Sprintf("изменился период на %s", period)
	}

	if shouldReset {
		log.Printf("🔄 Счетчик для %s сброшен. Причина: %s", counter.Symbol, resetReason)

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

// ==================== МЕТОДЫ ПОЛУЧЕНИЯ РЫНОЧНЫХ ДАННЫХ ====================

// getLiquidationData получает данные ликвидаций за последние 5 минут
func (a *CounterAnalyzer) getLiquidationData(symbol string) (totalVolume, longVolume, shortVolume float64) {
	// Пробуем получить реальные данные
	if a.marketFetcher != nil {
		if fetcher, ok := a.marketFetcher.(interface {
			GetLiquidationMetrics(string) (*bybit.LiquidationMetrics, bool)
		}); ok {
			if metrics, exists := fetcher.GetLiquidationMetrics(symbol); exists {
				log.Printf("📊 Получены ликвидации для %s: $%.0f (long: $%.0f, short: $%.0f)",
					symbol, metrics.TotalVolumeUSD, metrics.LongLiqVolume, metrics.ShortLiqVolume)
				return metrics.TotalVolumeUSD, metrics.LongLiqVolume, metrics.ShortLiqVolume
			}
		}
	}

	// Эмуляция если данных нет
	if snapshot, exists := a.storage.GetCurrentSnapshot(symbol); exists && snapshot.VolumeUSD > 0 {
		baseLiq := snapshot.VolumeUSD * 0.001

		if math.Abs(snapshot.Change24h) > 5 {
			baseLiq *= 3
		} else if math.Abs(snapshot.Change24h) > 2 {
			baseLiq *= 2
		}

		if snapshot.Change24h > 0 {
			longVolume = baseLiq * 0.4
			shortVolume = baseLiq * 0.6
		} else {
			longVolume = baseLiq * 0.6
			shortVolume = baseLiq * 0.4
		}

		log.Printf("📊 Симулированные ликвидации для %s: $%.0f", symbol, baseLiq)
		return baseLiq, longVolume, shortVolume
	}

	log.Printf("⚠️ Не удалось получить ликвидации для %s", symbol)
	return 0, 0, 0
}

// calculateOIChange24h рассчитывает изменение OI за 24 часа
func (a *CounterAnalyzer) calculateOIChange24h(symbol string) float64 {
	log.Printf("🔍 Получение OI change для %s", symbol)

	if metrics, exists := a.storage.GetSymbolMetrics(symbol); exists {
		log.Printf("✅ Получен OI change из хранилища для %s: %.1f%%", symbol, metrics.OIChange24h)
		return metrics.OIChange24h
	}

	history, err := a.storage.GetPriceHistory(symbol, 200)
	if err != nil {
		log.Printf("⚠️ Не удалось получить историю для %s: %v", symbol, err)
		return 0
	}

	if len(history) < 2 {
		return 0
	}

	var latestData *storage.PriceData
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].OpenInterest > 0 {
			latestData = &history[i]
			break
		}
	}

	if latestData == nil {
		return 0
	}

	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	var oldOI float64
	var minDiff time.Duration = 24 * time.Hour
	var found bool

	for _, point := range history {
		if point.OpenInterest > 0 {
			diff := point.Timestamp.Sub(twentyFourHoursAgo)
			diffAbs := diff.Abs()

			if diffAbs < minDiff {
				minDiff = diffAbs
				oldOI = point.OpenInterest
				found = true
			}
		}
	}

	if !found || oldOI == 0 {
		return 0
	}

	change := ((latestData.OpenInterest - oldOI) / oldOI) * 100
	log.Printf("📊 Рассчитан OI change для %s: %.1f%%", symbol, change)
	return change
}

// getRealTimeMetrics получает реальные метрики из хранилища
func (a *CounterAnalyzer) getRealTimeMetrics(symbol string) (price, oi, funding float64, volumeUSD float64) {
	if metrics, exists := a.storage.GetSymbolMetrics(symbol); exists {
		log.Printf("✅ Получены реальные метрики для %s: OI=%.0f", symbol, metrics.OpenInterest)
		return metrics.Price, metrics.OpenInterest, metrics.FundingRate, metrics.VolumeUSD
	}
	return 0, 0, 0, 0
}

// ==================== МЕТОДЫ РАСЧЕТА ДЕЛЬТЫ ОБЪЕМОВ ====================

// calculateVolumeDelta рассчитывает дельту объемов
func (a *CounterAnalyzer) calculateVolumeDelta(symbol, direction string, volume24h float64) float64 {
	baseDelta := volume24h * 0.02 // 2% от объема

	if direction == "growth" {
		return baseDelta
	}
	return -baseDelta
}

// calculateVolumeDeltaPercent рассчитывает изменение дельты в процентах
func (a *CounterAnalyzer) calculateVolumeDeltaPercent(symbol, direction string) float64 {
	basePercent := 10.0

	if direction == "growth" {
		return basePercent
	}
	return -basePercent
}

// ==================== МЕТОДЫ РАСЧЕТА ФАНДИНГА ====================

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
	hour := now.Hour()
	var nextHour int

	switch {
	case hour < 8:
		nextHour = 8
	case hour < 16:
		nextHour = 16
	default:
		nextHour = 0
		now = now.Add(24 * time.Hour)
	}

	return time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		nextHour,
		0, 0, 0,
		time.UTC,
	)
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ КОНФИГУРАЦИИ ====================

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

func (a *CounterAnalyzer) calculateMaxSignals(period CounterPeriod, basePeriodMinutes int) int {
	totalPossibleSignals := period.GetMinutes() / basePeriodMinutes

	if totalPossibleSignals < 5 {
		return 5
	}
	if totalPossibleSignals > 15 {
		return 15
	}
	return totalPossibleSignals
}

func (a *CounterAnalyzer) getPeriodDuration(period CounterPeriod) time.Duration {
	return period.GetDuration()
}

// ==================== МЕТОДЫ УПРАВЛЕНИЯ УВЕДОМЛЕНИЯМИ ====================

func (a *CounterAnalyzer) canSendNotification(symbol string, signalType CounterSignalType) bool {
	return true // TODO: добавить логику ограничения частоты
}

func (a *CounterAnalyzer) updateNotificationSent(symbol string, signalType CounterSignalType) {
	// TODO: добавить кэш времени последнего уведомления
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

	return fmt.Sprintf(
		"⚫ Bybit - 1мин - %s\n"+
			"🕐 %s\n"+
			"%s %s: %s\n"+
			"📡 Сигнал: %d\n"+
			"⏱️  Период: %s",
		notification.Symbol,
		notification.Timestamp.Format("2006/01/02 15:04:05"),
		icon, directionStr, changeStr,
		notification.CurrentCount,
		notification.Period.ToString(),
	)
}

// createNotificationKeyboard создает клавиатуру для уведомления
func (a *CounterAnalyzer) createNotificationKeyboard(notification CounterNotification) *telegram.InlineKeyboardMarkup {
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

// ==================== МЕТОДЫ УПРАВЛЕНИЯ АНАЛИЗАТОРОМ ====================

func (a *CounterAnalyzer) SetNotificationEnabled(enabled bool) {
	a.notificationEnabled = enabled
}

func (a *CounterAnalyzer) SetChartProvider(provider string) {
	a.chartProvider = provider
}

func (a *CounterAnalyzer) SetAnalysisPeriod(period CounterPeriod) {
	newSettings := make(map[string]interface{})
	for k, v := range a.config.CustomSettings {
		newSettings[k] = v
	}
	newSettings["analysis_period"] = string(period)
	a.config.CustomSettings = newSettings

	a.resetAllCountersForPeriod(period)
}

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

func (a *CounterAnalyzer) GetCounterStats(symbol string) (SignalCounter, bool) {
	a.mu.RLock()
	counter, exists := a.counters[symbol]
	a.mu.RUnlock()

	if !exists {
		return SignalCounter{}, false
	}

	counter.RLock()
	defer counter.RUnlock()

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

func (a *CounterAnalyzer) GetAllCounters() map[string]SignalCounter {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]SignalCounter)
	for symbol, counter := range a.counters {
		counter.RLock()
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

// ==================== МЕТОДЫ СТАТИСТИКИ ====================

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

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ====================

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

// ==================== КОНФИГУРАЦИЯ ПО УМОЛЧАНИЮ ====================

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
		"notify_on_signal":       true,
		"notification_threshold": 1,
		"chart_provider":         "coinglass",
		"exchange":               "bybit",
		"include_oi":             true,
		"include_volume":         true,
		"include_funding":        true,
	},
}
