// internal/core/domain/signals/detectors/counter/signal_processor.go
package counter

import (
	"fmt"
	"math"
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/counter/manager"
	"crypto-exchange-screener-bot/internal/types"

	"github.com/google/uuid"
)

type SignalProcessor struct {
	analyzer *CounterAnalyzer
}

func NewSignalProcessor(analyzer *CounterAnalyzer) *SignalProcessor {
	return &SignalProcessor{analyzer: analyzer}
}

func (sp *SignalProcessor) Process(data []types.PriceData, cfg common.AnalyzerConfig) ([]analysis.Signal, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("insufficient data points")
	}

	symbol := data[0].Symbol

	// Получаем конфигурацию
	basePeriodMinutes := sp.getBasePeriodMinutes(cfg)
	analysisPeriod := sp.getCurrentPeriod(cfg)
	maxSignals := sp.calculateMaxSignals(analysisPeriod, basePeriodMinutes)

	// Получаем или создаем счетчик
	counter := sp.analyzer.counterManager.GetOrCreateCounter(symbol, analysisPeriod, basePeriodMinutes)

	// Проверяем и сбрасываем период если нужно
	sp.analyzer.periodManager.CheckAndResetPeriod(counter, analysisPeriod, maxSignals)

	// Рассчитываем изменение цены
	change := sp.calculateChange(data)

	var signals []analysis.Signal

	// Блокируем счетчик для записи
	counter.Lock()
	counter.BasePeriodCount++

	// Проверяем рост
	growthThreshold := sp.getGrowthThreshold(cfg)
	if change > growthThreshold && counter.Settings.TrackGrowth {
		counter.GrowthCount++
		counter.SignalCount++
		counter.LastSignalTime = time.Now()

		signal := sp.createSignal(symbol, "growth", math.Abs(change), counter.SignalCount, maxSignals, data)
		signals = append(signals, signal)
	}

	// Проверяем падение
	fallThreshold := sp.getFallThreshold(cfg)
	if change < -fallThreshold && counter.Settings.TrackFall {
		counter.FallCount++
		counter.SignalCount++
		counter.LastSignalTime = time.Now()

		signal := sp.createSignal(symbol, "fall", math.Abs(change), counter.SignalCount, maxSignals, data)
		signals = append(signals, signal)
	}

	counter.Unlock()

	return signals, nil
}

func (sp *SignalProcessor) calculateChange(data []types.PriceData) float64 {
	startPrice := data[0].Price
	endPrice := data[len(data)-1].Price
	return ((endPrice - startPrice) / startPrice) * 100
}

func (sp *SignalProcessor) createSignal(
	symbol, direction string,
	change float64,
	count, maxSignals int,
	data []types.PriceData,
) analysis.Signal {
	confidence := sp.calculateConfidence(count, maxSignals)
	latestData := data[len(data)-1]

	return analysis.Signal{
		ID:            uuid.New().String(),
		Symbol:        symbol,
		Type:          "counter_" + direction,
		Direction:     direction,
		ChangePercent: change,
		Confidence:    confidence,
		DataPoints:    2,
		StartPrice:    data[0].Price,
		EndPrice:      latestData.Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "counter_analyzer_v2",
			Tags: []string{
				"counter",
				direction,
				fmt.Sprintf("count_%d", count),
				fmt.Sprintf("max_%d", maxSignals),
			},
			Indicators: map[string]float64{
				"count":         float64(count),
				"max_signals":   float64(maxSignals),
				"percentage":    float64(count) / float64(maxSignals) * 100,
				"volume_24h":    latestData.Volume24h,
				"open_interest": latestData.OpenInterest,
				"funding_rate":  latestData.FundingRate,
				"current_price": latestData.Price,
			},
		},
	}
}

func (sp *SignalProcessor) calculateConfidence(count, maxSignals int) float64 {
	if maxSignals == 0 {
		return 0
	}
	return float64(count) / float64(maxSignals) * 100
}

func (sp *SignalProcessor) getGrowthThreshold(cfg common.AnalyzerConfig) float64 {
	if val, ok := cfg.CustomSettings["growth_threshold"].(float64); ok {
		return val
	}
	return 0.1
}

func (sp *SignalProcessor) getFallThreshold(cfg common.AnalyzerConfig) float64 {
	if val, ok := cfg.CustomSettings["fall_threshold"].(float64); ok {
		return val
	}
	return 0.1
}

func (sp *SignalProcessor) getBasePeriodMinutes(cfg common.AnalyzerConfig) int {
	if val, ok := cfg.CustomSettings["base_period_minutes"].(int); ok {
		return val
	}
	return 1
}

func (sp *SignalProcessor) getCurrentPeriod(cfg common.AnalyzerConfig) string {
	if val, ok := cfg.CustomSettings["analysis_period"].(string); ok {
		return val
	}
	return "15m"
}

func (sp *SignalProcessor) calculateMaxSignals(period string, basePeriodMinutes int) int {
	periodMinutes := manager.GetPeriodMinutes(period)
	totalPossibleSignals := periodMinutes / basePeriodMinutes

	if totalPossibleSignals < 5 {
		return 5
	}
	if totalPossibleSignals > 15 {
		return 15
	}
	return totalPossibleSignals
}
