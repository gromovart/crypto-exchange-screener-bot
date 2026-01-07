// internal/core/domain/signals/detectors/open_interest_analyzer/adapter.go
package oianalyzer

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

// Adapter - адаптер для совместимости с оригинальным интерфейсом
type Adapter struct {
	analyzer *OpenInterestAnalyzer
	config   AnalyzerConfigCopy
	stats    common.AnalyzerStats
}

// NewAdapter создает новый адаптер с конфигурацией по умолчанию
func NewAdapter() *Adapter {
	return &Adapter{
		analyzer: NewOpenInterestAnalyzer(),
		config: AnalyzerConfigCopy{
			Enabled:        true,
			Weight:         0.6,
			MinConfidence:  50.0,
			MinDataPoints:  3,
			CustomSettings: make(map[string]interface{}),
		},
		stats: common.AnalyzerStats{
			TotalCalls:   0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
		},
	}
}

// NewAdapterWithConfig создает адаптер с кастомной конфигурацией
func NewAdapterWithConfig(config AnalyzerConfigCopy) *Adapter {
	return &Adapter{
		analyzer: NewOpenInterestAnalyzer(),
		config:   config,
		stats: common.AnalyzerStats{
			TotalCalls:   0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
		},
	}
}

// Name возвращает имя анализатора
func (a *Adapter) Name() string {
	return a.analyzer.Name()
}

// Version возвращает версию анализатора
func (a *Adapter) Version() string {
	return a.analyzer.Version()
}

// Supports проверяет поддержку символа
func (a *Adapter) Supports(symbol string) bool {
	return a.analyzer.Supports(symbol)
}

// Analyze анализирует данные (реализация оригинального интерфейса Analyzer)
func (a *Adapter) Analyze(data []types.PriceData, config AnalyzerConfigCopy) ([]analysis.Signal, error) {
	// Обновляем конфигурацию
	a.config = config

	// Конвертируем AnalyzerConfigCopy в map для внутреннего анализатора
	cfgMap := make(map[string]interface{})
	cfgMap["enabled"] = config.Enabled
	cfgMap["weight"] = config.Weight
	cfgMap["min_confidence"] = config.MinConfidence
	cfgMap["min_data_points"] = config.MinDataPoints

	// Добавляем кастомные настройки
	if config.CustomSettings != nil {
		for k, v := range config.CustomSettings {
			cfgMap[k] = v
		}
	}

	// Вызываем внутренний анализатор
	signals, err := a.analyzer.Analyze(data, cfgMap)

	// Обновляем статистику
	duration := time.Since(a.stats.LastCallTime)
	a.updateStats(duration, err == nil && len(signals) > 0)

	return signals, err
}

// GetConfig возвращает конфигурацию
func (a *Adapter) GetConfig() AnalyzerConfigCopy {
	return a.config
}

// GetStats возвращает статистику
func (a *Adapter) GetStats() common.AnalyzerStats {
	return a.stats
}

// updateStats обновляет статистику адаптера
func (a *Adapter) updateStats(duration time.Duration, success bool) {
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
