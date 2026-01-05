// internal/core/domain/signals/detectors/counter/calculator/technical_calculator.go
package calculator

import (
	"math"

	"crypto-exchange-screener-bot/internal/types"
)

// TechnicalCalculator - калькулятор технических индикаторов
type TechnicalCalculator struct{}

// NewTechnicalCalculator создает новый калькулятор технических индикаторов
func NewTechnicalCalculator() *TechnicalCalculator {
	return &TechnicalCalculator{}
}

// CalculateRSI рассчитывает RSI
func (c *TechnicalCalculator) CalculateRSI(prices []types.PriceData) float64 {
	if len(prices) < 14 {
		return 50.0 // Нейтральное значение
	}

	var gains, losses float64
	for i := 1; i < len(prices); i++ {
		change := prices[i].Price - prices[i-1].Price
		if change > 0 {
			gains += change
		} else {
			losses += math.Abs(change)
		}
	}

	if gains+losses == 0 {
		return 50.0
	}

	avgGain := gains / float64(len(prices)-1)
	avgLoss := losses / float64(len(prices)-1)

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	// Ограничиваем RSI в пределах 0-100
	if rsi > 100 {
		return 100
	}
	if rsi < 0 {
		return 0
	}

	return rsi
}

// CalculateMACD рассчитывает MACD
func (c *TechnicalCalculator) CalculateMACD(prices []types.PriceData) float64 {
	if len(prices) < 26 {
		return 0
	}

	// EMA12 - EMA26
	period12 := 12
	period26 := 26

	if len(prices) < period26 {
		return 0
	}

	// EMA12
	var sum12 float64
	for i := len(prices) - period12; i < len(prices); i++ {
		sum12 += prices[i].Price
	}
	ema12 := sum12 / float64(period12)

	// EMA26
	var sum26 float64
	for i := len(prices) - period26; i < len(prices); i++ {
		sum26 += prices[i].Price
	}
	ema26 := sum26 / float64(period26)

	// MACD сигнал (разница)
	macd := ema12 - ema26

	return macd
}

// CalculateVolatility рассчитывает волатильность
func (c *TechnicalCalculator) CalculateVolatility(prices []types.PriceData) float64 {
	if len(prices) < 2 {
		return 0
	}

	var sum float64
	for _, point := range prices {
		sum += point.Price
	}
	mean := sum / float64(len(prices))

	var variance float64
	for _, point := range prices {
		diff := point.Price - mean
		variance += diff * diff
	}
	variance /= float64(len(prices))

	// Возвращаем стандартное отклонение в процентах от средней цены
	return (math.Sqrt(variance) / mean) * 100
}

// CalculateTrendStrength рассчитывает силу тренда
func (c *TechnicalCalculator) CalculateTrendStrength(prices []types.PriceData) float64 {
	if len(prices) < 2 {
		return 0
	}

	var totalChange float64
	for i := 1; i < len(prices); i++ {
		change := ((prices[i].Price - prices[i-1].Price) / prices[i-1].Price) * 100
		totalChange += change
	}

	avgChange := totalChange / float64(len(prices)-1)
	return math.Abs(avgChange)
}

// CalculateAverageChange рассчитывает среднее изменение
func (c *TechnicalCalculator) CalculateAverageChange(prices []types.PriceData) float64 {
	if len(prices) < 2 {
		return 0
	}

	startPrice := prices[0].Price
	endPrice := prices[len(prices)-1].Price

	if startPrice == 0 {
		return 0
	}

	return ((endPrice - startPrice) / startPrice) * 100
}

// IsContinuousGrowth проверяет непрерывный рост
func (c *TechnicalCalculator) IsContinuousGrowth(prices []types.PriceData, threshold float64) bool {
	if len(prices) < 2 {
		return false
	}

	continuousPoints := 0
	totalPoints := len(prices) - 1

	for i := 1; i < len(prices); i++ {
		change := ((prices[i].Price - prices[i-1].Price) / prices[i-1].Price) * 100
		if change > 0 {
			continuousPoints++
		}
	}

	continuousRatio := float64(continuousPoints) / float64(totalPoints)
	return continuousRatio > threshold
}

// IsContinuousFall проверяет непрерывное падение
func (c *TechnicalCalculator) IsContinuousFall(prices []types.PriceData, threshold float64) bool {
	if len(prices) < 2 {
		return false
	}

	continuousPoints := 0
	totalPoints := len(prices) - 1

	for i := 1; i < len(prices); i++ {
		change := ((prices[i].Price - prices[i-1].Price) / prices[i-1].Price) * 100
		if change < 0 {
			continuousPoints++
		}
	}

	continuousRatio := float64(continuousPoints) / float64(totalPoints)
	return continuousRatio > threshold
}

// CalculateMinMax рассчитывает минимум и максимум
func (c *TechnicalCalculator) CalculateMinMax(prices []types.PriceData) (float64, float64) {
	if len(prices) == 0 {
		return 0, 0
	}

	min := prices[0].Price
	max := prices[0].Price

	for _, point := range prices {
		if point.Price < min {
			min = point.Price
		}
		if point.Price > max {
			max = point.Price
		}
	}

	return min, max
}

// GetRSIStatus возвращает статус RSI
func (c *TechnicalCalculator) GetRSIStatus(rsi float64) string {
	switch {
	case rsi >= 70:
		return "перекупленность"
	case rsi >= 62:
		return "близко к перекупленности"
	case rsi >= 55:
		return "бычий настрой"
	case rsi >= 45:
		return "нейтральный"
	case rsi >= 38:
		return "медвежий настрой"
	default:
		return "перепроданность"
	}
}

// GetMACDStatus возвращает статус MACD
func (c *TechnicalCalculator) GetMACDStatus(macd float64) string {
	switch {
	case macd > 0.1:
		return "сильный бычий"
	case macd > 0.01:
		return "бычий"
	case macd > -0.01:
		return "нейтральный"
	case macd > -0.1:
		return "медвежий"
	default:
		return "сильный медвежий"
	}
}
