// internal/adapters/market/bybit_fetcher.go
package market

import (
	bybit "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
	"sync"
	"time"
)

// BybitPriceFetcher реализация
type BybitPriceFetcher struct {
	client   *bybit.BybitClient
	storage  storage.PriceStorage
	eventBus *events.EventBus
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
	config   *config.Config
}

// NewPriceFetcher создает новый PriceFetcher
func NewPriceFetcher(apiClient *bybit.BybitClient, storage storage.PriceStorage, eventBus *events.EventBus) *BybitPriceFetcher {
	return &BybitPriceFetcher{
		client:   apiClient,
		storage:  storage,
		eventBus: eventBus,
		stopChan: make(chan struct{}),
		running:  false,
	}
}

func (f *BybitPriceFetcher) Start(interval time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return fmt.Errorf("price fetcher already running")
	}

	f.running = true
	f.wg.Add(1)

	go func() {
		defer f.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Первоначальный запрос
		if err := f.fetchPrices(); err != nil {
			logger.Info("Ошибка первоначального получения цен: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := f.fetchPrices(); err != nil {
					logger.Info("Ошибка получения цен: %v", err)
				}
			case <-f.stopChan:
				return
			}
		}
	}()

	log.Println("✅ PriceFetcher запущен")
	return nil
}

func (f *BybitPriceFetcher) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return nil
	}

	f.running = false
	close(f.stopChan)
	f.wg.Wait()

	log.Println("🛑 PriceFetcher остановлен")
	return nil
}

func (f *BybitPriceFetcher) fetchPrices() error {
	// 🔴 ДОБАВЛЯЕМ ЛОГИРОВАНИЕ
	logger.Info("🔄 BybitFetcher: начало получения цен...")

	// Получаем тикеры
	tickers, err := f.client.GetTickers(f.client.Category())
	if err != nil {
		logger.Error("❌ BybitFetcher: ошибка получения тикеров: %v", err)
		return fmt.Errorf("failed to get tickers: %w", err)
	}

	logger.Info("📊 BybitFetcher: получено %d тикеров", len(tickers.Result.List))

	now := time.Now()
	updatedCount := 0
	errorCount := 0

	// 🔴 СОБИРАЕМ ВСЕ ЦЕНЫ В МАССИВ
	var priceDataList []PriceData

	for i, ticker := range tickers.Result.List {
		// Парсим цену
		price, err := parseFloat(ticker.LastPrice)
		if err != nil {
			logger.Debug("⚠️  BybitFetcher: ошибка парсинга цены для %s: %v", ticker.Symbol, err)
			continue
		}

		// Парсим объем в базовой валюте
		volumeBase, _ := parseFloat(ticker.Volume24h)

		// Парсим объем в USDT (turnover) - ОСНОВНОЙ ДЛЯ АНАЛИЗА
		volumeUSD, _ := parseFloat(ticker.Turnover24h)

		// Если turnover недоступен, используем расчетный объем
		if volumeUSD == 0 && price > 0 && volumeBase > 0 {
			volumeUSD = price * volumeBase
			logger.Debug("📝 BybitFetcher: расчетный VolumeUSD для %s: %f", ticker.Symbol, volumeUSD)
		}

		// 🔴 ДОБАВЛЯЕМ ДЕБАГ ЛОГ
		logger.Debug("💰 BybitFetcher: сохранение %s: price=%f, volume24h=%f, volumeUSD=%f",
			ticker.Symbol, price, volumeBase, volumeUSD)

		// 🔴 ОБНОВЛЕННЫЙ ВЫЗОВ: 4 параметра вместо 3
		if err := f.storage.StorePrice(ticker.Symbol, price, volumeBase, volumeUSD, now); err != nil {
			errorCount++
			logger.Error("❌ BybitFetcher: ошибка StorePrice для %s: %v", ticker.Symbol, err)
			continue
		}

		// Добавляем в массив
		priceDataList = append(priceDataList, PriceData{
			Symbol:    ticker.Symbol,
			Price:     price,
			Volume24h: volumeBase,
			VolumeUSD: volumeUSD, // ← ДОБАВЛЕНО!
			Timestamp: now,
		})

		updatedCount++

		// Логируем каждый 50-й тикер
		if (i+1)%50 == 0 {
			logger.Debug("📈 BybitFetcher: обработано %d тикеров...", i+1)
		}
	}

	logger.Info("✅ BybitFetcher: успешно сохранено %d цен, ошибок: %d", updatedCount, errorCount)

	// 🔴 ПУБЛИКУЕМ ОДНО СОБЫТИЕ СО ВСЕМИ ЦЕНАМИ
	if updatedCount > 0 && f.eventBus != nil {
		event := events.Event{
			Type:      events.EventPriceUpdated,
			Source:    "bybit_price_fetcher",
			Data:      priceDataList,
			Timestamp: now,
		}

		err := f.eventBus.Publish(event)
		if err != nil {
			logger.Error("❌ BybitFetcher: ошибка публикации события: %v", err)
		} else {
			logger.Info("📨 BybitFetcher: опубликовано событие с %d ценами", updatedCount)
		}
	}

	return nil
}

func (f *BybitPriceFetcher) IsRunning() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.running
}

func (f *BybitPriceFetcher) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"running": f.running,
		"type":    "bybit",
	}
}

// Вспомогательная функция для парсинга
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}
