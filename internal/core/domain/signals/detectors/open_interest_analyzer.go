// internal/core/domain/signals/detectors/open_interest_analyzer.go
package analyzers

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"math"
	"sync"
	"time"
)

// OpenInterestAnalyzer - анализатор открытого интереса
type OpenInterestAnalyzer struct {
	config AnalyzerConfig
	stats  AnalyzerStats
	mu     sync.RWMutex
}

// Name возвращает имя анализатора
func (a *OpenInterestAnalyzer) Name() string {
	return "open_interest_analyzer"
}

// Version возвращает версию
func (a *OpenInterestAnalyzer) Version() string {
	return "1.0.0"
}

// Supports проверяет поддержку символа
func (a *OpenInterestAnalyzer) Supports(symbol string) bool {
	// Поддерживаем все символы, но проверяем доступность данных OI при анализе
	return true
}

// Analyze анализирует данные на основе открытого интереса
func (a *OpenInterestAnalyzer) Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	logger.Info("🔍 OpenInterestAnalyzer: начало анализа %s, точек данных: %d",
		data[0].Symbol, len(data))

	if len(data) < config.MinDataPoints {
		a.updateStats(time.Since(startTime), false)
		logger.Debug("⚠️  OpenInterestAnalyzer: недостаточно точек для %s (нужно %d, есть %d)",
			data[0].Symbol, config.MinDataPoints, len(data))
		return nil, fmt.Errorf("insufficient data points")
	}

	// Проверяем наличие данных OI
	validOIData := 0
	for _, point := range data {
		if point.OpenInterest > 0 {
			validOIData++
		}
	}

	logger.Debug("📊 OpenInterestAnalyzer: %s - доступно %d/%d точек с OI",
		data[0].Symbol, validOIData, len(data))

	var signals []analysis.Signal

	// 1. Проверка роста OI вместе с ценой
	if signal := a.checkOIGrowthWithPrice(data); signal != nil {
		signals = append(signals, *signal)
		logger.Debug("✅ OpenInterestAnalyzer: обнаружен рост OI+цена для %s", data[0].Symbol)
	}

	// 2. Проверка падения OI вместе с ценой
	if signal := a.checkOIFallWithPrice(data); signal != nil {
		signals = append(signals, *signal)
		logger.Debug("✅ OpenInterestAnalyzer: обнаружен рост OI при падении цены для %s", data[0].Symbol)
	}

	// 3. Проверка экстремальных значений OI
	if signal := a.checkExtremeOI(data); signal != nil {
		signals = append(signals, *signal)
		logger.Debug("✅ OpenInterestAnalyzer: обнаружены экстремальные значения OI для %s", data[0].Symbol)
	}

	// 4. Проверка дивергенций OI-цена
	if signal := a.checkOIPriceDivergence(data); signal != nil {
		signals = append(signals, *signal)
		logger.Debug("✅ OpenInterestAnalyzer: обнаружена дивергенция OI-цена для %s", data[0].Symbol)
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)

	if len(signals) > 0 {
		logger.Info("🎯 OpenInterestAnalyzer: найдено %d сигналов OI для %s",
			len(signals), data[0].Symbol)
		for i, signal := range signals {
			logger.Debug("   %d. %s: изменение=%.2f%%, уверенность=%.1f%%",
				i+1, signal.Type, signal.ChangePercent, signal.Confidence)
		}
	} else {
		logger.Debug("📭 OpenInterestAnalyzer: для %s сигналов OI не найдено", data[0].Symbol)
	}

	return signals, nil
}

// checkOIGrowthWithPrice проверяет рост OI вместе с ростом цены
func (a *OpenInterestAnalyzer) checkOIGrowthWithPrice(data []types.PriceData) *analysis.Signal {
	if len(data) < 2 {
		return nil
	}

	// Используем поле OpenInterest из PriceData
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
	// 3. Оба растут существенно
	minPriceChange := a.getMinPriceChange()
	minOIChange := a.getMinOIChange()

	logger.Debug("📈 OpenInterestAnalyzer: %s - цена: %.2f%%, OI: %.2f%% (пороги: цена>%.1f%%, OI>%.1f%%)",
		data[0].Symbol, priceChange, oiChange, minPriceChange, minOIChange)

	if priceChange > minPriceChange && oiChange > minOIChange {
		confidence := a.calculateOIGrowthConfidence(priceChange, oiChange)

		if confidence >= a.config.MinConfidence {
			logger.Debug("✅ OpenInterestAnalyzer: %s - РОСТ OI+цена: цена↑%.2f%%, OI↑%.2f%%, уверенность=%.1f%%",
				data[0].Symbol, priceChange, oiChange, confidence)

			return &analysis.Signal{
				Symbol:        data[0].Symbol,
				Type:          "oi_growth_with_price",
				Direction:     "up",
				ChangePercent: priceChange,
				Confidence:    confidence,
				DataPoints:    len(data),
				StartPrice:    data[0].Price,
				EndPrice:      data[len(data)-1].Price,
				Timestamp:     time.Now(),
				Metadata: analysis.Metadata{
					Strategy: "oi_price_growth",
					Tags:     []string{"open_interest", "bullish", "oi_growth"},
					Indicators: map[string]float64{
						"price_change":       priceChange,
						"oi_change":          oiChange,
						"oi_start":           startOI,
						"oi_end":             endOI,
						"oi_change_absolute": endOI - startOI,
						"oi_to_price_ratio":  oiChange / priceChange,
					},
				},
			}
		} else {
			logger.Debug("📉 OpenInterestAnalyzer: %s - рост есть, но уверенность низкая (%.1f%% < %.1f%%)",
				data[0].Symbol, confidence, a.config.MinConfidence)
		}
	}

	return nil
}

// checkOIFallWithPrice проверяет рост OI вместе с падением цены
func (a *OpenInterestAnalyzer) checkOIFallWithPrice(data []types.PriceData) *analysis.Signal {
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
	minPriceFall := a.getMinPriceFall()
	minOIChange := a.getMinOIChange()

	logger.Debug("📉 OpenInterestAnalyzer: %s - цена: %.2f%%, OI: %.2f%% (пороги падения: |цена|>%.1f%%, OI>%.1f%%)",
		data[0].Symbol, priceChange, oiChange, minPriceFall, minOIChange)

	if priceChange < -minPriceFall && oiChange > minOIChange {
		confidence := a.calculateOIFallConfidence(math.Abs(priceChange), oiChange)

		if confidence >= a.config.MinConfidence {
			logger.Debug("✅ OpenInterestAnalyzer: %s - РОСТ OI при ПАДЕНИИ цены: цена↓%.2f%%, OI↑%.2f%%, уверенность=%.1f%%",
				data[0].Symbol, math.Abs(priceChange), oiChange, confidence)

			return &analysis.Signal{
				Symbol:        data[0].Symbol,
				Type:          "oi_growth_with_fall",
				Direction:     "down",
				ChangePercent: priceChange,
				Confidence:    confidence,
				DataPoints:    len(data),
				StartPrice:    data[0].Price,
				EndPrice:      data[len(data)-1].Price,
				Timestamp:     time.Now(),
				Metadata: analysis.Metadata{
					Strategy: "oi_price_fall",
					Tags:     []string{"open_interest", "bearish", "short_accumulation"},
					Indicators: map[string]float64{
						"price_change":       priceChange,
						"oi_change":          oiChange,
						"oi_start":           startOI,
						"oi_end":             endOI,
						"oi_change_absolute": endOI - startOI,
						"oi_to_price_ratio":  oiChange / math.Abs(priceChange),
					},
				},
			}
		}
	}

	return nil
}

// checkExtremeOI проверяет экстремальные значения OI
func (a *OpenInterestAnalyzer) checkExtremeOI(data []types.PriceData) *analysis.Signal {
	if len(data) < 3 {
		return nil
	}

	// Собираем все значения OI
	var oiValues []float64
	var totalOI float64
	validPoints := 0

	for _, point := range data {
		if point.OpenInterest > 0 {
			oiValues = append(oiValues, point.OpenInterest)
			totalOI += point.OpenInterest
			validPoints++
		}
	}

	if validPoints < 3 {
		logger.Debug("📭 OpenInterestAnalyzer: недостаточно точек с OI для экстремального анализа (%d < 3)", validPoints)
		return nil
	}

	// Рассчитываем среднее OI
	avgOI := totalOI / float64(validPoints)

	// Находим последнее значение OI
	lastOI := data[len(data)-1].OpenInterest

	if lastOI <= 0 {
		return nil
	}

	// Рассчитываем, насколько последнее OI отличается от среднего
	oiRatio := lastOI / avgOI
	extremeThreshold := a.getExtremeOIThreshold()

	logger.Debug("📊 OpenInterestAnalyzer: %s - OI анализ: текущее=%.0f, среднее=%.0f, отношение=%.2f (порог=%.1f)",
		data[0].Symbol, lastOI, avgOI, oiRatio, extremeThreshold)

	// Проверяем экстремальное значение
	if oiRatio > extremeThreshold {
		// Высокий OI относительно среднего
		confidence := math.Min((oiRatio-1)*100, 90)

		// Определяем направление по цене
		priceChange := ((data[len(data)-1].Price - data[0].Price) / data[0].Price) * 100
		direction := "neutral"
		if priceChange > 0 {
			direction = "up"
		} else if priceChange < 0 {
			direction = "down"
		}

		logger.Debug("⚠️  OpenInterestAnalyzer: %s - ЭКСТРЕМАЛЬНЫЙ OI! отношение=%.2f, уверенность=%.1f%%, цена=%.2f%%",
			data[0].Symbol, oiRatio, confidence, priceChange)

		return &analysis.Signal{
			Symbol:        data[0].Symbol,
			Type:          "extreme_oi",
			Direction:     direction,
			ChangePercent: priceChange,
			Confidence:    confidence,
			DataPoints:    validPoints,
			StartPrice:    data[0].Price,
			EndPrice:      data[len(data)-1].Price,
			Timestamp:     time.Now(),
			Metadata: analysis.Metadata{
				Strategy: "extreme_oi_detection",
				Tags:     []string{"open_interest", "extreme", "overbought_oversold"},
				Indicators: map[string]float64{
					"current_oi":        lastOI,
					"avg_oi":            avgOI,
					"oi_ratio":          oiRatio,
					"oi_deviation":      (oiRatio - 1) * 100,
					"price_change":      priceChange,
					"oi_values_count":   float64(validPoints),
					"extreme_threshold": extremeThreshold,
				},
			},
		}
	}

	return nil
}

// checkOIPriceDivergence проверяет дивергенции между OI и ценой
func (a *OpenInterestAnalyzer) checkOIPriceDivergence(data []types.PriceData) *analysis.Signal {
	if len(data) < 4 {
		logger.Debug("📭 OpenInterestAnalyzer: недостаточно точек для анализа дивергенции (%d < 4)", len(data))
		return nil
	}

	// Собираем цены и OI
	var prices, oiValues []float64
	var priceChanges, oiChanges []float64

	for i, point := range data {
		if point.OpenInterest > 0 {
			prices = append(prices, point.Price)
			oiValues = append(oiValues, point.OpenInterest)
		}

		// Рассчитываем изменения
		if i > 0 && i < len(data) {
			if data[i].OpenInterest > 0 && data[i-1].OpenInterest > 0 {
				prevOI := data[i-1].OpenInterest
				currOI := data[i].OpenInterest

				priceChange := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100
				oiChange := ((currOI - prevOI) / prevOI) * 100

				priceChanges = append(priceChanges, priceChange)
				oiChanges = append(oiChanges, oiChange)
			}
		}
	}

	if len(priceChanges) < 3 || len(oiChanges) < 3 {
		logger.Debug("📭 OpenInterestAnalyzer: недостаточно изменений для дивергенции (цена:%d, OI:%d)",
			len(priceChanges), len(oiChanges))
		return nil
	}

	// Ищем дивергенции
	divergenceType := a.findDivergence(priceChanges, oiChanges)

	if divergenceType != "" {
		priceChange := ((data[len(data)-1].Price - data[0].Price) / data[0].Price) * 100
		confidence := a.calculateDivergenceConfidence(divergenceType, priceChanges, oiChanges)

		if confidence >= a.config.MinConfidence {
			var direction, signalType string
			if divergenceType == "bullish" {
				direction = "up"
				signalType = "bullish_oi_divergence"
			} else {
				direction = "down"
				signalType = "bearish_oi_divergence"
			}

			logger.Debug("🔀 OpenInterestAnalyzer: %s - ДИВЕРГЕНЦИЯ %s! уверенность=%.1f%%, цена=%.2f%%",
				data[0].Symbol, divergenceType, confidence, priceChange)

			// Создаем indicators map отдельно
			indicators := make(map[string]float64)
			if divergenceType == "bullish" {
				indicators["divergence_type"] = 1.0
			} else {
				indicators["divergence_type"] = -1.0
			}
			indicators["price_change"] = priceChange
			indicators["avg_price_change"] = a.calculateAverage(priceChanges)
			indicators["avg_oi_change"] = a.calculateAverage(oiChanges)
			indicators["divergence_strength"] = confidence / 100
			indicators["price_volatility"] = a.calculateVolatility(prices)
			indicators["oi_volatility"] = a.calculateVolatility(oiValues)

			return &analysis.Signal{
				Symbol:        data[0].Symbol,
				Type:          signalType,
				Direction:     direction,
				ChangePercent: priceChange,
				Confidence:    confidence,
				DataPoints:    len(data),
				StartPrice:    data[0].Price,
				EndPrice:      data[len(data)-1].Price,
				Timestamp:     time.Now(),
				Metadata: analysis.Metadata{
					Strategy:   "oi_price_divergence",
					Tags:       []string{"open_interest", "divergence", divergenceType},
					Indicators: indicators,
				},
			}
		}
	}

	return nil
}

// findDivergence ищет дивергенции между ценами и OI
func (a *OpenInterestAnalyzer) findDivergence(priceChanges, oiChanges []float64) string {
	if len(priceChanges) < 3 || len(oiChanges) < 3 {
		return ""
	}

	// Простая логика дивергенции:
	// Бычья дивергенция: цена делает новые минимумы, а OI растет
	// Медвежья дивергенция: цена делает новые максимумы, а OI падает

	// Проверяем последние 3 точки
	lastPrice1 := priceChanges[len(priceChanges)-3]
	lastPrice2 := priceChanges[len(priceChanges)-2]
	lastPrice3 := priceChanges[len(priceChanges)-1]

	lastOI1 := oiChanges[len(oiChanges)-3]
	lastOI2 := oiChanges[len(oiChanges)-2]
	lastOI3 := oiChanges[len(oiChanges)-1]

	logger.Debug("🔍 OpenInterestAnalyzer: проверка дивергенции - цена: [%.2f, %.2f, %.2f], OI: [%.2f, %.2f, %.2f]",
		lastPrice1, lastPrice2, lastPrice3, lastOI1, lastOI2, lastOI3)

	// Бычья дивергенция
	if lastPrice1 > lastPrice2 && lastPrice2 < lastPrice3 && // цена делает higher low
		lastOI1 < lastOI2 && lastOI2 > lastOI3 { // OI делает lower high
		logger.Debug("✅ OpenInterestAnalyzer: обнаружена БЫЧЬЯ дивергенция")
		return "bullish"
	}

	// Медвежья дивергенция
	if lastPrice1 < lastPrice2 && lastPrice2 > lastPrice3 && // цена делает lower high
		lastOI1 > lastOI2 && lastOI2 < lastOI3 { // OI делает higher low
		logger.Debug("✅ OpenInterestAnalyzer: обнаружена МЕДВЕЖЬЯ дивергенция")
		return "bearish"
	}

	return ""
}

// Вспомогательные методы для получения настроек
func (a *OpenInterestAnalyzer) getMinPriceChange() float64 {
	if val, ok := a.config.CustomSettings["min_price_change"].(float64); ok {
		return val
	}
	return 1.0
}

func (a *OpenInterestAnalyzer) getMinPriceFall() float64 {
	if val, ok := a.config.CustomSettings["min_price_fall"].(float64); ok {
		return val
	}
	return 1.0
}

func (a *OpenInterestAnalyzer) getMinOIChange() float64 {
	if val, ok := a.config.CustomSettings["min_oi_change"].(float64); ok {
		return val
	}
	return 5.0
}

func (a *OpenInterestAnalyzer) getExtremeOIThreshold() float64 {
	if val, ok := a.config.CustomSettings["extreme_oi_threshold"].(float64); ok {
		return val
	}
	return 1.5
}

// Методы расчета уверенности
func (a *OpenInterestAnalyzer) calculateOIGrowthConfidence(priceChange, oiChange float64) float64 {
	// Базовая уверенность на основе изменения цены (макс 40%)
	priceConfidence := math.Min(priceChange*2, 40)

	// Уверенность на основе изменения OI (макс 30%)
	oiConfidence := math.Min(oiChange/2, 30)

	// Дополнительный бонус за синхронность (макс 30%)
	syncBonus := 0.0
	if oiChange > priceChange*0.5 && oiChange < priceChange*2 {
		syncBonus = math.Min(30, (oiChange/priceChange)*15)
	}

	totalConfidence := priceConfidence + oiConfidence + syncBonus
	result := math.Min(totalConfidence, 100)

	logger.Debug("📊 OpenInterestAnalyzer: уверенность роста OI+цена = %.1f%% (цена:%.1f%%, OI:%.1f%%, синхр:%.1f%%)",
		result, priceConfidence, oiConfidence, syncBonus)

	return result
}

func (a *OpenInterestAnalyzer) calculateOIFallConfidence(priceFall, oiGrowth float64) float64 {
	// Чем сильнее падение цены при росте OI, тем увереннее сигнал
	baseConfidence := math.Min(priceFall*3, 60)
	oiConfidence := math.Min(oiGrowth, 30)

	totalConfidence := baseConfidence + oiConfidence
	result := math.Min(totalConfidence, 100)

	logger.Debug("📊 OpenInterestAnalyzer: уверенность роста OI при падении = %.1f%% (падение:%.1f%%, OI:%.1f%%)",
		result, baseConfidence, oiConfidence)

	return result
}

func (a *OpenInterestAnalyzer) calculateDivergenceConfidence(divergenceType string, priceChanges, oiChanges []float64) float64 {
	if len(priceChanges) < 3 {
		return 0
	}

	// Рассчитываем силу дивергенции
	var divergenceStrength float64

	// Для бычьей дивергенции: чем ниже цена и выше OI, тем сильнее
	if divergenceType == "bullish" {
		priceDecrease := math.Abs(priceChanges[len(priceChanges)-2]) // самый низкий
		oiIncrease := oiChanges[len(oiChanges)-2]                    // самый высокий
		divergenceStrength = priceDecrease + oiIncrease
	} else {
		// Для медвежьей дивергенции: чем выше цена и ниже OI, тем сильнее
		priceIncrease := priceChanges[len(priceChanges)-2]  // самый высокий
		oiDecrease := math.Abs(oiChanges[len(oiChanges)-2]) // самый низкий
		divergenceStrength = priceIncrease + oiDecrease
	}

	// Нормализуем до 0-100%
	confidence := math.Min(divergenceStrength*10, 80)

	// Добавляем бонус за количество точек
	if len(priceChanges) >= 5 {
		confidence += 10
	}

	logger.Debug("📊 OpenInterestAnalyzer: уверенность дивергенции %s = %.1f%% (сила=%.2f, точек=%d)",
		divergenceType, confidence, divergenceStrength, len(priceChanges))

	return math.Min(confidence, 100)
}

// Вспомогательные математические методы
func (a *OpenInterestAnalyzer) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (a *OpenInterestAnalyzer) calculateVolatility(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := a.calculateAverage(values)
	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance) / mean * 100 // возвращаем в процентах
}

// GetConfig возвращает конфигурацию
func (a *OpenInterestAnalyzer) GetConfig() AnalyzerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// GetStats возвращает статистику
func (a *OpenInterestAnalyzer) GetStats() AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

// updateStats обновляет статистику
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
		logger.Info("📈 OpenInterestAnalyzer статистика: вызовов=%d, успехов=%d, ошибок=%d, среднее время=%v",
			a.stats.TotalCalls, a.stats.SuccessCount, a.stats.ErrorCount, a.stats.AverageTime)
	}
}

// DefaultOpenInterestConfig - конфигурация по умолчанию для Open Interest Analyzer
var DefaultOpenInterestConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        0.6,
	MinConfidence: 50.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_price_change":      1.0, // минимальное изменение цены для сигнала (%)
		"min_price_fall":        1.0, // минимальное падение цены для сигнала (%)
		"min_oi_change":         5.0, // минимальное изменение OI для сигнала (%)
		"extreme_oi_threshold":  1.5, // порог экстремального OI (1.5 = на 50% выше среднего)
		"divergence_min_points": 4,   // минимальное количество точек для дивергенции
		"volume_weight":         0.3, // вес объема в расчетах
	},
}
