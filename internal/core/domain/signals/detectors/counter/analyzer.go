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

	// ✅ СЧЕТЧИКИ ДЛЯ ОТЛАДКИ AnalyzeCandle С РАЗДЕЛЕНИЕМ НА ЗАКРЫТЫЕ/АКТИВНЫЕ
	candleStatsMu sync.RWMutex
	candleStats   CandleAnalyzeStats
}

// CandleAnalyzeStats статистика анализа свечей для отладки с разделением
type CandleAnalyzeStats struct {
	TotalCalls int // Всего вызовов AnalyzeCandle

	// Статистика по ЗАКРЫТЫМ свечам
	ClosedCandleStats struct {
		Attempts         int // Попыток анализа
		Success          int // Успешных анализов (сигнал создан)
		NoData           int // Нет закрытых свечей
		Unreal           int // Нереальные свечи
		AlreadyProcessed int // Уже обработаны
		BelowThreshold   int // Ниже порога
		GetCandleError   int // Ошибки получения
		MarkCandleError  int // Ошибки отметки
		GrowthSignals    int // Ростовые сигналы
		FallSignals      int // Падающие сигналы
	}

	// Статистика по АКТИВНЫМ свечам
	ActiveCandleStats struct {
		Attempts         int // Попыток анализа
		Success          int // Успешных анализов (сигнал создан)
		NoActiveCandle   int // Нет активной свечи
		BelowMinTime     int // Меньше минимального времени
		BelowThreshold   int // Ниже порога
		GetCandleError   int // Ошибки получения
		InsufficientData int // Недостаточно данных
		GrowthSignals    int // Ростовые сигналы
		FallSignals      int // Падающие сигналы
	}

	// Интервальная статистика сигналов
	IntervalStats struct {
		StartTime     time.Time
		ClosedSignals int // Сигналов по закрытым свечам
		ActiveSignals int // Сигналов по активным свечам
		TotalSignals  int // Всего сигналов
		GrowthSignals int // Ростовых сигналов
		FallSignals   int // Падающих сигналов
	}
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
		analyzeCallsCount:  0,
		analyzeTotalPoints: 0,
		analyzeTotalTime:   0,
		candleStats: CandleAnalyzeStats{
			IntervalStats: struct {
				StartTime     time.Time
				ClosedSignals int
				ActiveSignals int
				TotalSignals  int
				GrowthSignals int
				FallSignals   int
			}{
				StartTime: time.Now(),
			},
		},
	}

	logger.Warn("✅ [CounterAnalyzer] Создан анализатор счетчика с разделенной статистикой")
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
	// ✅ ДОБАВЛЯЕМ ПЕРИОД 1m
	supportedPeriods := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d"}

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
				if errStr == "нет закрытых свечей" || errStr == "нет активной свечи" {
					candleNoDataErrors++
				} else if errStr == "нереальная свеча" || errStr == "недостаточно данных" {
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

	// ✅ СБОР АГРЕГИРОВАННОЙ СТАТИСТИКИ ДЛЯ ИНТЕРВАЛА
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

// logAggregatedStatsIfNeeded логирует агрегированную статистику с разделением закрытые/активные
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

	// ✅ СТАТИСТИКА ПО ЗАКРЫТЫМ И АКТИВНЫМ СВЕЧАМ
	a.candleStatsMu.Lock()
	candleStats := a.candleStats
	a.candleStatsMu.Unlock()

	// ✅ СТАТИСТИКА ПО ЗАКРЫТЫМ СВЕЧАМ
	if candleStats.ClosedCandleStats.Attempts > 0 {
		closedAttempts := candleStats.ClosedCandleStats.Attempts
		closedSuccessRate := float64(candleStats.ClosedCandleStats.Success) / float64(closedAttempts) * 100
		closedNoDataRate := float64(candleStats.ClosedCandleStats.NoData) / float64(closedAttempts) * 100
		closedProcessedRate := float64(candleStats.ClosedCandleStats.AlreadyProcessed) / float64(closedAttempts) * 100

		logger.Warn("   🕯️  ЗАКРЫТЫЕ свечи (попыток: %d):", closedAttempts)
		logger.Warn("      • Успешно: %d (%.1f%%)", candleStats.ClosedCandleStats.Success, closedSuccessRate)
		logger.Warn("      • Нет данных: %d (%.1f%%)", candleStats.ClosedCandleStats.NoData, closedNoDataRate)
		logger.Warn("      • Уже обработаны: %d (%.1f%%)", candleStats.ClosedCandleStats.AlreadyProcessed, closedProcessedRate)
		logger.Warn("      • Ниже порога: %d", candleStats.ClosedCandleStats.BelowThreshold)
		logger.Warn("      • Ошибки получения: %d", candleStats.ClosedCandleStats.GetCandleError)
		logger.Warn("      • Ростовые: %d, Падающие: %d",
			candleStats.ClosedCandleStats.GrowthSignals,
			candleStats.ClosedCandleStats.FallSignals)
	}

	// ✅ СТАТИСТИКА ПО АКТИВНЫМ СВЕЧАМ
	if candleStats.ActiveCandleStats.Attempts > 0 {
		activeAttempts := candleStats.ActiveCandleStats.Attempts
		activeSuccessRate := float64(candleStats.ActiveCandleStats.Success) / float64(activeAttempts) * 100
		activeNoDataRate := float64(candleStats.ActiveCandleStats.NoActiveCandle) / float64(activeAttempts) * 100
		activeBelowTimeRate := float64(candleStats.ActiveCandleStats.BelowMinTime) / float64(activeAttempts) * 100

		logger.Warn("   🔥 АКТИВНЫЕ свечи (попыток: %d):", activeAttempts)
		logger.Warn("      • Успешно: %d (%.1f%%)", candleStats.ActiveCandleStats.Success, activeSuccessRate)
		logger.Warn("      • Нет активной свечи: %d (%.1f%%)", candleStats.ActiveCandleStats.NoActiveCandle, activeNoDataRate)
		logger.Warn("      • Мало времени: %d (%.1f%%)", candleStats.ActiveCandleStats.BelowMinTime, activeBelowTimeRate)
		logger.Warn("      • Ниже порога: %d", candleStats.ActiveCandleStats.BelowThreshold)
		logger.Warn("      • Ростовые: %d, Падающие: %d",
			candleStats.ActiveCandleStats.GrowthSignals,
			candleStats.ActiveCandleStats.FallSignals)
	}

	// ✅ ИНТЕРВАЛЬНАЯ СТАТИСТИКА СИГНАЛОВ
	if candleStats.IntervalStats.TotalSignals > 0 {
		intervalDuration := now.Sub(candleStats.IntervalStats.StartTime)
		logger.Warn("   📈 СИГНАЛЫ за интервал (%v):", intervalDuration.Round(time.Second))
		logger.Warn("      • Всего: %d", candleStats.IntervalStats.TotalSignals)
		logger.Warn("      • По закрытым свечам: %d", candleStats.IntervalStats.ClosedSignals)
		logger.Warn("      • По активным свечам: %d", candleStats.IntervalStats.ActiveSignals)
		logger.Warn("      • Ростовые: %d, Падающие: %d",
			candleStats.IntervalStats.GrowthSignals,
			candleStats.IntervalStats.FallSignals)
	}

	// Сбрасываем общую статистику для следующего интервала
	a.analyzeCallsCount = 0
	a.analyzeTotalPoints = 0
	a.analyzeTotalTime = 0
	a.aggregatedStats = AggregatedStats{}

	// ✅ СБРАСЫВАЕМ СТАТИСТИКУ СВЕЧЕЙ (сохраняем только интервальную)
	a.candleStatsMu.Lock()
	// Сохраняем интервальную статистику, сбрасываем остальное
	a.candleStats = CandleAnalyzeStats{
		IntervalStats: struct {
			StartTime     time.Time
			ClosedSignals int
			ActiveSignals int
			TotalSignals  int
			GrowthSignals int
			FallSignals   int
		}{
			StartTime: now, // Начинаем новый интервал
		},
	}
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
