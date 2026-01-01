// internal/core/domain/signals/detectors/volume_analyzer.go
package analyzers

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"math"
	"sync"
	"time"
)

// VolumeAnalyzer - анализатор объема
type VolumeAnalyzer struct {
	config AnalyzerConfig
	stats  AnalyzerStats
	mu     sync.RWMutex
}

func (a *VolumeAnalyzer) Name() string {
	return "volume_analyzer"
}

func (a *VolumeAnalyzer) Version() string {
	return "1.0.0"
}

func (a *VolumeAnalyzer) Supports(symbol string) bool {
	return true
}

func (a *VolumeAnalyzer) Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) < config.MinDataPoints {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("insufficient data points")
	}

	var signals []analysis.Signal

	// 1. Проверка среднего объема
	if signal := a.checkAverageVolume(data); signal != nil {
		signals = append(signals, *signal)
	}

	// 2. Проверка скачков объема
	if spikeSignal := a.checkVolumeSpike(data); spikeSignal != nil {
		signals = append(signals, *spikeSignal)
	}

	// 3. Проверка согласованности объема и цены
	if confirmationSignal := a.checkVolumePriceConfirmation(data); confirmationSignal != nil {
		signals = append(signals, *confirmationSignal)
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

func (a *VolumeAnalyzer) checkAverageVolume(data []types.PriceData) *analysis.Signal {
	var totalVolume float64
	validPoints := 0

	for _, point := range data {
		if point.Volume24h > 0 {
			totalVolume += point.Volume24h
			validPoints++
		}
	}

	if validPoints == 0 {
		return nil
	}

	avgVolume := totalVolume / float64(validPoints)
	minVolume := a.getMinVolume()

	if avgVolume < minVolume {
		return nil
	}

	confidence := a.calculateVolumeConfidence(avgVolume, minVolume)

	if confidence < a.config.MinConfidence {
		return nil
	}

	return &analysis.Signal{
		Symbol:        data[0].Symbol,
		Type:          "high_volume",
		Direction:     "neutral",
		ChangePercent: 0,
		Confidence:    confidence,
		DataPoints:    validPoints,
		StartPrice:    data[0].Price,
		EndPrice:      data[len(data)-1].Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "average_volume",
			Tags:     []string{"volume", "liquidity", "high_volume"},
			Indicators: map[string]float64{
				"avg_volume":   avgVolume,
				"min_volume":   minVolume,
				"volume_ratio": avgVolume / minVolume,
			},
		},
	}
}

func (a *VolumeAnalyzer) checkVolumeSpike(data []types.PriceData) *analysis.Signal {
	if len(data) < 2 {
		return nil
	}

	// Находим максимальный объем
	maxVolume := 0.0
	maxIndex := 0

	for i, point := range data {
		if point.Volume24h > maxVolume {
			maxVolume = point.Volume24h
			maxIndex = i
		}
	}

	// Вычисляем средний объем без максимума
	var totalWithoutMax float64
	countWithoutMax := 0
	for i, point := range data {
		if i != maxIndex && point.Volume24h > 0 {
			totalWithoutMax += point.Volume24h
			countWithoutMax++
		}
	}

	if countWithoutMax == 0 {
		return nil
	}

	avgWithoutMax := totalWithoutMax / float64(countWithoutMax)

	// Проверяем, является ли это скачком
	if avgWithoutMax > 0 && maxVolume > avgWithoutMax*3 { // В 3 раза больше среднего
		spikeRatio := maxVolume / avgWithoutMax
		confidence := math.Min(spikeRatio*15, 90) // До 90% уверенности

		return &analysis.Signal{
			Symbol:        data[0].Symbol,
			Type:          "volume_spike",
			Direction:     "neutral",
			ChangePercent: 0,
			Confidence:    confidence,
			DataPoints:    len(data),
			StartPrice:    data[0].Price,
			EndPrice:      data[len(data)-1].Price,
			Timestamp:     time.Now(),
			Metadata: analysis.Metadata{
				Strategy: "volume_spike_detection",
				Tags:     []string{"volume", "spike", "unusual"},
				Indicators: map[string]float64{
					"spike_volume":   maxVolume,
					"avg_volume":     avgWithoutMax,
					"spike_ratio":    spikeRatio,
					"spike_position": float64(maxIndex),
				},
			},
		}
	}

	return nil
}

func (a *VolumeAnalyzer) calculateVolumeConfidence(volume, minVolume float64) float64 {
	if volume < minVolume {
		return 0
	}

	// Нормализация уверенности на основе объема
	if volume > minVolume*10 {
		return 90.0
	} else if volume > minVolume*5 {
		return 70.0
	} else if volume > minVolume*2 {
		return 50.0
	}
	return 30.0
}

func (a *VolumeAnalyzer) getMinVolume() float64 {
	if minVolume, ok := a.config.CustomSettings["min_volume"].(float64); ok {
		return minVolume
	}
	return 100000.0 // значение по умолчанию
}

func (a *VolumeAnalyzer) checkVolumePriceConfirmation(data []types.PriceData) *analysis.Signal {
	if len(data) < 2 {
		return nil
	}

	// Рассчитываем изменение цены
	priceChange := ((data[len(data)-1].Price - data[0].Price) / data[0].Price) * 100

	// Рассчитываем изменение объема
	var volumeChange float64
	if data[0].Volume24h > 0 {
		volumeChange = ((data[len(data)-1].Volume24h - data[0].Volume24h) / data[0].Volume24h) * 100
	} else {
		return nil
	}

	// Проверяем согласованность
	if math.Abs(priceChange) < 0.1 || math.Abs(volumeChange) < 10 {
		// Изменения слишком малы
		return nil
	}

	var signalType, direction string
	var confidence float64

	if priceChange > 0 && volumeChange > 0 {
		// Рост цены + рост объема = сильный бычий сигнал
		signalType = "volume_confirmation"
		direction = "up"
		confirmationStrength := math.Min(priceChange, volumeChange) / 2
		confidence = 50 + math.Min(confirmationStrength, 40) // 50-90%
	} else if priceChange < 0 && volumeChange > 0 {
		// Падение цены + рост объема = сильный медвежий сигнал
		signalType = "volume_confirmation"
		direction = "down"
		confirmationStrength := math.Min(math.Abs(priceChange), volumeChange) / 2
		confidence = 50 + math.Min(confirmationStrength, 40)
	} else if priceChange > 0 && volumeChange < -20 {
		// Рост цены + падение объема = бычья дивергенция (слабый сигнал)
		signalType = "volume_divergence"
		direction = "up"
		confidence = 30
	} else if priceChange < 0 && volumeChange < -20 {
		// Падение цены + падение объема = медвежья дивергенция (слабый сигнал)
		signalType = "volume_divergence"
		direction = "down"
		confidence = 30
	} else {
		// Нет значимой корреляции
		return nil
	}

	if confidence < a.config.MinConfidence {
		return nil
	}

	return &analysis.Signal{
		Symbol:        data[0].Symbol,
		Type:          signalType,
		Direction:     direction,
		ChangePercent: priceChange,
		Confidence:    confidence,
		DataPoints:    len(data),
		StartPrice:    data[0].Price,
		EndPrice:      data[len(data)-1].Price,
		Timestamp:     time.Now(),
		Metadata: analysis.Metadata{
			Strategy: "volume_price_analysis",
			Tags:     []string{"volume", "confirmation", "divergence"},
			Indicators: map[string]float64{
				"price_change":  priceChange,
				"volume_change": volumeChange,
				"correlation":   a.calculateVolumePriceCorrelation(data),
			},
		},
	}
}

func (a *VolumeAnalyzer) calculateVolumePriceCorrelation(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	var priceChanges, volumeChanges []float64

	for i := 1; i < len(data); i++ {
		priceChange := ((data[i].Price - data[i-1].Price) / data[i-1].Price) * 100
		priceChanges = append(priceChanges, priceChange)

		if data[i-1].Volume24h > 0 {
			volumeChange := ((data[i].Volume24h - data[i-1].Volume24h) / data[i-1].Volume24h) * 100
			volumeChanges = append(volumeChanges, volumeChange)
		}
	}

	// Простая корреляция (чем больше, тем сильнее связь)
	if len(priceChanges) != len(volumeChanges) || len(priceChanges) == 0 {
		return 0
	}

	// Считаем сколько раз изменение цены и объема в одном направлении
	sameDirection := 0
	for i := 0; i < len(priceChanges); i++ {
		if (priceChanges[i] > 0 && volumeChanges[i] > 0) ||
			(priceChanges[i] < 0 && volumeChanges[i] < 0) {
			sameDirection++
		}
	}

	correlation := float64(sameDirection) / float64(len(priceChanges)) * 100
	return correlation
}

func (a *VolumeAnalyzer) GetConfig() AnalyzerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

func (a *VolumeAnalyzer) GetStats() AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

func (a *VolumeAnalyzer) updateStats(duration time.Duration, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.stats.TotalCalls++
	a.stats.TotalTime += duration
	a.stats.LastCallTime = time.Now()

	if success {
		a.stats.SuccessCount++
	} else {
		a.stats.ErrorCount++
	}

	if a.stats.TotalCalls > 0 {
		a.stats.AverageTime = time.Duration(
			int64(a.stats.TotalTime) / int64(a.stats.TotalCalls),
		)
	}
}

var DefaultVolumeConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        0.5,
	MinConfidence: 30.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_volume":              100000.0,
		"volume_change_threshold": 50.0,
	},
}
