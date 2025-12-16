// internal/monitor/growth_monitor.go
package monitor

import (
	"crypto-exchange-screener-bot/internal/api"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"sync"
	"time"
)

// GrowthMonitor - монитор непрерывного роста/падения
type GrowthMonitor struct {
	client       *api.BybitClient
	config       *config.Config
	priceMonitor *PriceMonitor
	signals      chan types.GrowthSignal // Используем types.GrowthSignal
	mu           sync.RWMutex
	stopChan     chan bool
	active       bool
	lastCheck    map[int]time.Time // Кэш времени последней проверки по периоду
}

// NewGrowthMonitor создает новый монитор роста
func NewGrowthMonitor(cfg *config.Config, priceMonitor *PriceMonitor) *GrowthMonitor {
	return &GrowthMonitor{
		client:       api.NewBybitClient(cfg),
		config:       cfg,
		priceMonitor: priceMonitor,
		signals:      make(chan types.GrowthSignal, 100),
		stopChan:     make(chan bool),
		active:       false,
		lastCheck:    make(map[int]time.Time),
	}
}

// Start запускает мониторинг роста
func (gm *GrowthMonitor) Start() {
	if gm.active {
		return
	}

	gm.active = true
	go gm.monitoringLoop()
	log.Println("🚀 Growth monitor started")
}

// Stop останавливает мониторинг роста
func (gm *GrowthMonitor) Stop() {
	if !gm.active {
		return
	}

	gm.active = false
	gm.stopChan <- true
	close(gm.signals)
	log.Println("🛑 Growth monitor stopped")
}

// monitoringLoop основной цикл мониторинга
func (gm *GrowthMonitor) monitoringLoop() {
	ticker := time.NewTicker(time.Duration(gm.config.UpdateInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gm.checkAllSymbols()
		case <-gm.stopChan:
			return
		}
	}
}

// checkAllSymbols проверяет все символы
func (gm *GrowthMonitor) checkAllSymbols() {
	// Используем ТЕ ЖЕ символы, что и основной монитор
	// Получаем символы из priceMonitor
	symbols := gm.priceMonitor.GetSymbols()

	if len(symbols) == 0 {
		return
	}

	// Ограничиваем ТОП-15 популярными парами
	popularSymbols := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT",
		"ADAUSDT", "DOGEUSDT", "MATICUSDT", "DOTUSDT", "AVAXUSDT",
		"LINKUSDT", "UNIUSDT", "LTCUSDT", "ATOMUSDT", "ETCUSDT",
	}

	// Фильтруем только те, что есть в нашем списке
	var filteredSymbols []string
	for _, symbol := range popularSymbols {
		for _, availableSymbol := range symbols {
			if symbol == availableSymbol {
				filteredSymbols = append(filteredSymbols, symbol)
				break
			}
		}
	}

	// Если не нашли достаточно, берем первые 15 из доступных
	if len(filteredSymbols) < 10 {
		maxSymbols := 15
		if len(symbols) < maxSymbols {
			maxSymbols = len(symbols)
		}
		filteredSymbols = symbols[:maxSymbols]
	}

	log.Printf("📊 Growth monitor checking %d popular symbols", len(filteredSymbols))

	for _, period := range gm.config.GrowthPeriods {
		gm.checkPeriod(filteredSymbols, period)
	}
}

// checkPeriod проверяет символы для конкретного периода
func (gm *GrowthMonitor) checkPeriod(symbols []string, periodMinutes int) {
	// Проверяем, не слишком ли часто проверяем этот период
	gm.mu.RLock()
	lastCheck, exists := gm.lastCheck[periodMinutes]
	gm.mu.RUnlock()

	if exists {
		// Не проверяем период чаще, чем половина его длительности
		minInterval := time.Duration(periodMinutes/2) * time.Minute
		if time.Since(lastCheck) < minInterval {
			return
		}
	}

	log.Printf("🔍 Checking growth for period %d minutes", periodMinutes)

	signals, err := gm.client.FindGrowthSignals(
		symbols,
		periodMinutes,
		gm.config.GrowthThreshold,
		gm.config.FallThreshold,
		gm.config.CheckContinuity,
	)

	if err != nil {
		log.Printf("❌ Error checking growth signals: %v", err)
		return
	}

	// Обновляем время последней проверки
	gm.mu.Lock()
	gm.lastCheck[periodMinutes] = time.Now()
	gm.mu.Unlock()

	for _, signal := range signals {
		gm.processSignal(signal)
	}
}

// processSignal обрабатывает сигнал роста
func (gm *GrowthMonitor) processSignal(signal types.GrowthSignal) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Отправляем сигнал в канал
	select {
	case gm.signals <- signal:
		gm.printSignal(signal)
	default:
		log.Printf("⚠️ Signal channel full, dropping signal for %s", signal.Symbol)
	}
}

// printSignal выводит сигнал в терминал
func (gm *GrowthMonitor) printSignal(signal types.GrowthSignal) {
	var icon, direction, changeStr string

	if signal.Direction == "growth" {
		icon = "🟢"
		direction = "Continuous GROWTH"
		changeStr = fmt.Sprintf("+%.2f%%", signal.GrowthPercent)
	} else {
		icon = "🔴"
		direction = "Continuous FALL"
		changeStr = fmt.Sprintf("-%.2f%%", signal.FallPercent)
	}

	periodStr := gm.formatPeriod(signal.PeriodMinutes)

	// Форматируем время сигнала
	timeStr := signal.Timestamp.Format("2006/01/02 15:04:05")

	// Выводим напрямую, без среза строк
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Printf("%s %s - %s - %s\n", icon, direction, periodStr, signal.Symbol)
	fmt.Printf("🕐 %s\n", timeStr) // Добавляем время сигнала
	fmt.Printf("📈 Change: %s\n", changeStr)
	fmt.Printf("🎯 Period: %d minutes\n", signal.PeriodMinutes)
	fmt.Printf("📊 Confidence: %.1f%%\n", signal.Confidence)
	fmt.Printf("💰 Price: %.4f → %.4f\n", signal.StartPrice, signal.EndPrice)
	fmt.Printf("🔗 https://www.bybit.com/trade/usdt/%s\n", signal.Symbol)
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println()

	// Логируем в файл
	gm.logSignal(signal)
}

// formatPeriod форматирует период для отображения
func (gm *GrowthMonitor) formatPeriod(minutes int) string {
	switch {
	case minutes < 60:
		return fmt.Sprintf("%d min", minutes)
	case minutes == 60:
		return "1 hour"
	case minutes < 1440:
		return fmt.Sprintf("%d hours", minutes/60)
	default:
		return fmt.Sprintf("%d days", minutes/1440)
	}
}

// logSignal логирует сигнал в файл
func (gm *GrowthMonitor) logSignal(signal types.GrowthSignal) {
	// Логирование в файл (можно реализовать)
	log.Printf("📝 Signal logged: %s %s %.2f%%",
		signal.Symbol, signal.Direction,
		signal.GrowthPercent+signal.FallPercent)
}

// GetSignals возвращает канал сигналов
func (gm *GrowthMonitor) GetSignals() <-chan types.GrowthSignal {
	return gm.signals
}

// GetActiveSignals возвращает активные сигналы
func (gm *GrowthMonitor) GetActiveSignals() []types.GrowthSignal {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Собираем все сигналы из канала
	var signals []types.GrowthSignal
	for {
		select {
		case signal := <-gm.signals:
			signals = append(signals, signal)
		default:
			return signals
		}
	}
}

// AnalyzeSymbol анализирует конкретный символ
func (gm *GrowthMonitor) AnalyzeSymbol(symbol string) ([]types.GrowthSignal, error) {
	var allSignals []types.GrowthSignal

	for _, period := range gm.config.GrowthPeriods {
		signals, err := gm.client.FindGrowthSignals(
			[]string{symbol},
			period,
			gm.config.GrowthThreshold,
			gm.config.FallThreshold,
			gm.config.CheckContinuity,
		)

		if err != nil {
			continue
		}

		allSignals = append(allSignals, signals...)
	}

	return allSignals, nil
}

// GetGrowthStats возвращает статистику по мониторингу роста
func (gm *GrowthMonitor) GetGrowthStats() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	signals := gm.GetActiveSignals()

	growthCount := 0
	fallCount := 0
	var avgGrowth, avgFall float64

	for _, signal := range signals {
		if signal.Direction == "growth" {
			growthCount++
			avgGrowth += signal.GrowthPercent
		} else {
			fallCount++
			avgFall += signal.FallPercent
		}
	}

	if growthCount > 0 {
		avgGrowth /= float64(growthCount)
	}
	if fallCount > 0 {
		avgFall /= float64(fallCount)
	}

	return map[string]interface{}{
		"total_signals":      len(signals),
		"growth_signals":     growthCount,
		"fall_signals":       fallCount,
		"avg_growth":         avgGrowth,
		"avg_fall":           avgFall,
		"monitoring_periods": gm.config.GrowthPeriods,
		"growth_threshold":   gm.config.GrowthThreshold,
		"fall_threshold":     gm.config.FallThreshold,
		"active":             gm.active,
	}
}
func (gm *GrowthMonitor) GetDetailedStats() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	signals := gm.GetActiveSignals()

	stats := map[string]interface{}{
		"total_signals":  len(signals),
		"growth_signals": 0,
		"fall_signals":   0,
		"active":         gm.active,
		"last_check":     time.Now(),
	}

	// Группируем по периодам
	periodStats := make(map[int]int)
	for _, signal := range signals {
		if signal.Direction == "growth" {
			stats["growth_signals"] = stats["growth_signals"].(int) + 1
		} else {
			stats["fall_signals"] = stats["fall_signals"].(int) + 1
		}
		periodStats[signal.PeriodMinutes] = periodStats[signal.PeriodMinutes] + 1
	}

	stats["period_stats"] = periodStats

	return stats
}
