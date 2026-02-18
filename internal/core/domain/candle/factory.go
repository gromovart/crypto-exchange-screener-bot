// internal/core/domain/candle/factory.go
package candle

import (
	"fmt"
	"time"

	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	candletracker "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/candle_tracker"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
)

// CandleSystemFactory - фабрика для создания свечной системы
type CandleSystemFactory struct {
	config storage.CandleConfig
}

// CandleSystem - полная свечная система
type CandleSystem struct {
	Storage       storage.CandleStorageInterface
	Engine        *CandleEngine
	Calculator    *CandleCalculator
	candleTracker *candletracker.CandleTracker
	priceStorage  storage.PriceStorageInterface
	config        storage.CandleConfig
	eventBus      *events.EventBus
}

// NewCandleSystemFactory создает новую фабрику
func NewCandleSystemFactory() *CandleSystemFactory {
	return &CandleSystemFactory{
		config: storage.CandleConfig{
			// ✅ ДОБАВЛЯЕМ ПЕРИОД 1m
			SupportedPeriods: []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d"},
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
	eventBus *events.EventBus, // НОВЫЙ параметр: EventBus
) (*CandleSystem, error) {
	if priceStorage == nil {
		return nil, fmt.Errorf("price storage не инициализирован")
	}

	if candleStorage == nil {
		return nil, fmt.Errorf("candle storage не инициализирован")
	}

	logger.Info("🏗️ Создание свечной системы (Redis хранилище) с периодами: %v", f.config.SupportedPeriods)

	// Создаем движок с передачей EventBus
	candleEngine := NewCandleEngine(candleStorage, f.config, eventBus)

	// Создаем калькулятор
	candleCalculator := NewCandleCalculator(priceStorage)

	// Создаем систему
	system := &CandleSystem{
		Storage:      candleStorage,
		Engine:       candleEngine,
		Calculator:   candleCalculator,
		priceStorage: priceStorage,
		config:       f.config,
		eventBus:     eventBus,
	}

	logger.Info("✅ Свечная система с Redis хранилищем создана успешно")
	return system, nil
}

// CreateSystemWithRedis создает свечную систему с RedisService для CandleTracker
func (f *CandleSystemFactory) CreateSystemWithRedis(
	priceStorage storage.PriceStorageInterface,
	candleStorage storage.CandleStorageInterface,
	redisService *redis_service.RedisService,
	eventBus *events.EventBus, // ДОБАВЛЯЕМ параметр
) (*CandleSystem, error) {
	system, err := f.CreateSystem(priceStorage, candleStorage, eventBus)
	if err != nil {
		return nil, err
	}

	// Создаем CandleTracker если есть RedisService
	if redisService != nil {
		tracker := candletracker.NewCandleTracker(redisService, 2*time.Hour)
		if err := tracker.Initialize(); err != nil {
			logger.Warn("⚠️ Не удалось инициализировать CandleTracker: %v", err)
			// Не прерываем создание системы
		} else {
			system.SetCandleTracker(tracker)
			logger.Info("✅ CandleTracker добавлен в CandleSystem (TTL: 2 часа)")
		}
	} else {
		logger.Warn("⚠️ RedisService не передан, CandleTracker не будет создан")
	}

	return system, nil
}

// SetCandleTracker устанавливает трекер свечей
func (cs *CandleSystem) SetCandleTracker(tracker *candletracker.CandleTracker) {
	cs.candleTracker = tracker
	logger.Info("✅ CandleTracker установлен в CandleSystem")
}

// GetCandleTracker возвращает трекер свечей
func (cs *CandleSystem) GetCandleTracker() *candletracker.CandleTracker {
	return cs.candleTracker
}

// HasCandleTracker проверяет есть ли трекер свечей
func (cs *CandleSystem) HasCandleTracker() bool {
	return cs.candleTracker != nil
}

// Start запускает свечную систему
func (cs *CandleSystem) Start() error {
	logger.Info("🚀 Запуск свечной системы...")

	// Устанавливаем EventBus в Engine если он еще не установлен
	if cs.Engine != nil && cs.eventBus != nil {
		// CandleEngine уже получает eventBus через конструктор,
		// но дополнительно убеждаемся что подписка настроена
		logger.Debug("🔄 CandleSystem: EventBus настроен для Engine")
	}

	// Запускаем движок
	if err := cs.Engine.Start(); err != nil {
		return err
	}

	// Предзагружаем свечи для существующих символов
	cs.preloadCandles()

	logger.Info("✅ Свечная система запущена (трекер свечей: %v)", cs.HasCandleTracker())
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

// GetCandle получает свечу для символа и периода
func (cs *CandleSystem) GetCandle(symbol, period string) (*storage.Candle, error) {
	// Получаем свечу из Redis хранилища
	candleInterface, err := cs.Storage.GetCandle(symbol, period)
	if err != nil {
		return nil, err
	}

	// Конвертируем интерфейс в *Candle
	if candle, ok := candleInterface.(*storage.Candle); ok {
		return candle, nil
	}

	// Создаем *Candle из интерфейса
	return &storage.Candle{
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

// MarkCandleProcessedAtomically атомарно помечает свечу как обработанную
func (cs *CandleSystem) MarkCandleProcessedAtomically(symbol, period string, startTime int64) (bool, error) {
	if cs.candleTracker == nil {
		return false, fmt.Errorf("candle tracker не инициализирован")
	}
	return cs.candleTracker.MarkCandleProcessedAtomically(symbol, period, startTime)
}

// IsCandleProcessed проверяет была ли свеча обработана
func (cs *CandleSystem) IsCandleProcessed(symbol, period string, startTime int64) (bool, error) {
	if cs.candleTracker == nil {
		return false, fmt.Errorf("candle tracker не инициализирован")
	}
	return cs.candleTracker.IsCandleProcessed(symbol, period, startTime)
}

// GetLatestClosedCandle получает последнюю закрытую свечу с проверкой трекера
func (cs *CandleSystem) GetLatestClosedCandle(symbol, period string) (*storage.Candle, error) {
	// Получаем историю (последние 5 свечей)
	history, err := cs.GetHistory(symbol, period, 10) // Увеличиваем лимит для надежности
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, nil
	}

	// Идем от новых к старым свечам
	for i := len(history) - 1; i >= 0; i-- {
		candle := history[i]

		// Проверяем что свеча закрыта
		if !candle.IsClosedFlag {
			continue
		}

		// Проверяем что свеча реальная
		if !candle.IsRealFlag || candle.Open == 0 {
			continue
		}

		// ⭐ КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: проверяем через трекер если он доступен
		if cs.candleTracker != nil {
			processed, err := cs.IsCandleProcessed(symbol, period, candle.StartTime.Unix())
			if err != nil {
				// logger.Warn("⚠️ Ошибка проверки свечи %s/%s через трекер (начало: %s): %v",
				// 	symbol, period, candle.StartTime.Format("15:04:05"), err)
				// Продолжаем, но возвращаем свечу (может быть дублирование)
			} else if processed {
				// logger.Debug("⏭️ Свеча %s/%s уже обработана (начало: %s, изменение: %.2f%%)",
				// 	symbol, period, candle.StartTime.Format("15:04:05"),
				// 	((candle.Close-candle.Open)/candle.Open)*100)
				continue // Пропускаем уже обработанные свечи
			}
		}
		//Расскомментировать для отладки
		// Нашли подходящую свечу
		// logger.Debug("🔍 Найдена необработанная закрытая свеча %s/%s (начало: %s, изменение: %.2f%%)",
		// 	symbol, period, candle.StartTime.Format("15:04:05"),
		// 	((candle.Close-candle.Open)/candle.Open)*100)
		return candle, nil
	}
	//Раскомментировать для отладки
	// Если все свечи уже обработаны или нет подходящих
	// logger.Debug("📭 Все закрытые свечи %s/%s уже обработаны или нет подходящих", symbol, period)
	return nil, nil
}

// GetCandleOrLatestClosed получает свечу (активную или последнюю закрытую)
func (cs *CandleSystem) GetCandleOrLatestClosed(symbol, period string) (*storage.Candle, error) {
	// Сначала пробуем получить активную свечу
	candle, err := cs.GetCandle(symbol, period)
	if err != nil {
		return nil, err
	}

	// Если активная свеча есть и она закрыта - возвращаем её
	if candle != nil && candle.IsClosedFlag {
		return candle, nil
	}

	// Если активная свеча не закрыта или её нет, ищем последнюю закрытую
	return cs.GetLatestClosedCandle(symbol, period)
}

// GetHistory возвращает историю свечей
func (cs *CandleSystem) GetHistory(symbol, period string, limit int) ([]*storage.Candle, error) {
	historyInterfaces, err := cs.Storage.GetHistory(symbol, period, limit)
	if err != nil {
		return nil, err
	}

	// Конвертируем интерфейсы в *Candle
	candles := make([]*storage.Candle, len(historyInterfaces))
	for i, candleInterface := range historyInterfaces {
		if candle, ok := candleInterface.(*storage.Candle); ok {
			candles[i] = candle
		} else {
			// Создаем *Candle из интерфейса
			candles[i] = &storage.Candle{
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

	// Получаем статистику трекера если есть
	var trackerStats map[string]interface{}
	if cs.candleTracker != nil {
		stats, err := cs.candleTracker.GetStats()
		if err == nil {
			trackerStats = stats
		}
	}

	return map[string]interface{}{
		"system_config": map[string]interface{}{
			"supported_periods":  cs.config.SupportedPeriods,
			"max_history":        cs.config.MaxHistory,
			"cleanup_interval":   cs.config.CleanupInterval.String(),
			"auto_build":         cs.config.AutoBuild,
			"has_candle_tracker": cs.HasCandleTracker(),
		},
		"engine_stats":   engineStats,
		"storage_stats":  storageStats,
		"candle_tracker": trackerStats, // Статистика трекера
		"storage_type":   "redis",
	}
}

// CreateSimpleSystem создает упрощенную свечную систему с Redis
func CreateSimpleSystem(
	priceStorage storage.PriceStorageInterface,
	candleStorage storage.CandleStorageInterface,
	eventBus *events.EventBus, // НОВЫЙ параметр: EventBus
) (*CandleSystem, error) {
	factory := NewCandleSystemFactory()
	return factory.CreateSystem(priceStorage, candleStorage, eventBus)
}
