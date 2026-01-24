// internal/delivery/telegram/services/counter/extractors.go
package counter

import (
	"crypto-exchange-screener-bot/pkg/logger"
	"time"
)

// extractRawDataFromParams извлекает сырые данные счетчика из CounterParams
func (s *serviceImpl) extractRawDataFromParams(params CounterParams) (RawCounterData, error) {
	data := RawCounterData{
		Symbol:                params.Symbol,
		Direction:             params.Direction,
		ChangePercent:         params.ChangePercent,
		Period:                params.Period,
		Timestamp:             params.Timestamp,
		Confirmations:         params.Confirmations,
		RequiredConfirmations: GetRequiredConfirmations(params.Period),
		CurrentPrice:          params.CurrentPrice,
		Volume24h:             params.Volume24h,
		OpenInterest:          params.OpenInterest,
		FundingRate:           params.FundingRate,
		RSI:                   params.RSI,
		MACDSignal:            params.MACDSignal,
		VolumeDelta:           params.VolumeDelta,
		VolumeDeltaPercent:    params.VolumeDeltaPercent,

		// НОВОЕ: Используем данные прогресса из параметров если есть
		FilledSlots:        params.ProgressFilledGroups,
		TotalSlots:         params.ProgressTotalGroups,
		ProgressPercentage: params.ProgressPercentage,

		// Инициализируем остальные поля значениями по умолчанию
		OIChange24h:       0.0,
		NextFundingTime:   time.Time{},
		LiquidationVolume: 0.0,
		LongLiqVolume:     0.0,
		ShortLiqVolume:    0.0,
		DeltaSource:       "",
		Confidence:        0.0,
		SignalCount:       params.Confirmations,                    // для обратной совместимости
		MaxSignals:        GetRequiredConfirmations(params.Period), // для обратной совместимости
	}

	// После создания данных добавить:
	if params.ProgressFilledGroups > 0 || params.ProgressTotalGroups > 0 {
		logger.Warn("📊 Service: Использованы данные прогресса из параметров: заполнено %d из %d (%.0f%%)",
			data.FilledSlots, data.TotalSlots, data.ProgressPercentage)
	} else {
		logger.Warn("📊 Service: Прогресс рассчитан автоматически: заполнено %d из %d (%.0f%%)",
			data.FilledSlots, data.TotalSlots, data.ProgressPercentage)
	}

	// Если данные прогресса не переданы, рассчитываем как раньше
	if data.TotalSlots == 0 {
		totalGroups, _ := s.getGroupedSlotsInfo(params.Period)
		data.TotalSlots = totalGroups
	}

	if data.FilledSlots == 0 && params.Confirmations > 0 {
		data.FilledSlots = s.calculateFilledGroups(params.Confirmations, data.TotalSlots)
	}

	if data.ProgressPercentage == 0 && data.RequiredConfirmations > 0 {
		data.ProgressPercentage = float64(data.Confirmations) / float64(data.RequiredConfirmations) * 100
	}

	// Рассчитываем дополнительные поля
	data.NextAnalysis = s.calculateNextAnalysis(data.Timestamp, data.Period)
	data.NextSignal = s.calculateNextSignal(data.Timestamp, data.Period, data.Confirmations, data.RequiredConfirmations)

	logger.Debug("🔍 extractRawDataFromParams: RSI=%.2f, MACD=%.2f, Прогресс: %d/%d (%.0f%%)",
		params.RSI, params.MACDSignal, data.FilledSlots, data.TotalSlots, data.ProgressPercentage)

	return data, nil
}
