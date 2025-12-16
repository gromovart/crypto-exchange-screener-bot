// cmd/bot/main.go
package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/monitor"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func RunMainBot() {
	main() // Просто вызываем основную функцию
}

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Выводим информацию о конфигурации
	printHeader("Crypto Exchange Screener Bot - FULL MODE")
	fmt.Printf("🔧 Конфигурация:\n")
	fmt.Printf("   Сеть: %s\n", map[bool]string{true: "Testnet 🧪", false: "Mainnet ⚡"}[cfg.UseTestnet])
	fmt.Printf("   Категория: %s фьючерсы\n", cfg.FuturesCategory)
	fmt.Printf("   Интервал обновления: %d секунд\n", cfg.UpdateInterval)
	fmt.Printf("   Отслеживаемые интервалы: %s\n", formatIntervals(cfg.TrackedIntervals))
	fmt.Printf("   Порог сигнала: %.2f%%\n", cfg.AlertThreshold)
	fmt.Println()

	// Создаем монитор цен
	priceMonitor := monitor.NewPriceMonitor(cfg)

	// Создаем монитор роста
	growthMonitor := monitor.NewGrowthMonitor(cfg, priceMonitor)

	// Запускаем мониторинг роста
	growthMonitor.Start()
	fmt.Println("📈 Growth monitoring started")

	// Обработка сигналов роста в отдельной горутине
	go func() {
		for signal := range growthMonitor.GetSignals() {
			// Обработка сигналов роста
			log.Printf("🎯 Growth signal: %s %s %.2f%%",
				signal.Symbol, signal.Direction,
				signal.GrowthPercent+signal.FallPercent)
		}
	}()

	// Получаем все фьючерсные USDT пары
	fmt.Println("📈 Получение фьючерсных торговых пар...")
	pairs, err := priceMonitor.FetchAllFuturesPairs()
	if err != nil {
		log.Fatalf("Failed to fetch futures pairs: %v", err)
	}

	fmt.Printf("✅ Найдено %d фьючерсных USDT-пар\n", len(pairs))
	fmt.Println()

	// Выбираем символы для мониторинга
	symbolsToMonitor := selectSymbolsForMonitoring(pairs, 15)
	fmt.Printf("🎯 Отслеживается %d символов:\n", len(symbolsToMonitor))
	for i, symbol := range symbolsToMonitor {
		fmt.Printf("   %d. %s\n", i+1, symbol)
	}
	fmt.Println()

	// Конвертируем интервалы
	var intervals []monitor.Interval
	trackedIntervals := cfg.TrackedIntervals
	if len(trackedIntervals) > 3 {
		trackedIntervals = trackedIntervals[:3] // Берем только первые 3 интервала для теста
	}

	for _, interval := range trackedIntervals {
		intervals = append(intervals, monitor.Interval(fmt.Sprintf("%d", interval)))
	}

	fmt.Printf("⏱️  Отслеживаемые интервалы: %s\n", formatIntervals(trackedIntervals))
	fmt.Println()

	// Запускаем мониторинг цен
	priceMonitor.StartMonitoring(time.Duration(cfg.UpdateInterval) * time.Second)
	fmt.Printf("🔄 Мониторинг цен запущен (обновление каждые %d сек)\n", cfg.UpdateInterval)

	// Создаем монитор сигналов
	signalMonitor := monitor.NewSignalMonitor(priceMonitor, cfg.AlertThreshold)
	fmt.Println("🚨 Мониторинг сигналов инициализирован")
	fmt.Println()

	// Даем время на первоначальную загрузку данных
	fmt.Println("📥 Загрузка первоначальных данных...")
	time.Sleep(5 * time.Second)
	fmt.Println("✅ Данные загружены")
	fmt.Println()

	// Запускаем HTTP сервер (если включен)
	if cfg.HttpEnabled {
		go func() {
			fmt.Printf("🌐 Запуск HTTP сервера на порту %s...\n", cfg.HttpPort)
			priceMonitor.StartHTTPServer(cfg.HttpPort)
		}()
		fmt.Printf("   API доступен по адресу: http://localhost:%s\n", cfg.HttpPort)
		fmt.Println()
	}

	// Обработка сигналов для graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Переменные для статистики
	startTime := time.Now()
	var updateCount int32 = 0
	var signalCount int32 = 0
	totalSymbols := len(pairs)

	// Канал для сигналов (пока не используется, но оставим)
	signalChan := make(chan monitor.Signal, 100)

	// Горутина для проверки сигналов
	go func() {
		fmt.Println("🔍 Горутина проверки сигналов запущена")

		ticker := time.NewTicker(10 * time.Second) // Проверяем каждые 10 секунд
		defer ticker.Stop()

		checkCounter := 1

		for range ticker.C {
			fmt.Printf("👁️  Проверка #%d в %s\n",
				checkCounter, time.Now().Format("15:04:05"))
			fmt.Println(strings.Repeat("─", 40))

			var wg sync.WaitGroup
			for _, symbol := range symbolsToMonitor {
				for _, interval := range intervals {
					wg.Add(1)
					go func(s string, i monitor.Interval) {
						defer wg.Done()
						signalMonitor.CheckSignalNow(s, i)
					}(symbol, interval)
				}
			}
			wg.Wait()

			checkCounter++
			fmt.Println()
		}
	}()

	// Горутина для сбора статистики обновлений
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.UpdateInterval) * time.Second)
		defer ticker.Stop()

		counter := 1
		for range ticker.C {
			atomic.AddInt32(&updateCount, 1)
			fmt.Printf("📊 Обновление цен #%d завершено в %s\n",
				counter,
				time.Now().Format("15:04:05"))
			counter++
		}
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := growthMonitor.GetGrowthStats()
			fmt.Printf("📊 Growth Stats: %d signals (↑%d ↓%d)\n",
				stats["total_signals"],
				stats["growth_signals"],
				stats["fall_signals"])
		}
	}()

	// Горутина для периодического вывода статистики
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		iteration := 1
		for range ticker.C {
			currentUpdates := atomic.LoadInt32(&updateCount)
			currentSignals := atomic.LoadInt32(&signalCount)

			printStats(startTime, int(currentUpdates), int(currentSignals), cfg, totalSymbols, iteration)
			iteration++
		}
	}()

	// Выводим информацию о горячих клавишах
	fmt.Println("🎮 Управление:")
	fmt.Println("   Ctrl+C - Остановить бота")
	fmt.Println()
	printSeparator()

	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("                СИСТЕМА СИГНАЛОВ                  ")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println()

	// Ожидание сигнала остановки
	<-stopChan

	fmt.Println()
	printHeader("Завершение работы")
	fmt.Printf("⏱️  Время работы: %s\n", formatDuration(time.Since(startTime)))
	fmt.Printf("📊 Всего обновлений цен: %d\n", atomic.LoadInt32(&updateCount))
	fmt.Printf("🚨 Всего обнаружено сигналов: %d\n", atomic.LoadInt32(&signalCount))

	// Остановка мониторинга
	priceMonitor.StopMonitoring()
	close(signalChan)

	fmt.Println("✅ Бот остановлен корректно")
}

// Вспомогательные функции
func selectSymbolsForMonitoring(pairs []string, limit int) []string {
	popularSymbols := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT",
		"ADAUSDT", "DOGEUSDT", "MATICUSDT", "DOTUSDT", "AVAXUSDT",
		"LINKUSDT", "UNIUSDT", "LTCUSDT", "ATOMUSDT", "ETCUSDT",
	}

	var selected []string
	for _, symbol := range popularSymbols {
		for _, pair := range pairs {
			if pair == symbol && !contains(selected, symbol) {
				selected = append(selected, symbol)
				break
			}
		}
		if len(selected) >= limit {
			break
		}
	}

	// Если не нашли достаточно популярных, добавляем первые из списка
	if len(selected) < limit {
		for _, pair := range pairs {
			if !contains(selected, pair) {
				selected = append(selected, pair)
				if len(selected) >= limit {
					break
				}
			}
		}
	}

	return selected
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func printStats(startTime time.Time, updates int, signals int, cfg *config.Config, totalSymbols int, iteration int) {
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("📊 СТАТУС СИСТЕМЫ (итерация #%d)\n", iteration)
	fmt.Printf("   ⏱️  Время работы: %s\n", formatDuration(time.Since(startTime)))
	fmt.Printf("   🔄 Обновлений цен: %d\n", updates)
	fmt.Printf("   🚨 Сигналов: %d\n", signals)
	fmt.Printf("   📈 Всего пар: %d\n", totalSymbols)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("   💾 Память: %.2f MB\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("   🧵 Горутин: %d\n", runtime.NumGoroutine())
	fmt.Printf("   🕐 Текущее время: %s\n", time.Now().Format("15:04:05"))
	fmt.Printf("   ⏭️  След. проверка: %s\n",
		time.Now().Add(10*time.Second).Format("15:04:05"))
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
}

func printHeader(text string) {
	width := 50
	padding := (width - len(text)) / 2
	fmt.Println(strings.Repeat("═", width))
	fmt.Printf("%s%s%s\n",
		strings.Repeat(" ", padding),
		text,
		strings.Repeat(" ", width-len(text)-padding))
	fmt.Println(strings.Repeat("═", width))
}

func printSeparator() {
	fmt.Println(strings.Repeat("─", 50))
}

func formatIntervals(intervals []int) string {
	var result []string
	for _, interval := range intervals {
		switch interval {
		case 1:
			result = append(result, "1м")
		case 5:
			result = append(result, "5м")
		case 10:
			result = append(result, "10м")
		case 15:
			result = append(result, "15м")
		case 30:
			result = append(result, "30м")
		case 60:
			result = append(result, "1ч")
		case 120:
			result = append(result, "2ч")
		case 240:
			result = append(result, "4ч")
		case 480:
			result = append(result, "8ч")
		case 720:
			result = append(result, "12ч")
		case 1440:
			result = append(result, "1д")
		case 10080:
			result = append(result, "7д")
		case 43200:
			result = append(result, "30д")
		default:
			result = append(result, fmt.Sprintf("%dм", interval))
		}
	}
	return strings.Join(result, ", ")
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dч %dм %dс", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dм %dс", minutes, seconds)
	}
	return fmt.Sprintf("%dс", seconds)
}
