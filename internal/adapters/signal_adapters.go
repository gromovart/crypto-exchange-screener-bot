// internal/adapters/signal_adapters.go
package adapters

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	redis_storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
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
	direction := signal.Direction

	if signal.Direction == "up" {
		growthPercent = signal.ChangePercent
		direction = "growth"
	} else {
		fallPercent = -signal.ChangePercent
		direction = "fall"
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
		Direction:     direction, // Исправлено: должно быть "growth" или "fall"
		Confidence:    signal.Confidence,
		Timestamp:     signal.Timestamp,
		Type:          signal.Type,      // Добавлено: передаем тип сигнала
		Metadata:      &signal.Metadata, // КРИТИЧЕСКОЕ ИЗМЕНЕНИЕ: передаем метаданные
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

// PriceDataToPriceDataPoint конвертирует redis_storage.PriceData в redis_storage.PriceDataPoint
func PriceDataToPriceDataPoint(data redis_storage.PriceData) redis_storage.PriceDataPoint {
	return redis_storage.PriceDataPoint{
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
