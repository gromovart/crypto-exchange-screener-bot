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
func (s *serviceImpl) convertToFormatterData(raw RawCounterData) formatters.CounterData {
	// Рассчитываем процент прогресса
	progressPercentage := 0.0
	if raw.RequiredConfirmations > 0 {
		progressPercentage = float64(raw.Confirmations) / float64(raw.RequiredConfirmations) * 100
	} else if raw.MaxSignals > 0 {
		// Обратная совместимость
		progressPercentage = float64(raw.SignalCount) / float64(raw.MaxSignals) * 100
	}

	// Рассчитываем следующий анализ (всегда через 1 минуту)
	nextAnalysis := s.calculateNextAnalysis(raw.Timestamp, raw.Period)

	// Рассчитываем следующий сигнал
	nextSignal := s.calculateNextSignal(raw.Timestamp, raw.Period, raw.Confirmations, raw.RequiredConfirmations)

	// Рассчитываем группировку для прогресс-бара
	totalGroups, _ := s.getGroupedSlotsInfo(raw.Period)
	filledGroups := s.calculateFilledGroups(raw.Confirmations, raw.RequiredConfirmations, totalGroups)

	return formatters.CounterData{
		Symbol:                raw.Symbol,
		Direction:             raw.Direction,
		ChangePercent:         raw.ChangePercent,
		SignalCount:           raw.Confirmations,         // теперь это подтверждения
		MaxSignals:            raw.RequiredConfirmations, // теперь это требуемые подтверждения
		Period:                raw.Period,
		CurrentPrice:          raw.CurrentPrice,
		Volume24h:             raw.Volume24h,
		OpenInterest:          raw.OpenInterest,
		OIChange24h:           raw.OIChange24h,
		FundingRate:           raw.FundingRate,
		NextFundingTime:       raw.NextFundingTime,
		LiquidationVolume:     raw.LiquidationVolume,
		LongLiqVolume:         raw.LongLiqVolume,
		ShortLiqVolume:        raw.ShortLiqVolume,
		VolumeDelta:           raw.VolumeDelta,
		VolumeDeltaPercent:    raw.VolumeDeltaPercent,
		RSI:                   raw.RSI,
		MACDSignal:            raw.MACDSignal,
		DeltaSource:           raw.DeltaSource,
		Confidence:            raw.Confidence,
		Timestamp:             raw.Timestamp,
		Confirmations:         raw.Confirmations,
		RequiredConfirmations: raw.RequiredConfirmations,
		TotalSlots:            totalGroups,  // Теперь это группы (не отдельные минуты)
		FilledSlots:           filledGroups, // Заполненные группы
		ProgressPercentage:    progressPercentage,
		NextAnalysis:          nextAnalysis,
		NextSignal:            nextSignal,
	}
}

// calculateFilledGroups рассчитывает заполненные группы для прогресс-бара
func (s *serviceImpl) calculateFilledGroups(confirmations, requiredConfirmations, totalGroups int) int {
	if requiredConfirmations == 0 {
		return 0
	}

	// Каждая группа подтверждается если большинство минут в ней подтверждены
	filled := float64(confirmations) / float64(requiredConfirmations) * float64(totalGroups)

	// Округляем вверх, но не больше totalGroups
	filledInt := int(math.Ceil(filled))
	if filledInt > totalGroups {
		filledInt = totalGroups
	}

	return filledInt
}
