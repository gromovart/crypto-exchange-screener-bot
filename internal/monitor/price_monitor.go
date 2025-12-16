// internal/monitor/price_monitor.go
package monitor

import (
	"crypto-exchange-screener-bot/internal/api"
	"crypto-exchange-screener-bot/internal/config"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

// PriceMonitor - монитор цен
type PriceMonitor struct {
	client        *api.BybitClient
	config        *config.Config
	symbols       []string
	priceHistory  map[string][]PriceData
	currentPrices map[string]float64
	intervals     map[Interval]time.Duration
	mu            sync.RWMutex
	updateTicker  *time.Ticker
	stopChan      chan bool
}

// NewPriceMonitor создает новый монитор
func NewPriceMonitor(cfg *config.Config) *PriceMonitor {
	intervals := map[Interval]time.Duration{
		Interval1Min:   1 * time.Minute,
		Interval5Min:   5 * time.Minute,
		Interval10Min:  10 * time.Minute,
		Interval15Min:  15 * time.Minute,
		Interval30Min:  30 * time.Minute,
		Interval1Hour:  1 * time.Hour,
		Interval2Hour:  2 * time.Hour,
		Interval4Hour:  4 * time.Hour,
		Interval8Hour:  8 * time.Hour,
		Interval12Hour: 12 * time.Hour,
		Interval24Hour: 24 * time.Hour,
	}

	return &PriceMonitor{
		client:        api.NewBybitClient(cfg),
		config:        cfg,
		priceHistory:  make(map[string][]PriceData),
		currentPrices: make(map[string]float64),
		intervals:     intervals,
		stopChan:      make(chan bool),
	}
}

// FetchAllUSDTPairs получает все USDT пары
func (pm *PriceMonitor) FetchAllUSDTPairs() ([]string, error) {
	// Используем API клиент
	tickerResp, err := pm.client.GetTickers("spot")
	if err != nil {
		return nil, err
	}

	var usdtPairs []string
	for _, ticker := range tickerResp.Result.List {
		symbol := ticker.Symbol
		if len(symbol) > 4 && symbol[len(symbol)-4:] == "USDT" {
			usdtPairs = append(usdtPairs, symbol)
		}
	}

	sort.Strings(usdtPairs)

	pm.mu.Lock()
	pm.symbols = usdtPairs
	for _, symbol := range usdtPairs {
		if _, exists := pm.priceHistory[symbol]; !exists {
			pm.priceHistory[symbol] = make([]PriceData, 0)
		}
	}
	pm.mu.Unlock()

	return usdtPairs, nil
}

// UpdateAllPrices обновляет текущие цены для всех пар
func (pm *PriceMonitor) UpdateAllPrices() error {
	// Используем API клиент
	tickerResp, err := pm.client.GetTickers("spot")
	if err != nil {
		return err
	}

	pm.mu.Lock()
	now := time.Now()

	for _, ticker := range tickerResp.Result.List {
		symbol := ticker.Symbol

		// Пропускаем не-USDT пары
		if len(symbol) <= 4 || symbol[len(symbol)-4:] != "USDT" {
			continue
		}

		// Парсим цену
		price, err := strconv.ParseFloat(ticker.LastPrice, 64)
		if err != nil {
			log.Printf("Failed to parse price for %s: %v", symbol, err)
			continue
		}

		// Парсим объем
		volume, _ := strconv.ParseFloat(ticker.Volume24h, 64)

		// Обновляем текущую цену
		pm.currentPrices[symbol] = price

		// Добавляем в историю
		priceData := PriceData{
			Symbol:    symbol,
			Price:     price,
			Timestamp: now,
			Volume24h: volume,
		}

		// Сохраняем в историю (ограничиваем размер)
		history := pm.priceHistory[symbol]
		history = append(history, priceData)

		// Ограничиваем историю последними 10000 записями
		if len(history) > 10000 {
			history = history[len(history)-10000:]
		}

		pm.priceHistory[symbol] = history
	}

	pm.mu.Unlock()
	return nil
}

// StartMonitoring запускает периодическое обновление цен
func (pm *PriceMonitor) StartMonitoring(updateInterval time.Duration) {
	pm.updateTicker = time.NewTicker(updateInterval)

	// Первоначальное обновление
	if err := pm.UpdateAllPrices(); err != nil {
		log.Printf("Initial price update failed: %v", err)
	}

	go func() {
		for {
			select {
			case <-pm.updateTicker.C:
				if err := pm.UpdateAllPrices(); err != nil {
					log.Printf("Price update failed: %v", err)
				} else {
					log.Printf("Prices updated at %s", time.Now().Format("15:04:05"))
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

// GetPriceChange получает изменение цены за указанный интервал
func (pm *PriceMonitor) GetPriceChange(symbol string, interval Interval) (*PriceChange, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Получаем текущую цену
	currentPrice, currentExists := pm.currentPrices[symbol]
	if !currentExists {
		return nil, fmt.Errorf("current price for %s not found", symbol)
	}

	// Получаем исторические данные из API Bybit
	changePercent, err := pm.getPriceChangeFromAPI(symbol, interval)
	if err != nil {
		return nil, err
	}

	// Рассчитываем предыдущую цену на основе изменения
	previousPrice := currentPrice / (1 + changePercent/100)

	// Получаем объем из истории, если есть
	var volume24h float64
	if history, exists := pm.priceHistory[symbol]; exists && len(history) > 0 {
		volume24h = history[len(history)-1].Volume24h
	}

	return &PriceChange{
		Symbol:        symbol,
		CurrentPrice:  currentPrice,
		PreviousPrice: previousPrice,
		Change:        currentPrice - previousPrice,
		ChangePercent: changePercent,
		Interval:      string(interval),
		Volume24h:     volume24h,
		Timestamp:     time.Now().Format(time.RFC3339),
	}, nil
}

func (pm *PriceMonitor) getPriceChangeFromAPI(symbol string, interval Interval) (float64, error) {
	// Конвертируем интервал в минуты
	intervalStr := string(interval)
	intervalMinutes, err := strconv.Atoi(intervalStr)
	if err != nil {
		return 0, fmt.Errorf("invalid interval format: %s", intervalStr)
	}

	// Получаем свечные данные из API
	klineResp, err := pm.client.GetKlineData(symbol, "spot", "1", intervalMinutes+1)
	if err != nil {
		return 0, err
	}

	if len(klineResp.Result.List) < 2 {
		return 0, fmt.Errorf("insufficient historical data for %s", symbol)
	}

	// Берем самую старую и самую новую свечу
	oldestCandle := klineResp.Result.List[0] // [timestamp, open, high, low, close, volume, turnover]
	newestCandle := klineResp.Result.List[len(klineResp.Result.List)-1]

	// Парсим цены закрытия
	oldestPrice, err := strconv.ParseFloat(oldestCandle[4], 64) // Индекс 4 = цена закрытия
	if err != nil {
		return 0, fmt.Errorf("failed to parse oldest price: %w", err)
	}

	newestPrice, err := strconv.ParseFloat(newestCandle[4], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse newest price: %w", err)
	}

	// Рассчитываем процентное изменение
	changePercent := ((newestPrice - oldestPrice) / oldestPrice) * 100

	// Округляем до 2 знаков после запятой
	return math.Round(changePercent*100) / 100, nil
}

// GetTopPerformers получает топ N монет по росту/падению
func (pm *PriceMonitor) GetTopPerformers(interval Interval, topN int, ascending bool) ([]PriceChange, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var allChanges []PriceChange

	for symbol := range pm.currentPrices {
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

// GetMarketOverview получает статистику по всем парам
func (pm *PriceMonitor) GetMarketOverview(interval Interval) (map[string]interface{}, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	totalPairs := len(pm.currentPrices)
	var risingPairs, fallingPairs int
	var totalVolume24h float64
	var maxRise, maxFall float64
	var topRiser, topFall string

	for symbol := range pm.currentPrices {
		change, err := pm.GetPriceChange(symbol, interval)
		if err != nil {
			continue
		}

		totalVolume24h += change.Volume24h

		if change.ChangePercent > 0 {
			risingPairs++
			if change.ChangePercent > maxRise {
				maxRise = change.ChangePercent
				topRiser = symbol
			}
		} else {
			fallingPairs++
			if change.ChangePercent < maxFall {
				maxFall = change.ChangePercent
				topFall = symbol
			}
		}
	}

	return map[string]interface{}{
		"total_pairs":       totalPairs,
		"rising_pairs":      risingPairs,
		"falling_pairs":     fallingPairs,
		"rising_percentage": float64(risingPairs) / float64(totalPairs) * 100,
		"total_volume_24h":  totalVolume24h,
		"max_rise":          maxRise,
		"max_fall":          maxFall,
		"top_riser":         topRiser,
		"top_fall":          topFall,
		"monitoring_since":  time.Now().Format("2006-01-02 15:04:05"),
		"interval":          string(interval),
	}, nil
}

// StartHTTPServer запускает HTTP сервер для API
func (pm *PriceMonitor) StartHTTPServer(port string) {
	http.HandleFunc("/api/prices", func(w http.ResponseWriter, r *http.Request) {
		pm.mu.RLock()
		defer pm.mu.RUnlock()

		json.NewEncoder(w).Encode(pm.currentPrices)
	})

	http.HandleFunc("/api/change", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		interval := r.URL.Query().Get("interval")

		if symbol == "" || interval == "" {
			http.Error(w, "Missing symbol or interval parameter", http.StatusBadRequest)
			return
		}

		change, err := pm.GetPriceChange(symbol, Interval(interval))
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

		top, err := pm.GetTopPerformers(Interval(interval), topN, ascending)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(top)
	})

	http.HandleFunc("/api/overview", func(w http.ResponseWriter, r *http.Request) {
		interval := r.URL.Query().Get("interval")
		if interval == "" {
			interval = string(Interval1Hour)
		}

		overview, err := pm.GetMarketOverview(Interval(interval))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(overview)
	})

	log.Printf("Starting HTTP server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("HTTP server failed:", err)
	}
}

// GetCurrentPrices возвращает текущие цены (для использования в других пакетах)
func (pm *PriceMonitor) GetCurrentPrices() map[string]float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]float64)
	for k, v := range pm.currentPrices {
		result[k] = v
	}
	return result
}

// GetSymbols возвращает список отслеживаемых символов
func (pm *PriceMonitor) GetSymbols() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.symbols
}
