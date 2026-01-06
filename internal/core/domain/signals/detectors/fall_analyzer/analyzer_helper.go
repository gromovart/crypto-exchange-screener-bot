package fallanalyzer

import (
	"sort"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/fall_analyzer/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/fall_analyzer/config"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/fall_analyzer/manager"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// sortDataByTime сортирует данные по времени
func (a *FallAnalyzer) sortDataByTime(data []types.PriceData) []types.PriceData {
	sorted := make([]types.PriceData, len(data))
	copy(sorted, data)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	return sorted
}

// convertConfig конвертирует конфигурацию из map в FallConfig
func (a *FallAnalyzer) convertConfig(cfg map[string]interface{}) config.FallConfig {
	fallConfig := a.config.GetConfig()

	// Обновляем основные параметры
	if enabled, ok := cfg["enabled"].(bool); ok {
		fallConfig.Enabled = enabled
	}
	if weight, ok := cfg["weight"].(float64); ok {
		fallConfig.Weight = weight
	}
	if minConfidence, ok := cfg["min_confidence"].(float64); ok {
		fallConfig.MinConfidence = minConfidence
	}
	if minDataPoints, ok := cfg["min_data_points"].(int); ok {
		fallConfig.MinDataPoints = minDataPoints
	}

	// Обновляем кастомные настройки
	for key, value := range cfg {
		switch key {
		case "min_fall":
			if v, ok := value.(float64); ok {
				fallConfig.MinFall = v
			}
		case "continuity_threshold":
			if v, ok := value.(float64); ok {
				fallConfig.ContinuityThreshold = v
			}
		case "volume_weight":
			if v, ok := value.(float64); ok {
				fallConfig.VolumeWeight = v
			}
		case "check_all_algorithms":
			if v, ok := value.(bool); ok {
				fallConfig.CheckAllAlgorithms = v
			}
		}
	}

	return fallConfig
}

// convertResultToSignal конвертирует результат расчета в сигнал
func (a *FallAnalyzer) convertResultToSignal(result *calculator.FallResult, state *manager.FallState) analysis.Signal {
	if result == nil || result.Symbol == "" {
		return analysis.Signal{}
	}

	// Определяем тип сигнала
	var signalType FallSignalType
	switch result.Type {
	case "single_fall":
		signalType = FallTypeSingle
	case "interval_fall":
		signalType = FallTypeInterval
	case "continuous_fall":
		signalType = FallTypeContinuous
	default:
		signalType = FallTypeSingle
	}

	fallSignal := NewFallSignal(
		result.Symbol,
		signalType,
		result.Direction,
		result.ChangePercent,
		result.Confidence,
		result.Period,
	)

	fallSignal.DataPoints = result.DataPoints
	fallSignal.StartPrice = result.StartPrice
	fallSignal.EndPrice = result.EndPrice
	fallSignal.Volume = result.Volume

	// Заполняем метаданные
	fallSignal.Metadata.IsContinuous = result.IsContinuous
	fallSignal.Metadata.Indicators = result.Indicators

	// Добавляем дополнительные теги
	if result.IsContinuous {
		fallSignal.Metadata.Tags = append(fallSignal.Metadata.Tags, "continuous")
	}
	if result.Period < 10 {
		fallSignal.Metadata.Tags = append(fallSignal.Metadata.Tags, "fast_fall")
	}

	return fallSignal.ConvertToAnalysisSignal()
}

// logResults логирует результаты анализа
func (a *FallAnalyzer) logResults(symbol string, signals []analysis.Signal) {
	if len(signals) > 0 {
		logger.Info("🎯 FallAnalyzer v2: найдено %d сигналов падения для %s",
			len(signals), symbol)
		for i, signal := range signals {
			logger.Debug("   %d. %s: падение=%.2f%%, уверенность=%.1f%%, период=%dмин",
				i+1, signal.Type, signal.ChangePercent, signal.Confidence, signal.Period)
		}
	} else {
		logger.Debug("📭 FallAnalyzer v2: для %s сигналов падения не найдено", symbol)
	}
}
