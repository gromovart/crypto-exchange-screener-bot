// internal/delivery/telegram/services/counter/calculations.go
package counter

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"time"
)

// calculateNextAnalysis рассчитывает время следующего анализа (через 1 минуту)
func (s *serviceImpl) calculateNextAnalysis(timestamp time.Time, period string) time.Time {
	// Анализ всегда через 1 минуту
	next := timestamp.Add(1 * time.Minute)

	// Округляем до следующей целой минуты
	next = next.Truncate(time.Minute)
	if next.Before(timestamp) || next.Equal(timestamp) {
		next = next.Add(1 * time.Minute)
	}

	return next
}

// calculateNextSignal рассчитывает время следующего сигнала
func (s *serviceImpl) calculateNextSignal(timestamp time.Time, period string, confirmations, requiredConfirmations int) time.Time {
	if requiredConfirmations == 0 {
		requiredConfirmations = GetRequiredConfirmations(period)
	}

	if confirmations >= requiredConfirmations {
		// Если уже есть все подтверждения, следующий сигнал = начало следующего периода
		return s.calculateNextPeriodStart(timestamp, period)
	}

	// Если не все подтверждения, следующий сигнал = когда будет следующее подтверждение
	remainingConfirmations := requiredConfirmations - confirmations
	next := timestamp.Add(time.Duration(remainingConfirmations) * time.Minute)

	// Округляем до целой минуты
	next = next.Truncate(time.Minute)
	if next.Before(timestamp) || next.Equal(timestamp) {
		next = next.Add(1 * time.Minute)
	}

	return next
}

// calculateNextPeriodStart рассчитывает начало следующего периода
func (s *serviceImpl) calculateNextPeriodStart(timestamp time.Time, period string) time.Time {
	periodMinutes := s.periodToMinutes(period)
	currentMinute := timestamp.Minute()

	// Находим следующий период
	periodsPassed := currentMinute / periodMinutes
	nextPeriodStartMinute := (periodsPassed + 1) * periodMinutes

	// Если следующее начало периода в этом часу
	if nextPeriodStartMinute < 60 {
		next := time.Date(
			timestamp.Year(), timestamp.Month(), timestamp.Day(),
			timestamp.Hour(), nextPeriodStartMinute, 0, 0,
			timestamp.Location(),
		)

		// Если следующее начало уже прошло, берем следующее
		if !next.After(timestamp) {
			next = next.Add(time.Duration(periodMinutes) * time.Minute)
		}

		return next
	}

	// Иначе в следующем часу
	next := time.Date(
		timestamp.Year(), timestamp.Month(), timestamp.Day(),
		timestamp.Hour()+1, 0, 0, 0,
		timestamp.Location(),
	)
	return next
}

// convertToFormatterData конвертирует сырые данные в форматтер данные
func (s *serviceImpl) convertToFormatterData(rawData RawCounterData) formatters.CounterData {
	return formatters.CounterData{
		Symbol:             rawData.Symbol,
		Direction:          rawData.Direction,
		ChangePercent:      rawData.ChangePercent,
		SignalCount:        rawData.SignalCount,
		MaxSignals:         rawData.MaxSignals,
		Period:             rawData.Period,
		CurrentPrice:       rawData.CurrentPrice,
		Volume24h:          rawData.Volume24h,
		OpenInterest:       rawData.OpenInterest,
		OIChange24h:        rawData.OIChange24h,
		FundingRate:        rawData.FundingRate,
		NextFundingTime:    rawData.NextFundingTime,
		LiquidationVolume:  rawData.LiquidationVolume,
		LongLiqVolume:      rawData.LongLiqVolume,
		ShortLiqVolume:     rawData.ShortLiqVolume,
		VolumeDelta:        rawData.VolumeDelta,
		VolumeDeltaPercent: rawData.VolumeDeltaPercent,
		RSI:                rawData.RSI,
		MACDSignal:         rawData.MACDSignal,
		DeltaSource:        rawData.DeltaSource,
		Confidence:         rawData.Confidence,
		Timestamp:          rawData.Timestamp,

		// Используем переданные данные прогресса, НЕ пересчитываем!
		Confirmations:         rawData.Confirmations,
		RequiredConfirmations: rawData.RequiredConfirmations,
		TotalSlots:            rawData.TotalSlots,         // Используем переданное
		FilledSlots:           rawData.FilledSlots,        // Используем переданное
		ProgressPercentage:    rawData.ProgressPercentage, // Используем переданное
		NextAnalysis:          rawData.NextAnalysis,
		NextSignal:            rawData.NextSignal,

		// Зоны S/R
		SRSupport:    buildSRZoneData(rawData.SRSupportPrice, rawData.SRSupportStrength, rawData.SRSupportDistPct, rawData.SRSupportHasWall, rawData.SRSupportWallUSD),
		SRResistance: buildSRZoneData(rawData.SRResistancePrice, rawData.SRResistanceStrength, rawData.SRResistanceDistPct, rawData.SRResistanceHasWall, rawData.SRResistanceWallUSD),
	}
}

// buildSRZoneData строит SRZoneData если цена > 0, иначе nil.
func buildSRZoneData(price, strength, distPct float64, hasWall bool, wallUSD float64) *formatters.SRZoneData {
	if price <= 0 {
		return nil
	}
	return &formatters.SRZoneData{
		Price:       price,
		Strength:    strength,
		DistPct:     distPct,
		HasWall:     hasWall,
		WallSizeUSD: wallUSD,
	}
}

// getTotalGroupsForPeriod возвращает количество групп для периода
func (s *serviceImpl) getTotalGroupsForPeriod(period string) int {
	switch period {
	case "5m", "15m":
		return 5
	case "30m", "1h":
		return 6
	case "4h":
		return 8
	case "1d":
		return 12
	default:
		return 5
	}
}
