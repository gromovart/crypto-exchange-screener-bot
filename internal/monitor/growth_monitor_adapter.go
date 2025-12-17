package monitor

import (
	"crypto-exchange-screener-bot/internal/storage"
	"crypto-exchange-screener-bot/internal/types"
)

// GrowthMonitorAdapter адаптирует GrowthMonitor для работы с хранилищем
type GrowthMonitorAdapter struct {
	growthMonitor *GrowthMonitor
	storage       storage.PriceStorage
}

// NewGrowthMonitorAdapter создает адаптер
func NewGrowthMonitorAdapter(gm *GrowthMonitor, storage storage.PriceStorage) *GrowthMonitorAdapter {
	return &GrowthMonitorAdapter{
		growthMonitor: gm,
		storage:       storage,
	}
}

// GetSignals возвращает канал сигналов
func (gma *GrowthMonitorAdapter) GetSignals() <-chan types.GrowthSignal {
	return gma.growthMonitor.GetSignals()
}

// GetGrowthStats возвращает статистику
func (gma *GrowthMonitorAdapter) GetGrowthStats() map[string]interface{} {
	return gma.growthMonitor.GetGrowthStats()
}

// GetDetailedStats возвращает детальную статистику
func (gma *GrowthMonitorAdapter) GetDetailedStats() map[string]interface{} {
	return gma.growthMonitor.GetDetailedStats()
}

// FlushDisplay выводит накопленные сигналы
func (gma *GrowthMonitorAdapter) FlushDisplay() {
	gma.growthMonitor.FlushDisplay()
}

// FlushBuffers очищает буферы
func (gma *GrowthMonitorAdapter) FlushBuffers() {
	gma.growthMonitor.FlushBuffers()
}

// SendTelegramTest отправляет тестовое сообщение
func (gma *GrowthMonitorAdapter) SendTelegramTest() error {
	return gma.growthMonitor.SendTelegramTest()
}

// Start запускает мониторинг
func (gma *GrowthMonitorAdapter) Start() {
	gma.growthMonitor.Start()
}

// Stop останавливает мониторинг
func (gma *GrowthMonitorAdapter) Stop() {
	gma.growthMonitor.Stop()
}
