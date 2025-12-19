// internal/analysis/analyzers/volume_analyzer.go
package analyzers

import (
	"crypto-exchange-screener-bot/internal/analysis"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
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

	// Рассчитываем средний объем
	var totalVolume float64
	hasVolume := false
	for _, point := range data {
		// Проверяем, есть ли данные об объеме
		// Если нет поля Volume24h, используем 0
		if point.Volume24h > 0 {
			totalVolume += point.Volume24h
			hasVolume = true
		}
	}

	if !hasVolume {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	avgVolume := totalVolume / float64(len(data))

	minVolume := a.config.CustomSettings["min_volume"].(float64)
	if avgVolume < minVolume {
		a.updateStats(time.Since(startTime), false)
		return nil, nil
	}

	signal := analysis.Signal{
		Symbol:        data[0].Symbol,
		Type:          "volume_analysis",
		Direction:     "neutral",
		ChangePercent: 0,
		Confidence:    a.calculateVolumeConfidence(avgVolume),
		DataPoints:    len(data),
		StartPrice:    data[0].Price,
		EndPrice:      data[len(data)-1].Price,
		Metadata: analysis.Metadata{
			Strategy: "volume_analysis",
			Tags:     []string{"volume", "liquidity"},
			Indicators: map[string]float64{
				"avg_volume":    avgVolume,
				"volume_change": a.calculateVolumeChange(data),
			},
		},
	}

	a.updateStats(time.Since(startTime), true)

	return []analysis.Signal{signal}, nil
}

func (a *VolumeAnalyzer) calculateVolumeConfidence(volume float64) float64 {
	minVolume := a.config.CustomSettings["min_volume"].(float64)
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

func (a *VolumeAnalyzer) calculateVolumeChange(data []types.PriceData) float64 {
	if len(data) < 2 {
		return 0
	}

	firstVolume := data[0].Volume24h
	lastVolume := data[len(data)-1].Volume24h

	if firstVolume == 0 {
		return 0
	}

	return ((lastVolume - firstVolume) / firstVolume) * 100
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
