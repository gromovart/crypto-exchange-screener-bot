// internal/monitor/growth_monitor.go
package monitor

import (
	"crypto-exchange-screener-bot/internal/api"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/telegram"
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// GrowthMonitor - монитор непрерывного роста/падения
type GrowthMonitor struct {
	client         *api.BybitClient
	config         *config.Config
	priceMonitor   *PriceMonitor
	signals        chan types.GrowthSignal
	filter         *SignalFilter
	display        *DisplayManager
	telegramBot    *telegram.TelegramBot // Добавляем Telegram бота
	mu             sync.RWMutex
	stopChan       chan bool
	active         bool
	lastCheck      map[int]time.Time
	signalsHistory []types.GrowthSignal
	signalsCount   map[string]int
}

// NewGrowthMonitor создает новый монитор роста
func NewGrowthMonitor(cfg *config.Config, priceMonitor *PriceMonitor) *GrowthMonitor {
	// Настройки отображения
	minChange := 0.5
	maxSignals := 15

	// Создаем Telegram бота если включен
	var telegramBot *telegram.TelegramBot
	if cfg.TelegramEnabled && cfg.TelegramAPIKey != "" && cfg.TelegramChatID != 0 {
		telegramBot = telegram.NewTelegramBot(cfg)
		log.Printf("🤖 Telegram бот инициализирован для чата ID: %d", cfg.TelegramChatID)
	}

	return &GrowthMonitor{
		client:         api.NewBybitClient(cfg),
		config:         cfg,
		priceMonitor:   priceMonitor,
		signals:        make(chan types.GrowthSignal, 100),
		filter:         NewSignalFilter(cfg),
		display:        NewDisplayManager(true, minChange, 50.0, maxSignals),
		telegramBot:    telegramBot, // Сохраняем бота
		stopChan:       make(chan bool),
		active:         false,
		lastCheck:      make(map[int]time.Time),
		signalsHistory: make([]types.GrowthSignal, 0),
		signalsCount:   make(map[string]int),
	}
}

// Start запускает мониторинг роста
func (gm *GrowthMonitor) Start() {
	if gm.active {
		return
	}

	gm.active = true
	go gm.monitoringLoop()
	log.Println("🚀 Монитор роста запущен")
}

// Stop останавливает мониторинг роста
func (gm *GrowthMonitor) Stop() {
	if !gm.active {
		return
	}

	gm.active = false
	gm.stopChan <- true
	close(gm.signals)
	log.Println("🛑 Монитор роста остановлен")
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
	// Получаем символы с учетом фильтров
	symbols, err := gm.GetSymbolsToMonitor()
	if err != nil {
		log.Printf("❌ Ошибка получения символов: %v", err)
		return
	}

	if len(symbols) == 0 {
		log.Println("⚠️ Нет символов для мониторинга")
		return
	}

	log.Printf("📊 Монитор роста проверяет %d символов", len(symbols))

	// Выводим список отслеживаемых символов (первые 20)
	if len(symbols) > 20 {
		log.Printf("   Отслеживаемые символы: %s...", strings.Join(symbols[:20], ", "))
	} else {
		log.Printf("   Отслеживаемые символы: %s", strings.Join(symbols, ", "))
	}

	// Проверяем каждый период
	for _, period := range gm.config.GrowthPeriods {
		gm.checkPeriod(symbols, period)
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

	log.Printf("🔍 Проверка роста за период %d минут", periodMinutes)

	signals, err := gm.client.FindGrowthSignals(
		symbols,
		periodMinutes,
		gm.config.GrowthThreshold,
		gm.config.FallThreshold,
		gm.config.CheckContinuity,
	)

	if err != nil {
		log.Printf("❌ Ошибка при проверке сигналов роста: %v", err)
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
	// Применяем фильтры
	if gm.config.SignalFilters.Enabled && !gm.filter.ApplyFilters(signal) {
		return
	}

	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Сохраняем сигнал в историю
	gm.signalsHistory = append(gm.signalsHistory, signal)

	// Увеличиваем счетчик сигналов
	key := fmt.Sprintf("%s_%s", signal.Direction, signal.Symbol)
	gm.signalsCount[key] = gm.signalsCount[key] + 1

	// Добавляем в DisplayManager для группового вывода
	gm.display.AddSignal(signal)

	// Отправляем в Telegram если бот активен
	if gm.telegramBot != nil {
		go func(s types.GrowthSignal) {
			if err := gm.telegramBot.SendNotification(s); err != nil {
				log.Printf("❌ Ошибка отправки в Telegram: %v", err)
			}
		}(signal)
	}

	// Отправляем сигнал в канал
	select {
	case gm.signals <- signal:
		// Сигнал отправлен
	default:
		log.Printf("⚠️ Канал сигналов переполнен, сигнал для %s пропущен", signal.Symbol)
	}
}

// printSignal выводит сигнал в терминал
func (gm *GrowthMonitor) printSignal(signal types.GrowthSignal) {
	var icon string
	changePercent := signal.GrowthPercent + signal.FallPercent

	if signal.Direction == "growth" {
		icon = "🟢"
		fmt.Printf("%s %s ↑%.2f%% (%dмин)\n",
			icon, signal.Symbol, changePercent, signal.PeriodMinutes)
	} else {
		icon = "🔴"
		fmt.Printf("%s %s ↓%.2f%% (%dмин)\n",
			icon, signal.Symbol, -changePercent, signal.PeriodMinutes)
	}
}

// formatPeriod форматирует период для отображения
func (gm *GrowthMonitor) formatPeriod(minutes int) string {
	switch {
	case minutes < 60:
		return fmt.Sprintf("%d мин", minutes)
	case minutes == 60:
		return "1 час"
	case minutes < 1440:
		return fmt.Sprintf("%d часов", minutes/60)
	default:
		return fmt.Sprintf("%d дней", minutes/1440)
	}
}

// logSignal логирует сигнал в файл
// func (gm *GrowthMonitor) logSignal(signal types.GrowthSignal) {
// 	timestamp := time.Now().Format("2006/01/02 15:04:05")
// 	changePercent := signal.GrowthPercent + signal.FallPercent

// 	fmt.Printf("📝 [%s] Сигнал записан: %s %s %.2f%% (период: %d мин)\n",
// 		timestamp,
// 		signal.Symbol,
// 		signal.Direction,
// 		changePercent,
// 		signal.PeriodMinutes)
// }

func (gm *GrowthMonitor) logSignal(signal types.GrowthSignal) {
	// Только для логирования в файл, не выводить в консоль
	// Это поможет избежать дублирования вывода
}

// GetSignals возвращает канал сигналов
func (gm *GrowthMonitor) GetSignals() <-chan types.GrowthSignal {
	return gm.signals
}

// GetGrowthStats возвращает статистику по мониторингу роста
func (gm *GrowthMonitor) GetGrowthStats() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	totalSignals := len(gm.signalsHistory)
	growthSignals := 0
	fallSignals := 0

	// Считаем сигналы за последние 5 минут
	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)

	for _, signal := range gm.signalsHistory {
		if signal.Timestamp.After(fiveMinutesAgo) {
			if signal.Direction == "growth" {
				growthSignals++
			} else {
				fallSignals++
			}
		}
	}

	return map[string]interface{}{
		"total_signals":      totalSignals,
		"growth_signals":     growthSignals,
		"fall_signals":       fallSignals,
		"monitoring_periods": gm.config.GrowthPeriods,
		"growth_threshold":   gm.config.GrowthThreshold,
		"fall_threshold":     gm.config.FallThreshold,
		"active":             gm.active,
	}
}

// GetDetailedStats возвращает детальную статистику
func (gm *GrowthMonitor) GetDetailedStats() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_signals":  len(gm.signalsHistory),
		"growth_signals": 0,
		"fall_signals":   0,
		"active":         gm.active,
		"last_check":     time.Now(),
	}

	// Группируем по периодам
	periodStats := make(map[int]int)

	// Считаем сигналы за последние 5 минут
	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)

	for _, signal := range gm.signalsHistory {
		if signal.Timestamp.After(fiveMinutesAgo) {
			if signal.Direction == "growth" {
				stats["growth_signals"] = stats["growth_signals"].(int) + 1
			} else {
				stats["fall_signals"] = stats["fall_signals"].(int) + 1
			}
			periodStats[signal.PeriodMinutes] = periodStats[signal.PeriodMinutes] + 1
		}
	}

	stats["period_stats"] = periodStats
	return stats
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

// GetSymbolsToMonitor возвращает список символов для мониторинга с учетом конфигурации
func (gm *GrowthMonitor) GetSymbolsToMonitor() ([]string, error) {
	// Получаем все символы из priceMonitor
	allSymbols := gm.priceMonitor.GetSymbols()

	// Если в конфигурации указаны конкретные символы для мониторинга
	if gm.config.SymbolFilter != "" {
		// Парсим фильтр символов
		filterSymbols := gm.parseSymbolFilter(gm.config.SymbolFilter)

		// Если фильтр "all" или пустой массив, используем все символы
		if len(filterSymbols) == 0 {
			// Используем все символы, но с ограничением по количеству если указано
			symbols := allSymbols

			// Ограничиваем количество если указано
			if gm.config.MaxSymbolsToMonitor > 0 && len(symbols) > gm.config.MaxSymbolsToMonitor {
				// Сортируем по объему и берем топ-N
				symbols = gm.filterByVolume(symbols, gm.config.MaxSymbolsToMonitor)
			}

			return symbols, nil
		}

		// Фильтруем только те, что есть в общем списке
		var symbols []string
		for _, symbol := range filterSymbols {
			for _, availableSymbol := range allSymbols {
				if strings.EqualFold(symbol, availableSymbol) {
					symbols = append(symbols, availableSymbol)
					break
				}
			}
		}

		if len(symbols) == 0 {
			log.Printf("⚠️ Не найдено символов по фильтру, использую все доступные")
			symbols = allSymbols
		}

		// Ограничиваем количество если указано
		if gm.config.MaxSymbolsToMonitor > 0 && len(symbols) > gm.config.MaxSymbolsToMonitor {
			// Сортируем по объему и берем топ-N
			symbols = gm.filterByVolume(symbols, gm.config.MaxSymbolsToMonitor)
		}

		return symbols, nil
	}

	// Если фильтр не указан, используем все символы с ограничением
	symbols := allSymbols

	// Ограничиваем количество если указано
	if gm.config.MaxSymbolsToMonitor > 0 && len(symbols) > gm.config.MaxSymbolsToMonitor {
		// Сортируем по объему и берем топ-N
		symbols = gm.filterByVolume(symbols, gm.config.MaxSymbolsToMonitor)
	}

	return symbols, nil
}

// parseSymbolFilter парсит фильтр символов
func (gm *GrowthMonitor) parseSymbolFilter(filter string) []string {
	// Если фильтр "all", возвращаем пустой массив - это будет означать все символы
	if strings.ToUpper(strings.TrimSpace(filter)) == "ALL" {
		return []string{} // Пустой массив = все символы
	}

	var symbols []string

	// Поддерживаем несколько форматов:
	// 1. Через запятую: BTCUSDT,ETHUSDT,BNBUSDT
	// 2. Через пробел: BTCUSDT ETHUSDT BNBUSDT
	// 3. С префиксом: BTC,ETH,BNB (автоматически добавляем USDT)

	// Разделяем по запятой
	if strings.Contains(filter, ",") {
		parts := strings.Split(filter, ",")
		for _, part := range parts {
			symbol := strings.TrimSpace(part)
			if symbol != "" {
				// Автоматически добавляем USDT если нужно
				if !strings.HasSuffix(strings.ToUpper(symbol), "USDT") {
					symbol = strings.ToUpper(symbol) + "USDT"
				}
				symbols = append(symbols, symbol)
			}
		}
	} else {
		// Разделяем по пробелу
		parts := strings.Fields(filter)
		for _, part := range parts {
			symbol := strings.TrimSpace(part)
			if symbol != "" && strings.ToUpper(symbol) != "ALL" {
				if !strings.HasSuffix(strings.ToUpper(symbol), "USDT") {
					symbol = strings.ToUpper(symbol) + "USDT"
				}
				symbols = append(symbols, symbol)
			}
		}
	}

	return symbols
}

// filterByVolume фильтрует символы по объему
func (gm *GrowthMonitor) filterByVolume(symbols []string, maxCount int) []string {
	if len(symbols) <= maxCount {
		return symbols
	}

	// Получаем объемы для символов
	volumes, err := gm.client.GetSymbolVolume(symbols)
	if err != nil {
		log.Printf("⚠️ Не удалось получить объемы: %v, использую первые %d символов", err, maxCount)
		return symbols[:maxCount]
	}

	// Создаем структуру для сортировки
	type SymbolVolume struct {
		Symbol string
		Volume float64
	}

	var sv []SymbolVolume
	for _, symbol := range symbols {
		if volume, ok := volumes[symbol]; ok {
			sv = append(sv, SymbolVolume{symbol, volume})
		} else {
			sv = append(sv, SymbolVolume{symbol, 0})
		}
	}

	// Сортируем по объему (по убыванию)
	sort.Slice(sv, func(i, j int) bool {
		return sv[i].Volume > sv[j].Volume
	})

	// Берем топ-N
	result := make([]string, maxCount)
	for i := 0; i < maxCount; i++ {
		result[i] = sv[i].Symbol
	}

	return result
}

// FlushDisplay очищает и выводит накопленные сигналы
func (gm *GrowthMonitor) FlushDisplay() {
	if gm.display != nil {
		gm.display.Flush()
	}
}

// SendTelegramTest отправляет тестовое сообщение в Telegram
func (gm *GrowthMonitor) SendTelegramTest() error {
	if gm.telegramBot != nil {
		return gm.telegramBot.SendTestMessage()
	}
	return fmt.Errorf("telegram bot not initialized")
}
