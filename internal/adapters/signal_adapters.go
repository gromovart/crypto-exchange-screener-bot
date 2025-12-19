// internal/adapters/signal_adapters.go
package adapters

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/types"
)

// AnalysisSignalToTrendSignal конвертирует analysis.Signal в types.TrendSignal
func AnalysisSignalToTrendSignal(signal analysis.Signal) types.TrendSignal {
	direction := "growth"
	if signal.Direction == "down" {
		direction = "fall"
	}

	return types.TrendSignal{
		Symbol:        signal.Symbol,
		Direction:     direction,
		ChangePercent: signal.ChangePercent,
		PeriodMinutes: signal.Period,
		Confidence:    signal.Confidence,
		Timestamp:     signal.Timestamp,
		DataPoints:    signal.DataPoints,
	}
}

// AnalysisSignalToGrowthSignal конвертирует analysis.Signal в types.GrowthSignal
func AnalysisSignalToGrowthSignal(signal analysis.Signal) types.GrowthSignal {
	growthPercent := 0.0
	fallPercent := 0.0

	if signal.Direction == "up" {
		growthPercent = signal.ChangePercent
	} else {
		fallPercent = -signal.ChangePercent
	}

	return types.GrowthSignal{
		Symbol:        signal.Symbol,
		PeriodMinutes: signal.Period,
		GrowthPercent: growthPercent,
		FallPercent:   fallPercent,
		IsContinuous:  signal.Metadata.IsContinuous,
		DataPoints:    signal.DataPoints,
		StartPrice:    signal.StartPrice,
		EndPrice:      signal.EndPrice,
		Direction:     signal.Direction,
		Confidence:    signal.Confidence,
		Timestamp:     signal.Timestamp,
	}
}

// TrendSignalToGrowthSignal конвертирует types.TrendSignal в types.GrowthSignal
func TrendSignalToGrowthSignal(signal types.TrendSignal) types.GrowthSignal {
	growthPercent := 0.0
	fallPercent := 0.0
	direction := "up"

	if signal.Direction == "growth" {
		growthPercent = signal.ChangePercent
		direction = "growth"
	} else {
		fallPercent = signal.ChangePercent
		direction = "fall"
	}

	return types.GrowthSignal{
		Symbol:        signal.Symbol,
		PeriodMinutes: signal.PeriodMinutes,
		GrowthPercent: growthPercent,
		FallPercent:   fallPercent,
		IsContinuous:  false, // неизвестно из TrendSignal
		DataPoints:    signal.DataPoints,
		StartPrice:    0, // неизвестно из TrendSignal
		EndPrice:      0, // неизвестно из TrendSignal
		Direction:     direction,
		Confidence:    signal.Confidence,
		Timestamp:     signal.Timestamp,
	}
}

// PriceDataToPriceDataPoint конвертирует types.PriceData в types.PriceDataPoint
func PriceDataToPriceDataPoint(data types.PriceData) types.PriceDataPoint {
	return types.PriceDataPoint{
		Price:     data.Price,
		Timestamp: data.Timestamp,
		Volume:    data.Volume24h,
	}
}

// BatchAnalysisSignalToTrendSignal конвертирует пакет analysis.Signal
func BatchAnalysisSignalToTrendSignal(signals []analysis.Signal) []types.TrendSignal {
	result := make([]types.TrendSignal, len(signals))
	for i, signal := range signals {
		result[i] = AnalysisSignalToTrendSignal(signal)
	}
	return result
}

// BatchAnalysisSignalToGrowthSignal конвертирует пакет analysis.Signal
func BatchAnalysisSignalToGrowthSignal(signals []analysis.Signal) []types.GrowthSignal {
	result := make([]types.GrowthSignal, len(signals))
	for i, signal := range signals {
		result[i] = AnalysisSignalToGrowthSignal(signal)
	}
	return result
}
