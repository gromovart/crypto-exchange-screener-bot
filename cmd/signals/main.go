// cmd/signals/main.go (исправленная версия)
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/monitor"
)

func main() {
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("      CRYPTO FUTURES SIGNAL MONITOR - BYBIT       ")
	fmt.Println("══════════════════════════════════════════════════")

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Настраиваем для фьючерсов
	cfg.FuturesCategory = "linear"
	cfg.UpdateInterval = 5
	cfg.AlertThreshold = 0.1
	cfg.HttpEnabled = false

	fmt.Printf("🔧 Конфигурация:\n")
	fmt.Printf("   Сеть: %s\n", map[bool]string{true: "Testnet 🧪", false: "Mainnet ⚡"}[cfg.UseTestnet])
	fmt.Printf("   Категория фьючерсов: %s\n", cfg.FuturesCategory)
	fmt.Printf("   Порог сигнала: %.2f%%\n", cfg.AlertThreshold)
	fmt.Printf("   Интервал проверки: %d сек\n", cfg.UpdateInterval)
	fmt.Println()

	// Создаем монитор цен
	priceMonitor := monitor.NewPriceMonitor(cfg)

	// Получаем фьючерсные пары
	fmt.Println("📈 Получение фьючерсных торговых пар...")
	pairs, err := priceMonitor.FetchAllFuturesPairs()
	if err != nil {
		log.Fatalf("Failed to fetch futures pairs: %v", err)
	}

	// Выбираем топ-10 фьючерсных пар для мониторинга
	var symbolsToMonitor []string
	topFuturesSymbols := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT",
		"ADAUSDT", "DOGEUSDT", "MATICUSDT", "DOTUSDT", "AVAXUSDT",
		"LINKUSDT", "UNIUSDT", "LTCUSDT", "ATOMUSDT", "ETCUSDT",
	}

	for _, symbol := range topFuturesSymbols {
		for _, pair := range pairs {
			if pair == symbol {
				symbolsToMonitor = append(symbolsToMonitor, symbol)
				break
			}
		}
		if len(symbolsToMonitor) >= 10 {
			break
		}
	}

	fmt.Printf("✅ Отслеживается %d фьючерсных пар:\n", len(symbolsToMonitor))
	for i, symbol := range symbolsToMonitor {
		fmt.Printf("   %d. %s\n", i+1, symbol)
	}
	fmt.Println()

	// Конвертируем интервалы
	var intervals []monitor.Interval
	trackedIntervals := []int{1, 5, 15} // Только короткие интервалы для теста
	for _, interval := range trackedIntervals {
		intervals = append(intervals, monitor.Interval(fmt.Sprintf("%d", interval)))
	}

	fmt.Printf("⏱️  Отслеживаемые интервалы: 1 мин, 5 мин, 15 мин\n")
	fmt.Println()

	// Создаем монитор сигналов
	signalMonitor := monitor.NewSignalMonitor(priceMonitor, cfg.AlertThreshold)

	// Запускаем мониторинг цен
	priceMonitor.StartMonitoring(time.Duration(cfg.UpdateInterval) * time.Second)

	// Даем время на первоначальную загрузку
	fmt.Println("🔄 Загрузка первоначальных данных...")
	time.Sleep(5 * time.Second)

	fmt.Println("🚀 Система сигналов запущена!")
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Println()

	// Переменные для статистики
	var totalSignals int
	startTime := time.Now()

	// Основной цикл проверки сигналов
	ticker := time.NewTicker(time.Duration(cfg.UpdateInterval) * time.Second)

	for {
		select {
		case <-ticker.C:
			currentTime := time.Now()

			// Выводим время проверки
			fmt.Printf("⏰ Проверка в %s | ", currentTime.Format("15:04:05"))
			fmt.Printf("Работаем: %s\n", formatDuration(currentTime.Sub(startTime)))
			fmt.Println(strings.Repeat("─", 50))

			signalsInThisCheck := 0
			for _, symbol := range symbolsToMonitor {
				for _, interval := range intervals {
					if signalMonitor.CheckSignalNow(symbol, interval) {
						signalsInThisCheck++
						totalSignals++
					}
				}
			}

			// Статистика проверки
			fmt.Printf("📊 Результат проверки:\n")
			fmt.Printf("   Обнаружено сигналов: %d\n", signalsInThisCheck)
			fmt.Printf("   Всего сигналов за сессию: %d\n", totalSignals)
			fmt.Printf("   Время до следующей проверки: %d сек\n", cfg.UpdateInterval)

			if signalsInThisCheck == 0 {
				fmt.Println("   ℹ️  Сигналов не обнаружено")
			}

			fmt.Println()
		}
	}
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dч %dм", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dм %dс", minutes, seconds)
	}
	return fmt.Sprintf("%dс", seconds)
}
