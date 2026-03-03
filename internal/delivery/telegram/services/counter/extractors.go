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
		OIChange24h:           params.OIChange24h,
		FundingRate:           params.FundingRate,
		RSI:                   params.RSI,
		MACDSignal:            params.MACDSignal,
		VolumeDelta:           params.VolumeDelta,
		VolumeDeltaPercent:    params.VolumeDeltaPercent,

		// Используем данные прогресса из параметров
		FilledSlots:        params.ProgressFilledGroups,
		TotalSlots:         params.ProgressTotalGroups,
		ProgressPercentage: params.ProgressPercentage,

		// Инициализируем остальные поля значениями по умолчанию
		NextFundingTime:   time.Time{},
		LiquidationVolume: 0.0,
		LongLiqVolume:     0.0,
		ShortLiqVolume:    0.0,
		DeltaSource:       "",
		Confidence:        0.0,
		SignalCount:       params.Confirmations,
		MaxSignals:        GetRequiredConfirmations(params.Period),
	}

	// Зоны S/R
	data.SRSupportPrice = params.SRSupportPrice
	data.SRSupportStrength = params.SRSupportStrength
	data.SRSupportDistPct = params.SRSupportDistPct
	data.SRSupportHasWall = params.SRSupportHasWall
	data.SRSupportWallUSD = params.SRSupportWallUSD
	data.SRResistancePrice = params.SRResistancePrice
	data.SRResistanceStrength = params.SRResistanceStrength
	data.SRResistanceDistPct = params.SRResistanceDistPct
	data.SRResistanceHasWall = params.SRResistanceHasWall
	data.SRResistanceWallUSD = params.SRResistanceWallUSD

	// Логируем полученные данные прогресса
	logger.Debug("📊 Service: Использованы данные прогресса из параметров: заполнено %d из %d (%.0f%%)",
		data.FilledSlots, data.TotalSlots, data.ProgressPercentage)

	// Рассчитываем дополнительные временные поля
	data.NextAnalysis = s.calculateNextAnalysis(data.Timestamp, data.Period)
	data.NextSignal = s.calculateNextSignal(data.Timestamp, data.Period, data.Confirmations, data.RequiredConfirmations)

	logger.Debug("🔍 extractRawDataFromParams: RSI=%.2f, MACD=%.2f, Прогресс: %d/%d (%.0f%%)",
		params.RSI, params.MACDSignal, data.FilledSlots, data.TotalSlots, data.ProgressPercentage)

	return data, nil
}
