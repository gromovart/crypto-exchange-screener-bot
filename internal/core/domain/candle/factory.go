// internal/core/domain/candle/factory.go
package candle

import (
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/pkg/logger"
)

// CandleSystemFactory - фабрика для создания свечной системы
type CandleSystemFactory struct {
	config storage.CandleConfig
}

// NewCandleSystemFactory создает новую фабрику
func NewCandleSystemFactory() *CandleSystemFactory {
	return &CandleSystemFactory{
		config: storage.CandleConfig{
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

// CreateSystem создает свечную систему с RedisCandleStorage
func (f *CandleSystemFactory) CreateSystem(
	priceStorage storage.PriceStorageInterface,
	candleStorage storage.CandleStorageInterface,
) (*CandleSystem, error) {
	if priceStorage == nil {
		return nil, fmt.Errorf("price storage не инициализирован")
	}

	if candleStorage == nil {
		return nil, fmt.Errorf("candle storage не инициализирован")
	}

	logger.Info("🏗️ Создание свечной системы (Redis хранилище) с периодами: %v", f.config.SupportedPeriods)

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

	logger.Info("✅ Свечная система с Redis хранилищем создана успешно")
	return system, nil
}

// CandleSystem - полная свечная система
type CandleSystem struct {
	Storage      storage.CandleStorageInterface
	Engine       *CandleEngine
	Calculator   *CandleCalculator
	priceStorage storage.PriceStorageInterface
	config       storage.CandleConfig
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
			if err == nil && candle != nil && candle.IsRealFlag {
				// Сохраняем как историческую свечу
				candle.IsClosedFlag = true
				// Для RedisCandleStorage используем SaveActiveCandle
				cs.Storage.SaveActiveCandle(candle)
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
func (cs *CandleSystem) GetCandle(symbol, period string) (*redis_storage.Candle, error) {
	// Получаем свечу из Redis хранилища
	candleInterface, err := cs.Storage.GetCandle(symbol, period)
	if err != nil {
		return nil, err
	}

	// Конвертируем интерфейс в *Candle
	if candle, ok := candleInterface.(*redis_storage.Candle); ok {
		return candle, nil
	}

	// Создаем *Candle из интерфейса
	return &redis_storage.Candle{
		Symbol:       candleInterface.GetSymbol(),
		Period:       candleInterface.GetPeriod(),
		Open:         candleInterface.GetOpen(),
		High:         candleInterface.GetHigh(),
		Low:          candleInterface.GetLow(),
		Close:        candleInterface.GetClose(),
		Volume:       candleInterface.GetVolume(),
		VolumeUSD:    candleInterface.GetVolumeUSD(),
		Trades:       candleInterface.GetTrades(),
		StartTime:    candleInterface.GetStartTime(),
		EndTime:      candleInterface.GetEndTime(),
		IsClosedFlag: candleInterface.IsClosed(),
		IsRealFlag:   candleInterface.IsReal(),
	}, nil
}

// GetHistory возвращает историю свечей
func (cs *CandleSystem) GetHistory(symbol, period string, limit int) ([]*redis_storage.Candle, error) {
	historyInterfaces, err := cs.Storage.GetHistory(symbol, period, limit)
	if err != nil {
		return nil, err
	}

	// Конвертируем интерфейсы в *Candle
	candles := make([]*redis_storage.Candle, len(historyInterfaces))
	for i, candleInterface := range historyInterfaces {
		if candle, ok := candleInterface.(*redis_storage.Candle); ok {
			candles[i] = candle
		} else {
			// Создаем *Candle из интерфейса
			candles[i] = &redis_storage.Candle{
				Symbol:       candleInterface.GetSymbol(),
				Period:       candleInterface.GetPeriod(),
				Open:         candleInterface.GetOpen(),
				High:         candleInterface.GetHigh(),
				Low:          candleInterface.GetLow(),
				Close:        candleInterface.GetClose(),
				Volume:       candleInterface.GetVolume(),
				VolumeUSD:    candleInterface.GetVolumeUSD(),
				Trades:       candleInterface.GetTrades(),
				StartTime:    candleInterface.GetStartTime(),
				EndTime:      candleInterface.GetEndTime(),
				IsClosedFlag: candleInterface.IsClosed(),
				IsRealFlag:   candleInterface.IsReal(),
			}
		}
	}

	return candles, nil
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
		"storage_type":  "redis",
	}
}

// CreateSimpleSystem создает упрощенную свечную систему с Redis
func CreateSimpleSystem(
	priceStorage storage.PriceStorageInterface,
	candleStorage storage.CandleStorageInterface,
) (*CandleSystem, error) {
	factory := NewCandleSystemFactory()
	return factory.CreateSystem(priceStorage, candleStorage)
}
