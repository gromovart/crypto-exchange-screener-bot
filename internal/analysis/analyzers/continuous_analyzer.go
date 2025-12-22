package analyzers

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/common"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// ContinuousAnalyzer - анализатор непрерывности
type ContinuousAnalyzer struct {
	config analysis.AnalyzerConfig
	stats  analysis.AnalyzerStats
	mu     sync.RWMutex
}

type SequenceInfo struct {
	StartIdx  int
	Length    int
	Direction analysis.TrendDirection
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

func (a *ContinuousAnalyzer) Analyze(data []common.PriceData, config analysis.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	minPoints := a.getMinContinuousPoints()
	if len(data) < minPoints {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("недостаточно точек данных: нужно минимум %d, получено %d", minPoints, len(data))
	}

	var signals []analysis.Signal

	// Ищем непрерывные последовательности роста
	if growthSignals := a.checkContinuousGrowth(data, minPoints); len(growthSignals) > 0 {
		signals = append(signals, growthSignals...)
	}

	// Ищем непрерывные последовательности падения
	if fallSignals := a.checkContinuousFall(data, minPoints); len(fallSignals) > 0 {
		signals = append(signals, fallSignals...)
	}

	// Если не нашли конкретных последовательностей, ищем общую тенденцию
	if len(signals) == 0 {
		bestSequence := a.findBestSequence(data)
		if bestSequence.Length >= minPoints {
			signal := a.createSequenceSignal(data, bestSequence)
			signals = append(signals, signal)
		}
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

// checkContinuousGrowth ищет непрерывные последовательности роста
func (a *ContinuousAnalyzer) checkContinuousGrowth(data []common.PriceData, minPoints int) []analysis.Signal {
	var signals []analysis.Signal
	symbol := data[0].Symbol

	for i := 0; i <= len(data)-minPoints; i++ {
		continuous := true
		startPrice := data[i].Price

		// Проверяем minPoints подряд
		for j := i; j < i+minPoints-1; j++ {
			if j+1 >= len(data) {
				continuous = false
				break
			}

			change := ((data[j+1].Price - data[j].Price) / data[j].Price) * 100
			if change <= 0 { // Не рост
				continuous = false
				break
			}
		}

		if continuous {
			endPrice := data[i+minPoints-1].Price
			totalChangePercent := ((endPrice - startPrice) / startPrice) * 100

			signal := a.createSignal(
				symbol,
				analysis.TrendBullish,
				totalChangePercent,
				minPoints,
				i,
				i+minPoints-1,
				startPrice,
				endPrice,
			)
			signals = append(signals, signal)
		}
	}

	return signals
}

// checkContinuousFall ищет непрерывные последовательности падения
func (a *ContinuousAnalyzer) checkContinuousFall(data []common.PriceData, minPoints int) []analysis.Signal {
	var signals []analysis.Signal
	symbol := data[0].Symbol

	for i := 0; i <= len(data)-minPoints; i++ {
		continuous := true
		startPrice := data[i].Price

		// Проверяем minPoints подряд
		for j := i; j < i+minPoints-1; j++ {
			if j+1 >= len(data) {
				continuous = false
				break
			}

			change := ((data[j+1].Price - data[j].Price) / data[j].Price) * 100
			if change >= 0 { // Не падение
				continuous = false
				break
			}
		}

		if continuous {
			endPrice := data[i+minPoints-1].Price
			totalChangePercent := ((endPrice - startPrice) / startPrice) * 100

			signal := a.createSignal(
				symbol,
				analysis.TrendBearish,
				totalChangePercent,
				minPoints,
				i,
				i+minPoints-1,
				startPrice,
				endPrice,
			)
			signals = append(signals, signal)
		}
	}

	return signals
}

// createSignal создает сигнал непрерывного движения
func (a *ContinuousAnalyzer) createSignal(
	symbol common.Symbol,
	direction analysis.TrendDirection,
	change float64,
	points, startIdx, endIdx int,
	startPrice, endPrice float64,
) analysis.Signal {
	confidence := a.calculateConfidence(points, math.Abs(change))

	// Определяем тип сигнала
	var signalType analysis.SignalType
	switch direction {
	case analysis.TrendBullish:
		signalType = analysis.SignalTypeContinuous
	case analysis.TrendBearish:
		signalType = analysis.SignalTypeContinuous
	default:
		signalType = analysis.SignalTypeContinuous
	}

	return analysis.Signal{
		Symbol:        symbol,
		Type:          signalType,
		Direction:     direction,
		ChangePercent: change,
		Confidence:    confidence,
		Strength:      confidence / 100.0, // Преобразуем из 0-100 в 0-1
		DataPoints:    points,
		StartPrice:    startPrice,
		EndPrice:      endPrice,
		Timestamp:     time.Now(),
		Metadata: analysis.SignalMetadata{
			Strategy:       "continuous_analyzer",
			Tags:           []string{"continuous", string(direction), fmt.Sprintf("points_%d", points)},
			IsContinuous:   true,
			ContinuousFrom: startIdx,
			ContinuousTo:   endIdx,
			Indicators: map[string]float64{
				"continuous_points": float64(points),
				"total_change":      change,
				"avg_change":        change / float64(points),
			},
		},
	}
}

// createSequenceSignal создает сигнал на основе найденной последовательности
func (a *ContinuousAnalyzer) createSequenceSignal(data []common.PriceData, sequence SequenceInfo) analysis.Signal {
	symbol := data[0].Symbol
	startIdx := sequence.StartIdx
	endIdx := sequence.StartIdx + sequence.Length - 1

	startPrice := data[startIdx].Price
	endPrice := data[endIdx].Price
	change := ((endPrice - startPrice) / startPrice) * 100

	confidence := a.calculateConfidence(sequence.Length, math.Abs(change))

	return analysis.Signal{
		Symbol:        symbol,
		Type:          analysis.SignalTypeContinuous,
		Direction:     sequence.Direction,
		ChangePercent: change,
		Confidence:    confidence,
		Strength:      confidence / 100.0, // Преобразуем из 0-100 в 0-1
		DataPoints:    sequence.Length,
		StartPrice:    startPrice,
		EndPrice:      endPrice,
		Timestamp:     time.Now(),
		Metadata: analysis.SignalMetadata{
			Strategy:       "continuous_analyzer",
			Tags:           []string{"continuous", "trend", string(sequence.Direction)},
			IsContinuous:   true,
			ContinuousFrom: startIdx,
			ContinuousTo:   endIdx,
			Indicators: map[string]float64{
				"trend_length":     float64(sequence.Length),
				"avg_gap":          sequence.AvgGap,
				"avg_change":       sequence.AvgChange,
				"continuity_score": confidence,
			},
		},
	}
}

// getMinContinuousPoints получает минимальное количество непрерывных точек
func (a *ContinuousAnalyzer) getMinContinuousPoints() int {
	if a.config.CustomSettings == nil {
		return 3
	}

	val := a.config.CustomSettings["min_continuous_points"]
	if val == nil {
		return 3
	}

	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 3
	}
}

// calculateConfidence рассчитывает уверенность сигнала
func (a *ContinuousAnalyzer) calculateConfidence(points int, absoluteChange float64) float64 {
	// Базовая уверенность на основе количества точек (максимум 60%)
	baseConfidence := math.Min(float64(points)*20.0, 60.0)

	// Дополнительная уверенность на основе величины изменения (максимум 40%)
	changeConfidence := math.Min(absoluteChange*2.0, 40.0)

	totalConfidence := baseConfidence + changeConfidence

	// Ограничиваем 100%
	return math.Min(totalConfidence, 100.0)
}

// findBestSequence находит лучшую непрерывную последовательность
func (a *ContinuousAnalyzer) findBestSequence(data []common.PriceData) SequenceInfo {
	if len(data) < 2 {
		return SequenceInfo{}
	}

	best := SequenceInfo{}
	current := SequenceInfo{
		StartIdx:  0,
		Length:    1,
		Direction: analysis.TrendSideways,
	}

	for i := 1; i < len(data); i++ {
		prevPrice := data[i-1].Price
		currPrice := data[i].Price

		// Определяем направление изменения
		var direction analysis.TrendDirection
		if currPrice > prevPrice {
			direction = analysis.TrendBullish
		} else if currPrice < prevPrice {
			direction = analysis.TrendBearish
		} else {
			direction = analysis.TrendSideways
		}

		// Если направление совпадает или мы только начинаем
		if current.Length == 1 || current.Direction == direction || direction == analysis.TrendSideways {
			if current.Direction == analysis.TrendSideways && direction != analysis.TrendSideways {
				current.Direction = direction
			}
			current.Length++

			// Обновляем средние значения
			if prevPrice != 0 {
				gap := math.Abs((currPrice - prevPrice) / prevPrice)
				change := ((currPrice - prevPrice) / prevPrice) * 100

				if current.Length == 2 {
					current.AvgGap = gap
					current.AvgChange = change
				} else {
					current.AvgGap = (current.AvgGap*float64(current.Length-2) + gap) / float64(current.Length-1)
					current.AvgChange = (current.AvgChange*float64(current.Length-2) + change) / float64(current.Length-1)
				}
			}
		} else {
			// Сохраняем лучшую последовательность
			if current.Length > best.Length {
				best = current
			}
			// Начинаем новую последовательность
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

// getTrendDirection преобразует строку в TrendDirection
func getTrendDirection(dir string) analysis.TrendDirection {
	switch strings.ToLower(dir) {
	case "up", "growth", "bullish":
		return analysis.TrendBullish
	case "down", "fall", "bearish":
		return analysis.TrendBearish
	default:
		return analysis.TrendSideways
	}
}

// Вспомогательные методы

func (a *ContinuousAnalyzer) calculateGap(prev, curr float64) float64 {
	if prev == 0 {
		return 0
	}
	return math.Abs((curr - prev) / prev)
}

func (a *ContinuousAnalyzer) determineDirection(data []common.PriceData) analysis.TrendDirection {
	if len(data) < 2 {
		return analysis.TrendSideways
	}

	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price

	if endPrice > startPrice {
		return analysis.TrendBullish
	} else if endPrice < startPrice {
		return analysis.TrendBearish
	}
	return analysis.TrendSideways
}

func (a *ContinuousAnalyzer) calculateChange(data []common.PriceData) float64 {
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

func (a *ContinuousAnalyzer) GetConfig() analysis.AnalyzerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

func (a *ContinuousAnalyzer) GetStats() analysis.AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

// DefaultContinuousConfig - конфигурация по умолчанию
var DefaultContinuousConfig = analysis.AnalyzerConfig{
	Enabled:       true,
	Weight:        0.8,
	MinConfidence: 60.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_continuous_points": 3,
		"max_gap_ratio":         0.3,
	},
}
