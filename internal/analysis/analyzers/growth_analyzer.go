// internal/analysis/analyzers/growth_analyzer.go
package analyzers

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/common"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// GrowthAnalyzer - анализатор роста
type GrowthAnalyzer struct {
	config analysis.AnalyzerConfig
	stats  analysis.AnalyzerStats
	mu     sync.RWMutex
}

// Name возвращает имя анализатора
func (a *GrowthAnalyzer) Name() string {
	return "growth_analyzer"
}

// Version возвращает версию
func (a *GrowthAnalyzer) Version() string {
	return "1.0.0"
}

// Supports проверяет поддержку символа
func (a *GrowthAnalyzer) Supports(symbol string) bool {
	// Поддерживаем все символы
	return true
}

// Analyze анализирует данные на рост
func (a *GrowthAnalyzer) Analyze(data []common.PriceData, config analysis.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) < config.MinDataPoints {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("insufficient data points")
	}

	// Сортируем по времени
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	// Рассчитываем рост
	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price
	change := ((endPrice - startPrice) / startPrice) * 100

	// Если рост меньше порога, пропускаем
	if change < a.config.CustomSettings["min_growth"].(float64) {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	// Проверяем непрерывность
	isContinuous := a.checkContinuity(data)

	// Рассчитываем уверенность
	confidence := a.calculateConfidence(data, change, isContinuous)

	// Если уверенность ниже порога, пропускаем
	if confidence < config.MinConfidence {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	signal := analysis.Signal{
		Symbol:        data[0].Symbol,
		Type:          "growth",
		Direction:     "up",
		ChangePercent: change,
		Confidence:    confidence,
		DataPoints:    len(data),
		StartPrice:    startPrice,
		EndPrice:      endPrice,
		Timestamp:     time.Now(),
		Metadata: analysis.SignalMetadata{
			Strategy:     "growth_detection",
			Tags:         []string{"growth", "bullish"},
			IsContinuous: isContinuous,
			Indicators: map[string]float64{
				"trend_strength": a.calculateTrendStrength(data),
				"volatility":     a.calculateVolatility(data),
			},
		},
	}

	// Обновляем статистику
	a.updateStats(time.Since(startTime), true)

	return []analysis.Signal{signal}, nil
}

// GetConfig возвращает конфигурацию
func (a *GrowthAnalyzer) GetConfig() analysis.AnalyzerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// GetStats возвращает статистику
func (a *GrowthAnalyzer) GetStats() analysis.AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

// checkContinuity проверяет непрерывность роста
func (a *GrowthAnalyzer) checkContinuity(data []common.PriceData) bool {
	continuousPoints := 0
	totalPoints := len(data) - 1

	for i := 1; i < len(data); i++ {
		if data[i].Price >= data[i-1].Price {
			continuousPoints++
		}
	}

	continuousRatio := float64(continuousPoints) / float64(totalPoints)
	return continuousRatio > a.config.CustomSettings["continuity_threshold"].(float64)
}

// calculateConfidence рассчитывает уверенность
func (a *GrowthAnalyzer) calculateConfidence(data []common.PriceData, change float64, isContinuous bool) float64 {
	confidence := 0.0

	// 1. Изменение цены (макс 40%)
	confidence += math.Min(change*2, 40)

	// 2. Непрерывность (макс 30%)
	if isContinuous {
		confidence += 30
	}

	// 3. Количество точек данных (макс 30%)
	dataPointScore := math.Min(float64(len(data))/50.0*30, 30)
	confidence += dataPointScore

	return math.Min(confidence, 100)
}

// calculateTrendStrength рассчитывает силу тренда
func (a *GrowthAnalyzer) calculateTrendStrength(data []common.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	var totalChange float64
	for i := 1; i < len(data); i++ {
		change := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100
		totalChange += change
	}

	return totalChange / float64(len(data)-1)
}

// calculateVolatility рассчитывает волатильность
func (a *GrowthAnalyzer) calculateVolatility(data []common.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	var sum, mean float64
	for _, point := range data {
		sum += point.Price
	}
	mean = sum / float64(len(data))

	var variance float64
	for _, point := range data {
		variance += (point.Price - mean) * (point.Price - mean)
	}
	variance /= float64(len(data))

	// Возвращаем стандартное отклонение в процентах от средней цены
	return (math.Sqrt(variance) / mean) * 100
}

// updateStats обновляет статистику
func (a *GrowthAnalyzer) updateStats(duration time.Duration, success bool) {
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

// DefaultGrowthConfig - конфигурация по умолчанию
var DefaultGrowthConfig = analysis.AnalyzerConfig{
	Enabled:       true,
	Weight:        1.0,
	MinConfidence: 60.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_growth":           2.0,
		"continuity_threshold": 0.7,
		"volume_weight":        0.2,
	},
}
