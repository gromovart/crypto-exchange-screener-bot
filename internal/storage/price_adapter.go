package storage

import (
	"crypto-exchange-screener-bot/internal/monitor"
	"log"
	"sync"
	"time"
)

// PriceMonitorAdapter адаптирует PriceMonitor для работы с хранилищем
type PriceMonitorAdapter struct {
	priceMonitor *monitor.PriceMonitor
	storage      PriceStorage
	mu           sync.RWMutex
}

// NewPriceMonitorAdapter создает адаптер
func NewPriceMonitorAdapter(pm *monitor.PriceMonitor, storage PriceStorage) *PriceMonitorAdapter {
	return &PriceMonitorAdapter{
		priceMonitor: pm,
		storage:      storage,
	}
}

// Start начинает мониторинг и сохранение цен
func (pma *PriceMonitorAdapter) Start(updateInterval time.Duration) {
	// Запускаем существующий мониторинг
	pma.priceMonitor.StartMonitoring(updateInterval)

	// Запускаем сохранение цен в хранилище
	go pma.startPriceSaver(updateInterval)
}

// startPriceSaver сохраняет цены в хранилище
func (pma *PriceMonitorAdapter) startPriceSaver(updateInterval time.Duration) {
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for range ticker.C {
		pma.saveCurrentPrices()
	}
}

// saveCurrentPrices сохраняет текущие цены в хранилище
func (pma *PriceMonitorAdapter) saveCurrentPrices() {
	pma.mu.Lock()
	defer pma.mu.Unlock()

	// Получаем текущие цены из PriceMonitor
	prices := pma.priceMonitor.GetCurrentPrices()

	now := time.Now()
	for symbol, price := range prices {
		// Получаем объем из истории
		var volume float64
		if history, err := pma.priceMonitor.GetPriceHistory(symbol, 1); err == nil && len(history) > 0 {
			volume = history[0].Volume24h
		}

		// Сохраняем в хранилище
		if err := pma.storage.StorePrice(symbol, price, volume, now); err != nil {
			log.Printf("❌ Ошибка сохранения цены %s: %v", symbol, err)
		}
	}
}

// GetSymbols возвращает список символов
func (pma *PriceMonitorAdapter) GetSymbols() []string {
	return pma.priceMonitor.GetSymbols()
}

// GetCurrentPrices возвращает текущие цены
func (pma *PriceMonitorAdapter) GetCurrentPrices() map[string]float64 {
	return pma.priceMonitor.GetCurrentPrices()
}

// GetPriceHistory возвращает историю цен
func (pma *PriceMonitorAdapter) GetPriceHistory(symbol string, limit int) ([]monitor.PriceData, error) {
	// Используем существующий метод PriceMonitor
	return pma.priceMonitor.GetPriceHistory(symbol, limit)
}

// Stop останавливает адаптер
func (pma *PriceMonitorAdapter) Stop() {
	pma.priceMonitor.StopMonitoring()
}
