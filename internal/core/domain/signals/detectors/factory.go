// internal/core/domain/signals/detectors/factory.go
package analyzers

import (
	"fmt"
	"time"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	fallanalyzer "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/fall_analyzer"
	oianalyzer "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/open_interest_analyzer"
	"crypto-exchange-screener-bot/internal/types"
)

// NewGrowthAnalyzer создает анализатор роста
func NewGrowthAnalyzer(config AnalyzerConfig) *GrowthAnalyzer {
	return &GrowthAnalyzer{
		config: config,
		stats: AnalyzerStats{
			TotalCalls:   0,
			TotalTime:    0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
			AverageTime:  0,
		},
	}
}

// fallAnalyzerWrapper - враппер для адаптера нового FallAnalyzer
type fallAnalyzerWrapper struct {
	analyzer *fallanalyzer.FallAnalyzer
	config   AnalyzerConfig
}

// NewFallAnalyzer создает анализатор падения (новая модульная версия)
func NewFallAnalyzer(config AnalyzerConfig) Analyzer {
	// Создаем враппер для новой модульной версии
	wrapper := &fallAnalyzerWrapper{
		config: config,
	}

	return wrapper
}

// Name возвращает имя анализатора
func (w *fallAnalyzerWrapper) Name() string {
	if w.analyzer == nil {
		w.analyzer = fallanalyzer.NewFallAnalyzer()
	}
	return w.analyzer.Name()
}

// Version возвращает версию анализатора
func (w *fallAnalyzerWrapper) Version() string {
	if w.analyzer == nil {
		w.analyzer = fallanalyzer.NewFallAnalyzer()
	}
	return w.analyzer.Version()
}

// Supports проверяет поддержку символа
func (w *fallAnalyzerWrapper) Supports(symbol string) bool {
	if w.analyzer == nil {
		w.analyzer = fallanalyzer.NewFallAnalyzer()
	}
	return w.analyzer.Supports(symbol)
}

// Analyze анализирует данные
func (w *fallAnalyzerWrapper) Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error) {
	if w.analyzer == nil {
		w.analyzer = fallanalyzer.NewFallAnalyzer()
	}

	// Конвертируем AnalyzerConfig в map для новой версии
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

	return w.analyzer.Analyze(data, cfgMap)
}

// GetConfig возвращает конфигурацию
func (w *fallAnalyzerWrapper) GetConfig() AnalyzerConfig {
	return w.config
}

// GetStats возвращает статистику
func (w *fallAnalyzerWrapper) GetStats() AnalyzerStats {
	if w.analyzer == nil {
		return AnalyzerStats{}
	}

	stats := w.analyzer.GetStats()
	return AnalyzerStats{
		TotalCalls:   stats.TotalCalls,
		SuccessCount: stats.SuccessCount,
		ErrorCount:   stats.ErrorCount,
		TotalTime:    stats.TotalTime,
		AverageTime:  stats.AverageTime,
		LastCallTime: stats.LastCallTime,
	}
}

// NewVolumeAnalyzer создает анализатор объема
func NewVolumeAnalyzer(config AnalyzerConfig) *VolumeAnalyzer {
	return &VolumeAnalyzer{
		config: config,
		stats: AnalyzerStats{
			TotalCalls:   0,
			TotalTime:    0,
			SuccessCount: 0,
			ErrorCount:   0,
			LastCallTime: time.Time{},
			AverageTime:  0,
		},
	}
}

// NewContinuousAnalyzer создает анализатор непрерывности
func NewContinuousAnalyzer(config AnalyzerConfig) *ContinuousAnalyzer {
	// Гарантируем наличие необходимых настроек
	if config.CustomSettings == nil {
		config.CustomSettings = make(map[string]interface{})
	}

	// Устанавливаем значения по умолчанию
	defaults := map[string]interface{}{
		"min_continuous_points": 3,
		"max_gap_ratio":         0.3,
	}

	for key, defaultValue := range defaults {
		if _, ok := config.CustomSettings[key]; !ok {
			config.CustomSettings[key] = defaultValue
		}
	}

	return &ContinuousAnalyzer{
		config: config,
		stats:  AnalyzerStats{},
	}
}

// openInterestAnalyzerWrapper - враппер для адаптера нового OI анализатора
type openInterestAnalyzerWrapper struct {
	adapter *oianalyzer.Adapter
	config  AnalyzerConfig
}

// NewOpenInterestAnalyzer создает анализатор открытого интереса (новая модульная версия)
func NewOpenInterestAnalyzer(config AnalyzerConfig) Analyzer {
	// Конвертируем AnalyzerConfig в AnalyzerConfigCopy для адаптера
	adapterConfig := oianalyzer.AnalyzerConfigCopy{
		Enabled:        config.Enabled,
		Weight:         config.Weight,
		MinConfidence:  config.MinConfidence,
		MinDataPoints:  config.MinDataPoints,
		CustomSettings: make(map[string]interface{}),
	}

	// Копируем кастомные настройки
	if config.CustomSettings != nil {
		for k, v := range config.CustomSettings {
			adapterConfig.CustomSettings[k] = v
		}
	}

	// Создаем адаптер
	adapter := oianalyzer.NewAdapterWithConfig(adapterConfig)

	// Возвращаем враппер
	return &openInterestAnalyzerWrapper{
		adapter: adapter,
		config:  config,
	}
}

// Реализуем интерфейс Analyzer для враппера:

func (w *openInterestAnalyzerWrapper) Name() string {
	return w.adapter.Name()
}

func (w *openInterestAnalyzerWrapper) Version() string {
	return w.adapter.Version()
}

func (w *openInterestAnalyzerWrapper) Supports(symbol string) bool {
	return w.adapter.Supports(symbol)
}

func (w *openInterestAnalyzerWrapper) Analyze(data []types.PriceData, config AnalyzerConfig) ([]analysis.Signal, error) {
	// Конвертируем AnalyzerConfig в AnalyzerConfigCopy
	adapterConfig := oianalyzer.AnalyzerConfigCopy{
		Enabled:        config.Enabled,
		Weight:         config.Weight,
		MinConfidence:  config.MinConfidence,
		MinDataPoints:  config.MinDataPoints,
		CustomSettings: make(map[string]interface{}),
	}

	if config.CustomSettings != nil {
		for k, v := range config.CustomSettings {
			adapterConfig.CustomSettings[k] = v
		}
	}

	return w.adapter.Analyze(data, adapterConfig)
}

func (w *openInterestAnalyzerWrapper) GetConfig() AnalyzerConfig {
	return w.config
}

func (w *openInterestAnalyzerWrapper) GetStats() AnalyzerStats {
	adapterStats := w.adapter.GetStats()

	return AnalyzerStats{
		TotalCalls:   adapterStats.TotalCalls,
		SuccessCount: adapterStats.SuccessCount,
		ErrorCount:   adapterStats.ErrorCount,
		TotalTime:    adapterStats.TotalTime,
		AverageTime:  adapterStats.AverageTime,
		LastCallTime: adapterStats.LastCallTime,
	}
}

// AnalyzerFactory - фабрика для создания анализаторов (обновленная)
type AnalyzerFactory struct{}

// NewAnalyzerFactory создает новую фабрику анализаторов
func NewAnalyzerFactory() *AnalyzerFactory {
	return &AnalyzerFactory{}
}

// CreateAnalyzer создает анализатор по имени
func (f *AnalyzerFactory) CreateAnalyzer(name string, config AnalyzerConfig) Analyzer {
	switch name {
	case "growth_analyzer":
		return NewGrowthAnalyzer(config)
	case "fall_analyzer":
		return NewFallAnalyzer(config)
	case "continuous_analyzer":
		return NewContinuousAnalyzer(config)
	case "volume_analyzer":
		return NewVolumeAnalyzer(config)
	case "open_interest_analyzer":
		return NewOpenInterestAnalyzer(config)
	case "counter_analyzer":
		// CounterAnalyzer требует дополнительные параметры
		// Он должен создаваться отдельно через NewCounterAnalyzer
		// с передачей storage и telegramBot
		return nil
	default:
		// Возвращаем анализатор по умолчанию
		return NewGrowthAnalyzer(config)
	}
}

// GetAllAnalyzerConfigs возвращает конфигурации всех анализаторов
func GetAllAnalyzerConfigs() map[string]AnalyzerConfig {
	return map[string]AnalyzerConfig{
		"growth_analyzer":        DefaultGrowthConfig,
		"fall_analyzer":          DefaultFallConfig,
		"continuous_analyzer":    DefaultContinuousConfig,
		"volume_analyzer":        DefaultVolumeConfig,
		"open_interest_analyzer": DefaultOpenInterestConfig,
		"counter_analyzer":       DefaultCounterConfig,
	}
}

// GetAnalyzerNames возвращает список всех доступных анализаторов
func GetAnalyzerNames() []string {
	return []string{
		"growth_analyzer",
		"fall_analyzer",
		"continuous_analyzer",
		"volume_analyzer",
		"open_interest_analyzer",
		"counter_analyzer",
	}
}

// GetEnabledAnalyzers возвращает список включенных анализаторов на основе конфигурации
func GetEnabledAnalyzers(configs map[string]AnalyzerConfig) []string {
	var enabled []string
	for name, config := range configs {
		if config.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// CreateAllEnabledAnalyzers создает все включенные анализаторы
func (f *AnalyzerFactory) CreateAllEnabledAnalyzers(configs map[string]AnalyzerConfig) map[string]Analyzer {
	analyzers := make(map[string]Analyzer)

	for name, config := range configs {
		if config.Enabled {
			analyzer := f.CreateAnalyzer(name, config)
			if analyzer != nil {
				analyzers[name] = analyzer
			}
		}
	}

	return analyzers
}

// MergeConfigs объединяет пользовательские настройки с настройками по умолчанию
func MergeConfigs(customConfigs map[string]AnalyzerConfig) map[string]AnalyzerConfig {
	defaultConfigs := GetAllAnalyzerConfigs()
	result := make(map[string]AnalyzerConfig)

	for name, defaultConfig := range defaultConfigs {
		if customConfig, exists := customConfigs[name]; exists {
			// Объединяем настройки
			mergedConfig := defaultConfig
			mergedConfig.Enabled = customConfig.Enabled
			mergedConfig.Weight = customConfig.Weight
			mergedConfig.MinConfidence = customConfig.MinConfidence
			mergedConfig.MinDataPoints = customConfig.MinDataPoints

			// Объединяем CustomSettings
			if mergedConfig.CustomSettings == nil {
				mergedConfig.CustomSettings = make(map[string]interface{})
			}
			if customConfig.CustomSettings != nil {
				for key, value := range customConfig.CustomSettings {
					mergedConfig.CustomSettings[key] = value
				}
			}

			result[name] = mergedConfig
		} else {
			result[name] = defaultConfig
		}
	}

	return result
}

// ValidateConfig проверяет корректность конфигурации анализатора
func ValidateConfig(config AnalyzerConfig) error {
	if config.Weight < 0 || config.Weight > 1 {
		return fmt.Errorf("weight must be between 0 and 1")
	}

	if config.MinConfidence < 0 || config.MinConfidence > 100 {
		return fmt.Errorf("min_confidence must be between 0 and 100")
	}

	if config.MinDataPoints < 1 {
		return fmt.Errorf("min_data_points must be at least 1")
	}

	return nil
}

// GetDefaultConfig возвращает конфигурацию по умолчанию для анализатора
func GetDefaultConfig(analyzerName string) AnalyzerConfig {
	configs := GetAllAnalyzerConfigs()
	if config, exists := configs[analyzerName]; exists {
		return config
	}

	// Возвращаем конфигурацию по умолчанию для growth_analyzer
	return DefaultGrowthConfig
}

// IsAnalyzerAvailable проверяет, доступен ли анализатор
func IsAnalyzerAvailable(analyzerName string) bool {
	availableAnalyzers := GetAnalyzerNames()
	for _, name := range availableAnalyzers {
		if name == analyzerName {
			return true
		}
	}
	return false
}

// DefaultCounterConfig - конфигурация по умолчанию для CounterAnalyzer
var DefaultCounterConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        0.7,
	MinConfidence: 10.0,
	MinDataPoints: 2,
	CustomSettings: map[string]interface{}{
		"base_period_minutes":    1,
		"analysis_period":        "15m",
		"growth_threshold":       0.1,
		"fall_threshold":         0.1,
		"track_growth":           true,
		"track_fall":             true,
		"notify_on_signal":       true,
		"notification_threshold": 1,
		"chart_provider":         "coinglass",
		"exchange":               "bybit",
		"include_oi":             true,
		"include_volume":         true,
		"include_funding":        true,
		"volume_delta_ttl":       30,
		"delta_fallback_enabled": true,
		"show_delta_source":      true,
	},
}

// DefaultOpenInterestConfig - конфигурация по умолчанию для Open Interest Analyzer
var DefaultOpenInterestConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        0.6,
	MinConfidence: 50.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_price_change":      1.0,  // минимальное изменение цены для сигнала (%)
		"min_price_fall":        1.0,  // минимальное падение цены для сигнала (%)
		"min_oi_change":         5.0,  // минимальное изменение OI для сигнала (%)
		"extreme_oi_threshold":  1.5,  // порог экстремального OI (1.5 = на 50% выше среднего)
		"divergence_min_points": 4,    // минимальное количество точек для дивергенции
		"volume_weight":         0.3,  // вес объема в расчетах
		"check_all_algorithms":  true, // проверять все алгоритмы
		"use_new_version":       true, // использовать новую модульную версию
	},
}

// DefaultFallConfig - конфигурация по умолчанию для Fall Analyzer
var DefaultFallConfig = AnalyzerConfig{
	Enabled:       true,
	Weight:        1.0,
	MinConfidence: 60.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_fall":             2.0,
		"continuity_threshold": 0.7,
		"volume_weight":        0.2,
		"check_all_algorithms": true,
		"use_new_version":      true,
	},
}
