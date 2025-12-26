package analyzers

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// FallAnalyzer - анализатор падения
type FallAnalyzer struct {
	config AnalyzerConfig
	stats  AnalyzerStats
	mu     sync.RWMutex
}

func (a *FallAnalyzer) Name() string {
	return "fall_analyzer"
}

func (a *FallAnalyzer) Version() string {
	return "2.0.0"
}

func (a *FallAnalyzer) Supports(symbol string) bool {
	return true
}

func (a *FallAnalyzer) Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) < config.MinDataPoints {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("insufficient data points")
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	// Ищем все значимые падения между любыми двумя точками
	var signals []analysis.Signal
	minFall := a.config.CustomSettings["min_fall"].(float64)

	// Проверяем падения между последовательными точками
	for i := 1; i < len(data); i++ {
		change := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100

		// Это падение?
		if change < 0 && math.Abs(change) >= minFall {
			// Нашли значимое падение!
			confidence := a.calculateSingleFallConfidence(data[i-1:i+1], math.Abs(change))

			if confidence >= config.MinConfidence {
				signal := a.createFallSignal(data[i-1], data[i], change, confidence)
				signals = append(signals, signal)
			}
		}
	}

	// Также проверяем максимальное падение за любой интервал
	if len(data) >= 3 {
		maxFall, startIdx, endIdx := a.findMaxFall(data)
		fallValue := math.Abs(maxFall)

		if maxFall < 0 && fallValue >= minFall {
			// Проверяем, не дублируем ли мы уже найденное падение
			isNew := true
			for _, sig := range signals {
				if int(sig.StartPrice) == int(data[startIdx].Price) &&
					int(sig.EndPrice) == int(data[endIdx].Price) {
					isNew = false
					break
				}
			}

			if isNew {
				confidence := a.calculateIntervalConfidence(data[startIdx:endIdx+1], fallValue)
				if confidence >= config.MinConfidence {
					signal := a.createFallSignal(data[startIdx], data[endIdx], maxFall, confidence)
					signals = append(signals, signal)
				}
			}
		}
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

// Находит максимальное падение за любой интервал
func (a *FallAnalyzer) findMaxFall(data []types.PriceData) (float64, int, int) {
	maxFall := 0.0
	startIdx, endIdx := 0, 0

	for i := 0; i < len(data); i++ {
		for j := i + 1; j < len(data); j++ {
			change := ((data[j].Price - data[i].Price) / data[i].Price) * 100
			if change < maxFall {
				maxFall = change
				startIdx, endIdx = i, j
			}
		}
	}

	return maxFall, startIdx, endIdx
}

// Создает сигнал падения
func (a *FallAnalyzer) createFallSignal(startPoint, endPoint types.PriceData, change, confidence float64) analysis.Signal {
	intervalData := []types.PriceData{startPoint, endPoint}
	isContinuous := a.checkContinuity(intervalData)

	// Рассчитываем период в минутах
	periodMinutes := int(endPoint.Timestamp.Sub(startPoint.Timestamp).Minutes())
	if periodMinutes < 1 {
		periodMinutes = 1
	}

	// Средний объем
	avgVolume := (startPoint.Volume24h + endPoint.Volume24h) / 2

	return analysis.Signal{
		Symbol:        startPoint.Symbol,
		Type:          "fall",
		Direction:     "down",
		ChangePercent: change,
		Period:        periodMinutes,
		Confidence:    confidence,
		DataPoints:    2,
		StartPrice:    startPoint.Price,
		EndPrice:      endPoint.Price,
		Volume:        avgVolume,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy:     "fall_detection",
			Tags:         []string{"fall", "bearish", "local_drop"},
			IsContinuous: isContinuous,
			Indicators: map[string]float64{
				"trend_strength":  a.calculateTrendStrength(intervalData),
				"volatility":      a.calculateVolatility(intervalData),
				"duration_min":    endPoint.Timestamp.Sub(startPoint.Timestamp).Minutes(),
				"start_time_unix": float64(startPoint.Timestamp.Unix()),
				"end_time_unix":   float64(endPoint.Timestamp.Unix()),
				"start_price":     startPoint.Price,
				"end_price":       endPoint.Price,
			},
		},
	}
}

// Уверенность для одиночного падения между двумя точками
func (a *FallAnalyzer) calculateSingleFallConfidence(data []types.PriceData, fallPercent float64) float64 {
	if len(data) < 2 {
		return 0.0
	}

	baseConfidence := math.Min(fallPercent*10, 70)

	volumeFactor := 0.0
	avgVolume := (data[0].Volume24h + data[1].Volume24h) / 2
	if avgVolume > 1000000 {
		volumeFactor = 10.0
	} else if avgVolume < 100000 {
		volumeFactor = -5.0
	}

	timeDiff := data[1].Timestamp.Sub(data[0].Timestamp).Minutes()
	timeFactor := 0.0
	if timeDiff < 5 {
		timeFactor = 15.0
	} else if timeDiff > 30 {
		timeFactor = -10.0
	}

	confidence := baseConfidence + volumeFactor + timeFactor
	return math.Max(0, math.Min(100, confidence))
}

// Уверенность для падения за интервал
func (a *FallAnalyzer) calculateIntervalConfidence(data []types.PriceData, fallPercent float64) float64 {
	if len(data) < 2 {
		return 0.0
	}

	baseConfidence := math.Min(fallPercent*8, 80)

	isContinuous := a.checkContinuity(data)
	continuityBonus := 0.0
	if isContinuous {
		continuityBonus = 20.0
	}

	trendStrength := a.calculateTrendStrength(data)
	trendFactor := math.Min(trendStrength/2, 10)

	confidence := baseConfidence + continuityBonus + trendFactor
	return math.Max(0, math.Min(100, confidence))
}

func (a *FallAnalyzer) checkContinuity(data []types.PriceData) bool {
	if len(data) < 2 {
		return false
	}

	continuousPoints := 0
	totalPoints := len(data) - 1

	for i := 1; i < len(data); i++ {
		if data[i].Price < data[i-1].Price {
			continuousPoints++
		}
	}

	if totalPoints == 0 {
		return false
	}

	continuousRatio := float64(continuousPoints) / float64(totalPoints)
	return continuousRatio > a.config.CustomSettings["continuity_threshold"].(float64)
}

func (a *FallAnalyzer) calculateTrendStrength(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0.0
	}

	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(data))

	for i, point := range data {
		x := float64(i)
		y := point.Price
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	b := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	trendStrength := math.Abs(b) * 1000
	return math.Max(0, math.Min(100, trendStrength))
}

func (a *FallAnalyzer) calculateVolatility(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0.0
	}

	var sum float64
	for _, point := range data {
		sum += point.Price
	}
	mean := sum / float64(len(data))

	var variance float64
	for _, point := range data {
		diff := point.Price - mean
		variance += diff * diff
	}
	variance /= float64(len(data))

	return math.Sqrt(variance) / mean
}

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
}

func (a *FallAnalyzer) GetConfig() AnalyzerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

func (a *FallAnalyzer) GetStats() AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

var DefaultFallConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        1.0,
	MinConfidence: 60.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_fall":             2.0,
		"continuity_threshold": 0.7,
		"volume_weight":        0.2,
	},
}
