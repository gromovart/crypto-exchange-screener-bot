// internal/fetcher/bybit_fetcher.go
package fetcher

import (
	"crypto-exchange-screener-bot/internal/api/bybit"
	"crypto-exchange-screener-bot/internal/config"
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
	// Получаем тикеры
	tickers, err := f.client.GetTickers(f.client.Category())
	if err != nil {
		return fmt.Errorf("failed to get tickers: %w", err)
	}

	now := time.Now()
	updatedCount := 0

	// 🔴 СОБИРАЕМ ВСЕ ЦЕНЫ В МАССИВ
	var priceDataList []PriceData

	for _, ticker := range tickers.Result.List {
		// Парсим цену
		price, err := parseFloat(ticker.LastPrice)
		if err != nil {
			continue
		}

		// Парсим объем
		volume, _ := parseFloat(ticker.Volume24h)

		// Сохраняем в хранилище
		if err := f.storage.StorePrice(ticker.Symbol, price, volume, now); err != nil {
			logger.Info("Ошибка сохранения цены для %s: %v", ticker.Symbol, err)
			continue
		}

		// Добавляем в массив
		priceDataList = append(priceDataList, PriceData{
			Symbol:    ticker.Symbol,
			Price:     price,
			Volume24h: volume,
			Timestamp: now,
		})

		updatedCount++
	}

	// 🔴 ПУБЛИКУЕМ ОДНО СОБЫТИЕ СО ВСЕМИ ЦЕНАМИ
	if updatedCount > 0 && f.eventBus != nil {
		event := events.Event{
			Type:      events.EventPriceUpdated,
			Source:    "price_fetcher",
			Data:      priceDataList, // ← МАССИВ ВСЕХ ЦЕН
			Timestamp: now,
		}

		err := f.eventBus.Publish(event)
		if err != nil {
			logger.Info("Ошибка публикации события: %v", err)
		} else {
			logger.Info("✅ Опубликовано событие с %d ценами", updatedCount)
		}
	}

	if updatedCount > 0 {
		logger.Info("💰 Обновлено %d цен", updatedCount)
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
