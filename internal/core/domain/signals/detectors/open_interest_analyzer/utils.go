// internal/core/domain/signals/detectors/open_interest_analyzer/analyzer_utils.go
package oianalyzer

import (
	"math"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/open_interest_analyzer/calculator"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/open_interest_analyzer/config"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/open_interest_analyzer/manager"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/pkg/logger"
)

// countValidOIData подсчитывает количество точек с валидными данными OI
func (a *OpenInterestAnalyzer) countValidOIData(data []redis_storage.PriceData) int {
	validCount := 0
	for _, point := range data {
		if point.OpenInterest > 0 {
			validCount++
		}
	}
	return validCount
}

// convertOISignals конвертирует OI сигналы в общие сигналы
func (a *OpenInterestAnalyzer) convertOISignals(oiSignals []*OISignal, oiConfig config.OIConfig) []analysis.Signal {
	var signals []analysis.Signal
	for _, oiSignal := range oiSignals {
		if oiSignal.Confidence >= oiConfig.MinConfidence {
			signals = append(signals, oiSignal.ConvertToAnalysisSignal())
		}
	}
	return signals
}

// logResults логирует результаты анализа
func (a *OpenInterestAnalyzer) logResults(symbol string, signals []analysis.Signal) {
	if len(signals) > 0 {
		logger.Info("🎯 OpenInterestAnalyzer v2: найдено %d сигналов OI для %s",
			len(signals), symbol)
		for i, signal := range signals {
			logger.Debug("   %d. %s: изменение=%.2f%%, уверенность=%.1f%%",
				i+1, signal.Type, signal.ChangePercent, signal.Confidence)
		}
	} else {
		logger.Debug("📭 OpenInterestAnalyzer v2: для %s сигналов OI не найдено", symbol)
	}
}

// convertConfig конвертирует конфигурацию из map в OIConfig
func (a *OpenInterestAnalyzer) convertConfig(cfg map[string]interface{}) config.OIConfig {
	oiConfig := a.config.GetConfig()

	// Обновляем основные параметры
	if enabled, ok := cfg["enabled"].(bool); ok {
		oiConfig.Enabled = enabled
	}
	if weight, ok := cfg["weight"].(float64); ok {
		oiConfig.Weight = weight
	}
	if minConfidence, ok := cfg["min_confidence"].(float64); ok {
		oiConfig.MinConfidence = minConfidence
	}
	if minDataPoints, ok := cfg["min_data_points"].(int); ok {
		oiConfig.MinDataPoints = minDataPoints
	}

	// Обновляем кастомные настройки
	for key, value := range cfg {
		switch key {
		case "min_price_change":
			if v, ok := value.(float64); ok {
				oiConfig.MinPriceChange = v
			}
		case "min_price_fall":
			if v, ok := value.(float64); ok {
				oiConfig.MinPriceFall = v
			}
		case "min_oi_change":
			if v, ok := value.(float64); ok {
				oiConfig.MinOIChange = v
			}
		case "extreme_oi_threshold":
			if v, ok := value.(float64); ok {
				oiConfig.ExtremeOIThreshold = v
			}
		case "divergence_min_points":
			if v, ok := value.(int); ok {
				oiConfig.DivergenceMinPoints = v
			} else if v, ok := value.(float64); ok {
				oiConfig.DivergenceMinPoints = int(v)
			}
		case "volume_weight":
			if v, ok := value.(float64); ok {
				oiConfig.VolumeWeight = v
			}
		case "check_all_algorithms":
			if v, ok := value.(bool); ok {
				oiConfig.CheckAllAlgorithms = v
			}
		}
	}

	return oiConfig
}

// analyzeGrowthWithPrice анализирует рост OI вместе с ростом цены
func (a *OpenInterestAnalyzer) analyzeGrowthWithPrice(data []redis_storage.PriceData, oiConfig config.OIConfig, state *manager.OIState) *OISignal {
	if len(data) < 2 {
		return nil
	}

	startOI := data[0].OpenInterest
	endOI := data[len(data)-1].OpenInterest

	if startOI <= 0 || endOI <= 0 {
		logger.Debug("📭 OpenInterestAnalyzer: нет данных OI для анализа роста (начало=%.0f, конец=%.0f)",
			startOI, endOI)
		return nil
	}

	// Рассчитываем изменение цены и OI
	priceChange := ((data[len(data)-1].Price - data[0].Price) / data[0].Price) * 100
	oiChange := ((endOI - startOI) / startOI) * 100

	// Проверяем условия:
	// 1. Цена растет (рост > порога)
	// 2. OI растет (рост > порога)
	logger.Debug("📈 OpenInterestAnalyzer: %s - цена: %.2f%%, OI: %.2f%% (пороги: цена>%.1f%%, OI>%.1f%%)",
		data[0].Symbol, priceChange, oiChange, oiConfig.MinPriceChange, oiConfig.MinOIChange)

	if priceChange > oiConfig.MinPriceChange && oiChange > oiConfig.MinOIChange {
		// Рассчитываем уверенность
		duration := data[len(data)-1].Timestamp.Sub(data[0].Timestamp)
		confidence := a.confidenceCalc.CalculateGrowthWithPriceConfidence(
			priceChange, oiChange, duration, len(data),
		)

		// Корректируем уверенность с учетом объема
		if oiConfig.VolumeWeight > 0 && len(data) > 0 {
			volumeRatio := 1.0 // По умолчанию
			confidence = a.confidenceCalc.AdjustConfidenceForVolume(
				confidence, volumeRatio, oiConfig.VolumeWeight,
			)
		}

		if confidence >= oiConfig.MinConfidence {
			logger.Debug("✅ OpenInterestAnalyzer: %s - РОСТ OI+цена: цена↑%.2f%%, OI↑%.2f%%, уверенность=%.1f%%",
				data[0].Symbol, priceChange, oiChange, confidence)

			signal := NewOISignal(
				data[0].Symbol,
				OITypeGrowthWithPrice,
				"up",
				priceChange,
				confidence,
			)

			signal.DataPoints = len(data)
			signal.StartPrice = data[0].Price
			signal.EndPrice = data[len(data)-1].Price
			signal.StartOI = startOI
			signal.EndOI = endOI

			// Заполняем метаданные
			signal.Metadata.Tags = append(signal.Metadata.Tags, "bullish", "oi_growth")
			signal.Metadata.Indicators = map[string]float64{
				"price_change":       priceChange,
				"oi_change":          oiChange,
				"oi_start":           startOI,
				"oi_end":             endOI,
				"oi_change_absolute": endOI - startOI,
				"oi_to_price_ratio":  oiChange / priceChange,
				"duration_minutes":   duration.Minutes(),
			}

			return signal
		} else {
			logger.Debug("📉 OpenInterestAnalyzer: %s - рост есть, но уверенность низкая (%.1f%% < %.1f%%)",
				data[0].Symbol, confidence, oiConfig.MinConfidence)
		}
	}

	return nil
}

// analyzeGrowthWithFall анализирует рост OI при падении цены
func (a *OpenInterestAnalyzer) analyzeGrowthWithFall(data []redis_storage.PriceData, oiConfig config.OIConfig, state *manager.OIState) *OISignal {
	if len(data) < 2 {
		return nil
	}

	startOI := data[0].OpenInterest
	endOI := data[len(data)-1].OpenInterest

	if startOI <= 0 || endOI <= 0 {
		return nil
	}

	// Рассчитываем изменение цены и OI
	priceChange := ((data[len(data)-1].Price - data[0].Price) / data[0].Price) * 100
	oiChange := ((endOI - startOI) / startOI) * 100

	// Проверяем условия:
	// 1. Цена падает (падение > порога)
	// 2. OI растет (рост > порога)
	logger.Debug("📉 OpenInterestAnalyzer: %s - цена: %.2f%%, OI: %.2f%% (пороги падения: |цена|>%.1f%%, OI>%.1f%%)",
		data[0].Symbol, priceChange, oiChange, oiConfig.MinPriceFall, oiConfig.MinOIChange)

	priceFall := math.Abs(priceChange)
	if priceChange < -oiConfig.MinPriceFall && oiChange > oiConfig.MinOIChange {
		// Рассчитываем уверенность
		duration := data[len(data)-1].Timestamp.Sub(data[0].Timestamp)
		confidence := a.confidenceCalc.CalculateGrowthWithFallConfidence(
			priceFall, oiChange, duration, len(data),
		)

		// Корректируем уверенность с учетом объема
		if oiConfig.VolumeWeight > 0 && len(data) > 0 {
			volumeRatio := 1.0 // По умолчанию
			confidence = a.confidenceCalc.AdjustConfidenceForVolume(
				confidence, volumeRatio, oiConfig.VolumeWeight,
			)
		}

		if confidence >= oiConfig.MinConfidence {
			logger.Debug("✅ OpenInterestAnalyzer: %s - РОСТ OI при ПАДЕНИИ цены: цена↓%.2f%%, OI↑%.2f%%, уверенность=%.1f%%",
				data[0].Symbol, priceFall, oiChange, confidence)

			signal := NewOISignal(
				data[0].Symbol,
				OITypeGrowthWithFall,
				"down",
				priceChange,
				confidence,
			)

			signal.DataPoints = len(data)
			signal.StartPrice = data[0].Price
			signal.EndPrice = data[len(data)-1].Price
			signal.StartOI = startOI
			signal.EndOI = endOI

			// Заполняем метаданные
			signal.Metadata.Tags = append(signal.Metadata.Tags, "bearish", "short_accumulation")
			signal.Metadata.Indicators = map[string]float64{
				"price_change":       priceChange,
				"oi_change":          oiChange,
				"oi_start":           startOI,
				"oi_end":             endOI,
				"oi_change_absolute": endOI - startOI,
				"oi_to_price_ratio":  oiChange / priceFall,
				"duration_minutes":   duration.Minutes(),
			}

			return signal
		}
	}

	return nil
}

// Добавить в analyzer_utils.go:

// convertExtremeResultToSignal конвертирует результат экстремального анализа в OISignal
func (a *OpenInterestAnalyzer) convertExtremeResultToSignal(result *calculator.ExtremeResult) *OISignal {
	if result == nil {
		return nil
	}

	signal := NewOISignal(
		result.Symbol,
		OITypeExtreme,
		result.Direction,
		result.ChangePercent,
		result.Confidence,
	)

	signal.DataPoints = result.DataPoints
	signal.StartPrice = result.StartPrice
	signal.EndPrice = result.EndPrice
	signal.StartOI = result.StartOI
	signal.EndOI = result.EndOI

	// Заполняем метаданные
	signal.Metadata.ExtremeType = result.ExtremeType
	signal.Metadata.Patterns = []string{"extreme_oi_" + result.ExtremeType}
	signal.Metadata.Indicators = result.Indicators

	return signal
}

// convertDivergenceResultToSignal конвертирует результат анализа дивергенции в OISignal
func (a *OpenInterestAnalyzer) convertDivergenceResultToSignal(result *calculator.OISignalForDivergence) *OISignal {
	if result == nil {
		return nil
	}

	var signalType OISignalType
	if result.Metadata.DivergenceType == "bullish" {
		signalType = OITypeBullishDiv
	} else {
		signalType = OITypeBearishDiv
	}

	signal := NewOISignal(
		result.Symbol,
		signalType,
		result.Direction,
		result.ChangePercent,
		result.Confidence,
	)

	signal.DataPoints = result.DataPoints
	signal.StartPrice = result.StartPrice
	signal.EndPrice = result.EndPrice
	signal.StartOI = result.StartOI
	signal.EndOI = result.EndOI

	// Заполняем метаданные
	signal.Metadata.DivergenceType = result.Metadata.DivergenceType
	signal.Metadata.Patterns = result.Metadata.Patterns
	signal.Metadata.Indicators = result.Metadata.Indicators

	return signal
}
