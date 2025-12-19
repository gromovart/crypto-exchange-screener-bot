// internal/fetcher/price_fetcher.go - правильная реализация
package fetcher

import (
	"crypto-exchange-screener-bot/internal/api/bybit"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/events"
	"crypto-exchange-screener-bot/internal/storage"
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
			log.Printf("Ошибка первоначального получения цен: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := f.fetchPrices(); err != nil {
					log.Printf("Ошибка получения цен: %v", err)
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
			log.Printf("Ошибка сохранения цены для %s: %v", ticker.Symbol, err)
			continue
		}

		// Публикуем событие
		f.eventBus.Publish(events.Event{
			Type:   events.EventPriceUpdated,
			Source: "price_fetcher",
			Data: map[string]interface{}{
				"symbol":    ticker.Symbol,
				"price":     price,
				"volume":    volume,
				"timestamp": now,
			},
			Timestamp: now,
		})

		updatedCount++
	}

	if updatedCount > 0 {
		log.Printf("💰 Обновлено %d цен", updatedCount)
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
