package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/monitor"
	"crypto-exchange-screener-bot/internal/telegram"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("Не удалось загрузить конфигурацию: %v", err)
	}

	// Выводим информацию о конфигурации
	printHeader("МОНИТОР РОСТА КРИПТОВАЛЮТНЫХ ФЬЮЧЕРСОВ")
	fmt.Printf("🔧 Конфигурация:\n")
	fmt.Printf("   Сеть: %s\n", map[bool]string{true: "Testnet 🧪", false: "Mainnet ⚡"}[cfg.UseTestnet])
	fmt.Printf("   Категория: %s фьючерсы\n", cfg.FuturesCategory)
	fmt.Printf("   Интервал обновления: %d секунд\n", cfg.UpdateInterval)
	fmt.Printf("   Периоды роста: %s\n", formatGrowthPeriods(cfg.GrowthPeriods))
	fmt.Printf("   Порог роста: %.2f%%\n", cfg.GrowthThreshold)
	fmt.Printf("   Порог падения: %.2f%%\n", cfg.FallThreshold)

	// Показываем настройки фильтров
	if cfg.SymbolFilter == "all" {
		fmt.Printf("   Режим: ВСЕ СИМВОЛЫ\n")
	} else if cfg.SymbolFilter != "" {
		fmt.Printf("   Фильтр символов: %s\n", cfg.SymbolFilter)
	}
	if cfg.MaxSymbolsToMonitor > 0 {
		fmt.Printf("   Макс. символов: %d\n", cfg.MaxSymbolsToMonitor)
	}
	fmt.Printf("   Мин. объем: $%.0f\n", cfg.MinVolumeFilter)
	fmt.Println()

	// Переменные для статистики
	startTime := time.Now()
	var updateCount int32 = 0
	var signalCount int32 = 0

	// Создаем монитор цен
	priceMonitor := monitor.NewPriceMonitor(cfg)

	// Получаем все фьючерсные USDT пары с фильтрацией
	fmt.Println("📈 Получение фьючерсных торговых пар...")

	var pairs []string
	if cfg.SymbolFilter == "all" {
		// Режим ALL - получаем все пары с фильтрацией по объему
		allPairs, err := priceMonitor.GetAllFuturesPairs(
			cfg.MinVolumeFilter,
			0,    // Без ограничения по количеству
			true, // Сортировать по объему
		)
		if err != nil {
			log.Fatalf("Не удалось получить фьючерсные пары: %v", err)
		}
		pairs = allPairs

		// Ограничиваем количество если указано
		if cfg.MaxSymbolsToMonitor > 0 && len(pairs) > cfg.MaxSymbolsToMonitor {
			pairs = pairs[:cfg.MaxSymbolsToMonitor]
			fmt.Printf("⚠️  Ограничено %d символами (MAX_SYMBOLS_TO_MONITOR)\n", cfg.MaxSymbolsToMonitor)
		}
	} else if cfg.SymbolFilter != "" {
		// Фильтр по конкретным символам
		allPairs, err := priceMonitor.GetAllFuturesPairs(
			cfg.MinVolumeFilter,
			0,    // Без ограничения по количеству
			true, // Сортировать по объему
		)
		if err != nil {
			log.Fatalf("Не удалось получить фьючерсные пары: %v", err)
		}

		// Фильтруем по SYMBOL_FILTER
		filterMap := make(map[string]bool)
		filterParts := strings.Split(strings.ToUpper(cfg.SymbolFilter), ",")
		for _, part := range filterParts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			filterMap[part] = true
			// Также добавляем версию с USDT если не указана
			if !strings.HasSuffix(part, "USDT") {
				filterMap[part+"USDT"] = true
			}
		}

		for _, pair := range allPairs {
			baseSymbol := strings.TrimSuffix(strings.ToUpper(pair), "USDT")
			if filterMap[pair] || filterMap[baseSymbol] {
				pairs = append(pairs, pair)
			}
		}

		// Если после фильтрации нет пар, используем первые N
		if len(pairs) == 0 && len(allPairs) > 0 {
			maxPairs := cfg.MaxSymbolsToMonitor
			if maxPairs <= 0 {
				maxPairs = 10
			}
			if maxPairs > len(allPairs) {
				maxPairs = len(allPairs)
			}
			pairs = allPairs[:maxPairs]
			fmt.Printf("⚠️  Фильтр не дал результатов, используем первые %d пар\n", maxPairs)
		}
	} else {
		// Если фильтр не задан, получаем ограниченное количество
		maxPairs := cfg.MaxSymbolsToMonitor
		if maxPairs <= 0 {
			maxPairs = 50
		}
		pairs, err = priceMonitor.GetAllFuturesPairs(
			cfg.MinVolumeFilter,
			maxPairs,
			true,
		)
		if err != nil {
			log.Fatalf("Не удалось получить фьючерсные пары: %v", err)
		}
	}

	fmt.Printf("✅ Найдено %d фьючерсных USDT-пар (фильтр: $%.0f)\n",
		len(pairs), cfg.MinVolumeFilter)

	// Показываем топ-10 символов по объему
	if len(pairs) > 0 {
		showCount := 10
		if len(pairs) < showCount {
			showCount = len(pairs)
		}
		fmt.Printf("   Топ-%d по объему: %s\n",
			showCount,
			strings.Join(pairs[:showCount], ", "))
	}
	fmt.Println()

	// Создаем монитор роста
	fmt.Println("📈 Инициализация монитора роста...")
	growthMonitor := monitor.NewGrowthMonitor(cfg, priceMonitor)

	// Инициализируем Telegram бота если включен
	if cfg.TelegramEnabled && cfg.TelegramAPIKey != "" {
		fmt.Println("🤖 Telegram бот инициализирован")

		// Запускаем webhook сервер если указан порт
		if cfg.TelegramWebhookPort != "" && cfg.TelegramWebhookURL != "" {
			telegramBot := telegram.NewTelegramBot(cfg)
			webhookServer := telegram.NewWebhookServer(
				telegramBot,
				cfg.TelegramWebhookPort,
				cfg.TelegramWebhookURL,
			)

			go func() {
				if err := webhookServer.Start(); err != nil {
					log.Printf("❌ Ошибка запуска Telegram webhook: %v", err)
				}
			}()

			fmt.Printf("🌐 Telegram webhook сервер запущен на порту %s\n", cfg.TelegramWebhookPort)
		}

		// Отправляем тестовое сообщение
		if cfg.TelegramChatID != 0 {
			go func() {
				time.Sleep(3 * time.Second) // Даем время на запуск
				if err := growthMonitor.SendTelegramTest(); err != nil {
					log.Printf("❌ Ошибка отправки тестового сообщения: %v", err)
				} else {
					fmt.Println("✅ Тестовое сообщение отправлено в Telegram")
				}
			}()
		}
	}

	fmt.Println("🎯 Режим вывода: КОМПАКТНЫЙ")
	fmt.Println("   Каждые 2 секунды будет групповой вывод сигналов")

	if cfg.TelegramEnabled {
		fmt.Printf("🤖 Telegram уведомления: ВКЛ\n")
		if cfg.TelegramNotifyOn.Growth {
			fmt.Printf("   Уведомления о росте: ВКЛ\n")
		}
		if cfg.TelegramNotifyOn.Fall {
			fmt.Printf("   Уведомления о падении: ВКЛ\n")
		}
	}
	fmt.Println()

	fmt.Println("🎯 Режим вывода: КОМПАКТНЫЙ")
	fmt.Println("   Каждые 2 секунды будет групповой вывод сигналов")
	fmt.Println()

	// Запускаем мониторинг цен
	priceMonitor.StartMonitoring(time.Duration(cfg.UpdateInterval) * time.Second)
	fmt.Printf("🔄 Мониторинг цен запущен (обновление каждые %d сек)\n", cfg.UpdateInterval)

	// Даем время на первоначальную загрузку данных
	fmt.Println("📥 Загрузка первоначальных данных...")
	time.Sleep(5 * time.Second)
	fmt.Println("✅ Данные загружены")
	fmt.Println()

	// Запускаем мониторинг роста
	growthMonitor.Start()

	fmt.Println("🚀 Мониторинг роста запущен")

	// Горутина для периодического вывода накопленных сигналов
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			growthMonitor.FlushDisplay()
		}
	}()

	// Обработка сигналов роста в отдельной горутине
	go func() {
		for range growthMonitor.GetSignals() {
			// Увеличиваем счетчик сигналов
			atomic.AddInt32(&signalCount, 1)

			// Выводим информацию о сигнале в новом формате
			// ВЫВОД ТЕПЕРЬ ДЕЛАЕТ DisplayManager - УДАЛИТЬ ЭТОТ ВЫВОД
			// timestamp := time.Now().Format("2006/01/02 15:04:05")
			// changePercent := signal.GrowthPercent + signal.FallPercent
			// fmt.Printf("📈 [%s] Получен сигнал: %s %s %.2f%% (период: %d мин)\n",
			//     timestamp,
			//     signal.Symbol,
			//     signal.Direction,
			//     changePercent,
			//     signal.PeriodMinutes)
		}
	}()

	// Обработка сигналов для graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем HTTP сервер (если включен)
	if cfg.HttpEnabled {
		go func() {
			fmt.Printf("🌐 Запуск HTTP сервера на порту %s...\n", cfg.HttpPort)
			priceMonitor.StartHTTPServer(cfg.HttpPort)
		}()
		fmt.Printf("   API доступен по адресу: http://localhost:%s\n", cfg.HttpPort)
		fmt.Println()
	}

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

	// Горутина для статистики роста
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := growthMonitor.GetGrowthStats()
			detailedStats := growthMonitor.GetDetailedStats()

			fmt.Printf("📈 Статистика роста: %d сигналов (↑%d ↓%d)\n",
				stats["total_signals"],
				stats["growth_signals"],
				stats["fall_signals"])

			// Выводим детальную статистику по периодам
			if periodStats, ok := detailedStats["period_stats"].(map[int]int); ok {
				fmt.Printf("   Периоды: ")
				for period, count := range periodStats {
					fmt.Printf("%dмин:%d ", period, count)
				}
				fmt.Println()
			}
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
			growthStats := growthMonitor.GetGrowthStats()

			printStats(startTime, int(currentUpdates), int(currentSignals),
				cfg, len(pairs), iteration, growthStats)
			iteration++
		}
	}()

	// Выводим информацию о горячих клавишах
	fmt.Println("🎮 Управление:")
	fmt.Println("   Ctrl+C - Остановить бота")
	fmt.Println()
	printSeparator()

	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("           МОНИТОР РОСТА - BYBIT FUTURES         ")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println()

	// Ожидание сигнала остановки
	<-stopChan

	fmt.Println()
	printHeader("Завершение работы")
	fmt.Printf("⏱️  Время работы: %s\n", formatDuration(time.Since(startTime)))
	fmt.Printf("📊 Всего обновлений цен: %d\n", atomic.LoadInt32(&updateCount))
	fmt.Printf("📈 Всего обнаружено сигналов: %d\n", atomic.LoadInt32(&signalCount))

	// Получаем финальную статистику по сигналам роста
	growthStats := growthMonitor.GetGrowthStats()
	fmt.Printf("📈 Сигналы роста: %d (↑%d ↓%d)\n",
		growthStats["total_signals"],
		growthStats["growth_signals"],
		growthStats["fall_signals"])

	// Остановка мониторинга
	priceMonitor.StopMonitoring()
	growthMonitor.Stop()

	fmt.Println("✅ Бот остановлен корректно")
}

func printStats(startTime time.Time, updates int, signals int,
	cfg *config.Config, totalSymbols int, iteration int,
	growthStats map[string]interface{}) {

	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("📊 СТАТУС СИСТЕМЫ (итерация #%d)\n", iteration)
	fmt.Printf("   ⏱️  Время работы: %s\n", formatDuration(time.Since(startTime)))
	fmt.Printf("   🔄 Обновлений цен: %d\n", updates)
	fmt.Printf("   📈 Всего сигналов: %d\n", signals)

	// Добавляем статистику роста
	if growthStats != nil {
		fmt.Printf("   📊 Сигналы роста: %d (↑%d ↓%d)\n",
			growthStats["total_signals"],
			growthStats["growth_signals"],
			growthStats["fall_signals"])
	}

	fmt.Printf("   📈 Всего пар: %d\n", totalSymbols)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("   💾 Память: %.2f MB\n", float64(m.Alloc)/1024/1024)
	fmt.Printf("   🧵 Горутин: %d\n", runtime.NumGoroutine())
	fmt.Printf("   🕐 Текущее время: %s\n", time.Now().Format("15:04:05"))
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
}

func printHeader(text string) {
	width := 80
	padding := (width - len(text)) / 2

	if padding < 0 {
		padding = 0
	}

	fmt.Println(strings.Repeat("═", width))
	fmt.Printf("%s%s%s\n",
		strings.Repeat(" ", padding),
		text,
		strings.Repeat(" ", width-len(text)-padding))
	fmt.Println(strings.Repeat("═", width))
}

func printSeparator() {
	fmt.Println(strings.Repeat("─", 80))
}

func formatGrowthPeriods(periods []int) string {
	var result []string
	for _, period := range periods {
		if period < 60 {
			result = append(result, fmt.Sprintf("%dм", period))
		} else if period == 60 {
			result = append(result, "1ч")
		} else if period < 1440 {
			result = append(result, fmt.Sprintf("%dч", period/60))
		} else {
			result = append(result, fmt.Sprintf("%dд", period/1440))
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
