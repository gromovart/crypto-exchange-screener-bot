// internal/core/domain/signals/detectors/continuous_analyzer/utils.go
package continuous_analyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"math"
	"strings"
	"time"
)

// CalculateSequenceMetrics вычисляет метрики последовательности
func CalculateSequenceMetrics(data []types.PriceData, startIdx, endIdx int) *SequenceMetrics {
	if len(data) == 0 || startIdx < 0 || endIdx >= len(data) || startIdx > endIdx {
		return nil
	}

	sequenceLength := endIdx - startIdx + 1
	if sequenceLength < 2 {
		return nil
	}

	metrics := &SequenceMetrics{
		Symbol:         data[0].Symbol,
		Timestamp:      time.Now().Unix(),
		SequenceLength: sequenceLength,
		StartIdx:       startIdx,
		EndIdx:         endIdx,
		StartPrice:     data[startIdx].Price,
		EndPrice:       data[endIdx].Price,
	}

	// Вычисляем общее изменение
	if metrics.StartPrice > 0 {
		metrics.TotalChange = ((metrics.EndPrice - metrics.StartPrice) / metrics.StartPrice) * 100
		metrics.AverageChange = metrics.TotalChange / float64(sequenceLength-1)
	}

	// Вычисляем средний gap
	var totalGap float64
	validGaps := 0
	directionChanges := 0
	prevDirection := ""

	for i := startIdx; i < endIdx; i++ {
		if i+1 < len(data) && data[i].Price > 0 {
			gap := math.Abs((data[i+1].Price - data[i].Price) / data[i].Price)
			totalGap += gap
			validGaps++

			// Определяем направление
			currentDirection := "neutral"
			if data[i+1].Price > data[i].Price {
				currentDirection = "up"
			} else if data[i+1].Price < data[i].Price {
				currentDirection = "down"
			}

			if i > startIdx && prevDirection != "" && currentDirection != prevDirection {
				directionChanges++
			}
			prevDirection = currentDirection
		}
	}

	if validGaps > 0 {
		metrics.AverageGap = totalGap / float64(validGaps)
	}

	// Определяем основное направление
	if metrics.TotalChange > 0.1 {
		metrics.Direction = "up"
	} else if metrics.TotalChange < -0.1 {
		metrics.Direction = "down"
	} else {
		metrics.Direction = "neutral"
	}

	// Качество последовательности
	metrics.IsContinuous = directionChanges == 0
	metrics.QualityScore = calculateQualityScore(metrics, directionChanges)

	return metrics
}

// calculateQualityScore вычисляет оценку качества последовательности
func calculateQualityScore(metrics *SequenceMetrics, directionChanges int) float64 {
	if metrics.SequenceLength < 2 {
		return 0
	}

	const (
		lengthWeight      = 0.3
		consistencyWeight = 0.4
		changeWeight      = 0.2
		gapWeight         = 0.1
	)

	// Оценка длины (нормализованная, максимум 10 точек = 1.0)
	lengthScore := math.Min(float64(metrics.SequenceLength)/10.0, 1.0)

	// Оценка консистентности
	consistencyScore := 1.0 - float64(directionChanges)/float64(metrics.SequenceLength-1)

	// Оценка изменения
	changeScore := math.Min(math.Abs(metrics.TotalChange)/10.0, 1.0)

	// Оценка gap (меньше gap = лучше)
	gapScore := 1.0 - math.Min(metrics.AverageGap/0.5, 1.0)

	// Итоговая оценка
	totalScore := lengthScore*lengthWeight +
		consistencyScore*consistencyWeight +
		changeScore*changeWeight +
		gapScore*gapWeight

	return totalScore * 100
}

// CreateContinuousSignal создает сигнал непрерывности
func CreateContinuousSignal(
	symbol string,
	direction string,
	changePercent float64,
	confidence float64,
	data []types.PriceData,
	startIdx, endIdx int,
	metadata map[string]float64,
) *analysis.Signal {
	if len(data) == 0 || startIdx < 0 || endIdx >= len(data) {
		return nil
	}

	// Формируем теги
	tags := []string{"continuous"}
	if direction != "neutral" {
		tags = append(tags, direction)
	}

	// Создаем сигнал
	return &analysis.Signal{
		Symbol:        symbol,
		Type:          "continuous_" + direction,
		Direction:     direction,
		ChangePercent: changePercent,
		Confidence:    math.Max(0, math.Min(100, confidence)),
		DataPoints:    endIdx - startIdx + 1,
		StartPrice:    data[startIdx].Price,
		EndPrice:      data[endIdx].Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy:       "continuous_analysis",
			Tags:           tags,
			IsContinuous:   true,
			ContinuousFrom: startIdx,
			ContinuousTo:   endIdx,
			Indicators:     metadata,
		},
	}
}

// ValidateContinuousData проверяет валидность данных для анализа непрерывности
func ValidateContinuousData(data []types.PriceData, minPoints int) error {
	if len(data) < minPoints {
		return fmt.Errorf("insufficient data points: got %d, need %d", len(data), minPoints)
	}

	// Проверяем целостность символов
	symbol := data[0].Symbol
	for i, point := range data {
		if point.Symbol != symbol {
			return fmt.Errorf("inconsistent symbols: %s != %s at index %d", point.Symbol, symbol, i)
		}
	}

	return nil
}

// FindLongestSequence находит самую длинную непрерывную последовательность
func FindLongestSequence(data []types.PriceData, maxGapRatio float64) (startIdx, endIdx int) {
	if len(data) < 2 {
		return 0, 0
	}

	longestStart := 0
	longestEnd := 0
	currentStart := 0

	for i := 1; i < len(data); i++ {
		prevPrice := data[i-1].Price
		currPrice := data[i].Price

		if prevPrice == 0 {
			continue
		}

		gap := math.Abs((currPrice - prevPrice) / prevPrice)

		// Если gap слишком большой, завершаем текущую последовательность
		if gap > maxGapRatio {
			// Проверяем, является ли текущая последовательность самой длинной
			currentLength := i - 1 - currentStart
			longestLength := longestEnd - longestStart

			if currentLength > longestLength {
				longestStart = currentStart
				longestEnd = i - 1
			}

			// Начинаем новую последовательность
			currentStart = i
		}
	}

	// Проверяем последнюю последовательность
	currentLength := len(data) - 1 - currentStart
	longestLength := longestEnd - longestStart

	if currentLength > longestLength {
		longestStart = currentStart
		longestEnd = len(data) - 1
	}

	return longestStart, longestEnd
}

// GenerateContinuousReport генерирует отчет по анализу непрерывности
func GenerateContinuousReport(metrics *SequenceMetrics, config common.AnalyzerConfig) string {
	if metrics == nil {
		return "No continuous metrics available"
	}

	var report strings.Builder

	report.WriteString(fmt.Sprintf("Continuous Analysis Report for %s\n", metrics.Symbol))
	report.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Unix(metrics.Timestamp, 0).Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("Sequence Length: %d points\n", metrics.SequenceLength))
	report.WriteString(fmt.Sprintf("Indices: %d - %d\n", metrics.StartIdx, metrics.EndIdx))
	report.WriteString("\n")

	report.WriteString("Price Metrics:\n")
	report.WriteString(fmt.Sprintf("  Start Price: $%.4f\n", metrics.StartPrice))
	report.WriteString(fmt.Sprintf("  End Price: $%.4f\n", metrics.EndPrice))
	report.WriteString(fmt.Sprintf("  Total Change: %.2f%%\n", metrics.TotalChange))
	report.WriteString(fmt.Sprintf("  Average Change: %.2f%% per point\n", metrics.AverageChange))
	report.WriteString("\n")

	report.WriteString("Sequence Quality:\n")
	report.WriteString(fmt.Sprintf("  Direction: %s\n", metrics.Direction))
	report.WriteString(fmt.Sprintf("  Average Gap: %.4f\n", metrics.AverageGap))
	report.WriteString(fmt.Sprintf("  Is Continuous: %v\n", metrics.IsContinuous))
	report.WriteString(fmt.Sprintf("  Quality Score: %.1f/100\n", metrics.QualityScore))
	report.WriteString("\n")

	report.WriteString("Configuration:\n")
	report.WriteString(fmt.Sprintf("  Min Points: %d\n", config.MinDataPoints))
	report.WriteString(fmt.Sprintf("  Min Confidence: %.1f%%\n", config.MinConfidence))

	return report.String()
}

// MergeContinuousConfigs объединяет конфигурации непрерывности
func MergeContinuousConfigs(base, override common.AnalyzerConfig) common.AnalyzerConfig {
	result := base

	// Обновляем основные поля
	if override.Enabled {
		result.Enabled = override.Enabled
	}
	if override.Weight > 0 {
		result.Weight = override.Weight
	}
	if override.MinConfidence > 0 {
		result.MinConfidence = override.MinConfidence
	}
	if override.MinDataPoints > 0 {
		result.MinDataPoints = override.MinDataPoints
	}

	// Обновляем кастомные настройки
	if result.CustomSettings == nil {
		result.CustomSettings = make(map[string]interface{})
	}

	for key, value := range override.CustomSettings {
		result.CustomSettings[key] = value
	}

	return result
}
