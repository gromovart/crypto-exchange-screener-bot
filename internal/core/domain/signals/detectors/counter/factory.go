// internal/core/domain/signals/detectors/counter/factory.go
package counter

import (
	candle "crypto-exchange-screener-bot/internal/core/domain/candle"
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage" // ДОБАВИТЬ
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
)

// CounterAnalyzerFactory - фабрика для создания CounterAnalyzer
type CounterAnalyzerFactory struct{}

// NewCounterAnalyzerFactory создает новую фабрику
func NewCounterAnalyzerFactory() *CounterAnalyzerFactory {
	return &CounterAnalyzerFactory{}
}

// CreateAnalyzer создает CounterAnalyzer с настройками по умолчанию
func (f *CounterAnalyzerFactory) CreateAnalyzer(
	storage storage.PriceStorage, // ИЗМЕНЕНО: конкретный тип
	eventBus types.EventBus,
	marketFetcher interface{},
	candleSystem *candle.CandleSystem,
) *CounterAnalyzer {
	config := f.DefaultConfig()
	return NewCounterAnalyzer(config, storage, eventBus, marketFetcher, candleSystem)
}

// CreateAnalyzerWithConfig создает CounterAnalyzer с пользовательской конфигурацией
func (f *CounterAnalyzerFactory) CreateAnalyzerWithConfig(
	config common.AnalyzerConfig,
	storage storage.PriceStorage, // ИЗМЕНЕНО: конкретный тип
	eventBus types.EventBus,
	marketFetcher interface{},
	candleSystem *candle.CandleSystem,
) *CounterAnalyzer {
	return NewCounterAnalyzer(config, storage, eventBus, marketFetcher, candleSystem)
}

// CreateFromCustomSettings создает CounterAnalyzer из пользовательских настроек
func (f *CounterAnalyzerFactory) CreateFromCustomSettings(
	customSettings map[string]interface{},
	storage storage.PriceStorage, // ИЗМЕНЕНО: конкретный тип
	eventBus types.EventBus,
	marketFetcher interface{},
	candleSystem *candle.CandleSystem,
) *CounterAnalyzer {
	config := common.AnalyzerConfig{
		Enabled:        true,
		Weight:         0.7,
		MinConfidence:  10.0,
		MinDataPoints:  2,
		CustomSettings: f.mergeWithDefaults(customSettings),
	}
	return NewCounterAnalyzer(config, storage, eventBus, marketFetcher, candleSystem)
}

// CreateTestAnalyzer создает тестовый анализатор (без внешних зависимостей)
func (f *CounterAnalyzerFactory) CreateTestAnalyzer() *CounterAnalyzer {
	config := common.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.7,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"base_period_minutes": 1,
			"analysis_period":     "15m",
			"growth_threshold":    0.1,
			"fall_threshold":      0.1,
			"track_growth":        true,
			"track_fall":          true,
			"notify_on_signal":    false,
			"chart_provider":      "coinglass",
		},
	}
	// Для тестов передаем nil storage
	return NewCounterAnalyzer(config, nil, nil, nil, nil)
}

// CreateMinimalAnalyzer создает минимальный анализатор
func (f *CounterAnalyzerFactory) CreateMinimalAnalyzer(storage storage.PriceStorage) *CounterAnalyzer { // ИЗМЕНЕНО
	config := common.AnalyzerConfig{
		Enabled:       true,
		Weight:        0.7,
		MinConfidence: 10.0,
		MinDataPoints: 2,
		CustomSettings: map[string]interface{}{
			"base_period_minutes": 1,
			"analysis_period":     "15m",
			"growth_threshold":    0.1,
			"fall_threshold":      0.1,
			"track_growth":        true,
			"track_fall":          true,
			"notify_on_signal":    false,
		},
	}
	return NewCounterAnalyzer(config, storage, nil, nil, nil)
}

// DefaultConfig возвращает конфигурацию по умолчанию
func (f *CounterAnalyzerFactory) DefaultConfig() common.AnalyzerConfig {
	return common.AnalyzerConfig{
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
}

// mergeWithDefaults объединяет пользовательские настройки с настройками по умолчанию
func (f *CounterAnalyzerFactory) mergeWithDefaults(customSettings map[string]interface{}) map[string]interface{} {
	defaults := map[string]interface{}{
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
	}

	result := make(map[string]interface{})
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range customSettings {
		result[k] = v
	}
	return result
}

// ValidateConfig проверяет корректность конфигурации
func (f *CounterAnalyzerFactory) ValidateConfig(config common.AnalyzerConfig) error {
	if config.Weight < 0 || config.Weight > 1 {
		return fmt.Errorf("weight must be between 0 and 1")
	}
	if config.MinConfidence < 0 || config.MinConfidence > 100 {
		return fmt.Errorf("min_confidence must be between 0 and 100")
	}
	if config.MinDataPoints < 1 {
		return fmt.Errorf("min_data_points must be at least 1")
	}
	requiredSettings := []string{
		"base_period_minutes",
		"analysis_period",
		"growth_threshold",
		"fall_threshold",
	}
	for _, setting := range requiredSettings {
		if _, exists := config.CustomSettings[setting]; !exists {
			return fmt.Errorf("required setting %s is missing", setting)
		}
	}
	return nil
}

// GetSupportedSettings возвращает список поддерживаемых настроек
func (f *CounterAnalyzerFactory) GetSupportedSettings() []string {
	return []string{
		"base_period_minutes",
		"analysis_period",
		"growth_threshold",
		"fall_threshold",
		"track_growth",
		"track_fall",
		"notify_on_signal",
		"notification_threshold",
		"chart_provider",
		"exchange",
		"include_oi",
		"include_volume",
		"include_funding",
		"volume_delta_ttl",
		"delta_fallback_enabled",
		"show_delta_source",
	}
}

// GetSettingDescription возвращает описание настройки
func (f *CounterAnalyzerFactory) GetSettingDescription(setting string) string {
	descriptions := map[string]string{
		"base_period_minutes":    "Базовый период в минутах (по умолчанию: 1)",
		"analysis_period":        "Период анализа (5m, 15m, 30m, 1h, 4h, 1d)",
		"growth_threshold":       "Порог роста в процентах (по умолчанию: 0.1)",
		"fall_threshold":         "Порог падения в процентах (по умолчанию: 0.1)",
		"track_growth":           "Отслеживать рост (true/false)",
		"track_fall":             "Отслеживать падение (true/false)",
		"notify_on_signal":       "Отправлять уведомления (true/false)",
		"notification_threshold": "Порог для уведомлений",
		"chart_provider":         "Провайдер графиков (coinglass/tradingview)",
		"exchange":               "Биржа (bybit/binance)",
		"include_oi":             "Включать открытый интерес (true/false)",
		"include_volume":         "Включать объемы (true/false)",
		"include_funding":        "Включать фандинг (true/false)",
		"volume_delta_ttl":       "TTL кэша дельты в секундах",
		"delta_fallback_enabled": "Включить fallback для дельты (true/false)",
		"show_delta_source":      "Показывать источник данных дельты (true/false)",
	}
	if desc, exists := descriptions[setting]; exists {
		return desc
	}
	return "Неизвестная настройка"
}
