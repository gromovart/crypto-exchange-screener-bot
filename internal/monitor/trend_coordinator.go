package monitor

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/storage"
	"crypto-exchange-screener-bot/internal/types"
	"log"
	"sync"
	"time"
)

// TrendCoordinator координатор трендового анализа
type TrendCoordinator struct {
	config      *config.Config
	storage     storage.PriceStorage
	fetcher     PriceFetcher
	analyzer    TrendAnalyzer
	notifier    types.NotificationService
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
	stats       map[string]interface{}
	lastAnalyze time.Time
}

// NewTrendCoordinator создает координатор
func NewTrendCoordinator(
	cfg *config.Config,
	storage storage.PriceStorage,
	fetcher PriceFetcher,
	analyzer TrendAnalyzer,
	notifier types.NotificationService,
) *TrendCoordinator {
	return &TrendCoordinator{
		config:   cfg,
		storage:  storage,
		fetcher:  fetcher,
		analyzer: analyzer,
		notifier: notifier,
		stopChan: make(chan struct{}),
		stats: map[string]interface{}{
			"total_analyzed": 0,
			"signals_found":  0,
			"last_analyze":   time.Time{},
			"start_time":     time.Now(),
		},
	}
}

// Start запускает координатор
func (c *TrendCoordinator) Start() error {
	if c.running {
		return nil
	}

	c.running = true

	// Запускаем получение данных
	if err := c.fetcher.StartFetching(time.Duration(c.config.UpdateInterval) * time.Second); err != nil {
		return err
	}

	// Запускаем анализ
	go c.analysisLoop()

	log.Println("🚀 TrendCoordinator запущен")
	return nil
}

// Stop останавливает координатор
func (c *TrendCoordinator) Stop() error {
	if !c.running {
		return nil
	}

	c.running = false
	close(c.stopChan)

	// Останавливаем получение данных
	if err := c.fetcher.StopFetching(); err != nil {
		return err
	}

	log.Println("🛑 TrendCoordinator остановлен")
	return nil
}

// analysisLoop цикл анализа
func (c *TrendCoordinator) analysisLoop() {
	ticker := time.NewTicker(time.Duration(c.config.UpdateInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.analyzeSymbols()
		case <-c.stopChan:
			return
		}
	}
}

// analyzeSymbols анализирует символы
func (c *TrendCoordinator) analyzeSymbols() {
	symbols := c.getSymbolsToAnalyze()
	if len(symbols) == 0 {
		return
	}

	var signals []types.TrendSignal

	for _, symbol := range symbols {
		for _, period := range c.analyzer.GetSupportedPeriods() {
			signal, err := c.analyzeSymbol(symbol, period)
			if err != nil {
				continue
			}

			if signal.ChangePercent > 0 { // Фильтруем пустые сигналы
				signals = append(signals, signal)
			}
		}
	}

	// Отправляем сигналы
	if len(signals) > 0 {
		for _, signal := range signals {
			c.notifier.Send(signal)
		}
	}

	// Обновляем статистику
	c.mu.Lock()
	c.stats["total_analyzed"] = c.stats["total_analyzed"].(int) + len(symbols)
	c.stats["signals_found"] = c.stats["signals_found"].(int) + len(signals)
	c.stats["last_analyze"] = time.Now()
	c.mu.Unlock()

	c.lastAnalyze = time.Now()
}

// analyzeSymbol анализирует конкретный символ
func (c *TrendCoordinator) analyzeSymbol(symbol string, periodMinutes int) (types.TrendSignal, error) {
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(periodMinutes) * time.Minute)

	history, err := c.storage.GetPriceHistoryRange(symbol, startTime, endTime)
	if err != nil {
		return types.TrendSignal{}, err
	}

	// Конвертируем в формат PriceData
	var priceData []types.PriceData
	for _, h := range history {
		priceData = append(priceData, types.PriceData{
			Symbol:    h.Symbol,
			Price:     h.Price,
			Volume24h: h.Volume24h,
			Timestamp: h.Timestamp,
		})
	}

	return c.analyzer.Analyze(symbol, priceData)
}

// getSymbolsToAnalyze возвращает символы для анализа
func (c *TrendCoordinator) getSymbolsToAnalyze() []string {
	symbols := c.storage.GetSymbols()

	// Фильтруем по минимальному объему
	if c.config.MinVolumeFilter > 0 {
		var filtered []string
		for _, symbol := range symbols {
			if snapshot, exists := c.storage.GetCurrentSnapshot(symbol); exists {
				if snapshot.Volume24h >= c.config.MinVolumeFilter {
					filtered = append(filtered, symbol)
				}
			}
		}
		symbols = filtered
	}

	// Ограничиваем количество
	if c.config.MaxSymbolsToMonitor > 0 && len(symbols) > c.config.MaxSymbolsToMonitor {
		symbols = symbols[:c.config.MaxSymbolsToMonitor]
	}

	return symbols
}

// GetStats возвращает статистику
func (c *TrendCoordinator) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := make(map[string]interface{})
	for k, v := range c.stats {
		stats[k] = v
	}

	// Добавляем статистику компонентов
	stats["fetcher"] = c.fetcher.GetStats()
	stats["analyzer"] = c.analyzer.GetStats()
	stats["notifier"] = c.notifier.GetStats()
	stats["running"] = c.running

	return stats
}
