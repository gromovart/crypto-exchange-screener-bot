// internal/core/candle/factory.go (исправленная)
package candle

import (
	"fmt"
	"time"

	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	"crypto-exchange-screener-bot/pkg/logger"
)

// CandleSystemFactory - фабрика для создания свечной системы
type CandleSystemFactory struct {
	config CandleConfig
}

// NewCandleSystemFactory создает новую фабрику
func NewCandleSystemFactory() *CandleSystemFactory {
	return &CandleSystemFactory{
		config: CandleConfig{
			SupportedPeriods: []string{"5m", "15m", "30m", "1h", "4h", "1d"},
			MaxHistory:       1000,
			CleanupInterval:  5 * time.Minute,
			AutoBuild:        true,
		},
	}
}

// WithSupportedPeriods устанавливает поддерживаемые периоды
func (f *CandleSystemFactory) WithSupportedPeriods(periods []string) *CandleSystemFactory {
	f.config.SupportedPeriods = periods
	return f
}

// WithMaxHistory устанавливает максимальную историю
func (f *CandleSystemFactory) WithMaxHistory(maxHistory int) *CandleSystemFactory {
	f.config.MaxHistory = maxHistory
	return f
}

// WithCleanupInterval устанавливает интервал очистки
func (f *CandleSystemFactory) WithCleanupInterval(interval time.Duration) *CandleSystemFactory {
	f.config.CleanupInterval = interval
	return f
}

// WithAutoBuild включает/выключает авто-построение
func (f *CandleSystemFactory) WithAutoBuild(autoBuild bool) *CandleSystemFactory {
	f.config.AutoBuild = autoBuild
	return f
}

// CreateSystem создает полную свечную систему
func (f *CandleSystemFactory) CreateSystem(priceStorage storage.PriceStorage) (*CandleSystem, error) {
	if priceStorage == nil {
		return nil, fmt.Errorf("price storage не инициализирован")
	}

	logger.Info("🏗️ Создание свечной системы с периодами: %v", f.config.SupportedPeriods)

	// Создаем хранилище свечей
	candleStorage := NewCandleStorage(f.config)

	// Создаем движок
	candleEngine := NewCandleEngine(candleStorage, f.config)

	// Создаем калькулятор
	candleCalculator := NewCandleCalculator(priceStorage)

	// Создаем систему
	system := &CandleSystem{
		Storage:      candleStorage,
		Engine:       candleEngine,
		Calculator:   candleCalculator,
		priceStorage: priceStorage,
		config:       f.config,
	}

	logger.Info("✅ Свечная система создана успешно")
	return system, nil
}

// CandleSystem - полная свечная система
type CandleSystem struct {
	Storage      *CandleStorage
	Engine       *CandleEngine
	Calculator   *CandleCalculator
	priceStorage storage.PriceStorage
	config       CandleConfig
}

// Start запускает свечную систему
func (cs *CandleSystem) Start() error {
	logger.Info("🚀 Запуск свечной системы...")

	// Запускаем движок
	if err := cs.Engine.Start(); err != nil {
		return err
	}

	// Предзагружаем свечи для существующих символов
	cs.preloadCandles()

	logger.Info("✅ Свечная система запущена")
	return nil
}

// Stop останавливает свечную систему
func (cs *CandleSystem) Stop() error {
	logger.Info("🛑 Остановка свечной системы...")

	if err := cs.Engine.Stop(); err != nil {
		return err
	}

	logger.Info("✅ Свечная система остановлена")
	return nil
}

// preloadCandles предзагружает свечи для существующих символов
func (cs *CandleSystem) preloadCandles() {
	symbols := cs.priceStorage.GetSymbols()
	logger.Debug("🔍 Предзагрузка свечей для %d символов", len(symbols))

	// Для каждого символа и периода строим начальные свечи
	for _, symbol := range symbols {
		for _, period := range cs.config.SupportedPeriods {
			// Пробуем построить свечу из истории
			candle, err := cs.Calculator.BuildCandleFromHistory(symbol, period)
			if err == nil && candle != nil && candle.IsReal {
				// Сохраняем как историческую свечу
				candle.IsClosed = true
				cs.Storage.CloseAndArchiveCandle(candle)
			}
		}
	}

	logger.Debug("✅ Предзагружены свечи для %d символов", len(symbols))
}

// OnPriceUpdate обрабатывает обновление цены
func (cs *CandleSystem) OnPriceUpdate(priceData storage.PriceData) {
	cs.Engine.OnPriceUpdate(priceData)
}

// GetCandle получает свечу для символа и периода
func (cs *CandleSystem) GetCandle(symbol, period string) (*Candle, error) {
	// Сначала проверяем активную свечу
	if candle, exists := cs.Storage.GetActiveCandle(symbol, period); exists {
		return candle, nil
	}

	// Затем проверяем последнюю историческую
	if candle, exists := cs.Storage.GetLatestCandle(symbol, period); exists {
		return candle, nil
	}

	// Если нет, строим из истории
	return cs.Calculator.BuildCandleFromHistory(symbol, period)
}

// GetHistory возвращает историю свечей
func (cs *CandleSystem) GetHistory(symbol, period string, limit int) ([]*Candle, error) {
	return cs.Storage.GetHistory(symbol, period, limit)
}

// GetStats возвращает статистику системы
func (cs *CandleSystem) GetStats() map[string]interface{} {
	engineStats := cs.Engine.GetStats()
	storageStats := cs.Storage.GetStats()

	return map[string]interface{}{
		"system_config": map[string]interface{}{
			"supported_periods": cs.config.SupportedPeriods,
			"max_history":       cs.config.MaxHistory,
			"cleanup_interval":  cs.config.CleanupInterval.String(),
			"auto_build":        cs.config.AutoBuild,
		},
		"engine_stats":  engineStats,
		"storage_stats": storageStats,
	}
}

// CreateSimpleSystem создает упрощенную свечную систему
func CreateSimpleSystem(priceStorage storage.PriceStorage) (*CandleSystem, error) {
	factory := NewCandleSystemFactory()
	return factory.CreateSystem(priceStorage)
}
