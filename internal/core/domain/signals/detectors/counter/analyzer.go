// internal/core/domain/signals/detectors/counter/analyzer.go
package counter

import (
	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"strings"
	"sync"
	"time"
)

// Dependencies зависимости для CounterAnalyzer
type Dependencies struct {
	Storage             storage.PriceStorageInterface
	EventBus            types.EventBus
	CandleSystem        *candle.CandleSystem
	MarketFetcher       interface{}
	VolumeCalculator    *calculator.VolumeDeltaCalculator
	TechnicalCalculator *calculator.TechnicalCalculator
}

// CounterAnalyzer - анализатор счетчика сигналов
type CounterAnalyzer struct {
	config common.AnalyzerConfig
	deps   Dependencies

	// Статистика отправленных сигналов
	stats              common.AnalyzerStats
	sentStatsMu        sync.RWMutex
	sentSignalsCount   int
	sentStatsStartTime time.Time
	lastLogTime        time.Time

	// Статистика вызовов Analyze()
	analyzeCallsCount  int
	analyzeTotalPoints int
	analyzeTotalTime   time.Duration
	analyzeCallMu      sync.RWMutex

	// Агрегированная статистика
	aggregatedStats AggregatedStats

	// ✅ СЧЕТЧИКИ ДЛЯ ОТЛАДКИ AnalyzeCandle
	candleStatsMu sync.RWMutex
	candleStats   CandleAnalyzeStats
}

// CandleAnalyzeStats статистика анализа свечей для отладки
type CandleAnalyzeStats struct {
	TotalCalls       int // Всего вызовов AnalyzeCandle
	NoCandleData     int // Нет свечей
	UnrealCandle     int // Нереальные свечи
	AlreadyProcessed int // Уже обработаны
	BelowThreshold   int // Ниже порога
	GrowthSignal     int // Ростовые сигналы
	FallSignal       int // Падающие сигналы
	MarkCandleError  int // Ошибки отметки свечи
	GetCandleError   int // Ошибки получения свечи
}

// AggregatedStats структура для агрегированной статистики
type AggregatedStats struct {
	TotalSymbols       int
	AnalyzeAttempts    int
	SignalsFound       int
	NoDataErrors       int
	UnrealCandleErrors int
	OtherErrors        int
	SignalsCreated     int
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(
	config common.AnalyzerConfig,
	deps Dependencies,
) *CounterAnalyzer {
	// ✅ ПРОВЕРЯЕМ И СОЗДАЕМ VolumeCalculator если не передан
	if deps.VolumeCalculator == nil && deps.MarketFetcher != nil && deps.Storage != nil {
		logger.Warn("🔧 [CounterAnalyzer] Создаем VolumeDeltaCalculator")
		deps.VolumeCalculator = calculator.NewVolumeDeltaCalculator(deps.MarketFetcher, deps.Storage)
	}

	// ✅ ПРОВЕРЯЕМ И СОЗДАЕМ TechnicalCalculator если не передан
	if deps.TechnicalCalculator == nil {
		logger.Warn("🔧 [CounterAnalyzer] Создаем TechnicalCalculator")
		deps.TechnicalCalculator = calculator.NewTechnicalCalculator()
	}

	analyzer := &CounterAnalyzer{
		config: config,
		deps:   deps,
		stats: common.AnalyzerStats{
			TotalCalls:   0,
			SuccessCount: 0,
			ErrorCount:   0,
			TotalTime:    0,
			AverageTime:  0,
			LastCallTime: time.Time{},
		},
		sentSignalsCount:   0,
		sentStatsStartTime: time.Now(),
		lastLogTime:        time.Now(),
		analyzeCallsCount:  0, // ✅ Инициализация счетчиков
		analyzeTotalPoints: 0, // ✅
		analyzeTotalTime:   0, // ✅
	}

	logger.Warn("✅ [CounterAnalyzer] Создан анализатор счетчика")
	return analyzer
}

// Analyze основной метод анализа
func (a *CounterAnalyzer) Analyze(data []storage.PriceDataInterface, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	a.analyzeCallMu.Lock()
	a.analyzeCallsCount++
	a.analyzeTotalPoints += len(data)
	a.analyzeCallMu.Unlock()

	a.stats.TotalCalls++
	defer func() {
		a.stats.LastCallTime = time.Now()
		a.stats.TotalTime += time.Since(startTime)
		if a.stats.TotalCalls > 0 {
			a.stats.AverageTime = a.stats.TotalTime / time.Duration(a.stats.TotalCalls)
		}

		a.analyzeCallMu.Lock()
		a.analyzeTotalTime += time.Since(startTime)
		a.analyzeCallMu.Unlock()
	}()

	// Обновляем конфигурацию
	a.config = config

	var signals []analysis.Signal
	supportedPeriods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	// Локальный счетчик для этого вызова
	localSentCount := 0
	candleAnalyzeAttempts := 0
	candleAnalyzeSuccess := 0
	candleErrors := 0
	candleNoDataErrors := 0
	candleUnrealErrors := 0

	// СЧЕТЧИК ДЛЯ АГРЕГИРОВАННОЙ СТАТИСТИКИ
	symbolsProcessed := len(data)

	// Анализируем каждую точку
	for _, point := range data {
		// Анализируем каждый период
		for _, period := range supportedPeriods {
			candleAnalyzeAttempts++
			signal, err := a.AnalyzeCandle(point.GetSymbol(), period)
			if err != nil {
				// АГРЕГИРУЕМ ОШИБКИ БЕЗ ЛОГОВ
				errStr := err.Error()
				if strings.Contains(errStr, "нет закрытых свечей") {
					candleNoDataErrors++
				} else if strings.Contains(errStr, "нереальная свеча") {
					candleUnrealErrors++
				} else {
					candleErrors++
				}
				continue
			}

			if signal != nil {
				candleAnalyzeSuccess++
				signals = append(signals, *signal)

				// Публикуем сигнал в EventBus
				a.PublishRawCounterSignal(*signal, period)

				// Увеличиваем локальный счетчик
				localSentCount++
			}
		}
	}

	// ✅ СБОР АГРЕГИРОВАННОЙ СТАТИСТИКИ ДЛЯ ИНТЕРВАЛА (без вывода здесь)
	a.analyzeCallMu.Lock()
	a.aggregatedStats = AggregatedStats{
		TotalSymbols:       symbolsProcessed,
		AnalyzeAttempts:    candleAnalyzeAttempts,
		SignalsFound:       candleAnalyzeSuccess,
		NoDataErrors:       candleNoDataErrors,
		UnrealCandleErrors: candleUnrealErrors,
		OtherErrors:        candleErrors,
		SignalsCreated:     localSentCount,
	}
	a.analyzeCallMu.Unlock()

	a.stats.SuccessCount++

	// Обновляем общую статистику отправленных сигналов
	if localSentCount > 0 {
		a.sentStatsMu.Lock()
		a.sentSignalsCount += localSentCount
		a.sentStatsMu.Unlock()
	}

	// ✅ ТОЛЬКО АГРЕГИРОВАННОЕ ЛОГИРОВАНИЕ РАЗ В 5 СЕКУНД
	a.logAggregatedStatsIfNeeded(5 * time.Second)

	return signals, nil
}

// logAggregatedAnalyzeStatsIfNeeded логирует агрегированную статистику вызовов Analyze() раз в 10 секунд
func (a *CounterAnalyzer) logAggregatedAnalyzeStatsIfNeeded(interval time.Duration) {
	now := time.Now()
	a.analyzeCallMu.RLock()
	shouldLog := now.Sub(a.lastLogTime) >= interval && a.analyzeCallsCount > 0
	a.analyzeCallMu.RUnlock()

	if !shouldLog {
		return
	}

	a.analyzeCallMu.Lock()
	defer a.analyzeCallMu.Unlock()

	// Проверяем еще раз после блокировки
	if now.Sub(a.lastLogTime) < interval || a.analyzeCallsCount == 0 {
		return
	}

	// Рассчитываем статистику
	var avgPointsPerCall float64
	var avgTimePerCall time.Duration
	if a.analyzeCallsCount > 0 {
		avgPointsPerCall = float64(a.analyzeTotalPoints) / float64(a.analyzeCallsCount)
		avgTimePerCall = a.analyzeTotalTime / time.Duration(a.analyzeCallsCount)
	}

	// ✅ ИСПОЛЬЗУЕМ WARN УРОВЕНЬ ДЛЯ АГРЕГИРОВАННЫХ ЛОГОВ
	logger.Warn("📊 [CounterAnalyzer] Статистика за последние %v:", interval)
	logger.Warn("   📞 Вызовов Analyze: %d", a.analyzeCallsCount)
	logger.Warn("   📍 Обработано точек: %d", a.analyzeTotalPoints)
	logger.Warn("   ⏱️  Среднее время: %v", avgTimePerCall.Round(time.Millisecond))
	logger.Warn("   📈 Среднее точек/вызов: %.1f", avgPointsPerCall)
	logger.Warn("   ⚡ Скорость: %.1f точек/сек",
		float64(a.analyzeTotalPoints)/interval.Seconds())

	// Сбрасываем статистику для следующего интервала
	a.analyzeCallsCount = 0
	a.analyzeTotalPoints = 0
	a.analyzeTotalTime = 0
	a.lastLogTime = now
}

// logAggregatedStatsIfNeeded логирует агрегированную статистику раз в 10 секунд
func (a *CounterAnalyzer) logAggregatedStatsIfNeeded(interval time.Duration) {
	now := time.Now()
	a.analyzeCallMu.RLock()
	shouldLog := now.Sub(a.lastLogTime) >= interval && a.analyzeCallsCount > 0
	a.analyzeCallMu.RUnlock()

	if !shouldLog {
		return
	}

	a.analyzeCallMu.Lock()
	defer a.analyzeCallMu.Unlock()

	// Проверяем еще раз после блокировки
	if now.Sub(a.lastLogTime) < interval || a.analyzeCallsCount == 0 {
		return
	}

	// Рассчитываем агрегированную статистику
	var avgPointsPerCall float64
	var avgTimePerCall time.Duration
	if a.analyzeCallsCount > 0 {
		avgPointsPerCall = float64(a.analyzeTotalPoints) / float64(a.analyzeCallsCount)
		avgTimePerCall = a.analyzeTotalTime / time.Duration(a.analyzeCallsCount)
	}

	// ✅ ОСНОВНАЯ АГРЕГИРОВАННАЯ СТАТИСТИКА
	logger.Warn("📊 [CounterAnalyzer] Статистика за последние %v:", interval)
	logger.Warn("   📞 Вызовов Analyze: %d", a.analyzeCallsCount)
	logger.Warn("   📍 Обработано символов: %d", a.analyzeTotalPoints)
	logger.Warn("   ⏱️  Среднее время: %v", avgTimePerCall.Round(time.Millisecond))
	logger.Warn("   📈 Среднее символов/вызов: %.1f", avgPointsPerCall)
	logger.Warn("   ⚡ Скорость: %.1f символов/сек", float64(a.analyzeTotalPoints)/interval.Seconds())

	// ✅ ДОБАВЛЯЕМ ОТЛАДОЧНУЮ СТАТИСТИКУ AnalyzeCandle
	a.candleStatsMu.Lock()
	candleStats := a.candleStats
	a.candleStatsMu.Unlock()

	if candleStats.TotalCalls > 0 {
		logger.Warn("   🕯️  Анализ свечей (вызовов: %d):", candleStats.TotalCalls)

		// Проценты для каждого типа результата
		noDataPercent := float64(candleStats.NoCandleData) / float64(candleStats.TotalCalls) * 100
		unrealPercent := float64(candleStats.UnrealCandle) / float64(candleStats.TotalCalls) * 100
		processedPercent := float64(candleStats.AlreadyProcessed) / float64(candleStats.TotalCalls) * 100
		belowThresholdPercent := float64(candleStats.BelowThreshold) / float64(candleStats.TotalCalls) * 100
		growthPercent := float64(candleStats.GrowthSignal) / float64(candleStats.TotalCalls) * 100
		fallPercent := float64(candleStats.FallSignal) / float64(candleStats.TotalCalls) * 100
		markErrorPercent := float64(candleStats.MarkCandleError) / float64(candleStats.TotalCalls) * 100
		getErrorPercent := float64(candleStats.GetCandleError) / float64(candleStats.TotalCalls) * 100

		// Выводим только значимые категории (>0%)
		if candleStats.NoCandleData > 0 {
			logger.Warn("      • Нет свечей: %d (%.1f%%)", candleStats.NoCandleData, noDataPercent)
		}
		if candleStats.UnrealCandle > 0 {
			logger.Warn("      • Нереальные свечи: %d (%.1f%%)", candleStats.UnrealCandle, unrealPercent)
		}
		if candleStats.AlreadyProcessed > 0 {
			logger.Warn("      • Уже обработаны: %d (%.1f%%)", candleStats.AlreadyProcessed, processedPercent)
		}
		if candleStats.BelowThreshold > 0 {
			logger.Warn("      • Ниже порога: %d (%.1f%%)", candleStats.BelowThreshold, belowThresholdPercent)
		}
		if candleStats.GrowthSignal > 0 {
			logger.Warn("      • Ростовые сигналы: %d (%.1f%%)", candleStats.GrowthSignal, growthPercent)
		}
		if candleStats.FallSignal > 0 {
			logger.Warn("      • Падающие сигналы: %d (%.1f%%)", candleStats.FallSignal, fallPercent)
		}
		if candleStats.MarkCandleError > 0 {
			logger.Warn("      • Ошибки отметки: %d (%.1f%%)", candleStats.MarkCandleError, markErrorPercent)
		}
		if candleStats.GetCandleError > 0 {
			logger.Warn("      • Ошибки получения: %d (%.1f%%)", candleStats.GetCandleError, getErrorPercent)
		}

		// Проверка суммы процентов
		totalPercent := noDataPercent + unrealPercent + processedPercent + belowThresholdPercent +
			growthPercent + fallPercent + markErrorPercent + getErrorPercent
		logger.Warn("      • Итого: %.1f%% (должно быть ~100%%)", totalPercent)
	}

	// Сбрасываем статистику для следующего интервала
	a.analyzeCallsCount = 0
	a.analyzeTotalPoints = 0
	a.analyzeTotalTime = 0
	a.aggregatedStats = AggregatedStats{}

	// ✅ СБРАСЫВАЕМ СТАТИСТИКУ АНАЛИЗА СВЕЧЕЙ
	a.candleStatsMu.Lock()
	a.candleStats = CandleAnalyzeStats{}
	a.candleStatsMu.Unlock()

	a.lastLogTime = now
}

// Stop останавливает анализатор счетчика
func (a *CounterAnalyzer) Stop() error {
	logger.Warn("🛑 [CounterAnalyzer] Остановка анализатора")

	// ✅ Останавливаем VolumeDeltaCalculator если есть
	if a.deps.VolumeCalculator != nil {
		logger.Warn("🛑 [CounterAnalyzer] Остановка VolumeDeltaCalculator")
		a.deps.VolumeCalculator.Stop()
	}

	// Сбрасываем статистику
	a.sentStatsMu.Lock()
	a.sentSignalsCount = 0
	a.sentStatsStartTime = time.Now()
	a.lastLogTime = time.Now()
	a.sentStatsMu.Unlock()

	// Сбрасываем общую статистику
	a.stats = common.AnalyzerStats{
		TotalCalls:   0,
		SuccessCount: 0,
		ErrorCount:   0,
		TotalTime:    0,
		AverageTime:  0,
		LastCallTime: time.Time{},
	}

	logger.Warn("✅ [CounterAnalyzer] Анализатор остановлен")
	return nil
}

// GetConfig возвращает конфигурацию
func (a *CounterAnalyzer) GetConfig() common.AnalyzerConfig {
	return a.config
}

// GetStats возвращает статистику
func (a *CounterAnalyzer) GetStats() common.AnalyzerStats {
	return a.stats
}

// Name возвращает имя анализатора
func (a *CounterAnalyzer) Name() string {
	return "counter"
}

// Version возвращает версию анализатора
func (a *CounterAnalyzer) Version() string {
	return "1.0.0"
}

// Supports проверяет, поддерживается ли символ
func (a *CounterAnalyzer) Supports(symbol string) bool {
	return true
}
