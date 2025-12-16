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
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Выводим информацию о конфигурации
	printHeader("Crypto Exchange Screener Bot")
	fmt.Printf("🔧 Конфигурация:\n")
	fmt.Printf("   Сеть: %s\n", map[bool]string{true: "Testnet 🧪", false: "Mainnet ⚡"}[cfg.UseTestnet])
	fmt.Printf("   Интервал обновления: %d секунд\n", cfg.UpdateInterval)
	fmt.Printf("   Отслеживаемые интервалы: %s\n", formatIntervals(cfg.TrackedIntervals))
	fmt.Println()

	// Создаем монитор цен
	priceMonitor := monitor.NewPriceMonitor(cfg)

	// Получаем все USDT пары
	pairs, err := priceMonitor.FetchAllUSDTPairs()
	if err != nil {
		log.Fatalf("Failed to fetch USDT pairs: %v", err)
	}

	fmt.Printf("📈 Мониторинг %d USDT-пар\n", len(pairs))

	// Выводим примеры отслеживаемых пар
	if len(pairs) > 0 {
		fmt.Printf("   Примеры: %s\n", formatSymbolsPreview(pairs))
	}
	fmt.Println()

	// Запускаем мониторинг
	priceMonitor.StartMonitoring(time.Duration(cfg.UpdateInterval) * time.Second)
	fmt.Printf("🔄 Мониторинг запущен (обновление каждые %d сек)\n", cfg.UpdateInterval)

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
	totalSymbols := len(pairs)

	// Демонстрационная работа - выводим после первой загрузки данных
	go func() {
		time.Sleep(3 * time.Second)
		fmt.Println("📊 Первоначальные данные загружены")
		fmt.Println()
	}()

	// Горутина для сбора статистики обновлений
	go func() {
		fmt.Println("📊 Статус-горутина запущена")

		ticker := time.NewTicker(time.Duration(cfg.UpdateInterval) * time.Second)
		defer ticker.Stop()

		counter := 1
		for range ticker.C {
			atomic.AddInt32(&updateCount, 1)
			current := atomic.LoadInt32(&updateCount)
			fmt.Printf("🔄 Обновление #%d завершено в %s (всего: %d)\n",
				counter,
				time.Now().Format("15:04:05"),
				current)
			counter++
		}
	}()

	// Горутина для периодического вывода статистики каждые 10 секунд
	go func() {
		fmt.Println("📈 Статус-монитор запущен")

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		iteration := 1
		for range ticker.C {
			currentCount := atomic.LoadInt32(&updateCount)

			// Получаем упрощенную статистику (без блокировок)
			stats := getSimpleStats(startTime, int(currentCount), cfg, totalSymbols)

			// Выводим статус
			printSimpleStatus(stats, iteration)

			iteration++
		}
	}()

	// Пример получения данных для демонстрации
	go func() {
		time.Sleep(8 * time.Second) // Даем время на накопление данных

		// Пример получения изменения цены BTCUSDT за 1 час
		change, err := priceMonitor.GetPriceChange("BTCUSDT", monitor.Interval1Hour)
		if err != nil {
			fmt.Printf("⚠️  Ошибка получения данных BTCUSDT: %v\n", err)
		} else {
			fmt.Printf("💰 BTCUSDT (1 час): %s\n", formatPriceChange(change.ChangePercent))
		}

		// Пример получения топ-5 растущих монет за 24 часа
		topGainers, err := priceMonitor.GetTopPerformers(monitor.Interval24Hour, 5, false)
		if err == nil && len(topGainers) > 0 {
			fmt.Printf("🚀 Топ-5 роста (24ч):\n")
			for i, gainer := range topGainers {
				fmt.Printf("   %d. %-10s %s\n", i+1, gainer.Symbol, formatPriceChange(gainer.ChangePercent))
			}
		}
		fmt.Println()
	}()

	// Выводим информацию о горячих клавишах
	fmt.Println("🎮 Управление:")
	fmt.Println("   Ctrl+C - Остановить бота")
	fmt.Println()
	printSeparator()

	// Ожидание сигнала остановки
	<-stopChan

	fmt.Println()
	printHeader("Завершение работы")
	fmt.Printf("⏱️  Время работы: %s\n", formatDuration(time.Since(startTime)))
	fmt.Printf("📊 Всего обновлений: %d\n", atomic.LoadInt32(&updateCount))

	// Остановка мониторинга
	priceMonitor.StopMonitoring()

	fmt.Println("✅ Бот остановлен корректно")
}

// Вспомогательные функции для форматирования

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

func formatSymbolsPreview(pairs []string) string {
	if len(pairs) == 0 {
		return "нет данных"
	}

	// Берем первые 5 популярных пар
	popularSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT"}
	var result []string

	for _, symbol := range popularSymbols {
		for _, pair := range pairs {
			if pair == symbol {
				result = append(result, symbol)
				break
			}
		}
		if len(result) >= 3 {
			break
		}
	}

	if len(result) == 0 && len(pairs) > 0 {
		result = append(result, pairs[0])
		if len(pairs) > 1 {
			result = append(result, pairs[1])
		}
		if len(pairs) > 2 {
			result = append(result, "...")
		}
	}

	return strings.Join(result, ", ")
}

func formatPriceChange(change float64) string {
	if change > 0 {
		return fmt.Sprintf("🟢 +%.2f%%", change)
	} else if change < 0 {
		return fmt.Sprintf("🔴 %.2f%%", change)
	}
	return fmt.Sprintf("⚪ %.2f%%", change)
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

func getSimpleStats(startTime time.Time, updateCount int, cfg *config.Config, totalSymbols int) map[string]interface{} {
	stats := make(map[string]interface{})

	// Базовые метрики
	stats["uptime"] = formatDuration(time.Since(startTime))
	stats["updates"] = updateCount
	stats["symbols"] = totalSymbols // Используем сохраненное значение

	// Использование памяти
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats["memory_mb"] = float64(m.Alloc) / 1024 / 1024
	stats["goroutines"] = runtime.NumGoroutine()

	// Текущее время
	now := time.Now()
	stats["time"] = now.Format("15:04:05")

	// Рассчитываем время до следующего обновления
	stats["next_update"] = now.Add(time.Duration(cfg.UpdateInterval) * time.Second).Format("15:04:05")

	return stats
}

func printSimpleStatus(stats map[string]interface{}, iteration int) {
	printSeparator()
	fmt.Printf("📊 СТАТУС СИСТЕМЫ (итерация #%d)\n", iteration)
	fmt.Printf("   ⏱️  Время работы: %s\n", stats["uptime"])
	fmt.Printf("   🔄 Обновлений: %d\n", stats["updates"])
	fmt.Printf("   📈 Пар: %d\n", stats["symbols"])
	fmt.Printf("   💾 Память: %.2f MB\n", stats["memory_mb"])
	fmt.Printf("   🧵 Горутин: %d\n", stats["goroutines"])
	fmt.Printf("   🕐 Текущее время: %s\n", stats["time"])
	fmt.Printf("   ⏭️  След. обновление: %s\n", stats["next_update"])
	printSeparator()
}
