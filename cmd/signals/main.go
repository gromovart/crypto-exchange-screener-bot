// cmd/signals/main.go
package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/manager"
	"fmt"
	"log"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/storage"
	"crypto-exchange-screener-bot/internal/types"
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

	// Создаем DataManager
	fmt.Println("🚀 Инициализация DataManager...")
	dm, err := manager.NewDataManager(cfg)
	if err != nil {
		log.Fatalf("Не удалось создать DataManager: %v", err)
	}

	// Получаем компоненты из DataManager
	priceMonitor := dm.GetPriceMonitor()
	growthMonitor := dm.GetGrowthMonitor()
	storage := dm.GetStorage()

	// Получаем все фьючерсные пары
	fmt.Println("📈 Получение фьючерсных торговых пар...")

	allPairs, err := priceMonitor.GetAllFuturesPairs(
		cfg.MinVolumeFilter,
		100, // Ограничиваем 100 символами
		true,
	)
	if err != nil {
		log.Fatalf("Не удалось получить фьючерсные пары: %v", err)
	}

	fmt.Printf("✅ Найдено %d фьючерсных USDT-пар (фильтр: $%.0f)\n",
		len(allPairs), cfg.MinVolumeFilter)

	// Показываем топ-20 символов
	if len(allPairs) > 0 {
		showCount := min(20, len(allPairs))
		fmt.Printf("   Топ-%d по объему: %s\n",
			showCount,
			strings.Join(allPairs[:showCount], ", "))
	}
	fmt.Println()

	// Запускаем DataManager
	fmt.Println("🚀 Запуск DataManager...")
	if err := dm.Start(); err != nil {
		log.Fatalf("Не удалось запустить DataManager: %v", err)
	}

	// Даем время на первоначальную загрузку
	fmt.Println("🔄 Загрузка первоначальных данных...")
	time.Sleep(5 * time.Second)

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
			displaySimpleSignal(signal)
		}
	}()

	// Основной цикл для статистики
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentTime := time.Now()
			displayStatsWithStorage(currentTime, startTime, growthMonitor, storage, totalSignals, len(allPairs))
		}
	}
}

func displaySimpleSignal(signal types.GrowthSignal) {
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
	fmt.Printf("%s %s: %s %s за %d минут\n",
		icon, direction, signal.Symbol, changeStr,
		signal.PeriodMinutes)
	fmt.Printf("   Уверенность: %.1f%% | Время: %s\n",
		signal.Confidence, signal.Timestamp.Format("15:04:05"))
	fmt.Printf("🔗 https://www.bybit.com/trade/usdt/%s\n", signal.Symbol)
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println()
}

func displayStatsWithStorage(currentTime, startTime time.Time,
	growthMonitor interface {
		GetGrowthStats() map[string]interface{}
	},
	storage storage.PriceStorage,
	totalSignals int,
	totalPairs int) {

	// Получаем статистику хранилища
	storageStats := storage.GetStats()

	// Выводим время и статистику
	fmt.Printf("⏰ Статистика в %s | ", currentTime.Format("15:04:05"))
	fmt.Printf("Работаем: %s\n", formatDuration(currentTime.Sub(startTime)))
	fmt.Println(strings.Repeat("─", 50))

	// Статистика роста
	stats := growthMonitor.GetGrowthStats()
	fmt.Printf("📊 Статистика роста:\n")
	fmt.Printf("   Всего сигналов: %d\n", stats["total_signals"])
	fmt.Printf("   Сигналов роста: %d\n", stats["growth_signals"])
	fmt.Printf("   Сигналов падения: %d\n", stats["fall_signals"])
	fmt.Printf("   Всего за сессию: %d\n", totalSignals)
	fmt.Printf("   Отслеживаем символов: %d\n", totalPairs)

	// Статистика хранилища
	fmt.Printf("📦 Хранилище:\n")
	fmt.Printf("   Символов: %d\n", storageStats.TotalSymbols)
	fmt.Printf("   Точек данных: %d\n", storageStats.TotalDataPoints)
	if storageStats.MemoryUsageBytes > 0 {
		fmt.Printf("   Память: %.2f MB\n", float64(storageStats.MemoryUsageBytes)/1024/1024)
	}
	fmt.Println()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
