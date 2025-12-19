// internal/analysis/analyzers/fall_analyzer.go (исправленная версия)
package analyzers

import (
	"crypto-exchange-screener-bot/internal/analysis"
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
	return "1.0.0"
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

	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price
	change := ((endPrice - startPrice) / startPrice) * 100

	// ДЛЯ ПАДЕНИЯ: изменение должно быть ОТРИЦАТЕЛЬНЫМ
	if change >= 0 {
		// Это рост или стагнация, не падение
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	// Проверяем, достаточно ли сильное падение (берем абсолютное значение)
	fallValue := math.Abs(change)
	if fallValue < a.config.CustomSettings["min_fall"].(float64) {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	isContinuous := a.checkContinuity(data)
	confidence := a.calculateConfidence(data, fallValue, isContinuous)

	if confidence < config.MinConfidence {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	signal := analysis.Signal{
		Symbol:        data[0].Symbol,
		Type:          "fall",
		Direction:     "down",
		ChangePercent: change, // Здесь change ОТРИЦАТЕЛЬНЫЙ
		Confidence:    confidence,
		DataPoints:    len(data),
		StartPrice:    startPrice,
		EndPrice:      endPrice,
		Metadata: analysis.Metadata{
			Strategy:     "fall_detection",
			Tags:         []string{"fall", "bearish"},
			IsContinuous: isContinuous,
			Indicators: map[string]float64{
				"trend_strength": a.calculateTrendStrength(data),
				"volatility":     a.calculateVolatility(data),
			},
		},
	}

	a.updateStats(time.Since(startTime), true)

	return []analysis.Signal{signal}, nil
}

func (a *FallAnalyzer) checkContinuity(data []types.PriceData) bool {
	continuousPoints := 0
	totalPoints := len(data) - 1

	for i := 1; i < len(data); i++ {
		if data[i].Price <= data[i-1].Price {
			continuousPoints++
		}
	}

	continuousRatio := float64(continuousPoints) / float64(totalPoints)
	return continuousRatio > a.config.CustomSettings["continuity_threshold"].(float64)
}

func (a *FallAnalyzer) calculateConfidence(data []types.PriceData, fallPercent float64, isContinuous bool) float64 {
	// Базовый confidence на основе процента падения
	baseConfidence := math.Min(fallPercent*5, 80) // До 80% за падение

	// Учет непрерывности
	continuityBonus := 0.0
	if isContinuous {
		continuityBonus = 15.0
	}

	// Учет волатильности
	volatility := a.calculateVolatility(data)
	volatilityFactor := 0.0
	if volatility < 0.05 { // Низкая волатильность
		volatilityFactor = 10.0
	} else if volatility > 0.15 { // Высокая волатильность
		volatilityFactor = -5.0
	}

	confidence := baseConfidence + continuityBonus + volatilityFactor
	return math.Max(0, math.Min(100, confidence))
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

	// Линейная регрессия: y = a + bx
	b := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Нормализуем наклон для получения силы тренда (0-100)
	// Отрицательный наклон указывает на падение
	trendStrength := math.Abs(b) * 1000
	return math.Max(0, math.Min(100, trendStrength))
}

func (a *FallAnalyzer) calculateVolatility(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0.0
	}

	// Рассчитываем среднюю цену
	var sum float64
	for _, point := range data {
		sum += point.Price
	}
	mean := sum / float64(len(data))

	// Рассчитываем стандартное отклонение
	var variance float64
	for _, point := range data {
		diff := point.Price - mean
		variance += diff * diff
	}
	variance /= float64(len(data))

	return math.Sqrt(variance) / mean // Относительная волатильность
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
