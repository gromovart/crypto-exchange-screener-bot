package monitor

import (
	"crypto-exchange-screener-bot/internal/api"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/storage"
	"crypto-exchange-screener-bot/internal/types"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BybitPriceFetcher реализация PriceFetcher для Bybit
type BybitPriceFetcher struct {
	client       *api.BybitClient
	storage      storage.PriceStorage
	mu           sync.RWMutex
	running      bool
	stopChan     chan struct{}
	lastFetch    time.Time
	fetchCount   int64
	errorCount   int64
	fetcherType  string
	updateTicker *time.Ticker
}

// NewBybitPriceFetcher создает новый fetcher для Bybit
func NewBybitPriceFetcher(cfg *config.Config, storage storage.PriceStorage) *BybitPriceFetcher {
	return &BybitPriceFetcher{
		client:      api.NewBybitClient(cfg),
		storage:     storage,
		stopChan:    make(chan struct{}),
		running:     false,
		fetcherType: "bybit",
	}
}

// FetchPrices получает текущие цены
func (f *BybitPriceFetcher) FetchPrices() ([]types.PriceData, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	tickerResp, err := f.client.GetTickers(f.client.Category())
	if err != nil {
		f.errorCount++
		return nil, err
	}

	var priceData []types.PriceData
	now := time.Now()
	updatedCount := 0

	for _, ticker := range tickerResp.Result.List {
		symbol := ticker.Symbol

		// Пропускаем не-USDT пары
		if !strings.HasSuffix(symbol, "USDT") {
			continue
		}

		// Парсим цену
		price, err := strconv.ParseFloat(ticker.LastPrice, 64)
		if err != nil {
			continue
		}

		// Парсим объем
		volume, _ := strconv.ParseFloat(ticker.Turnover24h, 64)

		// Сохраняем в хранилище
		if err := f.storage.StorePrice(symbol, price, volume, now); err != nil {
			continue
		}

		priceData = append(priceData, types.PriceData{
			Symbol:    symbol,
			Price:     price,
			Volume24h: volume,
			Timestamp: now,
		})

		updatedCount++
	}

	f.lastFetch = now
	f.fetchCount++

	log.Printf("✅ Получено %d цен (всего тикеров: %d)", updatedCount, len(tickerResp.Result.List))
	return priceData, nil
}

// StartFetching запускает периодическое получение данных
func (f *BybitPriceFetcher) StartFetching(interval time.Duration) error {
	if f.running {
		return nil
	}

	f.running = true
	f.updateTicker = time.NewTicker(interval)

	// Первоначальное получение данных
	if err := f.fetchAndLog(); err != nil {
		log.Printf("❌ Первоначальное получение данных не удалось: %v", err)
	}

	go f.fetchLoop()

	return nil
}

// fetchLoop цикл получения данных
func (f *BybitPriceFetcher) fetchLoop() {
	for {
		select {
		case <-f.updateTicker.C:
			f.fetchAndLog()
		case <-f.stopChan:
			return
		}
	}
}

// fetchAndLog получает данные и логирует результат
func (f *BybitPriceFetcher) fetchAndLog() error {
	_, err := f.FetchPrices()
	if err != nil {
		log.Printf("❌ Ошибка получения данных: %v", err)
		return err
	}
	return nil
}

// StopFetching останавливает получение данных
func (f *BybitPriceFetcher) StopFetching() error {
	if !f.running {
		return nil
	}

	f.running = false
	close(f.stopChan)
	if f.updateTicker != nil {
		f.updateTicker.Stop()
	}

	return nil
}

// IsRunning возвращает статус работы
func (f *BybitPriceFetcher) IsRunning() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.running
}

// GetLastFetchTime возвращает время последнего получения данных
func (f *BybitPriceFetcher) GetLastFetchTime() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lastFetch
}

// GetStats возвращает статистику
func (f *BybitPriceFetcher) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return map[string]interface{}{
		"running":      f.running,
		"last_fetch":   f.lastFetch,
		"fetch_count":  f.fetchCount,
		"error_count":  f.errorCount,
		"fetcher_type": f.fetcherType,
	}
}
