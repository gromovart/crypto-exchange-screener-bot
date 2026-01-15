// internal/delivery/telegram/services/counter/extractors.go
package counter

import (
	"log"
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

	// Рассчитываем дополнительные поля
	totalGroups, _ := s.getGroupedSlotsInfo(params.Period)
	data.TotalSlots = totalGroups
	data.FilledSlots = s.calculateFilledGroups(params.Confirmations, data.RequiredConfirmations, totalGroups)

	if data.RequiredConfirmations > 0 {
		data.ProgressPercentage = float64(data.Confirmations) / float64(data.RequiredConfirmations) * 100
	}

	data.NextAnalysis = s.calculateNextAnalysis(data.Timestamp, data.Period)
	data.NextSignal = s.calculateNextSignal(data.Timestamp, data.Period, data.Confirmations, data.RequiredConfirmations)

	log.Printf("🔍 extractRawDataFromParams: RSI=%.2f, MACD=%.2f",
		params.RSI, params.MACDSignal)

	return data, nil
}
