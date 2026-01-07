// internal/core/domain/signals/detectors/volume_analyzer/analyzer.go
package volume_analyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/volume_analyzer/calculator"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"sync"
	"time"
)

// VolumeAnalyzer - анализатор объема (модульный)
type VolumeAnalyzer struct {
	config           common.AnalyzerConfig
	stats            common.AnalyzerStats
	mu               sync.RWMutex
	averageCalc      *calculator.AverageVolumeCalculator
	spikeCalc        *calculator.VolumeSpikeCalculator
	confirmationCalc *calculator.VolumePriceConfirmationCalculator
}

// NewVolumeAnalyzer создает новый анализатор объема
func NewVolumeAnalyzer(config common.AnalyzerConfig) *VolumeAnalyzer {
	return &VolumeAnalyzer{
		config:           config,
		stats:            common.AnalyzerStats{},
		averageCalc:      calculator.NewAverageVolumeCalculator(config),
		spikeCalc:        calculator.NewVolumeSpikeCalculator(config),
		confirmationCalc: calculator.NewVolumePriceConfirmationCalculator(config),
	}
}

func (a *VolumeAnalyzer) Name() string {
	return "volume_analyzer"
}

func (a *VolumeAnalyzer) Version() string {
	return "1.1.0" // Обновляем версию для модульной версии
}

func (a *VolumeAnalyzer) Supports(symbol string) bool {
	return true
}

func (a *VolumeAnalyzer) Analyze(data []types.PriceData, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	startTime := time.Now()

	if len(data) < config.MinDataPoints {
		a.updateStats(time.Since(startTime), false)
		return nil, fmt.Errorf("insufficient data points")
	}

	// Обновляем конфигурацию если передана новая
	if !isZeroConfig(config) {
		a.updateConfig(config)
	}

	var signals []analysis.Signal

	// 1. Проверка среднего объема
	if signal := a.averageCalc.Calculate(data); signal != nil {
		signals = append(signals, *signal)
	}

	// 2. Проверка скачков объема
	if spikeSignal := a.spikeCalc.Calculate(data); spikeSignal != nil {
		signals = append(signals, *spikeSignal)
	}

	// 3. Проверка согласованности объема и цены
	if confirmationSignal := a.confirmationCalc.Calculate(data); confirmationSignal != nil {
		signals = append(signals, *confirmationSignal)
	}

	a.updateStats(time.Since(startTime), len(signals) > 0)
	return signals, nil
}

func (a *VolumeAnalyzer) GetConfig() common.AnalyzerConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

func (a *VolumeAnalyzer) GetStats() common.AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stats
}

func (a *VolumeAnalyzer) updateConfig(config common.AnalyzerConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.config = config
	// Обновляем конфигурацию калькуляторов
	a.averageCalc.UpdateConfig(config)
	a.spikeCalc.UpdateConfig(config)
	a.confirmationCalc.UpdateConfig(config)
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

// isZeroConfig проверяет, является ли конфигурация нулевой
func isZeroConfig(config common.AnalyzerConfig) bool {
	return !config.Enabled &&
		config.Weight == 0 &&
		config.MinConfidence == 0 &&
		config.MinDataPoints == 0 &&
		len(config.CustomSettings) == 0
}

// DefaultVolumeConfig возвращает конфигурацию по умолчанию
func DefaultVolumeConfig() common.AnalyzerConfig {
	return common.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.5,
		MinConfidence: 30.0,
		MinDataPoints: 3,
		CustomSettings: map[string]interface{}{
			"min_volume":              100000.0,
			"volume_change_threshold": 50.0,
		},
	}
}
