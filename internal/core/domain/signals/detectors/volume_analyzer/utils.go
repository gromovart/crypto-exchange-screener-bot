// internal/core/domain/signals/detectors/volume_analyzer/utils.go
package volume_analyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"math"
	"strings"
	"time"
)

// CalculateVolumeMetrics вычисляет метрики объема для данных
func CalculateVolumeMetrics(data []types.PriceData) *VolumeMetrics {
	if len(data) == 0 {
		return nil
	}

	lastPoint := data[len(data)-1]
	metrics := &VolumeMetrics{
		Symbol:        lastPoint.Symbol,
		Timestamp:     time.Now().Unix(),
		CurrentVolume: lastPoint.Volume24h,
		CurrentPrice:  lastPoint.Price,
		DataPoints:    len(data),
	}

	// Вычисляем средний объем
	var totalVolume float64
	validPoints := 0
	for _, point := range data {
		if point.Volume24h > 0 {
			totalVolume += point.Volume24h
			validPoints++
		}
	}

	if validPoints > 0 {
		metrics.AverageVolume = totalVolume / float64(validPoints)
		if metrics.AverageVolume > 0 {
			metrics.VolumeRatio = metrics.CurrentVolume / metrics.AverageVolume
		}
	}

	// Вычисляем изменение объема
	if len(data) > 1 && data[0].Volume24h > 0 {
		firstVolume := data[0].Volume24h
		metrics.VolumeChange = ((metrics.CurrentVolume - firstVolume) / firstVolume) * 100
	}

	// Вычисляем изменение цены
	if len(data) > 1 && data[0].Price > 0 {
		firstPrice := data[0].Price
		metrics.PriceChange = ((metrics.CurrentPrice - firstPrice) / firstPrice) * 100
	}

	// Вычисляем корреляцию
	metrics.PriceVolumeCorrelation = calculateCorrelation(data)

	return metrics
}

// calculateCorrelation вычисляет корреляцию между ценой и объемом
func calculateCorrelation(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	var priceChanges, volumeChanges []float64

	for i := 1; i < len(data); i++ {
		if data[i-1].Price > 0 && data[i-1].Volume24h > 0 {
			priceChange := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100
			volumeChange := ((data[i].Volume24h - data[i-1].Volume24h) / data[i-1].Volume24h) * 100

			priceChanges = append(priceChanges, priceChange)
			volumeChanges = append(volumeChanges, volumeChange)
		}
	}

	if len(priceChanges) == 0 || len(volumeChanges) == 0 {
		return 0
	}

	// Простая корреляция по направлению
	sameDirection := 0
	for i := 0; i < len(priceChanges); i++ {
		if (priceChanges[i] > 0 && volumeChanges[i] > 0) ||
			(priceChanges[i] < 0 && volumeChanges[i] < 0) {
			sameDirection++
		}
	}

	return float64(sameDirection) / float64(len(priceChanges)) * 100
}

// CreateVolumeSignal создает сигнал объема
func CreateVolumeSignal(
	symbol string,
	signalType string,
	direction string,
	changePercent float64,
	confidence float64,
	data []types.PriceData,
	metadata map[string]float64,
	tags []string,
) *analysis.Signal {
	if len(data) == 0 {
		return nil
	}

	// Формируем теги
	allTags := []string{"volume"}
	if tags != nil {
		allTags = append(allTags, tags...)
	}

	if direction != "neutral" {
		allTags = append(allTags, direction)
	}

	// Создаем сигнал
	return &analysis.Signal{
		Symbol:        symbol,
		Type:          signalType,
		Direction:     direction,
		ChangePercent: changePercent,
		Confidence:    math.Max(0, math.Min(100, confidence)), // Ограничиваем 0-100
		DataPoints:    len(data),
		StartPrice:    data[0].Price,
		EndPrice:      data[len(data)-1].Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy:   "volume_analysis",
			Tags:       allTags,
			Indicators: metadata,
		},
	}
}

// ValidateVolumeData проверяет валидность данных для анализа объема
func ValidateVolumeData(data []types.PriceData, minDataPoints int) error {
	if len(data) < minDataPoints {
		return fmt.Errorf("insufficient data points: got %d, need %d", len(data), minDataPoints)
	}

	// Проверяем, что есть данные об объеме
	hasVolume := false
	for _, point := range data {
		if point.Volume24h > 0 {
			hasVolume = true
			break
		}
	}

	if !hasVolume {
		return fmt.Errorf("no volume data available")
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

// FormatVolume форматирует объем для отображения
func FormatVolume(volume float64) string {
	if volume >= 1_000_000_000 {
		return fmt.Sprintf("%.2fB", volume/1_000_000_000)
	} else if volume >= 1_000_000 {
		return fmt.Sprintf("%.2fM", volume/1_000_000)
	} else if volume >= 1_000 {
		return fmt.Sprintf("%.2fK", volume/1_000)
	}
	return fmt.Sprintf("%.2f", volume)
}

// GetVolumeAlgorithmName возвращает читаемое имя алгоритма
func GetVolumeAlgorithmName(algorithmType VolumeAlgorithmType) string {
	switch algorithmType {
	case AlgorithmAverageVolume:
		return "Average Volume"
	case AlgorithmVolumeSpike:
		return "Volume Spike"
	case AlgorithmVolumeConfirmation:
		return "Volume-Price Confirmation"
	case AlgorithmVolumeDivergence:
		return "Volume-Price Divergence"
	default:
		return string(algorithmType)
	}
}

// MergeVolumeConfigs объединяет конфигурации объема
func MergeVolumeConfigs(base, override common.AnalyzerConfig) common.AnalyzerConfig {
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

// IsHighVolume проверяет, является ли объем высоким
func IsHighVolume(currentVolume, averageVolume, multiplier float64) bool {
	if averageVolume <= 0 {
		return false
	}
	return currentVolume > averageVolume*multiplier
}

// GetVolumeTrend определяет тренд объема
func GetVolumeTrend(data []types.PriceData) string {
	if len(data) < 2 {
		return "stable"
	}

	firstVolume := data[0].Volume24h
	lastVolume := data[len(data)-1].Volume24h

	if firstVolume <= 0 || lastVolume <= 0 {
		return "unknown"
	}

	change := ((lastVolume - firstVolume) / firstVolume) * 100

	if change > 20 {
		return "increasing"
	} else if change < -20 {
		return "decreasing"
	}
	return "stable"
}

// GenerateVolumeReport генерирует отчет по анализу объема
func GenerateVolumeReport(metrics *VolumeMetrics, config common.AnalyzerConfig) string {
	if metrics == nil {
		return "No volume metrics available"
	}

	var report strings.Builder

	report.WriteString(fmt.Sprintf("Volume Analysis Report for %s\n", metrics.Symbol))
	report.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Unix(metrics.Timestamp, 0).Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("Data Points: %d\n", metrics.DataPoints))
	report.WriteString("\n")

	report.WriteString("Volume Metrics:\n")
	report.WriteString(fmt.Sprintf("  Current Volume: %s\n", FormatVolume(metrics.CurrentVolume)))
	report.WriteString(fmt.Sprintf("  Average Volume: %s\n", FormatVolume(metrics.AverageVolume)))
	report.WriteString(fmt.Sprintf("  Volume Ratio: %.2f\n", metrics.VolumeRatio))
	report.WriteString(fmt.Sprintf("  Volume Change: %.2f%%\n", metrics.VolumeChange))
	report.WriteString("\n")

	report.WriteString("Price Metrics:\n")
	report.WriteString(fmt.Sprintf("  Current Price: $%.4f\n", metrics.CurrentPrice))
	report.WriteString(fmt.Sprintf("  Price Change: %.2f%%\n", metrics.PriceChange))
	report.WriteString("\n")

	report.WriteString("Correlation Analysis:\n")
	report.WriteString(fmt.Sprintf("  Price-Volume Correlation: %.2f%%\n", metrics.PriceVolumeCorrelation))

	// Флаги
	report.WriteString("\nFlags:\n")
	flags := []string{}
	if metrics.IsAboveMinVolume {
		flags = append(flags, "Above Min Volume")
	}
	if metrics.IsSpike {
		flags = append(flags, "Volume Spike")
	}
	if metrics.IsConfirmation {
		flags = append(flags, "Volume-Price Confirmation")
	}
	if metrics.IsDivergence {
		flags = append(flags, "Volume-Price Divergence")
	}

	if len(flags) > 0 {
		report.WriteString("  " + strings.Join(flags, ", ") + "\n")
	} else {
		report.WriteString("  No significant flags\n")
	}

	report.WriteString(fmt.Sprintf("\nConfidence: %.2f%%\n", metrics.Confidence))

	return report.String()
}
