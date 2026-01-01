// internal/adapters/market/binance_fetcher.go
package market

import (
	binance "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/binance"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/in_memory_storage"
	events "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

// BinancePriceFetcher реализация фетчера для Binance
type BinancePriceFetcher struct {
	client   *binance.BinanceClient
	storage  storage.PriceStorage
	eventBus *events.EventBus
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewBinancePriceFetcher создает новый BinancePriceFetcher
func NewBinancePriceFetcher(client *binance.BinanceClient, storage storage.PriceStorage, eventBus *events.EventBus) *BinancePriceFetcher {
	return &BinancePriceFetcher{
		client:   client,
		storage:  storage,
		eventBus: eventBus,
		stopChan: make(chan struct{}),
		running:  false,
	}
}

func (f *BinancePriceFetcher) Start(interval time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		return fmt.Errorf("binance price fetcher already running")
	}

	f.running = true
	f.wg.Add(1)

	go func() {
		defer f.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Первоначальный запрос
		if err := f.fetchPrices(); err != nil {
			log.Printf("Binance: Ошибка первоначального получения цен: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := f.fetchPrices(); err != nil {
					log.Printf("Binance: Ошибка получения цен: %v", err)
				}
			case <-f.stopChan:
				return
			}
		}
	}()

	log.Println("✅ Binance PriceFetcher запущен")
	return nil
}

func (f *BinancePriceFetcher) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.running {
		return nil
	}

	f.running = false
	close(f.stopChan)
	f.wg.Wait()

	log.Println("🛑 Binance PriceFetcher остановлен")
	return nil
}

func (f *BinancePriceFetcher) fetchPrices() error {
	// Получаем тикеры
	tickers, err := f.client.GetTickers(f.client.Category())
	if err != nil {
		return fmt.Errorf("failed to get binance tickers: %w", err)
	}

	now := time.Now()
	updatedCount := 0

	// 🔴 СОБИРАЕМ ВСЕ ЦЕНЫ В МАССИВ
	var priceDataList []PriceData

	for _, ticker := range tickers.Result.List {
		// Парсим цену
		price, err := strconv.ParseFloat(ticker.LastPrice, 64)
		if err != nil {
			continue
		}

		// Парсим объем в базовой валюте
		volumeBase, _ := strconv.ParseFloat(ticker.Volume24h, 64)

		// Binance не предоставляет turnover, рассчитываем сами
		volumeUSD := price * volumeBase

		// 🔴 ОБНОВЛЕННЫЙ ВЫЗОВ: 4 параметра вместо 3
		if err := f.storage.StorePrice(ticker.Symbol, price, volumeBase, volumeUSD, now); err != nil {
			log.Printf("Binance: Ошибка сохранения цены для %s: %v", ticker.Symbol, err)
			continue
		}

		// Добавляем в массив для batch события
		priceDataList = append(priceDataList, PriceData{
			Symbol:    ticker.Symbol,
			Price:     price,
			Volume24h: volumeBase,
			VolumeUSD: volumeUSD, // ← ДОБАВЛЕНО!
			Timestamp: now,
		})

		updatedCount++
	}

	// 🔴 ПУБЛИКУЕМ ОДНО СОБЫТИЕ СО ВСЕМИ ЦЕНАМИ (как в Bybit)
	if updatedCount > 0 && f.eventBus != nil {
		event := events.Event{
			Type:      events.EventPriceUpdated,
			Source:    "binance_price_fetcher",
			Data:      priceDataList,
			Timestamp: now,
		}

		err := f.eventBus.Publish(event)
		if err != nil {
			log.Printf("Binance: Ошибка публикации события: %v", err)
		} else {
			log.Printf("✅ Binance: Опубликовано событие с %d ценами", updatedCount)
		}
	}

	if updatedCount > 0 {
		log.Printf("💰 Binance: Обновлено %d цен", updatedCount)
	}

	return nil
}

func (f *BinancePriceFetcher) IsRunning() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.running
}

func (f *BinancePriceFetcher) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"running":  f.running,
		"type":     "binance",
		"exchange": "binance",
	}
}
