// internal/delivery/telegram/formatters/recommendation/analyzer.go
package recommendation

import (
	"fmt"
	"math"
)

// Analyzer анализирует данные для рекомендаций
type Analyzer struct{}

// NewAnalyzer создает новый анализатор
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// AnalysisResult результат анализа
type AnalysisResult struct {
	Recommendations []string
	Strength        string
}

// AnalyzeData анализирует все данные
func (a *Analyzer) AnalyzeData(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta float64,
	volumeDeltaPercent float64,
	longLiqVolume float64,
	shortLiqVolume float64,
) AnalysisResult {
	var recommendations []string

	// Анализ RSI
	recommendations = a.analyzeRSI(rsi, recommendations)

	// Анализ MACD
	recommendations = a.analyzeMACD(macdSignal, recommendations)

	// Анализ дельты объемов
	recommendations = a.analyzeVolumeDelta(direction, volumeDelta, volumeDeltaPercent, recommendations)

	// Анализ ликвидаций
	recommendations = a.analyzeLiquidations(longLiqVolume, shortLiqVolume, recommendations)

	// Определяем силу движения
	strength := a.determineStrength(volumeDelta)

	return AnalysisResult{
		Recommendations: recommendations,
		Strength:        strength,
	}
}

// analyzeRSI анализирует RSI
func (a *Analyzer) analyzeRSI(rsi float64, recs []string) []string {
	if rsi >= 70 {
		recs = append(recs, fmt.Sprintf("RSI в зоне перекупленности (%.1f) - осторожность с LONG", rsi))
	} else if rsi >= 62 {
		recs = append(recs, fmt.Sprintf("RSI показывает перекупленность (%.1f)", rsi))
	} else if rsi <= 30 {
		recs = append(recs, fmt.Sprintf("RSI в зоне перепроданности (%.1f) - осторожность с SHORT", rsi))
	} else if rsi <= 38 {
		recs = append(recs, fmt.Sprintf("RSI показывает перепроданность (%.1f)", rsi))
	} else if rsi >= 55 {
		recs = append(recs, fmt.Sprintf("RSI бычий настрой (%.1f)", rsi))
	} else if rsi < 45 {
		recs = append(recs, fmt.Sprintf("RSI медвежий настрой (%.1f)", rsi))
	}
	return recs
}

// analyzeMACD анализирует MACD
func (a *Analyzer) analyzeMACD(macdSignal float64, recs []string) []string {
	if macdSignal > 0.1 {
		recs = append(recs, "MACD: сильный бычий тренд")
	} else if macdSignal > 0.05 {
		recs = append(recs, "MACD: бычий тренд")
	} else if macdSignal > 0.01 {
		recs = append(recs, "MACD: слабый бычий сигнал")
	} else if macdSignal < -0.1 {
		recs = append(recs, "MACD: сильный медвежий тренд")
	} else if macdSignal < -0.05 {
		recs = append(recs, "MACD: медвежий тренд")
	} else if macdSignal < -0.01 {
		recs = append(recs, "MACD: слабый медвежий сигнал")
	} else {
		recs = append(recs, "MACD: нейтральный")
	}
	return recs
}

// analyzeVolumeDelta анализирует дельту объемов
func (a *Analyzer) analyzeVolumeDelta(direction string, delta, deltaPercent float64, recs []string) []string {
	if math.Abs(delta) > 0 {
		strengthLevel, deltaDescription := a.getDeltaStrength(delta)

		if delta > 0 {
			if direction == "growth" {
				recs = append(recs,
					fmt.Sprintf("%s дельта покупок ($%.0f) - %s покупателей",
						strengthLevel, delta, deltaDescription))
			} else {
				recs = append(recs,
					fmt.Sprintf("⚠️ %s дельта покупок при падении ($%.0f) - возможен разворот",
						strengthLevel, delta))
			}
		} else {
			if direction == "fall" {
				recs = append(recs,
					fmt.Sprintf("%s дельта продаж ($%.0f) - %s продавцов",
						strengthLevel, math.Abs(delta), deltaDescription))
			} else {
				recs = append(recs,
					fmt.Sprintf("⚠️ %s дельта продаж при росте ($%.0f) - возможна коррекция",
						strengthLevel, math.Abs(delta)))
			}
		}

		// Анализ согласованности
		if deltaPercent != 0 {
			if (delta > 0 && deltaPercent > 0) || (delta < 0 && deltaPercent < 0) {
				consistencyStrength := math.Min(math.Abs(deltaPercent)/10, 1.0)
				if consistencyStrength > 0.5 {
					recs = append(recs, "✅ Объемы подтверждают ценовое движение")
				} else {
					recs = append(recs, "🟡 Объемы слабо подтверждают движение")
				}
			} else {
				contradictionStrength := math.Min(math.Abs(deltaPercent)/10, 1.0)
				if contradictionStrength > 0.5 {
					recs = append(recs, "🔄 Сильное противоречие объемов - возможен разворот")
				} else {
					recs = append(recs, "⚠️ Объемы противоречат ценовому движению")
				}
			}
		}
	}
	return recs
}

// analyzeLiquidations анализирует ликвидации
func (a *Analyzer) analyzeLiquidations(longLiq, shortLiq float64, recs []string) []string {
	liquidationRatio := 0.0
	if shortLiq > 0 {
		liquidationRatio = longLiq / shortLiq
	}

	totalLiq := longLiq + shortLiq
	if totalLiq > 0 {
		liqDescription := a.getLiquidationDescription(totalLiq)

		if liquidationRatio > 2.0 {
			recs = append(recs,
				fmt.Sprintf("💥 %s LONG ликвидации ($%.0f) - возможен отскок вверх",
					liqDescription, longLiq))
		} else if liquidationRatio < 0.5 {
			recs = append(recs,
				fmt.Sprintf("💥 %s SHORT ликвидации ($%.0f) - возможен отскок вниз",
					liqDescription, shortLiq))
		} else if totalLiq > 50000 {
			recs = append(recs,
				fmt.Sprintf("💥 %s ликвидации ($%.0f) - повышенная волатильность",
					liqDescription, totalLiq))
		}
	}
	return recs
}

// getDeltaStrength определяет силу дельты
func (a *Analyzer) getDeltaStrength(delta float64) (string, string) {
	deltaAbs := math.Abs(delta)

	switch {
	case deltaAbs > 100000:
		return "сильная", "значительное давление"
	case deltaAbs > 10000:
		return "умеренная", "заметное давление"
	case deltaAbs > 1000:
		return "слабая", "небольшое давление"
	default:
		return "незначительная", "минимальное давление"
	}
}

// getLiquidationDescription определяет описание ликвидаций
func (a *Analyzer) getLiquidationDescription(totalLiq float64) string {
	if totalLiq > 100000 {
		return "значительные"
	} else if totalLiq > 10000 {
		return "заметные"
	}
	return "небольшие"
}

// determineStrength определяет силу движения
func (a *Analyzer) determineStrength(volumeDelta float64) string {
	deltaAbs := math.Abs(volumeDelta)

	if deltaAbs > 50000 {
		return "сильное"
	} else if deltaAbs > 10000 {
		return "умеренное"
	}
	return "слабое"
}
