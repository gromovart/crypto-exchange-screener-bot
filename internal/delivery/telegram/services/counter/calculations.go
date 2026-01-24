// internal/delivery/telegram/services/counter/calculations.go
package counter

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters"
	"math"
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

// getGroupedSlotsInfo возвращает информацию о группировке слотов
func (s *serviceImpl) getGroupedSlotsInfo(period string) (totalGroups int, minutesPerGroup int) {
	switch period {
	case "5m":
		return 5, 1 // 5 групп по 1 минуте
	case "15m":
		return 5, 3 // 5 групп по 3 минуты
	case "30m":
		return 6, 5 // 6 групп по 5 минут
	case "1h":
		return 6, 10 // 6 групп по 10 минут
	case "4h":
		return 8, 30 // 8 групп по 30 минут
	case "1d":
		return 12, 120 // 12 групп по 2 часа
	default:
		return 5, 1
	}
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
	// ВИЗУАЛЬНАЯ ЦЕЛЬ ВСЕГДА = 6
	visualTarget := 6

	// ПЕРЕСЧИТЫВАЕМ группы на основе периода
	totalGroups := s.getTotalGroupsForPeriod(rawData.Period)
	filledGroups := s.calculateFilledGroups(rawData.Confirmations, totalGroups)

	// Рассчитываем процент прогресса
	progressPercentage := math.Min(float64(rawData.Confirmations)/float64(visualTarget), 1.0) * 100

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

		// ПРАВИЛЬНЫЕ данные прогресса
		Confirmations:         rawData.Confirmations,
		RequiredConfirmations: visualTarget, // ВСЕГДА 6
		TotalSlots:            totalGroups,  // ПЕРЕСЧИТАНО
		FilledSlots:           filledGroups, // ПЕРЕСЧИТАНО
		ProgressPercentage:    progressPercentage,
		NextAnalysis:          rawData.NextAnalysis,
		NextSignal:            rawData.NextSignal,
	}
}

// calculateFilledGroups рассчитывает заполненные группы для прогресс-бара
func (s *serviceImpl) calculateFilledGroups(confirmations, totalGroups int) int {
	// ВИЗУАЛЬНАЯ ЦЕЛЬ ВСЕГДА = 6
	visualTarget := 6

	if confirmations <= 0 {
		return 0
	}

	// Ограничиваем подтверждения визуальной целью
	normalizedConfirmations := math.Min(float64(confirmations), float64(visualTarget))

	// Рассчитываем прогресс: подтверждения / 6
	progressRatio := normalizedConfirmations / float64(visualTarget)

	// Математическое округление
	filledGroups := int(math.Round(progressRatio * float64(totalGroups)))

	// Корректировки
	if filledGroups == 0 && confirmations > 0 {
		filledGroups = 1
	}
	if filledGroups > totalGroups {
		filledGroups = totalGroups
	}
	if filledGroups < 0 {
		filledGroups = 0
	}

	return filledGroups
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
