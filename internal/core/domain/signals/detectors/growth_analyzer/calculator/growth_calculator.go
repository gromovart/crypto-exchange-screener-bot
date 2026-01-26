// internal/core/domain/signals/detectors/growth_analyzer/calculator/growth_calculator.go
package calculator

import (
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"math"
	"sort"
	"time"
)

// GrowthCalculatorInput - входные данные для калькулятора роста
type GrowthCalculatorInput struct {
	PriceData   []redis_storage.PriceData `json:"price_data"`
	Config      CalculatorConfig          `json:"config"`
	CurrentTime time.Time                 `json:"current_time"`
}

// GrowthCalculatorOutput - выходные данные калькулятора роста
type GrowthCalculatorOutput struct {
	GrowthPercent   float64 `json:"growth_percent"`
	IsContinuous    bool    `json:"is_continuous"`
	ContinuityScore float64 `json:"continuity_score"`
	Acceleration    float64 `json:"acceleration"`
	TrendStrength   float64 `json:"trend_strength"`
	Volatility      float64 `json:"volatility"`
	SignalType      string  `json:"signal_type"`
	ConfidenceScore float64 `json:"confidence_score"`
	Recommendation  string  `json:"recommendation"`
}

// CalculateGrowth - основной калькулятор роста
func CalculateGrowth(input GrowthCalculatorInput) (GrowthCalculatorOutput, error) {
	if len(input.PriceData) < 2 {
		return GrowthCalculatorOutput{}, ErrInsufficientData(ErrInsufficientDataMsg)
	}

	// Сортируем данные по времени
	sortedData := make([]redis_storage.PriceData, len(input.PriceData))
	copy(sortedData, input.PriceData)
	sort.Slice(sortedData, func(i, j int) bool {
		return sortedData[i].Timestamp.Before(sortedData[j].Timestamp)
	})

	// Рассчитываем базовые метрики
	growthPercent := calculateGrowthPercent(sortedData)
	continuityScore, isContinuous := calculateContinuity(sortedData, input.Config.ContinuityThreshold)
	acceleration := calculateAcceleration(sortedData)
	trendStrength := calculateTrendStrength(sortedData)
	volatility := calculateVolatility(sortedData)

	// Определяем тип сигнала
	signalType := determineSignalType(growthPercent, continuityScore, acceleration, input.Config)

	// Рассчитываем уверенность
	confidenceScore := calculateBaseConfidence(
		growthPercent,
		continuityScore,
		acceleration,
		trendStrength,
		volatility,
		input.Config,
	)

	// Формируем рекомендацию
	recommendation := generateRecommendation(growthPercent, signalType, confidenceScore)

	return GrowthCalculatorOutput{
		GrowthPercent:   growthPercent,
		IsContinuous:    isContinuous,
		ContinuityScore: continuityScore,
		Acceleration:    acceleration,
		TrendStrength:   trendStrength,
		Volatility:      volatility,
		SignalType:      signalType,
		ConfidenceScore: confidenceScore,
		Recommendation:  recommendation,
	}, nil
}

// calculateGrowthPercent - рассчитывает процент роста
func calculateGrowthPercent(data []redis_storage.PriceData) float64 {
	if len(data) < 2 {
		return 0.0
	}
	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price
	return ((endPrice - startPrice) / startPrice) * 100
}

// calculateContinuity - рассчитывает непрерывность роста
func calculateContinuity(data []redis_storage.PriceData, threshold float64) (float64, bool) {
	if len(data) < 2 {
		return 0.0, false
	}

	continuousPoints := 0
	totalPoints := len(data) - 1

	for i := 1; i < len(data); i++ {
		if data[i].Price >= data[i-1].Price {
			continuousPoints++
		}
	}

	continuityScore := float64(continuousPoints) / float64(totalPoints)
	return continuityScore, continuityScore > threshold
}

// calculateAcceleration - рассчитывает ускорение роста
func calculateAcceleration(data []redis_storage.PriceData) float64 {
	if len(data) < 3 {
		return 0.0
	}

	// Делим данные на три равные части
	third := len(data) / 3
	if third < 1 {
		return 0.0
	}

	// Рассчитываем рост для каждой трети
	firstGrowth := calculateSegmentGrowth(data, 0, third*2)
	secondGrowth := calculateSegmentGrowth(data, third, len(data)-1)

	// Ускорение = рост второй трети - рост первой трети
	return secondGrowth - firstGrowth
}

// calculateSegmentGrowth - рассчитывает рост для сегмента данных
func calculateSegmentGrowth(data []redis_storage.PriceData, start, end int) float64 {
	if end <= start || start < 0 || end >= len(data) {
		return 0.0
	}

	startPrice := data[start].Price
	endPrice := data[end].Price
	return ((endPrice - startPrice) / startPrice) * 100
}

// calculateTrendStrength - рассчитывает силу тренда
func calculateTrendStrength(data []redis_storage.PriceData) float64 {
	if len(data) < 2 {
		return 0.0
	}

	var totalChange float64
	for i := 1; i < len(data); i++ {
		change := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100
		totalChange += change
	}

	return totalChange / float64(len(data)-1)
}

// calculateVolatility - рассчитывает волатильность
func calculateVolatility(data []redis_storage.PriceData) float64 {
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
		variance += (point.Price - mean) * (point.Price - mean)
	}
	variance /= float64(len(data))

	// Возвращаем стандартное отклонение в процентах от средней цены
	return (math.Sqrt(variance) / mean) * 100
}

// determineSignalType - определяет тип сигнала роста
func determineSignalType(growthPercent, continuityScore, acceleration float64, config CalculatorConfig) string {
	if acceleration > config.AccelerationThreshold && growthPercent > config.MinGrowthPercent*1.5 {
		return "accelerated_growth"
	}

	if continuityScore > config.ContinuityThreshold && growthPercent > config.MinGrowthPercent {
		return "continuous_growth"
	}

	if growthPercent > config.MinGrowthPercent*2 {
		return "breakout_growth"
	}

	return "continuous_growth" // default
}

// calculateBaseConfidence - рассчитывает базовую уверенность
func calculateBaseConfidence(growthPercent, continuityScore, acceleration, trendStrength, volatility float64, config CalculatorConfig) float64 {
	confidence := 0.0

	// 1. Изменение цены (макс 40%)
	priceScore := math.Min(growthPercent*2, 40)
	confidence += priceScore

	// 2. Непрерывность (макс 30%)
	if continuityScore > config.ContinuityThreshold {
		continuityScore := continuityScore * 30
		confidence += continuityScore
	}

	// 3. Ускорение (макс 15%)
	if acceleration > 0 {
		accelerationScore := math.Min(acceleration*10, 15)
		confidence += accelerationScore
	}

	// 4. Сила тренда (макс 10%)
	trendScore := math.Min(math.Abs(trendStrength)*5, 10)
	confidence += trendScore

	// 5. Низкая волатильность бонус (макс 5%)
	if volatility < 5 {
		volatilityBonus := (5 - volatility) * 1
		confidence += math.Min(volatilityBonus, 5)
	}

	return math.Min(confidence, 100)
}

// generateRecommendation - генерирует текстовую рекомендацию
func generateRecommendation(growthPercent float64, signalType string, confidence float64) string {
	if confidence < 50 {
		return "Слабый сигнал роста"
	}

	if confidence < 70 {
		switch signalType {
		case "continuous_growth":
			return "Умеренный устойчивый рост"
		case "accelerated_growth":
			return "Ускоренный рост с потенциалом"
		case "breakout_growth":
			return "Пробойной рост"
		default:
			return "Сигнал роста"
		}
	}

	// confidence >= 70
	switch signalType {
	case "continuous_growth":
		return "Сильный устойчивый рост"
	case "accelerated_growth":
		return "Сильный ускоренный рост"
	case "breakout_growth":
		return "Сильный пробойной рост"
	default:
		return "Очень сильный сигнал роста"
	}
}

// ErrInsufficientData - ошибка недостаточных данных
type ErrInsufficientData string

func (e ErrInsufficientData) Error() string {
	return string(e)
}

const ErrInsufficientDataMsg = "insufficient data for calculation"
