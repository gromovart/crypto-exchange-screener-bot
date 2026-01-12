// internal/core/domain/signals/detectors/fall_analyzer/analyzer.go
package fallanalyzer

import (
	"sync"
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/fall_analyzer/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/fall_analyzer/config"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/fall_analyzer/manager"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// FallAnalyzer - основной анализатор падений (фасад)
type FallAnalyzer struct {
	config         *config.ConfigManager
	stateManager   *manager.StateManager
	fallCalc       *calculator.FallCalculator
	confidenceCalc *calculator.ConfidenceCalculator
	stats          analyzerStats
	mu             sync.RWMutex
}

// analyzerStats - упрощенная версия статистики
type analyzerStats struct {
	TotalCalls   int           `json:"total_calls"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	LastCallTime time.Time     `json:"last_call_time"`
}

// NewFallAnalyzer создает новый анализатор падений
func NewFallAnalyzer() *FallAnalyzer {
	return &FallAnalyzer{
		config:         config.NewConfigManager(),
		stateManager:   manager.NewStateManager(),
		fallCalc:       calculator.NewFallCalculator(),
		confidenceCalc: calculator.NewConfidenceCalculator(),
		stats: analyzerStats{
			TotalCalls:   0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
		},
	}
}

// Name возвращает имя анализатора
func (a *FallAnalyzer) Name() string {
	return "fall_analyzer"
}

// Version возвращает версию анализатора
func (a *FallAnalyzer) Version() string {
	return "2.0.0" // Новая версия после рефакторинга
}

// Supports проверяет поддержку символа
func (a *FallAnalyzer) Supports(symbol string) bool {
	// Поддерживаем все символы
	return true
}

// Analyze анализирует данные на поиск падений
func (a *FallAnalyzer) Analyze(data []types.PriceData, cfg map[string]interface{}) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) == 0 {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	symbol := data[0].Symbol
	logger.Debug("🔻 FallAnalyzer v2: начало анализа %s, точек данных: %d",
		symbol, len(data))

	// Конвертируем конфигурацию
	fallConfig := a.convertConfig(cfg)

	if len(data) < fallConfig.MinDataPoints {
		a.updateStats(time.Since(startTime), false)
		logger.Debug("⚠️  FallAnalyzer: недостаточно точек для %s (нужно %d, есть %d)",
			symbol, fallConfig.MinDataPoints, len(data))
		return nil, nil
	}

	// Сортируем данные по времени
	sortedData := a.sortDataByTime(data)

	// Обновляем состояние символа
	stateConfig := manager.FallConfigForState{
		MinFall: fallConfig.MinFall,
	}
	state := a.stateManager.UpdateState(symbol, sortedData, stateConfig)

	// Анализируем падения
	calcConfig := calculator.FallConfigForCalculator{
		MinConfidence:       fallConfig.MinConfidence,
		MinFall:             fallConfig.MinFall,
		ContinuityThreshold: fallConfig.ContinuityThreshold,
		VolumeWeight:        fallConfig.VolumeWeight,
	}

	fallResults := a.fallCalc.AnalyzeFalls(sortedData, calcConfig)

	// Конвертируем результаты в сигналы
	var signals []analysis.Signal
	for _, result := range fallResults {
		if result.Confidence >= fallConfig.MinConfidence {
			signal := a.convertResultToSignal(result, state)
			// Проверяем, что сигнал валидный
			if signal.Symbol != "" {
				signals = append(signals, signal)
			}
		}
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	a.logResults(symbol, signals)

	return signals, nil
}

// GetConfig возвращает конфигурацию анализатора
func (a *FallAnalyzer) GetConfig() config.FallConfig {
	return a.config.GetConfig()
}

// GetStats возвращает статистику анализатора
func (a *FallAnalyzer) GetStats() analyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

// GetStateManager возвращает менеджер состояний (для тестирования и отладки)
func (a *FallAnalyzer) GetStateManager() *manager.StateManager {
	return a.stateManager
}

// GetState возвращает состояние для символа
func (a *FallAnalyzer) GetState(symbol string) *manager.FallState {
	return a.stateManager.GetState(symbol)
}

// Cleanup очищает старые состояния
func (a *FallAnalyzer) Cleanup(maxAge time.Duration) {
	a.stateManager.Cleanup(maxAge)
}

// GetAnalysisStats возвращает статистику анализа
func (a *FallAnalyzer) GetAnalysisStats() map[string]interface{} {
	return a.stateManager.GetStats()
}

// updateStats обновляет статистику анализатора
func (a *FallAnalyzer) updateStats(duration time.Duration, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

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

	// Логируем статистику каждые 100 вызовов
	if a.stats.TotalCalls%100 == 0 {
		logger.Info("📈 FallAnalyzer v2 статистика: вызовов=%d, успехов=%d, ошибок=%d, среднее время=%v",
			a.stats.TotalCalls, a.stats.SuccessCount, a.stats.ErrorCount, a.stats.AverageTime)
	}
}
