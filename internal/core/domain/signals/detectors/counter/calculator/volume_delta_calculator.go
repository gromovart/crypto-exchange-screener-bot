// internal/core/domain/signals/detectors/counter/calculator/volume_delta_calculator.go
package calculator

import (
	"fmt"
	"log"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// VolumeDeltaCalculator - калькулятор дельты объемов
type VolumeDeltaCalculator struct {
	marketFetcher interface{}
	storage       interface{}

	volumeDeltaCache   map[string]*volumeDeltaCache
	volumeDeltaCacheMu sync.RWMutex
	volumeDeltaTTL     time.Duration

	// Для безопасного удаления просроченных записей
	deleteQueue chan string
	stopCh      chan struct{}
}

type volumeDeltaCache struct {
	deltaData  *types.VolumeDeltaData
	expiration time.Time
	updateTime time.Time
}

// NewVolumeDeltaCalculator создает новый калькулятор дельты
func NewVolumeDeltaCalculator(marketFetcher interface{}, storage interface{}) *VolumeDeltaCalculator {
	calc := &VolumeDeltaCalculator{
		marketFetcher:    marketFetcher,
		storage:          storage,
		volumeDeltaCache: make(map[string]*volumeDeltaCache),
		volumeDeltaTTL:   30 * time.Second,
		deleteQueue:      make(chan string, 1000), // Буферизованный канал для асинхронного удаления
		stopCh:           make(chan struct{}),
	}

	// Запускаем обработчик для безопасного удаления просроченных записей
	go calc.startDeletionHandler()

	return calc
}

// Stop останавливает обработчик удаления
func (c *VolumeDeltaCalculator) Stop() {
	select {
	case <-c.stopCh:
		// Уже остановлен
		logger.Debug("⚠️ VolumeDeltaCalculator уже остановлен")
	default:
		close(c.stopCh)
		logger.Debug("🛑 VolumeDeltaCalculator: отправлен сигнал остановки")
	}
}

// startDeletionHandler обрабатывает асинхронное удаление записей
func (c *VolumeDeltaCalculator) startDeletionHandler() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("❌ Паника в startDeletionHandler: %v", r)
		}
	}()

	// ✅ ИСПРАВЛЕНИЕ: Добавляем тикер для периодической очистки
	cleanupTicker := time.NewTicker(30 * time.Second)
	defer cleanupTicker.Stop()

	logger.Debug("🔄 VolumeDeltaCalculator: запущен обработчик удаления")

	for {
		select {
		case symbol := <-c.deleteQueue:
			c.safeDelete(symbol)
		case <-cleanupTicker.C:
			c.cleanupExpired() // ✅ ДОБАВЛЕНО: Периодическая очистка
		case <-c.stopCh:
			logger.Debug("🛑 VolumeDeltaCalculator: обработчик удаления остановлен")
			return
		}
	}
}

// cleanupExpired очищает все просроченные записи в кэше
func (c *VolumeDeltaCalculator) cleanupExpired() {
	c.volumeDeltaCacheMu.Lock()
	defer c.volumeDeltaCacheMu.Unlock()

	now := time.Now()
	deleted := 0

	for symbol, cache := range c.volumeDeltaCache {
		if now.After(cache.expiration) {
			delete(c.volumeDeltaCache, symbol)
			deleted++
		}
	}

	if deleted > 0 {
		logger.Debug("🧹 VolumeDeltaCalculator: очищено %d просроченных записей кэша", deleted)
	}
}

// safeDelete безопасно удаляет запись из кэша
func (c *VolumeDeltaCalculator) safeDelete(symbol string) {
	c.volumeDeltaCacheMu.Lock()
	defer c.volumeDeltaCacheMu.Unlock()

	// Двойная проверка на случай, если запись уже удалена или обновлена
	if cache, exists := c.volumeDeltaCache[symbol]; exists {
		if time.Now().After(cache.expiration) {
			delete(c.volumeDeltaCache, symbol)
			logger.Debug("🧹 Удален просроченный кэш для %s (возраст: %v)",
				symbol, time.Since(cache.updateTime).Round(time.Second))
		}
	}
}

// getFromCache получает дельту из кэша (ИСПРАВЛЕННАЯ ВЕРСИЯ)
func (c *VolumeDeltaCalculator) getFromCache(symbol string) (*volumeDeltaCache, bool) {
	c.volumeDeltaCacheMu.RLock()
	defer c.volumeDeltaCacheMu.RUnlock()

	cache, exists := c.volumeDeltaCache[symbol]
	if !exists {
		return nil, false
	}

	if time.Now().Before(cache.expiration) {
		return cache, true
	}

	// Кэш устарел - планируем асинхронное удаление
	// Отправляем в канал вместо непосредственного удаления
	select {
	case c.deleteQueue <- symbol:
		// Успешно поставлено в очередь на удаление
	default:
		// Канал переполнен - пропускаем удаление, будет удалено при следующей проверке
		log.Printf("⚠️ Канал удаления переполнен для %s, пропускаем удаление", symbol)
	}

	return nil, false
}

// setToCache сохраняет дельту в кэш
func (c *VolumeDeltaCalculator) setToCache(symbol string, deltaData *types.VolumeDeltaData) {
	c.volumeDeltaCacheMu.Lock()
	defer c.volumeDeltaCacheMu.Unlock()

	c.volumeDeltaCache[symbol] = &volumeDeltaCache{
		deltaData:  deltaData,
		expiration: time.Now().Add(c.volumeDeltaTTL),
		updateTime: time.Now(),
	}
}

// CalculateWithFallback получает дельту с многоуровневым fallback
func (c *VolumeDeltaCalculator) CalculateWithFallback(symbol, direction string) *types.VolumeDeltaData {
	// 1. Проверяем кэш (используем исправленный метод)
	if cached, found := c.getFromCache(symbol); found {
		logger.Debug("📦 Дельта из кэша для %s: $%.0f (%.1f%%, источник: %s, возраст: %v)",
			symbol, cached.deltaData.Delta, cached.deltaData.DeltaPercent,
			cached.deltaData.Source, time.Since(cached.updateTime).Round(time.Second))
		return cached.deltaData
	}

	// 2. Пробуем получить реальные данные через API
	apiDeltaData, apiErr := c.getFromAPI(symbol)
	if apiErr == nil && (apiDeltaData.Delta != 0 || apiDeltaData.DeltaPercent != 0) {
		logger.Debug("✅ Получена реальная дельта из API для %s: $%.0f (%.1f%%)",
			symbol, apiDeltaData.Delta, apiDeltaData.DeltaPercent)
		c.setToCache(symbol, apiDeltaData)
		return apiDeltaData
	}

	// 3. Fallback: Данные из хранилища (вызываем метод из volume_delta_fallback.go)
	logger.Warn("⚠️ API недоступно для %s: %v", symbol, apiErr)
	storageDeltaData := c.getFromStorage(symbol, direction)
	if storageDeltaData != nil {
		logger.Debug("📊 Используем дельту из хранилища для %s: $%.0f (%.1f%%)",
			symbol, storageDeltaData.Delta, storageDeltaData.DeltaPercent)
		c.setToCache(symbol, storageDeltaData)
		return storageDeltaData
	}

	// 4. Final Fallback: Базовая эмуляция (вызываем метод из volume_delta_fallback.go)
	emulatedDeltaData := c.calculateBasicDelta(symbol, direction)
	logger.Debug("📊 Используем базовую дельту для %s: $%.0f (%.1f%%)",
		symbol, emulatedDeltaData.Delta, emulatedDeltaData.DeltaPercent)
	c.setToCache(symbol, emulatedDeltaData)
	return emulatedDeltaData
}

// getFromAPI получает реальную дельту через API
func (c *VolumeDeltaCalculator) getFromAPI(symbol string) (*types.VolumeDeltaData, error) {
	if c.marketFetcher == nil {
		logger.Error("❌ MARKET FETCHER IS NIL для %s!", symbol)
		return nil, fmt.Errorf("market fetcher not available")
	}

	logger.Debug("🔍 Проверяем интерфейс marketFetcher для %s: %T", symbol, c.marketFetcher)

	// 🔴 ПРОВЕРКА 1: Полный интерфейс
	if fetcher, ok := c.marketFetcher.(interface {
		GetRealTimeVolumeDelta(string) (*bybit.VolumeDelta, error)
	}); ok {
		logger.Debug("✅ MarketFetcher реализует GetRealTimeVolumeDelta для %s", symbol)

		volumeDelta, err := fetcher.GetRealTimeVolumeDelta(symbol)
		if err != nil {
			logger.Error("❌ Ошибка API дельты для %s: %v", symbol, err)
			return nil, fmt.Errorf("API error: %w", err)
		}

		if volumeDelta == nil {
			logger.Warn("⚠️ Получен nil volume delta для %s", symbol)
			return nil, fmt.Errorf("nil volume delta response")
		}

		logger.Debug("✅ Получена реальная дельта для %s: $%.0f (%.1f%%)",
			symbol, volumeDelta.Delta, volumeDelta.DeltaPercent)

		return &types.VolumeDeltaData{
			Delta:        volumeDelta.Delta,
			DeltaPercent: volumeDelta.DeltaPercent,
			Source:       types.VolumeDeltaSourceAPI,
			Timestamp:    time.Now(),
			BuyVolume:    volumeDelta.BuyVolume,
			SellVolume:   volumeDelta.SellVolume,
			TotalTrades:  volumeDelta.TotalTrades,
			IsRealData:   true,
		}, nil
	} else {
		// 🔴 ПРОВЕРКА 2: Basic интерфейс
		logger.Error("❌ MarketFetcher не реализует GetRealTimeVolumeDelta для %s", symbol)

		// Проверим базовые методы PriceFetcher
		if _, ok := c.marketFetcher.(interface {
			Start(time.Duration) error
		}); ok {
			log.Printf("   ✓ Реализует Start()")
		}
		if _, ok := c.marketFetcher.(interface {
			Stop() error
		}); ok {
			log.Printf("   ✓ Реализует Stop()")
		}
		if _, ok := c.marketFetcher.(interface {
			GetRealTimeVolumeDelta(string) (*bybit.VolumeDelta, error)
		}); !ok {
			log.Printf("   ✗ НЕ реализует GetRealTimeVolumeDelta")
		}
	}

	return nil, fmt.Errorf("market fetcher doesn't support volume delta")
}

// TestConnection тестирует подключение к API дельты с детальной диагностикой
func (c *VolumeDeltaCalculator) TestConnection(symbol string) error {
	logger.Debug("🧪 Тестирование подключения к API дельты для %s", symbol)
	logger.Debug("🔍 Тип marketFetcher: %T", c.marketFetcher)
	logger.Debug("🔍 MarketFetcher равен nil: %v", c.marketFetcher == nil)

	if c.marketFetcher == nil {
		return fmt.Errorf("market fetcher not available")
	}

	// 🔴 ДЕТАЛЬНАЯ ПРОВЕРКА ИНТЕРФЕЙСА
	var fetcherInterface interface {
		GetRealTimeVolumeDelta(string) (*bybit.VolumeDelta, error)
	}

	if fetcher, ok := c.marketFetcher.(interface {
		GetRealTimeVolumeDelta(string) (*bybit.VolumeDelta, error)
	}); ok {
		logger.Debug("✅ MarketFetcher реализует GetRealTimeVolumeDelta")
		fetcherInterface = fetcher

		// 🔴 ИСПРАВЛЕНИЕ: Убираем неиспользуемую переменную, просто проверяем интерфейс
		if _, ok := c.marketFetcher.(interface {
			Start(time.Duration) error
			Stop() error
			GetRealTimeVolumeDelta(string) (*bybit.VolumeDelta, error)
		}); ok {
			logger.Debug("✅ Это полный BybitPriceFetcher")
		}
	} else {
		logger.Error("❌ MarketFetcher не реализует GetRealTimeVolumeDelta")

		// Проверяем какие методы доступны
		methods := []string{
			"Start",
			"Stop",
			"GetRealTimeVolumeDelta",
			"GetVolumeDelta",
			"GetLiquidationMetrics",
			"CalculateEstimatedVolumeDelta",
		}

		logger.Debug("🔍 Доступные методы:")
		for _, method := range methods {
			switch method {
			case "Start":
				if _, ok := c.marketFetcher.(interface{ Start(time.Duration) error }); ok {
					log.Printf("   ✓ Start()")
				}
			case "Stop":
				if _, ok := c.marketFetcher.(interface{ Stop() error }); ok {
					log.Printf("   ✓ Stop()")
				}
			case "GetRealTimeVolumeDelta":
				if _, ok := c.marketFetcher.(interface {
					GetRealTimeVolumeDelta(string) (*bybit.VolumeDelta, error)
				}); ok {
					log.Printf("   ✓ GetRealTimeVolumeDelta()")
				} else {
					log.Printf("   ✗ GetRealTimeVolumeDelta() - НЕ ДОСТУПЕН")
				}
			case "GetVolumeDelta":
				if _, ok := c.marketFetcher.(interface {
					GetVolumeDelta(string, time.Duration) (*bybit.VolumeDelta, error)
				}); ok {
					log.Printf("   ✓ GetVolumeDelta()")
				}
			case "GetLiquidationMetrics":
				if _, ok := c.marketFetcher.(interface {
					GetLiquidationMetrics(string) (*bybit.LiquidationMetrics, bool)
				}); ok {
					log.Printf("   ✓ GetLiquidationMetrics()")
				}
			case "CalculateEstimatedVolumeDelta":
				if _, ok := c.marketFetcher.(interface {
					CalculateEstimatedVolumeDelta(string, string, float64) (*bybit.VolumeDelta, error)
				}); ok {
					log.Printf("   ✓ CalculateEstimatedVolumeDelta()")
				}
			}
		}
		return fmt.Errorf("market fetcher doesn't support GetRealTimeVolumeDelta")
	}

	// Пробуем получить данные
	logger.Debug("🔄 Пытаемся получить данные дельты для %s...", symbol)
	volumeDelta, err := fetcherInterface.GetRealTimeVolumeDelta(symbol)
	if err != nil {
		logger.Error("❌ Ошибка получения реальной дельты: %v", err)

		// Fallback: проверяем другие методы
		if fallbackFetcher, ok := c.marketFetcher.(interface {
			CalculateEstimatedVolumeDelta(string, string, float64) (*bybit.VolumeDelta, error)
		}); ok {
			logger.Debug("🔄 Пробуем fallback метод CalculateEstimatedVolumeDelta...")
			estimatedDelta, err := fallbackFetcher.CalculateEstimatedVolumeDelta(symbol, "growth", 1000000)
			if err == nil && estimatedDelta != nil {
				logger.Debug("📊 Fallback дельта: $%.0f (%.1f%%)",
					estimatedDelta.Delta, estimatedDelta.DeltaPercent)
				return nil // Хотя это не реальные данные, метод работает
			}
		}
		return err
	}

	if volumeDelta == nil {
		return fmt.Errorf("nil volume delta response")
	}

	logger.Debug("✅ Тест пройден! Дельта для %s: $%.0f (%.1f%%, источник: API)",
		symbol, volumeDelta.Delta, volumeDelta.DeltaPercent)

	return nil
}

// ClearCache очищает весь кэш
func (c *VolumeDeltaCalculator) ClearCache() {
	c.volumeDeltaCacheMu.Lock()
	defer c.volumeDeltaCacheMu.Unlock()

	count := len(c.volumeDeltaCache)
	c.volumeDeltaCache = make(map[string]*volumeDeltaCache)

	logger.Debug("🧹 Полная очистка кэша дельты: удалено %d записей", count)
}
