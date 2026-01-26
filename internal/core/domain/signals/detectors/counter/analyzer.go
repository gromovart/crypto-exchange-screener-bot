package counter

import (
	"crypto-exchange-screener-bot/internal/core/domain/candle"
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/calculator"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
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
	sentSignalsCount   int       // Общее количество отправленных сигналов
	sentStatsStartTime time.Time // Время начала сбора статистики
	lastLogTime        time.Time // Время последнего логирования статистики
}

// NewCounterAnalyzer создает новый анализатор счетчика
func NewCounterAnalyzer(
	config common.AnalyzerConfig,
	deps Dependencies,
) *CounterAnalyzer {
	// ✅ ПРОВЕРЯЕМ И СОЗДАЕМ VolumeCalculator если не передан
	if deps.VolumeCalculator == nil && deps.MarketFetcher != nil && deps.Storage != nil {
		logger.Info("🔧 Создаем VolumeDeltaCalculator для CounterAnalyzer")
		deps.VolumeCalculator = calculator.NewVolumeDeltaCalculator(deps.MarketFetcher, deps.Storage)
	}

	// ✅ ПРОВЕРЯЕМ И СОЗДАЕМ TechnicalCalculator если не передан
	if deps.TechnicalCalculator == nil {
		logger.Info("🔧 Создаем TechnicalCalculator для CounterAnalyzer")
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
	}

	logger.Info("✅ CounterAnalyzer создан")
	return analyzer
}

// Analyze основной метод анализа
func (a *CounterAnalyzer) Analyze(data []redis_storage.PriceData, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()
	a.stats.TotalCalls++
	defer func() {
		a.stats.LastCallTime = time.Now()
		a.stats.TotalTime += time.Since(startTime)
		if a.stats.TotalCalls > 0 {
			a.stats.AverageTime = a.stats.TotalTime / time.Duration(a.stats.TotalCalls)
		}
	}()

	// Обновляем конфигурацию
	a.config = config

	var signals []analysis.Signal
	supportedPeriods := []string{"5m", "15m", "30m", "1h", "4h", "1d"}

	// Локальный счетчик для этого вызова
	localSentCount := 0

	logger.Debug("🔍 CounterAnalyzer.Analyze - анализ свечей")

	for i, point := range data {
		logger.Debug("📊 Анализ точки #%d: Символ: %s", i+1, point.Symbol)

		// Анализируем каждый период
		for _, period := range supportedPeriods {
			signal, err := a.AnalyzeCandle(point.Symbol, period)
			if err != nil {
				logger.Warn("⚠️ Ошибка анализа свечи %s/%s: %v", point.Symbol, period, err)
				continue
			}

			if signal != nil {
				signals = append(signals, *signal)

				// Публикуем сигнал в EventBus
				a.PublishRawCounterSignal(*signal, period)

				// Увеличиваем локальный счетчик
				localSentCount++
			}
		}
	}

	a.stats.SuccessCount++

	// Обновляем общую статистику отправленных сигналов
	if localSentCount > 0 {
		a.sentStatsMu.Lock()
		a.sentSignalsCount += localSentCount
		a.sentStatsMu.Unlock()
	}

	// Логируем агрегированную статистику раз в 10 секунд
	a.logAggregatedStatsIfNeeded()

	return signals, nil
}

// logAggregatedStatsIfNeeded логирует агрегированную статистику раз в 10 секунд
func (a *CounterAnalyzer) logAggregatedStatsIfNeeded() {
	a.sentStatsMu.RLock()
	defer a.sentStatsMu.RUnlock()

	now := time.Now()
	timeSinceLastLog := now.Sub(a.lastLogTime)

	// Логируем раз в 10 секунд
	if timeSinceLastLog >= 10*time.Second && a.sentSignalsCount > 0 {
		totalDuration := now.Sub(a.sentStatsStartTime)
		var signalsPerMinute float64
		if totalDuration.Minutes() > 0 {
			signalsPerMinute = float64(a.sentSignalsCount) / totalDuration.Minutes()
		}

		logger.Info("📊 CounterAnalyzer - Агрегированная статистика: "+
			"всего отправлено=%d, за период=%v, скорость=%.1f сигн/мин",
			a.sentSignalsCount, totalDuration.Round(time.Second), signalsPerMinute)

		// Сбрасываем время последнего логирования
		a.lastLogTime = now
	}
}

// Stop останавливает анализатор счетчика
func (a *CounterAnalyzer) Stop() error {
	logger.Info("🛑 Останавливаем CounterAnalyzer")

	// ✅ Останавливаем VolumeDeltaCalculator если есть
	if a.deps.VolumeCalculator != nil {
		logger.Info("🛑 Останавливаем VolumeDeltaCalculator")
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

	logger.Info("✅ CounterAnalyzer остановлен")
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
