// internal/delivery/telegram/app/bot/formatters/recommendation/scorer.go
package recommendation

import (
	"math"
	"strings"
)

// Scorer подсчитывает баллы для рекомендаций
type Scorer struct{}

// NewScorer создает новый счетчик
func NewScorer() *Scorer {
	return &Scorer{}
}

// SignalScores баллы сигналов
type SignalScores struct {
	BullishScore int
	BearishScore int
	NeutralScore int
}

// CalculateSignalScores подсчитывает баллы для сигналов
func (s *Scorer) CalculateSignalScores(recommendations []string) SignalScores {
	scores := SignalScores{}

	for _, rec := range recommendations {
		lowerRec := strings.ToLower(rec)

		if strings.Contains(lowerRec, "long") || strings.Contains(lowerRec, "рост") ||
			strings.Contains(lowerRec, "бычий") || strings.Contains(lowerRec, "покуп") ||
			strings.Contains(lowerRec, "дельта покупок") ||
			strings.Contains(lowerRec, "сильный бычий") {

			if strings.Contains(lowerRec, "сильный") || strings.Contains(lowerRec, "значительное") {
				scores.BullishScore += 3
			} else if strings.Contains(lowerRec, "умерен") || strings.Contains(lowerRec, "заметное") {
				scores.BullishScore += 2
			} else {
				scores.BullishScore += 1
			}

		} else if strings.Contains(lowerRec, "short") || strings.Contains(lowerRec, "падение") ||
			strings.Contains(lowerRec, "медвежий") || strings.Contains(lowerRec, "продаж") ||
			strings.Contains(lowerRec, "дельта продаж") ||
			strings.Contains(lowerRec, "сильный медвежий") {

			if strings.Contains(lowerRec, "сильный") || strings.Contains(lowerRec, "значительное") {
				scores.BearishScore += 3
			} else if strings.Contains(lowerRec, "умерен") || strings.Contains(lowerRec, "заметное") {
				scores.BearishScore += 2
			} else {
				scores.BearishScore += 1
			}

		} else if strings.Contains(lowerRec, "нейтраль") || strings.Contains(lowerRec, "слабый") ||
			strings.Contains(lowerRec, "незначитель") {
			scores.NeutralScore += 1
		} else if strings.Contains(lowerRec, "⚠️") || strings.Contains(lowerRec, "🔄") {
			scores.BullishScore -= 1
			scores.BearishScore -= 1
			scores.NeutralScore += 2
		}
	}

	return scores
}

// DeterminePrimarySignal определяет основной сигнал
func (s *Scorer) DeterminePrimarySignal(
	scores SignalScores,
	recommendations []string,
) string {
	totalWeightedScore := scores.BullishScore + scores.BearishScore + scores.NeutralScore

	if totalWeightedScore == 0 {
		return ""
	}

	bullishRatio := float64(scores.BullishScore) / float64(totalWeightedScore)
	bearishRatio := float64(scores.BearishScore) / float64(totalWeightedScore)

	switch {
	case bullishRatio > 0.7:
		if scores.BullishScore >= 6 {
			return "🟢🔼 СИЛЬНЫЕ БЫЧЬИ СИГНАЛЫ"
		} else if scores.BullishScore >= 3 {
			return "🟢 Бычьи сигналы"
		} else {
			return "🟡 Слабые бычьи сигналы"
		}

	case bearishRatio > 0.7:
		if scores.BearishScore >= 6 {
			return "🔴🔽 СИЛЬНЫЕ МЕДВЕЖЬИ СИГНАЛЫ"
		} else if scores.BearishScore >= 3 {
			return "🔴 Медвежьи сигналы"
		} else {
			return "🟠 Слабые медвежьи сигналы"
		}

	case bullishRatio > bearishRatio && bullishRatio > 0.4:
		if scores.BullishScore-scores.BearishScore >= 3 {
			return "🟢 Преобладают бычьи сигналы"
		} else {
			return "🟡 Слабый бычий перевес"
		}

	case bearishRatio > bullishRatio && bearishRatio > 0.4:
		if scores.BearishScore-scores.BullishScore >= 3 {
			return "🔴 Преобладают медвежьи сигналы"
		} else {
			return "🟠 Слабый медвежий перевес"
		}

	default:
		if scores.NeutralScore > 0 {
			scoreDiff := math.Abs(float64(scores.BullishScore - scores.BearishScore))
			if scoreDiff <= 1 {
				return "⚪ СБАЛАНСИРОВАННЫЕ СИГНАЛЫ"
			} else {
				return "⚪ Смешанные сигналы"
			}
		} else {
			return "🟡 ПРОТИВОРЕЧИВЫЕ СИГНАЛЫ"
		}
	}
}
