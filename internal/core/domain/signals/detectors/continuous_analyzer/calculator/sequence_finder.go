// internal/core/domain/signals/detectors/continuous_analyzer/calculator/sequence_finder.go
package calculator

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"math"
)

// SequenceInfo информация о последовательности
type SequenceInfo struct {
	StartIdx   int     `json:"start_idx"`
	Length     int     `json:"length"`
	Direction  string  `json:"direction"`
	AvgGap     float64 `json:"avg_gap"`
	AvgChange  float64 `json:"avg_change"`
	StartPrice float64 `json:"start_price"`
	EndPrice   float64 `json:"end_price"`
}

// SequenceFinder ищет лучшие последовательности в данных
type SequenceFinder struct {
	config common.AnalyzerConfig
}

// NewSequenceFinder создает новый поисковик последовательностей
func NewSequenceFinder(config common.AnalyzerConfig) *SequenceFinder {
	return &SequenceFinder{
		config: config,
	}
}

// FindBestSequence находит лучшую непрерывную последовательность
func (s *SequenceFinder) FindBestSequence(data []redis_storage.PriceData) SequenceInfo {
	if len(data) < 2 {
		return SequenceInfo{}
	}

	best := SequenceInfo{}
	current := SequenceInfo{
		StartIdx:   0,
		Length:     1,
		Direction:  "neutral",
		StartPrice: data[0].Price,
	}

	maxGapRatio := s.getMaxGapRatio()

	for i := 1; i < len(data); i++ {
		prevPrice := data[i-1].Price
		currPrice := data[i].Price

		// Проверяем gap
		gap := s.calculateGap(prevPrice, currPrice)
		if gap > maxGapRatio {
			// Сохраняем текущую последовательность если она лучшая
			if current.Length > best.Length {
				best = current
				best.EndPrice = prevPrice
			}
			// Начинаем новую последовательность
			current = SequenceInfo{
				StartIdx:   i,
				Length:     1,
				Direction:  "neutral",
				StartPrice: currPrice,
			}
			continue
		}

		// Определяем направление изменения
		direction := "neutral"
		if currPrice > prevPrice {
			direction = "up"
		} else if currPrice < prevPrice {
			direction = "down"
		}

		// Если направление совпадает или мы только начинаем
		if current.Length == 1 || current.Direction == direction || direction == "neutral" {
			if current.Direction == "neutral" && direction != "neutral" {
				current.Direction = direction
			}
			current.Length++

			// Обновляем средние значения
			if prevPrice != 0 {
				gap := s.calculateGap(prevPrice, currPrice)
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
			current.EndPrice = prevPrice
			if current.Length > best.Length {
				best = current
			}
			// Начинаем новую последовательность
			current = SequenceInfo{
				StartIdx:   i - 1,
				Length:     2,
				Direction:  direction,
				StartPrice: prevPrice,
				AvgGap:     gap,
				AvgChange:  ((currPrice - prevPrice) / prevPrice) * 100,
			}
		}
	}

	// Проверяем последнюю последовательность
	current.EndPrice = data[len(data)-1].Price
	if current.Length > best.Length {
		best = current
	}

	// Если нашли последовательность, добавляем конечную цену
	if best.Length > 0 && best.EndPrice == 0 {
		endIdx := best.StartIdx + best.Length - 1
		if endIdx < len(data) {
			best.EndPrice = data[endIdx].Price
		}
	}

	return best
}

// FindAllSequences находит все последовательности в данных
func (s *SequenceFinder) FindAllSequences(data []redis_storage.PriceData, minPoints int) []SequenceInfo {
	var sequences []SequenceInfo
	if len(data) < minPoints {
		return sequences
	}

	maxGapRatio := s.getMaxGapRatio()
	current := SequenceInfo{
		StartIdx:   0,
		Length:     1,
		Direction:  "neutral",
		StartPrice: data[0].Price,
	}

	for i := 1; i < len(data); i++ {
		prevPrice := data[i-1].Price
		currPrice := data[i].Price

		// Проверяем gap
		gap := s.calculateGap(prevPrice, currPrice)
		if gap > maxGapRatio {
			// Сохраняем текущую последовательность если она достаточно длинная
			if current.Length >= minPoints {
				current.EndPrice = prevPrice
				sequences = append(sequences, current)
			}
			// Начинаем новую последовательность
			current = SequenceInfo{
				StartIdx:   i,
				Length:     1,
				Direction:  "neutral",
				StartPrice: currPrice,
			}
			continue
		}

		// Определяем направление
		direction := "neutral"
		if currPrice > prevPrice {
			direction = "up"
		} else if currPrice < prevPrice {
			direction = "down"
		}

		// Если направление совпадает или мы только начинаем
		if current.Length == 1 || current.Direction == direction || direction == "neutral" {
			if current.Direction == "neutral" && direction != "neutral" {
				current.Direction = direction
			}
			current.Length++

			// Обновляем средние значения
			if prevPrice != 0 {
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
			// Сохраняем текущую последовательность если она достаточно длинная
			if current.Length >= minPoints {
				current.EndPrice = prevPrice
				sequences = append(sequences, current)
			}
			// Начинаем новую последовательность
			current = SequenceInfo{
				StartIdx:   i - 1,
				Length:     2,
				Direction:  direction,
				StartPrice: prevPrice,
				AvgGap:     gap,
				AvgChange:  ((currPrice - prevPrice) / prevPrice) * 100,
			}
		}
	}

	// Сохраняем последнюю последовательность
	if current.Length >= minPoints {
		current.EndPrice = data[len(data)-1].Price
		sequences = append(sequences, current)
	}

	return sequences
}

// getMaxGapRatio возвращает максимальный допустимый gap
func (s *SequenceFinder) getMaxGapRatio() float64 {
	if s.config.CustomSettings == nil {
		return 0.3
	}

	val := s.config.CustomSettings["max_gap_ratio"]
	if val == nil {
		return 0.3
	}

	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case float32:
		return float64(v)
	default:
		return 0.3
	}
}

// calculateGap вычисляет относительный разрыв между ценами
func (s *SequenceFinder) calculateGap(prev, curr float64) float64 {
	if prev == 0 {
		return 0
	}
	diff := math.Abs(curr - prev)
	return diff / prev
}

// UpdateConfig обновляет конфигурацию
func (s *SequenceFinder) UpdateConfig(config common.AnalyzerConfig) {
	s.config = config
}

// GetName возвращает имя калькулятора
func (s *SequenceFinder) GetName() string {
	return "sequence_finder"
}
