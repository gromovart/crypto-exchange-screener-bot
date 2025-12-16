// internal/monitor/price_monitor.go
package monitor

import (
	// "crypto-exchange-screener-bot/internal/api"
	api "crypto-exchange-screener-bot/internal/api"
	"crypto-exchange-screener-bot/internal/config"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
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
	signalMonitor *SignalMonitor // Новое поле
	cronScheduler *cron.Cron     // Для запуска по расписанию
	symbolInfo    map[string]api.InstrumentInfo
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

	pm := &PriceMonitor{
		client:        api.NewBybitClient(cfg),
		config:        cfg,
		priceHistory:  make(map[string][]PriceData),
		currentPrices: make(map[string]float64),
		intervals:     intervals,
		stopChan:      make(chan bool),
		symbolInfo:    make(map[string]api.InstrumentInfo),
		cronScheduler: cron.New(cron.WithSeconds()), // Инициализируем cron scheduler
	}
	// Инициализируем SignalMonitor
	pm.signalMonitor = NewSignalMonitor(pm, cfg.AlertThreshold)

	return pm
}

// FetchAllFuturesPairs получает все фьючерсные USDT пары (линейные фьючерсы)
func (pm *PriceMonitor) FetchAllFuturesPairs() ([]string, error) {
	// Получаем информацию об инструментах фьючерсов
	instruments, err := pm.client.GetInstrumentsInfo(pm.client.Category(), "", "Trading")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch futures instruments: %w", err)
	}

	var futuresPairs []string
	pm.mu.Lock()

	for _, instrument := range instruments {
		// Фильтруем только USDT линейные фьючерсы
		if instrument.ContractType == "LinearPerpetual" &&
			strings.HasSuffix(instrument.Symbol, "USDT") {

			futuresPairs = append(futuresPairs, instrument.Symbol)

			// Сохраняем информацию о символе
			pm.symbolInfo[instrument.Symbol] = instrument
		}
	}

	pm.symbols = futuresPairs

	// Инициализируем историю цен для каждого символа
	for _, symbol := range futuresPairs {
		if _, exists := pm.priceHistory[symbol]; !exists {
			pm.priceHistory[symbol] = make([]PriceData, 0)
		}
	}

	pm.mu.Unlock()

	sort.Strings(futuresPairs)
	return futuresPairs, nil
}

// FetchAllUSDTPairs получает все USDT пары
func (pm *PriceMonitor) FetchAllUSDTPairs() ([]string, error) {
	// Используем API клиент
	tickerResp, err := pm.client.GetTickers("linear")
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
	fmt.Println(usdtPairs)

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
		if len(symbol) <= 4 || !strings.HasSuffix(symbol, "USDT") {
			continue
		}

		// Парсим цену
		price, err := strconv.ParseFloat(ticker.LastPrice, 64)
		if err != nil {
			log.Printf("⚠️  Ошибка парсинга цены для %s: %v", symbol, err)
			continue
		}
		updatedCount++

		// Парсим объем (в USDT для фьючерсов)
		volume, _ := strconv.ParseFloat(ticker.Turnover24h, 64)

		// Обновляем текущую цену
		pm.currentPrices[symbol] = price

		// Добавляем в историю
		priceData := PriceData{
			Symbol:    symbol,
			Price:     price,
			Timestamp: now,
			Volume24h: volume,
		}

		// Сохраняем в историю
		history := pm.priceHistory[symbol]
		history = append(history, priceData)

		// Ограничиваем историю
		if len(history) > 10000 {
			history = history[len(history)-10000:]
		}

		pm.priceHistory[symbol] = history
	}

	pm.mu.Unlock()
	log.Printf("✅ Обновлено %d цен фьючерсов в %s", updatedCount, now.Format("15:04:05"))
	return nil
}

// GetSymbolInfo возвращает информацию о символе фьючерса
func (pm *PriceMonitor) GetSymbolInfo(symbol string) (*api.InstrumentInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if info, exists := pm.symbolInfo[symbol]; exists {
		return &info, nil
	}

	return nil, fmt.Errorf("информация о символе %s не найдена", symbol)
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
		return nil, fmt.Errorf("текущая цена для %s не найдена", symbol)
	}

	// Получаем исторические данные из API Bybit для фьючерсов
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

	// Получаем свечные данные для фьючерсов
	klineResp, err := pm.client.GetKlineData(symbol, pm.client.Category(), "1", intervalMinutes+1)
	if err != nil {
		return 0, err
	}

	if len(klineResp.Result.List) < 2 {
		return 0, fmt.Errorf("insufficient historical data for %s", symbol)
	}

	// Берем самую старую и самую новую свечу
	oldestCandle := klineResp.Result.List[0]
	newestCandle := klineResp.Result.List[len(klineResp.Result.List)-1]

	// Парсим цены закрытия
	oldestPrice, err := strconv.ParseFloat(oldestCandle[4], 64)
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

// StartSignalMonitoring запускает мониторинг сигналов
func (pm *PriceMonitor) StartSignalMonitoring() {
	// Конвертируем интервалы из конфига
	var intervals []Interval
	for _, interval := range pm.config.TrackedIntervals {
		intervals = append(intervals, Interval(fmt.Sprintf("%d", interval)))
	}

	// Запускаем мониторинг сигналов в отдельной горутине
	go pm.signalMonitor.MonitorSymbols(pm.symbols, intervals,
		time.Duration(pm.config.UpdateInterval)*time.Second)

	// Настраиваем cron задания для проверки в конце каждого интервала
	pm.setupCronJobs(intervals)

	pm.cronScheduler.Start()
}

// setupCronJobs настраивает задания cron для проверки в конце интервалов
func (pm *PriceMonitor) setupCronJobs(intervals []Interval) {
	for _, interval := range intervals {
		minutes, err := parseIntervalToMinutes(string(interval))
		if err != nil {
			continue
		}

		// Создаем cron выражение для запуска каждые N минут
		cronExpr := fmt.Sprintf("*/%d * * * *", minutes)

		pm.cronScheduler.AddFunc(cronExpr, func() {
			pm.checkIntervalEnd(interval)
		})
	}
}

// checkIntervalEnd проверяет сигналы в конце интервала
func (pm *PriceMonitor) checkIntervalEnd(interval Interval) {
	pm.mu.RLock()
	symbols := make([]string, len(pm.symbols))
	copy(symbols, pm.symbols)
	pm.mu.RUnlock()

	// Проверяем все символы для данного интервала
	for _, symbol := range symbols {
		pm.signalMonitor.checkSignal(symbol, interval)
	}
}

// StopSignalMonitoring останавливает мониторинг сигналов
func (pm *PriceMonitor) StopSignalMonitoring() {
	if pm.cronScheduler != nil {
		pm.cronScheduler.Stop()
	}
}

func (pm *PriceMonitor) Config() *config.Config {
	return pm.config
}
