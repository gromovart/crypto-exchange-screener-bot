package monitor

import (
	"crypto-exchange-screener-bot/internal/api"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/storage"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PriceMonitor - монитор цен (только получение данных)
type PriceMonitor struct {
	client       *api.BybitClient
	config       *config.Config
	storage      storage.PriceStorage
	updateTicker *time.Ticker
	stopChan     chan bool
	mu           sync.RWMutex
	lastUpdate   time.Time
}

// NewPriceMonitor создает новый монитор цен
func NewPriceMonitor(cfg *config.Config, storage storage.PriceStorage) *PriceMonitor {
	return &PriceMonitor{
		client:     api.NewBybitClient(cfg),
		config:     cfg,
		storage:    storage,
		stopChan:   make(chan bool),
		lastUpdate: time.Now(),
	}
}

// FetchAndStorePrices получает и сохраняет текущие цены
func (pm *PriceMonitor) FetchAndStorePrices() error {
	// Используем API клиент с категорией фьючерсов
	tickerResp, err := pm.client.GetTickers(pm.client.Category())
	if err != nil {
		log.Printf("❌ Ошибка получения тикеров фьючерсов: %v", err)
		return err
	}

	log.Printf("📥 Получено %d тикеров фьючерсов от API", len(tickerResp.Result.List))

	pm.mu.Lock()
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
			log.Printf("⚠️ Ошибка парсинга цены для %s: %v", symbol, err)
			continue
		}

		// Парсим объем (в USDT для фьючерсов)
		volume, _ := strconv.ParseFloat(ticker.Turnover24h, 64)

		// Сохраняем в хранилище
		if err := pm.storage.StorePrice(symbol, price, volume, now); err != nil {
			log.Printf("⚠️ Ошибка сохранения цены %s: %v", symbol, err)
			continue
		}

		updatedCount++
	}

	pm.lastUpdate = now
	pm.mu.Unlock()

	log.Printf("✅ Сохранено %d цен фьючерсов в %s", updatedCount, now.Format("15:04:05"))
	return nil
}

// GetAllFuturesPairs получает все фьючерсные пары с фильтрацией
func (pm *PriceMonitor) GetAllFuturesPairs(minVolume float64, maxPairs int, sortByVolume bool) ([]string, error) {
	// Используем API клиент для получения тикеров
	tickerResp, err := pm.client.GetTickers(pm.client.Category())
	if err != nil {
		return nil, err
	}

	type SymbolVolume struct {
		Symbol string
		Volume float64
	}

	var symbolsWithVolume []SymbolVolume

	// Собираем все USDT фьючерсы с объемом
	for _, ticker := range tickerResp.Result.List {
		symbol := ticker.Symbol

		// Фильтруем только USDT пары
		if !strings.HasSuffix(symbol, "USDT") {
			continue
		}

		// Парсим объем
		volume, err := strconv.ParseFloat(ticker.Turnover24h, 64)
		if err != nil {
			volume = 0
		}

		// Фильтруем по минимальному объему
		if volume >= minVolume {
			symbolsWithVolume = append(symbolsWithVolume, SymbolVolume{
				Symbol: symbol,
				Volume: volume,
			})
		}
	}

	// Сортируем по объему если нужно
	if sortByVolume {
		sort.Slice(symbolsWithVolume, func(i, j int) bool {
			return symbolsWithVolume[i].Volume > symbolsWithVolume[j].Volume
		})
	} else {
		// Или сортируем по алфавиту
		sort.Slice(symbolsWithVolume, func(i, j int) bool {
			return symbolsWithVolume[i].Symbol < symbolsWithVolume[j].Symbol
		})
	}

	// Ограничиваем количество пар если указано
	if maxPairs > 0 && len(symbolsWithVolume) > maxPairs {
		symbolsWithVolume = symbolsWithVolume[:maxPairs]
	}

	// Извлекаем только символы
	symbols := make([]string, len(symbolsWithVolume))
	for i, sv := range symbolsWithVolume {
		symbols[i] = sv.Symbol
	}

	return symbols, nil
}

// StartMonitoring запускает периодическое обновление цен
func (pm *PriceMonitor) StartMonitoring(updateInterval time.Duration) {
	pm.updateTicker = time.NewTicker(updateInterval)

	// Первоначальное обновление
	if err := pm.FetchAndStorePrices(); err != nil {
		log.Printf("Первоначальное обновление цен не удалось: %v", err)
	}

	go func() {
		for {
			select {
			case <-pm.updateTicker.C:
				if err := pm.FetchAndStorePrices(); err != nil {
					log.Printf("Обновление цен не удалось: %v", err)
				} else {
					log.Printf("Цены обновлены в %s", time.Now().Format("15:04:05"))
				}
			case <-pm.stopChan:
				if pm.updateTicker != nil {
					pm.updateTicker.Stop()
				}
				return
			}
		}
	}()
}

// StopMonitoring останавливает мониторинг
func (pm *PriceMonitor) StopMonitoring() {
	if pm.stopChan != nil {
		close(pm.stopChan)
	}
}

// GetCurrentPrices возвращает текущие цены из хранилища
func (pm *PriceMonitor) GetCurrentPrices() map[string]float64 {
	snapshots := pm.storage.GetAllCurrentPrices()
	result := make(map[string]float64, len(snapshots))

	for symbol, snapshot := range snapshots {
		result[symbol] = snapshot.Price
	}

	return result
}

// GetSymbols возвращает список отслеживаемых символов из хранилища
func (pm *PriceMonitor) GetSymbols() []string {
	return pm.storage.GetSymbols()
}

// GetPriceChange рассчитывает изменение цены за интервал
func (pm *PriceMonitor) GetPriceChange(symbol string, interval string) (*PriceChange, error) {
	// Конвертируем интервал в time.Duration
	var duration time.Duration
	switch interval {
	case "1": // 1 минута
		duration = time.Minute
	case "5":
		duration = 5 * time.Minute
	case "15":
		duration = 15 * time.Minute
	case "30":
		duration = 30 * time.Minute
	case "60": // 1 час
		duration = time.Hour
	case "240": // 4 часа
		duration = 4 * time.Hour
	case "1440": // 1 день
		duration = 24 * time.Hour
	default:
		return nil, fmt.Errorf("unsupported interval: %s", interval)
	}

	// Используем хранилище для расчета изменения
	change, err := pm.storage.CalculatePriceChange(symbol, duration)
	if err != nil {
		return nil, err
	}

	// Конвертируем в наш формат
	return &PriceChange{
		Symbol:        change.Symbol,
		CurrentPrice:  change.CurrentPrice,
		PreviousPrice: change.PreviousPrice,
		Change:        change.Change,
		ChangePercent: change.ChangePercent,
		Interval:      interval,
		Timestamp:     change.Timestamp,
	}, nil
}

// GetTopPerformers получает топ N монет по росту/падению
func (pm *PriceMonitor) GetTopPerformers(interval string, topN int, ascending bool) ([]PriceChange, error) {
	// Получаем все символы из хранилища
	symbols := pm.storage.GetSymbols()

	var allChanges []PriceChange

	for _, symbol := range symbols {
		change, err := pm.GetPriceChange(symbol, interval)
		if err != nil {
			continue // Пропускаем пары с недостаточными данными
		}

		allChanges = append(allChanges, *change)
	}

	// Сортируем по проценту изменения
	if ascending {
		// По возрастанию (самое большое падение сначала)
		sort.Slice(allChanges, func(i, j int) bool {
			return allChanges[i].ChangePercent < allChanges[j].ChangePercent
		})
	} else {
		// По убыванию (самый большой рост сначала)
		sort.Slice(allChanges, func(i, j int) bool {
			return allChanges[i].ChangePercent > allChanges[j].ChangePercent
		})
	}

	// Ограничиваем количество
	if topN > len(allChanges) {
		topN = len(allChanges)
	}

	return allChanges[:topN], nil
}

// StartHTTPServer запускает HTTP сервер для API
func (pm *PriceMonitor) StartHTTPServer(port string) {
	http.HandleFunc("/api/prices", func(w http.ResponseWriter, r *http.Request) {
		prices := pm.GetCurrentPrices()
		json.NewEncoder(w).Encode(prices)
	})

	http.HandleFunc("/api/change", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		interval := r.URL.Query().Get("interval")

		if symbol == "" || interval == "" {
			http.Error(w, "Missing symbol or interval parameter", http.StatusBadRequest)
			return
		}

		change, err := pm.GetPriceChange(symbol, interval)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(change)
	})

	http.HandleFunc("/api/top", func(w http.ResponseWriter, r *http.Request) {
		interval := r.URL.Query().Get("interval")
		topN, _ := strconv.Atoi(r.URL.Query().Get("n"))
		order := r.URL.Query().Get("order")

		if topN <= 0 {
			topN = 10
		}

		ascending := order == "asc"

		top, err := pm.GetTopPerformers(interval, topN, ascending)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(top)
	})

	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := pm.storage.GetStats()
		json.NewEncoder(w).Encode(stats)
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("HTTP сервер не запустился:", err)
	}
}

// GetClient возвращает API клиент
func (pm *PriceMonitor) GetClient() *api.BybitClient {
	return pm.client
}

// Config возвращает конфигурацию
func (pm *PriceMonitor) Config() *config.Config {
	return pm.config
}

// GetLastUpdate возвращает время последнего обновления
func (pm *PriceMonitor) GetLastUpdate() time.Time {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.lastUpdate
}
