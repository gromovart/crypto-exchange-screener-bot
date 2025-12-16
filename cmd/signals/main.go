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
	fmt.Println("      МОНИТОР РОСТА КРИПТОВАЛЮТНЫХ ФЬЮЧЕРСОВ - BYBIT")
	fmt.Println("══════════════════════════════════════════════════")

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("Не удалось загрузить конфигурацию: %v", err)
	}

	// Настраиваем для роста
	cfg.FuturesCategory = "linear"
	cfg.UpdateInterval = 5
	cfg.HttpEnabled = false
	cfg.GrowthThreshold = 0.1
	cfg.FallThreshold = 0.1
	cfg.CheckContinuity = false

	fmt.Printf("🔧 Конфигурация:\n")
	fmt.Printf("   Сеть: %s\n", map[bool]string{true: "Testnet 🧪", false: "Mainnet ⚡"}[cfg.UseTestnet])
	fmt.Printf("   Категория фьючерсов: %s\n", cfg.FuturesCategory)
	fmt.Printf("   Порог роста: %.2f%%\n", cfg.GrowthThreshold)
	fmt.Printf("   Порог падения: %.2f%%\n", cfg.FallThreshold)

	// Показываем настройки фильтров
	if cfg.SymbolFilter != "" {
		fmt.Printf("   Фильтр символов: %s\n", cfg.SymbolFilter)
	}
	if cfg.MaxSymbolsToMonitor > 0 {
		fmt.Printf("   Макс. символов: %d\n", cfg.MaxSymbolsToMonitor)
	}
	if cfg.SignalFilters.Enabled {
		fmt.Printf("   Фильтры сигналов: ВКЛ\n")
		fmt.Printf("   Мин. уверенность: %.1f%%\n", cfg.SignalFilters.MinConfidence)
	}
	fmt.Println()

	// Создаем монитор цен
	priceMonitor := monitor.NewPriceMonitor(cfg)

	// Получаем ВСЕ фьючерсные пары с фильтрацией
	fmt.Println("📈 Получение всех фьючерсных торговых пар...")

	// Используем новый метод для получения всех пар
	var allPairs []string
	if cfg.SymbolFilter == "all" {
		// Режим ALL - получаем все пары
		allPairs, err = priceMonitor.GetAllFuturesPairs(
			cfg.MinVolumeFilter, // Минимальный объем
			0,                   // Без ограничения по количеству
			true,                // Сортировать по объему
		)
		fmt.Printf("✅ Режим ALL: отслеживаются ВСЕ фьючерсные пары\n")
	} else if cfg.SymbolFilter != "" {
		// Если задан фильтр символов, получаем все пары а затем фильтруем
		allPairs, err = priceMonitor.GetAllFuturesPairs(
			cfg.MinVolumeFilter, // Минимальный объем
			0,                   // Без ограничения по количеству
			true,                // Сортировать по объему
		)
	} else {
		// Иначе используем стандартный метод
		allPairs, err = priceMonitor.FetchAllFuturesPairs()
	}

	if err != nil {
		log.Fatalf("Не удалось получить фьючерсные пары: %v", err)
	}

	fmt.Printf("✅ Найдено %d фьючерсных USDT-пар\n", len(allPairs))

	// Показываем топ-20 символов
	if len(allPairs) > 0 {
		showCount := 20
		if len(allPairs) < showCount {
			showCount = len(allPairs)
		}
		fmt.Printf("   Топ-%d по объему: %s\n",
			showCount,
			strings.Join(allPairs[:showCount], ", "))
	}
	fmt.Println()

	// Создаем growth monitor
	growthMonitor := monitor.NewGrowthMonitor(cfg, priceMonitor)

	// Запускаем мониторинг цен
	priceMonitor.StartMonitoring(time.Duration(cfg.UpdateInterval) * time.Second)

	// Даем время на первоначальную загрузку
	fmt.Println("🔄 Загрузка первоначальных данных...")
	time.Sleep(5 * time.Second)

	// Запускаем growth monitor
	growthMonitor.Start()
	fmt.Println("🚀 Монитор роста запущен!")
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Println()

	// Переменные для статистики
	var totalSignals int
	startTime := time.Now()

	// Обработка сигналов роста
	go func() {
		for signal := range growthMonitor.GetSignals() {
			totalSignals++

			var icon, direction, changeStr string
			if signal.Direction == "growth" {
				icon = "🟢"
				direction = "РОСТ"
				changeStr = fmt.Sprintf("+%.4f%%", signal.GrowthPercent)
			} else {
				icon = "🔴"
				direction = "ПАДЕНИЕ"
				changeStr = fmt.Sprintf("-%.4f%%", signal.FallPercent)
			}

			fmt.Println("══════════════════════════════════════════════════")
			fmt.Printf("%s %s ОБНАРУЖЕН!\n", icon, direction)
			fmt.Printf("   Символ: %s\n", signal.Symbol)
			fmt.Printf("   Изменение: %s\n", changeStr)
			fmt.Printf("   Период: %d минут\n", signal.PeriodMinutes)
			fmt.Printf("   Время: %s\n", signal.Timestamp.Format("15:04:05"))
			fmt.Printf("   Уверенность: %.1f%%\n", signal.Confidence)
			fmt.Printf("   Начальная цена: %.4f\n", signal.StartPrice)
			fmt.Printf("   Конечная цена: %.4f\n", signal.EndPrice)
			fmt.Printf("🔗 https://www.bybit.com/trade/usdt/%s\n", signal.Symbol)
			fmt.Println("══════════════════════════════════════════════════")
			fmt.Println()
		}
	}()

	// Основной цикл для статистики
	ticker := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-ticker.C:
			currentTime := time.Now()

			// Выводим время и статистику
			fmt.Printf("⏰ Статистика в %s | ", currentTime.Format("15:04:05"))
			fmt.Printf("Работаем: %s\n", formatDuration(currentTime.Sub(startTime)))
			fmt.Println(strings.Repeat("─", 50))

			stats := growthMonitor.GetGrowthStats()
			fmt.Printf("📊 Статистика роста:\n")
			fmt.Printf("   Всего сигналов: %d\n", stats["total_signals"])
			fmt.Printf("   Сигналов роста: %d\n", stats["growth_signals"])
			fmt.Printf("   Сигналов падения: %d\n", stats["fall_signals"])
			fmt.Printf("   Всего за сессию: %d\n", totalSignals)
			fmt.Printf("   Отслеживаем символов: %d\n", len(allPairs))
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
