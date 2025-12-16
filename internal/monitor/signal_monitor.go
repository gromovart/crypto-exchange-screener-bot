// internal/monitor/signal_monitor.go
package monitor

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"time"
)

// SignalMonitor - монитор сигналов для отправки в терминал
type SignalMonitor struct {
	priceMonitor    *PriceMonitor
	history         map[string]map[Interval]*SignalHistory // symbol -> interval -> history
	mu              sync.RWMutex
	alertThreshold  float64
	activeSignals   map[string]bool // Уникальный ключ symbol+interval для активных сигналов
	signalIDCounter int
	logFile         *os.File             // Добавляем файл для логирования
	lastSignalTime  map[string]time.Time // Добавляем для cooldown
}

// NewSignalMonitor создает новый монитор сигналов
func NewSignalMonitor(priceMonitor *PriceMonitor, alertThreshold float64) *SignalMonitor {
	return &SignalMonitor{
		priceMonitor:    priceMonitor,
		history:         make(map[string]map[Interval]*SignalHistory),
		activeSignals:   make(map[string]bool),
		alertThreshold:  alertThreshold,
		signalIDCounter: 1,
		lastSignalTime:  make(map[string]time.Time), // Инициализируем
	}
}

// MonitorSymbols начинает мониторинг выбранных символов и интервалов
func (sm *SignalMonitor) MonitorSymbols(symbols []string, intervals []Interval, updateInterval time.Duration) {
	// Инициализируем историю для символов и интервалов
	sm.initializeHistory(symbols, intervals)

	// Запускаем тикер для проверки цен
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for range ticker.C {
		sm.checkAllSymbols(symbols, intervals)
	}
}

// initializeHistory инициализирует историю сигналов
func (sm *SignalMonitor) initializeHistory(symbols []string, intervals []Interval) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, symbol := range symbols {
		sm.history[symbol] = make(map[Interval]*SignalHistory)
		for _, interval := range intervals {
			sm.history[symbol][interval] = &SignalHistory{
				Symbol:    symbol,
				Interval:  interval,
				Signals:   make([]Signal, 0),
				LastTrend: "neutral",
			}
		}
	}
}

// checkAllSymbols проверяет все символы на наличие сигналов
func (sm *SignalMonitor) checkAllSymbols(symbols []string, intervals []Interval) {
	for _, symbol := range symbols {
		for _, interval := range intervals {
			sm.checkSignal(symbol, interval)
		}
	}
}

// checkSignal проверяет сигнал для конкретного символа и интервала
func (sm *SignalMonitor) checkSignal(symbol string, interval Interval) {
	// Получаем изменение цены
	priceChange, err := sm.priceMonitor.GetPriceChange(symbol, interval)
	if err != nil {
		return
	}

	// Проверяем, превышает ли изменение порог
	absChange := priceChange.ChangePercent
	if absChange < 0 {
		absChange = -absChange
	}

	if absChange < sm.alertThreshold {
		return
	}

	// Определяем направление
	direction := "pump"
	if priceChange.ChangePercent < 0 {
		direction = "dump"
	}

	// Создаем сигнал
	signal := Signal{
		Symbol:        symbol,
		Interval:      interval,
		ChangePercent: priceChange.ChangePercent,
		Direction:     direction,
		Timestamp:     time.Now(),
	}

	// Обрабатываем сигнал
	sm.processSignal(signal)
}

// processSignal обрабатывает новый сигнал
func (sm *SignalMonitor) processSignal(signal Signal) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	symbol := signal.Symbol
	interval := signal.Interval

	// Проверяем, существует ли история для этого символа и интервала
	if _, ok := sm.history[symbol]; !ok {
		sm.history[symbol] = make(map[Interval]*SignalHistory)
	}
	if _, ok := sm.history[symbol][interval]; !ok {
		sm.history[symbol][interval] = &SignalHistory{
			Symbol:    symbol,
			Interval:  interval,
			Signals:   make([]Signal, 0),
			LastTrend: "neutral",
		}
	}

	history := sm.history[symbol][interval]

	// Удаляем старые сигналы (старше 24 часов)
	sm.cleanOldSignals(history)

	// Проверяем, является ли это продолжением тренда
	isContinuation := sm.isTrendContinuation(signal, history)

	// Определяем ID сигнала
	if !isContinuation || len(history.Signals) == 0 {
		// Новый тренд - создаем новый ID
		signal.SignalID = sm.getNextSignalID()
	} else {
		// Продолжение тренда - используем последний ID
		lastSignal := history.Signals[len(history.Signals)-1]
		signal.SignalID = lastSignal.SignalID
	}

	// Добавляем сигнал в историю
	history.Signals = append(history.Signals, signal)
	history.LastTrend = signal.Direction

	// Считаем количество уникальных сигналов за 24 часа
	signalCount := sm.countUniqueSignals24h(history)

	// Отправляем сообщение в терминал
	sm.printSignalMessage(signal, signalCount, isContinuation)
}

func (sm *SignalMonitor) getNextSignalID() int {
	sm.signalIDCounter++
	return sm.signalIDCounter
}

// isTrendContinuation проверяет, является ли сигнал продолжением тренда
// Добавляем метод isTrendContinuation
func (sm *SignalMonitor) isTrendContinuation(signal Signal, history *SignalHistory) bool {
	if len(history.Signals) == 0 {
		return false
	}

	lastSignal := history.Signals[len(history.Signals)-1]

	// Проверяем, совпадает ли направление
	if lastSignal.Direction != signal.Direction {
		return false
	}

	// Проверяем, не прошло ли слишком много времени
	timeSinceLast := signal.Timestamp.Sub(lastSignal.Timestamp)
	intervalMinutes, _ := parseIntervalToMinutes(string(signal.Interval))
	maxTimeBetweenSignals := time.Duration(intervalMinutes*3) * time.Minute // Максимум 3 интервала

	return timeSinceLast < maxTimeBetweenSignals
}

// Добавляем cooldown между одинаковыми сигналами
func (sm *SignalMonitor) shouldProcessSignal(signal Signal, history *SignalHistory) bool {
	if len(history.Signals) == 0 {
		return true
	}

	lastSignal := history.Signals[len(history.Signals)-1]
	timeSinceLast := signal.Timestamp.Sub(lastSignal.Timestamp)

	// Cooldown: не выводить тот же сигнал чаще чем раз в N минут
	cooldownMinutes := 5
	cooldownDuration := time.Duration(cooldownMinutes) * time.Minute

	if timeSinceLast < cooldownDuration {
		// Проверяем, достаточно ли отличается новый сигнал
		changeDiff := math.Abs(signal.ChangePercent - lastSignal.ChangePercent)
		directionChanged := signal.Direction != lastSignal.Direction

		if changeDiff < 0.1 && !directionChanged {
			// Сигнал практически не изменился, пропускаем
			return false
		}
	}

	return true
}

// cleanOldSignals удаляет сигналы старше 24 часов
func (sm *SignalMonitor) cleanOldSignals(history *SignalHistory) {
	now := time.Now()
	cutoffTime := now.Add(-24 * time.Hour)

	var validSignals []Signal
	for _, signal := range history.Signals {
		if signal.Timestamp.After(cutoffTime) {
			validSignals = append(validSignals, signal)
		}
	}

	history.Signals = validSignals
}

// sendTerminalMessage отправляет сообщение в терминал
func (sm *SignalMonitor) sendTerminalMessage(signal Signal, history *SignalHistory, isContinuation bool, signalCount int) {
	// Создаем сообщение
	message := TerminalMessage{
		Exchange:      "Bybit",
		Interval:      string(signal.Interval),
		Symbol:        signal.Symbol,
		SymbolURL:     fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", signal.Symbol),
		ChangePercent: signal.ChangePercent,
		Direction:     signal.Direction,
		Signal24h:     signalCount,
		Timestamp:     signal.Timestamp,
	}

	// Отправляем сообщение
	sm.printTerminalMessage(message, isContinuation)
}

// countUniqueSignals24h считает количество уникальных сигналов за 24 часа
func (sm *SignalMonitor) countUniqueSignals24h(history *SignalHistory) int {
	uniqueIDs := make(map[int]bool)
	now := time.Now()
	cutoffTime := now.Add(-24 * time.Hour)

	for _, signal := range history.Signals {
		if signal.Timestamp.After(cutoffTime) {
			uniqueIDs[signal.SignalID] = true
		}
	}

	return len(uniqueIDs)
}

// printTerminalMessage выводит сообщение в терминал
func (sm *SignalMonitor) printTerminalMessage(message TerminalMessage, isContinuation bool) {
	// Форматируем интервал
	intervalStr := formatIntervalForDisplay(message.Interval)

	// Форматируем изменение цены
	changeStr := fmt.Sprintf("%.2f%%", message.ChangePercent)
	if message.ChangePercent > 0 {
		changeStr = "+" + changeStr
	}

	// Определяем цвет и иконку
	var icon, directionStr string
	if message.Direction == "pump" {
		icon = "🟢"
		directionStr = "Pump"
	} else {
		icon = "🔴"
		directionStr = "Dump"
	}

	// Форматируем время сигнала
	timeStr := message.Timestamp.Format("2006/01/02 15:04:05")

	// Формируем сообщение (используем fmt.Sprintf для форматирования строк)
	lines := []string{
		"══════════════════════════════════════════════════",
		fmt.Sprintf("⚫ %s - %s - %s", message.Exchange, intervalStr, message.Symbol),
		fmt.Sprintf("🕐 %s", timeStr), // Добавляем время сигнала
		fmt.Sprintf("%s %s: %s", icon, directionStr, changeStr),
		fmt.Sprintf("📡 Signal 24h: %d", message.Signal24h),
		fmt.Sprintf("🔗 %s", message.SymbolURL),
		"══════════════════════════════════════════════════",
		"", // Пустая строка для разделения
	}

	// Выводим в терминал
	for _, line := range lines {
		fmt.Println(line)
	}

	// Если это продолжение тренда, добавляем пояснение
	if isContinuation {
		fmt.Println("   ↪ Тренд продолжается")
		fmt.Println()
	}
}

// Вспомогательные функции

func parseIntervalToMinutes(interval string) (int, error) {
	switch interval {
	case "1":
		return 1, nil
	case "5":
		return 5, nil
	case "10":
		return 10, nil
	case "15":
		return 15, nil
	case "30":
		return 30, nil
	case "60":
		return 60, nil
	case "120":
		return 120, nil
	case "240":
		return 240, nil
	case "480":
		return 480, nil
	case "720":
		return 720, nil
	case "1440":
		return 1440, nil
	case "10080":
		return 10080, nil
	case "43200":
		return 43200, nil
	default:
		return 0, fmt.Errorf("неизвестный интервал: %s", interval)
	}
}

func formatIntervalForDisplay(interval string) string {
	minutes, err := parseIntervalToMinutes(interval)
	if err != nil {
		return interval
	}

	if minutes < 60 {
		return fmt.Sprintf("%d мин", minutes)
	} else if minutes == 60 {
		return "1 час"
	} else if minutes < 1440 {
		return fmt.Sprintf("%d час", minutes/60)
	} else {
		return fmt.Sprintf("%d дн", minutes/1440)
	}
}

// GetSignalHistory возвращает историю сигналов для символа
func (sm *SignalMonitor) GetSignalHistory(symbol string, interval Interval) []Signal {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if history, ok := sm.history[symbol]; ok {
		if sigHistory, ok := history[interval]; ok {
			return sigHistory.Signals
		}
	}
	return []Signal{}
}

// GetActiveSignals возвращает активные сигналы
func (sm *SignalMonitor) GetActiveSignals() map[string]Signal {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	active := make(map[string]Signal)
	now := time.Now()

	for symbol, intervals := range sm.history {
		for interval, history := range intervals {
			if len(history.Signals) > 0 {
				lastSignal := history.Signals[len(history.Signals)-1]
				// Считаем сигнал активным, если он был в последние 2 интервала
				intervalMinutes, _ := parseIntervalToMinutes(string(interval))
				if now.Sub(lastSignal.Timestamp) <= time.Duration(intervalMinutes*2)*time.Minute {
					key := fmt.Sprintf("%s_%s", symbol, interval)
					active[key] = lastSignal
				}
			}
		}
	}

	return active
}

// CheckSignalNow принудительно проверяет сигнал сейчас
func (sm *SignalMonitor) CheckSignalNow(symbol string, interval Interval) bool {
	// Получаем данные об объеме
	volume24h, _ := sm.priceMonitor.client.Get24hVolume(symbol)

	// Фильтруем по минимальному объему (например, $100,000)
	if volume24h < 100000 {
		return false
	}
	// Создаем ключ для cooldown
	cooldownKey := fmt.Sprintf("%s_%s", symbol, interval)

	sm.mu.RLock()
	lastTime, hasLastTime := sm.lastSignalTime[cooldownKey]
	sm.mu.RUnlock()

	// Проверяем cooldown (30 секунд между сигналами для одной пары и интервала)
	if hasLastTime {
		timeSinceLast := time.Since(lastTime)
		if timeSinceLast < 30*time.Second {
			return false
		}
	}

	// Получаем изменение цены
	priceChange, err := sm.priceMonitor.GetPriceChange(symbol, interval)
	if err != nil {
		fmt.Printf("[DEBUG] Ошибка получения изменения цены для %s %s: %v\n",
			symbol, interval, err)
		return false
	}

	// Проверяем, превышает ли изменение порог
	absChange := priceChange.ChangePercent
	if absChange < 0 {
		absChange = -absChange
	}

	if absChange >= sm.alertThreshold {
		// Определяем направление
		direction := "pump"
		if priceChange.ChangePercent < 0 {
			direction = "dump"
		}

		// Создаем сигнал
		signal := Signal{
			Symbol:        symbol,
			Interval:      interval,
			ChangePercent: priceChange.ChangePercent,
			Direction:     direction,
			Timestamp:     time.Now(),
		}

		// Обрабатываем сигнал
		sm.processSignal(signal)

		// Обновляем время последнего сигнала
		sm.mu.Lock()
		sm.lastSignalTime[cooldownKey] = time.Now()
		sm.mu.Unlock()

		return true
	}

	return false
}

// printDebugInfo выводит отладочную информацию
func (sm *SignalMonitor) printDebugInfo(signal Signal) {
	// Только для отладки - выводим все изменения
	changeStr := fmt.Sprintf("%.2f%%", signal.ChangePercent)
	if signal.ChangePercent > 0 {
		changeStr = "+" + changeStr
	}

	fmt.Printf("[DEBUG] %s %s: %s (порог: %.2f%%)\n",
		signal.Symbol,
		signal.Interval,
		changeStr,
		sm.alertThreshold)
}

// logSignalToFile логирует сигнал в файл
func (sm *SignalMonitor) logSignalToFile(signal Signal, signalCount int) {
	if sm.logFile == nil {
		return
	}

	logEntry := map[string]interface{}{
		"timestamp":      signal.Timestamp.Format(time.RFC3339),
		"symbol":         signal.Symbol,
		"interval":       string(signal.Interval),
		"change_percent": signal.ChangePercent,
		"direction":      signal.Direction,
		"signal_24h":     signalCount,
	}

	data, err := json.Marshal(logEntry)
	if err != nil {
		return
	}

	data = append(data, '\n')
	sm.logFile.Write(data)
	sm.logFile.Sync()
}

type VolumeFilter struct {
	minVolumeUSDT float64
	volumeCache   map[string]float64
	cacheTTL      time.Duration
	lastUpdate    time.Time
}

func NewVolumeFilter(minVolumeUSDT float64) *VolumeFilter {
	return &VolumeFilter{
		minVolumeUSDT: minVolumeUSDT,
		volumeCache:   make(map[string]float64),
		cacheTTL:      5 * time.Minute,
	}
}

func (vf *VolumeFilter) ShouldFilter(symbol string, volume24h float64) bool {
	// Проверяем объем
	if volume24h < vf.minVolumeUSDT {
		return true
	}
	return false
}

// Обновляем printSignalMessage
func (sm *SignalMonitor) printSignalMessage(signal Signal, signalCount int, isContinuation bool) {
	// Форматируем интервал
	intervalStr := sm.formatIntervalDisplay(string(signal.Interval))

	// Форматируем изменение
	changeStr := fmt.Sprintf("%.2f%%", signal.ChangePercent)
	if signal.ChangePercent > 0 {
		changeStr = "+" + changeStr
	}

	// Определяем иконку и направление
	var icon, direction string
	if signal.Direction == "pump" {
		icon = "🟢"
		direction = "Pump"
	} else {
		icon = "🔴"
		direction = "Dump"
	}

	// Ссылка на торговую пару
	symbolURL := fmt.Sprintf("https://www.bybit.com/trade/usdt/%s", signal.Symbol)
	timeStr := signal.Timestamp.Format("2006/01/02 15:04:05")
	// Выводим сообщение
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Printf("⚫ Bybit - %s - %s\n", intervalStr, signal.Symbol)
	fmt.Printf("%s %s: %s\n", icon, direction, changeStr)
	fmt.Printf("📡 Signal 24h: %d\n", signalCount)
	fmt.Printf("🔗 %s\n", symbolURL)
	fmt.Printf("🕐 %s\n", timeStr) // Добавляем время сигнала
	fmt.Println("══════════════════════════════════════════════════")

	// Добавляем указание на продолжение тренда
	if isContinuation {
		fmt.Println("   ↪ Тренд продолжается")
	}

	fmt.Println()
}

// formatIntervalDisplay форматирует интервал для отображения
func (sm *SignalMonitor) formatIntervalDisplay(interval string) string {
	switch interval {
	case "1":
		return "1 мин"
	case "5":
		return "5 мин"
	case "10":
		return "10 мин"
	case "15":
		return "15 мин"
	case "30":
		return "30 мин"
	case "60":
		return "1 час"
	case "120":
		return "2 час"
	case "240":
		return "4 час"
	default:
		return interval + " мин"
	}
}
