// internal/core/domain/signals/detectors/factory.go
package analyzers

import (
	"fmt"

	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	continuousanalyzer "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/continuous_analyzer"
	fallanalyzer "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/fall_analyzer"
	growthanalyzer "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/growth_analyzer"
	oianalyzer "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/open_interest_analyzer"
	volumeanalyzer "crypto-exchange-screener-bot/internal/core/domain/signals/detectors/volume_analyzer"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
)

// growthAnalyzerWrapper - враппер для нового модульного GrowthAnalyzer
type growthAnalyzerWrapper struct {
	analyzer *growthanalyzer.GrowthAnalyzer
	config   common.AnalyzerConfig
}

// NewGrowthAnalyzer создает анализатор роста (новая модульная версия)
func NewGrowthAnalyzer(config common.AnalyzerConfig) common.Analyzer {
	// Создаем враппер
	wrapper := &growthAnalyzerWrapper{
		config: config,
	}

	return wrapper
}

// Name возвращает имя анализатора
func (w *growthAnalyzerWrapper) Name() string {
	if w.analyzer == nil {
		w.analyzer = growthanalyzer.NewGrowthAnalyzer(w.config)
	}
	return w.analyzer.Name()
}

// Version возвращает версию анализатора
func (w *growthAnalyzerWrapper) Version() string {
	if w.analyzer == nil {
		w.analyzer = growthanalyzer.NewGrowthAnalyzer(w.config)
	}
	return w.analyzer.Version()
}

// Supports проверяет поддержку символа
func (w *growthAnalyzerWrapper) Supports(symbol string) bool {
	if w.analyzer == nil {
		w.analyzer = growthanalyzer.NewGrowthAnalyzer(w.config)
	}
	return w.analyzer.Supports(symbol)
}

// Analyze анализирует данные
func (w *growthAnalyzerWrapper) Analyze(data []redis_storage.PriceData, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	if w.analyzer == nil {
		w.analyzer = growthanalyzer.NewGrowthAnalyzer(w.config)
	}

	// Обновляем конфигурацию если передана новая
	w.config = config
	w.analyzer = growthanalyzer.NewGrowthAnalyzer(config)

	return w.analyzer.Analyze(data, config)
}

// GetConfig возвращает конфигурацию
func (w *growthAnalyzerWrapper) GetConfig() common.AnalyzerConfig {
	return w.config
}

// GetStats возвращает статистику
func (w *growthAnalyzerWrapper) GetStats() common.AnalyzerStats {
	if w.analyzer == nil {
		return common.AnalyzerStats{}
	}

	stats := w.analyzer.GetStats()
	return common.AnalyzerStats{
		TotalCalls:   stats.TotalCalls,
		SuccessCount: stats.SuccessCount,
		ErrorCount:   stats.ErrorCount,
		TotalTime:    stats.TotalTime,
		AverageTime:  stats.AverageTime,
		LastCallTime: stats.LastCallTime,
	}
}

// fallAnalyzerWrapper - враппер для адаптера нового FallAnalyzer
type fallAnalyzerWrapper struct {
	analyzer *fallanalyzer.FallAnalyzer
	config   common.AnalyzerConfig
}

// NewFallAnalyzer создает анализатор падения (новая модульная версия)
func NewFallAnalyzer(config common.AnalyzerConfig) common.Analyzer {
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
func (w *fallAnalyzerWrapper) Analyze(data []redis_storage.PriceData, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	if w.analyzer == nil {
		w.analyzer = fallanalyzer.NewFallAnalyzer()
	}

	// Конвертируем common.AnalyzerConfig в map для новой версии
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
func (w *fallAnalyzerWrapper) GetConfig() common.AnalyzerConfig {
	return w.config
}

// GetStats возвращает статистику
func (w *fallAnalyzerWrapper) GetStats() common.AnalyzerStats {
	if w.analyzer == nil {
		return common.AnalyzerStats{}
	}

	stats := w.analyzer.GetStats()
	return common.AnalyzerStats{
		TotalCalls:   stats.TotalCalls,
		SuccessCount: stats.SuccessCount,
		ErrorCount:   stats.ErrorCount,
		TotalTime:    stats.TotalTime,
		AverageTime:  stats.AverageTime,
		LastCallTime: stats.LastCallTime,
	}
}

// volumeAnalyzerWrapper - враппер для нового модульного VolumeAnalyzer
type volumeAnalyzerWrapper struct {
	analyzer *volumeanalyzer.VolumeAnalyzer
	config   common.AnalyzerConfig
}

// NewVolumeAnalyzer создает анализатор объема (новая модульная версия)
func NewVolumeAnalyzer(config common.AnalyzerConfig) common.Analyzer {
	// Создаем враппер для новой модульной версии
	wrapper := &volumeAnalyzerWrapper{
		config: config,
	}

	return wrapper
}

// Name возвращает имя анализатора
func (w *volumeAnalyzerWrapper) Name() string {
	if w.analyzer == nil {
		w.analyzer = volumeanalyzer.NewVolumeAnalyzer(w.config)
	}
	return w.analyzer.Name()
}

// Version возвращает версию анализатора
func (w *volumeAnalyzerWrapper) Version() string {
	if w.analyzer == nil {
		w.analyzer = volumeanalyzer.NewVolumeAnalyzer(w.config)
	}
	return w.analyzer.Version()
}

// Supports проверяет поддержку символа
func (w *volumeAnalyzerWrapper) Supports(symbol string) bool {
	if w.analyzer == nil {
		w.analyzer = volumeanalyzer.NewVolumeAnalyzer(w.config)
	}
	return w.analyzer.Supports(symbol)
}

// Analyze анализирует данные
func (w *volumeAnalyzerWrapper) Analyze(data []redis_storage.PriceData, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	if w.analyzer == nil {
		w.analyzer = volumeanalyzer.NewVolumeAnalyzer(w.config)
	}

	// Обновляем конфигурацию если передана новая
	w.config = config
	w.analyzer = volumeanalyzer.NewVolumeAnalyzer(config)

	return w.analyzer.Analyze(data, config)
}

// GetConfig возвращает конфигурацию
func (w *volumeAnalyzerWrapper) GetConfig() common.AnalyzerConfig {
	return w.config
}

// GetStats возвращает статистику
func (w *volumeAnalyzerWrapper) GetStats() common.AnalyzerStats {
	if w.analyzer == nil {
		return common.AnalyzerStats{}
	}

	stats := w.analyzer.GetStats()
	return common.AnalyzerStats{
		TotalCalls:   stats.TotalCalls,
		SuccessCount: stats.SuccessCount,
		ErrorCount:   stats.ErrorCount,
		TotalTime:    stats.TotalTime,
		AverageTime:  stats.AverageTime,
		LastCallTime: stats.LastCallTime,
	}
}

// openInterestAnalyzerWrapper - враппер для адаптера нового OI анализатора
type openInterestAnalyzerWrapper struct {
	adapter *oianalyzer.Adapter
	config  common.AnalyzerConfig
}

// NewOpenInterestAnalyzer создает анализатор открытого интереса (новая модульная версия)
func NewOpenInterestAnalyzer(config common.AnalyzerConfig) common.Analyzer {
	// Конвертируем common.AnalyzerConfig в common.AnalyzerConfigCopy для адаптера
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

func (w *openInterestAnalyzerWrapper) Analyze(data []redis_storage.PriceData, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	// Конвертируем common.AnalyzerConfig в common.AnalyzerConfigCopy
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

func (w *openInterestAnalyzerWrapper) GetConfig() common.AnalyzerConfig {
	return w.config
}

func (w *openInterestAnalyzerWrapper) GetStats() common.AnalyzerStats {
	adapterStats := w.adapter.GetStats()

	return common.AnalyzerStats{
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
func (f *AnalyzerFactory) CreateAnalyzer(name string, config common.AnalyzerConfig) common.Analyzer {
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

// GetAllcommon.AnalyzerConfigs возвращает конфигурации всех анализаторов
func GetAllAnalyzerConfigs() map[string]common.AnalyzerConfig {
	return map[string]common.AnalyzerConfig{
		"growth_analyzer":        DefaultGrowthConfig,
		"fall_analyzer":          DefaultFallConfig,
		"continuous_analyzer":    getContinuousConfig(),
		"volume_analyzer":        getVolumeConfig(),
		"open_interest_analyzer": DefaultOpenInterestConfig,
		"counter_analyzer":       DefaultCounterConfig,
	}
}

// Вспомогательные функции для получения конфигураций
func getContinuousConfig() common.AnalyzerConfig {
	return common.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.8,
		MinConfidence: 70.0,
		MinDataPoints: 4,
		CustomSettings: map[string]interface{}{
			"min_continuous_points": 3,
			"max_gap_ratio":         0.3,
			"require_confirmation":  true,
		},
	}
}

func getVolumeConfig() common.AnalyzerConfig {
	// Используем конфигурацию по умолчанию из нового volume_analyzer
	return volumeanalyzer.DefaultVolumeConfig()
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
func GetEnabledAnalyzers(configs map[string]common.AnalyzerConfig) []string {
	var enabled []string
	for name, config := range configs {
		if config.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// CreateAllEnabledAnalyzers создает все включенные анализаторы
func (f *AnalyzerFactory) CreateAllEnabledAnalyzers(configs map[string]common.AnalyzerConfig) map[string]common.Analyzer {
	analyzers := make(map[string]common.Analyzer)

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
func MergeConfigs(customConfigs map[string]common.AnalyzerConfig) map[string]common.AnalyzerConfig {
	defaultConfigs := GetAllAnalyzerConfigs()
	result := make(map[string]common.AnalyzerConfig)

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
func ValidateConfig(config common.AnalyzerConfig) error {
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
func GetDefaultConfig(analyzerName string) common.AnalyzerConfig {
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

// DefaultGrowthConfig - конфигурация по умолчанию для Growth Analyzer
var DefaultGrowthConfig = common.AnalyzerConfig{
	Enabled:       true,
	Weight:        1.0,
	MinConfidence: 60.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_growth_percent":     2.0,
		"continuity_threshold":   0.7,
		"acceleration_threshold": 0.5,
		"volume_weight":          0.2,
		"trend_strength_weight":  0.4,
		"volatility_weight":      0.2,
		"check_all_algorithms":   true,
		"use_new_version":        true,
	},
}

// DefaultCounterConfig - конфигурация по умолчанию для CounterAnalyzer
var DefaultCounterConfig = common.AnalyzerConfig{
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
var DefaultOpenInterestConfig = common.AnalyzerConfig{
	Enabled:       true,
	Weight:        0.6,
	MinConfidence: 50.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_price_change":      1.0,
		"min_price_fall":        1.0,
		"min_oi_change":         5.0,
		"extreme_oi_threshold":  1.5,
		"divergence_min_points": 4,
		"volume_weight":         0.3,
		"check_all_algorithms":  true,
		"use_new_version":       true,
	},
}

// DefaultFallConfig - конфигурация по умолчанию для Fall Analyzer
var DefaultFallConfig = common.AnalyzerConfig{
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

// DefaultVolumeConfig - конфигурация по умолчанию для Volume Analyzer
var DefaultVolumeConfig = common.AnalyzerConfig{
	Enabled:       true,
	Weight:        0.5,
	MinConfidence: 30.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_volume":              100000.0,
		"volume_change_threshold": 50.0,
		"spike_multiplier":        3.0,
		"confirmation_threshold":  10.0,
		"check_all_algorithms":    true,
		"use_new_version":         true,
	},
}

// DefaultContinuousConfig - конфигурация по умолчанию
var DefaultContinuousConfig = common.AnalyzerConfig{
	Enabled:       true,
	Weight:        0.8,
	MinConfidence: 60.0,
	MinDataPoints: 3,
	CustomSettings: map[string]interface{}{
		"min_continuous_points": 3,
		"max_gap_ratio":         0.3,
	},
}

// continuousAnalyzerWrapper - враппер для нового модульного ContinuousAnalyzer
type continuousAnalyzerWrapper struct {
	analyzer *continuousanalyzer.ContinuousAnalyzer
	config   common.AnalyzerConfig
}

// NewContinuousAnalyzer создает анализатор непрерывности (новая модульная версия)
func NewContinuousAnalyzer(config common.AnalyzerConfig) common.Analyzer {
	// Создаем враппер для новой модульной версии
	wrapper := &continuousAnalyzerWrapper{
		config: config,
	}

	return wrapper
}

// Name возвращает имя анализатора
func (w *continuousAnalyzerWrapper) Name() string {
	if w.analyzer == nil {
		w.analyzer = continuousanalyzer.NewContinuousAnalyzer(w.config)
	}
	return w.analyzer.Name()
}

// Version возвращает версию анализатора
func (w *continuousAnalyzerWrapper) Version() string {
	if w.analyzer == nil {
		w.analyzer = continuousanalyzer.NewContinuousAnalyzer(w.config)
	}
	return w.analyzer.Version()
}

// Supports проверяет поддержку символа
func (w *continuousAnalyzerWrapper) Supports(symbol string) bool {
	if w.analyzer == nil {
		w.analyzer = continuousanalyzer.NewContinuousAnalyzer(w.config)
	}
	return w.analyzer.Supports(symbol)
}

// Analyze анализирует данные
func (w *continuousAnalyzerWrapper) Analyze(data []redis_storage.PriceData, config common.AnalyzerConfig) ([]analysis.Signal, error) {
	if w.analyzer == nil {
		w.analyzer = continuousanalyzer.NewContinuousAnalyzer(w.config)
	}

	// Обновляем конфигурацию если передана новая
	w.config = config
	w.analyzer = continuousanalyzer.NewContinuousAnalyzer(config)

	return w.analyzer.Analyze(data, config)
}

// GetConfig возвращает конфигурацию
func (w *continuousAnalyzerWrapper) GetConfig() common.AnalyzerConfig {
	return w.config
}

// GetStats возвращает статистику
func (w *continuousAnalyzerWrapper) GetStats() common.AnalyzerStats {
	if w.analyzer == nil {
		return common.AnalyzerStats{}
	}

	stats := w.analyzer.GetStats()
	return common.AnalyzerStats{
		TotalCalls:   stats.TotalCalls,
		SuccessCount: stats.SuccessCount,
		ErrorCount:   stats.ErrorCount,
		TotalTime:    stats.TotalTime,
		AverageTime:  stats.AverageTime,
		LastCallTime: stats.LastCallTime,
	}
}
