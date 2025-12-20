// internal/analysis/analyzers/continuous_analyzer.go
package analyzers

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"math"
	"sync"
	"time"
)

// ContinuousAnalyzer - анализатор непрерывности
type ContinuousAnalyzer struct {
	config AnalyzerConfig
	stats  AnalyzerStats
	mu     sync.RWMutex
}

type SequenceInfo struct {
	StartIdx  int
	Length    int
	Direction string
	AvgGap    float64
	AvgChange float64
}

func (a *ContinuousAnalyzer) Name() string {
	return "continuous_analyzer"
}

func (a *ContinuousAnalyzer) Version() string {
	return "1.0.0"
}

func (a *ContinuousAnalyzer) Supports(symbol string) bool {
	return true
}

func (a *ContinuousAnalyzer) Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) < config.MinDataPoints {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("insufficient data points")
	}

	// Анализ непрерывности
	continuous, score := a.checkContinuity(data)

	if !continuous {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	signal := analysis.Signal{
		Symbol:        data[0].Symbol,
		Type:          "continuous_trend",
		Direction:     a.determineDirection(data),
		ChangePercent: a.calculateChange(data),
		Confidence:    score,
		DataPoints:    len(data),
		StartPrice:    data[0].Price,
		EndPrice:      data[len(data)-1].Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy:     "continuity_analysis",
			Tags:         []string{"continuous", "trend"},
			IsContinuous: true,
			Indicators: map[string]float64{
				"continuity_score": score,
				"trend_length":     float64(len(data)),
			},
		},
	}

	a.updateStats(time.Since(startTime), true)

	return []analysis.Signal{signal}, nil
}

func (a *ContinuousAnalyzer) checkContinuity(data []types.PriceData) (bool, float64) {
	minPoints := a.config.CustomSettings["min_continuous_points"].(int)
	maxGapRatio := a.config.CustomSettings["max_gap_ratio"].(float64)

	if len(data) < minPoints {
		return false, 0
	}

	// Ищем последовательные движения в одном направлении
	bestSequence := a.findBestSequence(data)

	if bestSequence.Length < minPoints {
		return false, 0
	}

	// Рассчитываем оценку
	score := float64(bestSequence.Length) / float64(len(data)) * 50
	score += (1.0 - bestSequence.AvgGap/maxGapRatio) * 30
	score += math.Min(math.Abs(bestSequence.AvgChange), 20)

	return true, math.Min(score, 100)
}

func (a *ContinuousAnalyzer) calculateGap(prev, curr float64) float64 {
	if prev == 0 {
		return 0
	}
	return math.Abs((curr - prev) / prev)
}

func (a *ContinuousAnalyzer) determineDirection(data []types.PriceData) string {
	if len(data) < 2 {
		return "neutral"
	}

	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price

	if endPrice > startPrice {
		return "up"
	} else if endPrice < startPrice {
		return "down"
	}
	return "neutral"
}

func (a *ContinuousAnalyzer) calculateChange(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price

	if startPrice == 0 {
		return 0
	}

	return ((endPrice - startPrice) / startPrice) * 100
}

func (a *ContinuousAnalyzer) updateStats(duration time.Duration, success bool) {
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

func (a *ContinuousAnalyzer) GetConfig() AnalyzerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

func (a *ContinuousAnalyzer) GetStats() AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

var DefaultContinuousConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        0.8,
	MinConfidence: 60.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_continuous_points": 3,
		"max_gap_ratio":         0.3,
	},
}

func (a *ContinuousAnalyzer) findBestSequence(data []types.PriceData) SequenceInfo {
	best := SequenceInfo{}
	current := SequenceInfo{StartIdx: 0, Length: 1}

	for i := 1; i < len(data); i++ {
		prevPrice := data[i-1].Price
		currPrice := data[i].Price

		// Определяем направление этого изменения
		var direction string
		if currPrice > prevPrice {
			direction = "up"
		} else if currPrice < prevPrice {
			direction = "down"
		} else {
			direction = "neutral"
		}

		// Если направление совпадает с текущей последовательностью или она только началась
		if current.Length == 1 || current.Direction == direction {
			current.Direction = direction
			current.Length++

			// Обновляем средние значения
			gap := a.calculateGap(prevPrice, currPrice)
			change := ((currPrice - prevPrice) / prevPrice) * 100

			current.AvgGap = (current.AvgGap*float64(current.Length-2) + gap) / float64(current.Length-1)
			current.AvgChange = (current.AvgChange*float64(current.Length-2) + change) / float64(current.Length-1)
		} else {
			// Завершаем текущую последовательность
			if current.Length > best.Length {
				best = current
			}
			// Начинаем новую
			current = SequenceInfo{
				StartIdx:  i - 1,
				Length:    2,
				Direction: direction,
			}
		}
	}

	// Проверяем последнюю последовательность
	if current.Length > best.Length {
		best = current
	}

	return best
}
