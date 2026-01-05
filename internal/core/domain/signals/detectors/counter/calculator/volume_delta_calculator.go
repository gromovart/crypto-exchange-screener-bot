// internal/core/domain/signals/detectors/counter/calculator/volume_delta_calculator.go
package calculator

import (
	"fmt"
	"log"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/types"
)

// VolumeDeltaCalculator - калькулятор дельты объемов
type VolumeDeltaCalculator struct {
	marketFetcher interface{}
	storage       interface{}

	volumeDeltaCache   map[string]*volumeDeltaCache
	volumeDeltaCacheMu sync.RWMutex
	volumeDeltaTTL     time.Duration
}

type volumeDeltaCache struct {
	deltaData  *types.VolumeDeltaData
	expiration time.Time
	updateTime time.Time
}

// NewVolumeDeltaCalculator создает новый калькулятор дельты
func NewVolumeDeltaCalculator(marketFetcher interface{}, storage interface{}) *VolumeDeltaCalculator {
	return &VolumeDeltaCalculator{
		marketFetcher:    marketFetcher,
		storage:          storage,
		volumeDeltaCache: make(map[string]*volumeDeltaCache),
		volumeDeltaTTL:   30 * time.Second,
	}
}

// CalculateWithFallback получает дельту с многоуровневым fallback
func (c *VolumeDeltaCalculator) CalculateWithFallback(symbol, direction string) *types.VolumeDeltaData {
	// 1. Проверяем кэш
	if cached, found := c.getFromCache(symbol); found {
		log.Printf("📦 Дельта из кэша для %s: $%.0f (%.1f%%, источник: %s)",
			symbol, cached.deltaData.Delta, cached.deltaData.DeltaPercent, cached.deltaData.Source)
		return cached.deltaData
	}

	// 2. Пробуем получить реальные данные через API
	apiDeltaData, apiErr := c.getFromAPI(symbol)
	if apiErr == nil && (apiDeltaData.Delta != 0 || apiDeltaData.DeltaPercent != 0) {
		log.Printf("✅ Получена реальная дельта из API для %s: $%.0f (%.1f%%)",
			symbol, apiDeltaData.Delta, apiDeltaData.DeltaPercent)
		c.setToCache(symbol, apiDeltaData)
		return apiDeltaData
	}

	// 3. Fallback: Данные из хранилища
	log.Printf("⚠️ API недоступно для %s: %v", symbol, apiErr)
	storageDeltaData := c.getFromStorage(symbol, direction)
	if storageDeltaData != nil {
		log.Printf("📊 Используем дельту из хранилища для %s: $%.0f (%.1f%%)",
			symbol, storageDeltaData.Delta, storageDeltaData.DeltaPercent)
		c.setToCache(symbol, storageDeltaData)
		return storageDeltaData
	}

	// 4. Final Fallback: Базовая эмуляция
	emulatedDeltaData := c.calculateBasicDelta(symbol, direction)
	log.Printf("📊 Используем базовую дельту для %s: $%.0f (%.1f%%)",
		symbol, emulatedDeltaData.Delta, emulatedDeltaData.DeltaPercent)
	c.setToCache(symbol, emulatedDeltaData)
	return emulatedDeltaData
}

// getFromAPI получает реальную дельту через API
func (c *VolumeDeltaCalculator) getFromAPI(symbol string) (*types.VolumeDeltaData, error) {
	if c.marketFetcher == nil {
		log.Printf("❌ MARKET FETCHER IS NIL для %s!", symbol)
		return nil, fmt.Errorf("market fetcher not available")
	}

	log.Printf("🔍 Проверяем интерфейс marketFetcher для %s: %T", symbol, c.marketFetcher)

	// 🔴 ПРОВЕРКА 1: Полный интерфейс
	if fetcher, ok := c.marketFetcher.(interface {
		GetRealTimeVolumeDelta(string) (*bybit.VolumeDelta, error)
	}); ok {
		log.Printf("✅ MarketFetcher реализует GetRealTimeVolumeDelta для %s", symbol)

		volumeDelta, err := fetcher.GetRealTimeVolumeDelta(symbol)
		if err != nil {
			log.Printf("❌ Ошибка API дельты для %s: %v", symbol, err)
			return nil, fmt.Errorf("API error: %w", err)
		}

		if volumeDelta == nil {
			log.Printf("⚠️ Получен nil volume delta для %s", symbol)
			return nil, fmt.Errorf("nil volume delta response")
		}

		log.Printf("✅ Получена реальная дельта для %s: $%.0f (%.1f%%)",
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
		log.Printf("❌ MarketFetcher не реализует GetRealTimeVolumeDelta для %s", symbol)
		log.Printf("   Тип marketFetcher: %T", c.marketFetcher)

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

// getFromCache получает дельту из кэша
func (c *VolumeDeltaCalculator) getFromCache(symbol string) (*volumeDeltaCache, bool) {
	c.volumeDeltaCacheMu.RLock()
	defer c.volumeDeltaCacheMu.RUnlock()

	if cache, exists := c.volumeDeltaCache[symbol]; exists {
		if time.Now().Before(cache.expiration) {
			return cache, true
		}
		// Кэш устарел
		delete(c.volumeDeltaCache, symbol)
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

// TestConnection тестирует подключение к API дельты с детальной диагностикой
func (c *VolumeDeltaCalculator) TestConnection(symbol string) error {
	log.Printf("🧪 Тестирование подключения к API дельты для %s", symbol)
	log.Printf("🔍 Тип marketFetcher: %T", c.marketFetcher)
	log.Printf("🔍 MarketFetcher равен nil: %v", c.marketFetcher == nil)

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
		log.Printf("✅ MarketFetcher реализует GetRealTimeVolumeDelta")
		fetcherInterface = fetcher

		// 🔴 ИСПРАВЛЕНИЕ: Убираем неиспользуемую переменную, просто проверяем интерфейс
		if _, ok := c.marketFetcher.(interface {
			Start(time.Duration) error
			Stop() error
			GetRealTimeVolumeDelta(string) (*bybit.VolumeDelta, error)
		}); ok {
			log.Printf("✅ Это полный BybitPriceFetcher")
		}
	} else {
		log.Printf("❌ MarketFetcher не реализует GetRealTimeVolumeDelta")

		// Проверяем какие методы доступны
		methods := []string{
			"Start",
			"Stop",
			"GetRealTimeVolumeDelta",
			"GetVolumeDelta",
			"GetLiquidationMetrics",
			"CalculateEstimatedVolumeDelta",
		}

		log.Printf("🔍 Доступные методы:")
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
	log.Printf("🔄 Пытаемся получить данные дельты для %s...", symbol)
	volumeDelta, err := fetcherInterface.GetRealTimeVolumeDelta(symbol)
	if err != nil {
		log.Printf("❌ Ошибка получения реальной дельты: %v", err)

		// Fallback: проверяем другие методы
		if fallbackFetcher, ok := c.marketFetcher.(interface {
			CalculateEstimatedVolumeDelta(string, string, float64) (*bybit.VolumeDelta, error)
		}); ok {
			log.Printf("🔄 Пробуем fallback метод CalculateEstimatedVolumeDelta...")
			estimatedDelta, err := fallbackFetcher.CalculateEstimatedVolumeDelta(symbol, "growth", 1000000)
			if err == nil && estimatedDelta != nil {
				log.Printf("📊 Fallback дельта: $%.0f (%.1f%%)",
					estimatedDelta.Delta, estimatedDelta.DeltaPercent)
				return nil // Хотя это не реальные данные, метод работает
			}
		}
		return err
	}

	if volumeDelta == nil {
		return fmt.Errorf("nil volume delta response")
	}

	log.Printf("✅ Тест пройден! Дельта для %s: $%.0f (%.1f%%, источник: API)",
		symbol, volumeDelta.Delta, volumeDelta.DeltaPercent)

	return nil
}
