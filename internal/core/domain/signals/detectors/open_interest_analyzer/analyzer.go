// internal/core/domain/signals/detectors/open_interest_analyzer/analyzer.go
package oianalyzer

import (
	"sync"
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/open_interest_analyzer/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/open_interest_analyzer/config"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/open_interest_analyzer/manager"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// OpenInterestAnalyzer - основной анализатор открытого интереса (фасад)
type OpenInterestAnalyzer struct {
	config         *config.ConfigManager
	stateManager   *manager.StateManager
	extremeCalc    *calculator.ExtremeCalculator
	divergenceCalc *calculator.DivergenceCalculator
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

// NewOpenInterestAnalyzer создает новый анализатор открытого интереса
func NewOpenInterestAnalyzer() *OpenInterestAnalyzer {
	return &OpenInterestAnalyzer{
		config:         config.NewConfigManager(),
		stateManager:   manager.NewStateManager(),
		extremeCalc:    calculator.NewExtremeCalculator(),
		divergenceCalc: calculator.NewDivergenceCalculator(),
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
func (a *OpenInterestAnalyzer) Name() string {
	return "open_interest_analyzer"
}

// Version возвращает версию анализатора
func (a *OpenInterestAnalyzer) Version() string {
	return "2.0.0" // Новая версия после рефакторинга
}

// Supports проверяет поддержку символа
func (a *OpenInterestAnalyzer) Supports(symbol string) bool {
	// Поддерживаем все символы, но проверяем доступность данных OI при анализе
	return true
}

// Analyze анализирует данные на основе открытого интереса
func (a *OpenInterestAnalyzer) Analyze(data []types.PriceData, cfg map[string]interface{}) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) == 0 {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	symbol := data[0].Symbol
	logger.Info("🔍 OpenInterestAnalyzer v2: начало анализа %s, точек данных: %d",
		symbol, len(data))

	// Конвертируем конфигурацию
	oiConfig := a.convertConfig(cfg)

	if len(data) < oiConfig.MinDataPoints {
		a.updateStats(time.Since(startTime), false)
		logger.Debug("⚠️  OpenInterestAnalyzer: недостаточно точек для %s (нужно %d, есть %d)",
			symbol, oiConfig.MinDataPoints, len(data))
		return nil, nil
	}

	// Обновляем состояние символа
	stateConfig := manager.OIConfigForState{
		ExtremeOIThreshold: oiConfig.ExtremeOIThreshold,
	}
	state := a.stateManager.UpdateState(symbol, data, stateConfig)

	// Проверяем наличие данных OI
	validOIData := a.countValidOIData(data)

	logger.Debug("📊 OpenInterestAnalyzer: %s - доступно %d/%d точек с OI",
		symbol, validOIData, len(data))

	var oiSignals []*OISignal

	// 1. Проверка роста OI вместе с ценой
	if signal := a.analyzeGrowthWithPrice(data, oiConfig, state); signal != nil {
		oiSignals = append(oiSignals, signal)
	}

	// 2. Проверка роста OI при падении цены
	if signal := a.analyzeGrowthWithFall(data, oiConfig, state); signal != nil {
		oiSignals = append(oiSignals, signal)
	}

	// 3. Проверка экстремальных значений OI
	if oiConfig.CheckAllAlgorithms || a.config.IsAlgorithmEnabled(config.AlgorithmExtremeOI) {
		extremeConfig := calculator.OIConfigForExtreme{
			MinConfidence:      oiConfig.MinConfidence,
			ExtremeOIThreshold: oiConfig.ExtremeOIThreshold,
		}
		if result := a.extremeCalc.AnalyzeExtremeOI(data, extremeConfig); result != nil {
			if signal := a.convertExtremeResultToSignal(result); signal != nil {
				oiSignals = append(oiSignals, signal)
			}
		}
	}

	// 4. Проверка дивергенций OI-цена
	if oiConfig.CheckAllAlgorithms || a.config.IsAlgorithmEnabled(config.AlgorithmDivergence) {
		divergenceConfig := calculator.OIConfigForDivergence{
			MinConfidence:       oiConfig.MinConfidence,
			DivergenceMinPoints: oiConfig.DivergenceMinPoints,
		}
		if result := a.divergenceCalc.AnalyzeDivergence(data, divergenceConfig); result != nil {
			if signal := a.convertDivergenceResultToSignal(result); signal != nil {
				oiSignals = append(oiSignals, signal)
			}
		}
	}

	// Конвертируем OI сигналы в общие сигналы
	signals := a.convertOISignals(oiSignals, oiConfig)

	a.updateStats(time.Since(startTime), len(signals) > 0)
	a.logResults(symbol, signals)

	return signals, nil
}

// GetConfig возвращает конфигурацию анализатора
func (a *OpenInterestAnalyzer) GetConfig() config.OIConfig {
	return a.config.GetConfig()
}

// GetStats возвращает статистику анализатора
func (a *OpenInterestAnalyzer) GetStats() analyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

// GetStateManager возвращает менеджер состояний (для тестирования и отладки)
func (a *OpenInterestAnalyzer) GetStateManager() *manager.StateManager {
	return a.stateManager
}

// GetState возвращает состояние для символа
func (a *OpenInterestAnalyzer) GetState(symbol string) *manager.OIState {
	return a.stateManager.GetState(symbol)
}

// Cleanup очищает старые состояния
func (a *OpenInterestAnalyzer) Cleanup(maxAge time.Duration) {
	a.stateManager.Cleanup(maxAge)
}

// GetAnalysisStats возвращает статистику анализа
func (a *OpenInterestAnalyzer) GetAnalysisStats() map[string]interface{} {
	return a.stateManager.GetStats()
}

// updateStats обновляет статистику анализатора
func (a *OpenInterestAnalyzer) updateStats(duration time.Duration, success bool) {
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
		logger.Info("📈 OpenInterestAnalyzer v2 статистика: вызовов=%d, успехов=%d, ошибок=%d, среднее время=%v",
			a.stats.TotalCalls, a.stats.SuccessCount, a.stats.ErrorCount, a.stats.AverageTime)
	}
}
