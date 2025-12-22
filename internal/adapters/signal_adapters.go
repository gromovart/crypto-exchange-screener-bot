// internal/adapters/signal_adapters.go
package adapters

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/common"
)

// AnalysisSignalToTrendSignal конвертирует analysis.Signal в types.TrendSignal
func AnalysisSignalToTrendSignal(signal analysis.Signal) analysis.TrendSignal {
	direction := "growth"
	if signal.Direction == analysis.TrendBearish || string(signal.Direction) == "down" {
		direction = "fall"
	}

	return analysis.TrendSignal{
		Symbol:        signal.Symbol, // Уже common.Symbol
		Direction:     direction,
		ChangePercent: signal.ChangePercent,
		PeriodMinutes: signal.Period,
		Confidence:    signal.Confidence,
		Timestamp:     signal.Timestamp,
		DataPoints:    signal.DataPoints,
	}
}

// AnalysisSignalToGrowthSignal конвертирует analysis.Signal в types.GrowthSignal
func AnalysisSignalToGrowthSignal(signal analysis.Signal) analysis.GrowthSignal {
	growthPercent := 0.0
	fallPercent := 0.0
	direction := "growth"

	// Сравниваем с правильными типами
	if signal.Direction == analysis.TrendBullish || string(signal.Direction) == "up" {
		growthPercent = signal.ChangePercent
		direction = "growth"
	} else {
		fallPercent = -signal.ChangePercent
		direction = "fall"
	}

	return analysis.GrowthSignal{
		Symbol:        signal.Symbol,
		PeriodMinutes: signal.Period,
		GrowthPercent: growthPercent,
		FallPercent:   fallPercent,
		IsContinuous:  signal.Metadata.IsContinuous,
		DataPoints:    signal.DataPoints,
		StartPrice:    signal.StartPrice,
		EndPrice:      signal.EndPrice,
		Direction:     direction,
		Confidence:    signal.Confidence,
		Timestamp:     signal.Timestamp,
		Type:          string(signal.Type), // Преобразуем SignalType в string
		Metadata:      &signal.Metadata,
	}
}

// TrendSignalToGrowthSignal конвертирует types.TrendSignal в types.GrowthSignal
func TrendSignalToGrowthSignal(signal analysis.TrendSignal) analysis.GrowthSignal {
	growthPercent := 0.0
	fallPercent := 0.0
	direction := "growth"

	if signal.Direction == "growth" {
		growthPercent = signal.ChangePercent
		direction = "growth"
	} else {
		fallPercent = signal.ChangePercent
		direction = "fall"
	}

	return analysis.GrowthSignal{
		Symbol:        signal.Symbol,
		PeriodMinutes: signal.PeriodMinutes,
		GrowthPercent: growthPercent,
		FallPercent:   fallPercent,
		IsContinuous:  false,
		DataPoints:    signal.DataPoints,
		StartPrice:    0,
		EndPrice:      0,
		Direction:     direction,
		Confidence:    signal.Confidence,
		Timestamp:     signal.Timestamp,
		Type:          "",  // Неизвестно из TrendSignal
		Metadata:      nil, // Нет метаданных в TrendSignal
	}
}

// PriceDataToPriceDataPoint конвертирует types.PriceData в types.PriceDataPoint
func PriceDataToPriceDataPoint(data common.PriceData) analysis.PriceDataPoint {
	return analysis.PriceDataPoint{
		Price:     data.Price,
		Timestamp: data.Timestamp,
		Volume:    data.Volume24h,
	}
}

// BatchAnalysisSignalToTrendSignal конвертирует пакет analysis.Signal
func BatchAnalysisSignalToTrendSignal(signals []analysis.Signal) []analysis.TrendSignal {
	result := make([]analysis.TrendSignal, len(signals))
	for i, signal := range signals {
		result[i] = AnalysisSignalToTrendSignal(signal)
	}
	return result
}

// BatchAnalysisSignalToGrowthSignal конвертирует пакет analysis.Signal
func BatchAnalysisSignalToGrowthSignal(signals []analysis.Signal) []analysis.GrowthSignal {
	result := make([]analysis.GrowthSignal, len(signals))
	for i, signal := range signals {
		result[i] = AnalysisSignalToGrowthSignal(signal)
	}
	return result
}
