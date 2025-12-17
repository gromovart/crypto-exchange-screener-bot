package monitor

import (
	"crypto-exchange-screener-bot/internal/types"
	"sort"
	"time"
)

// SimpleTrendAnalyzer простой анализатор трендов
type SimpleTrendAnalyzer struct {
	growthThreshold  float64
	fallThreshold    float64
	supportedPeriods []int
	minDataPoints    int
}

// NewSimpleTrendAnalyzer создает новый анализатор
func NewSimpleTrendAnalyzer(growthThreshold, fallThreshold float64) *SimpleTrendAnalyzer {
	return &SimpleTrendAnalyzer{
		growthThreshold:  growthThreshold,
		fallThreshold:    fallThreshold,
		supportedPeriods: []int{5, 15, 30, 60}, // 5м, 15м, 30м, 1ч
		minDataPoints:    2,
	}
}

// Analyze анализирует историю цен и возвращает сигнал
func (a *SimpleTrendAnalyzer) Analyze(symbol string, history []types.PriceData) (types.TrendSignal, error) {
	if len(history) < a.minDataPoints {
		return types.TrendSignal{}, nil
	}

	// Сортируем по времени (старые -> новые)
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.Before(history[j].Timestamp)
	})

	// Вычисляем изменение между первой и последней точкой
	firstPrice := history[0].Price
	lastPrice := history[len(history)-1].Price
	change := lastPrice - firstPrice
	changePercent := (change / firstPrice) * 100

	// Определяем период в минутах
	periodMinutes := int(history[len(history)-1].Timestamp.Sub(history[0].Timestamp).Minutes())

	// Определяем направление
	direction := "growth"
	if change < 0 {
		direction = "fall"
		changePercent = -changePercent
	}

	// Проверяем пороги
	if direction == "growth" && changePercent < a.growthThreshold {
		return types.TrendSignal{}, nil
	}
	if direction == "fall" && changePercent < a.fallThreshold {
		return types.TrendSignal{}, nil
	}

	// Рассчитываем уверенность (на основе количества точек и волатильности)
	confidence := a.calculateConfidence(history, direction)

	signal := types.TrendSignal{
		Symbol:        symbol,
		Direction:     direction,
		ChangePercent: changePercent,
		PeriodMinutes: periodMinutes,
		Confidence:    confidence,
		Timestamp:     time.Now(),
		DataPoints:    len(history),
	}

	return signal, nil
}

// calculateConfidence рассчитывает уверенность сигнала
func (a *SimpleTrendAnalyzer) calculateConfidence(history []types.PriceData, direction string) float64 {
	if len(history) < 3 {
		return 50.0
	}

	// Базовый уровень уверенности на основе количества точек (не более 100 точек)
	points := len(history)
	if points > 100 {
		points = 100
	}
	baseConfidence := float64(points) * 0.5

	// Проверяем непрерывность тренда
	continuityScore := a.checkContinuity(history, direction)

	// Рассчитываем волатильность
	volatilityScore := a.calculateVolatilityScore(history)

	// Итоговая уверенность
	confidence := baseConfidence + continuityScore*20 + volatilityScore*30

	return a.minFloat(confidence, 100.0)
}

// checkContinuity проверяет непрерывность тренда
func (a *SimpleTrendAnalyzer) checkContinuity(history []types.PriceData, direction string) float64 {
	continuousPoints := 0
	totalPoints := len(history) - 1

	for i := 1; i < len(history); i++ {
		if direction == "growth" && history[i].Price >= history[i-1].Price {
			continuousPoints++
		} else if direction == "fall" && history[i].Price <= history[i-1].Price {
			continuousPoints++
		}
	}

	return float64(continuousPoints) / float64(totalPoints)
}

// calculateVolatilityScore рассчитывает оценку волатильности
func (a *SimpleTrendAnalyzer) calculateVolatilityScore(history []types.PriceData) float64 {
	if len(history) < 2 {
		return 0.5
	}

	// Вычисляем стандартное отклонение
	var sum, mean, sd float64
	for _, point := range history {
		sum += point.Price
	}
	mean = sum / float64(len(history))

	for _, point := range history {
		sd += (point.Price - mean) * (point.Price - mean)
	}
	sd = sd / float64(len(history))

	// Нормализуем волатильность (чем меньше волатильность, тем лучше)
	volatility := sd / mean
	if volatility < 0.01 { // 1%
		return 1.0
	} else if volatility < 0.05 { // 5%
		return 0.8
	} else if volatility < 0.1 { // 10%
		return 0.5
	}
	return 0.2
}

// GetSupportedPeriods возвращает поддерживаемые периоды
func (a *SimpleTrendAnalyzer) GetSupportedPeriods() []int {
	return a.supportedPeriods
}

// SetThresholds устанавливает пороги
func (a *SimpleTrendAnalyzer) SetThresholds(growth, fall float64) {
	a.growthThreshold = growth
	a.fallThreshold = fall
}

// GetThresholds возвращает пороги
func (a *SimpleTrendAnalyzer) GetThresholds() (float64, float64) {
	return a.growthThreshold, a.fallThreshold
}

// GetStats возвращает статистику
func (a *SimpleTrendAnalyzer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"growth_threshold":  a.growthThreshold,
		"fall_threshold":    a.fallThreshold,
		"supported_periods": a.supportedPeriods,
		"min_data_points":   a.minDataPoints,
	}
}

// minFloat возвращает минимальное из двух float64
func (a *SimpleTrendAnalyzer) minFloat(x, y float64) float64 {
	if x < y {
		return x
	}
	return y
}
