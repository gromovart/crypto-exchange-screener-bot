// cmd/bot/main.go - исправленная версия
package main

import (
	"crypto-exchange-screener-bot/internal/api/bybit"
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/events"
	"crypto-exchange-screener-bot/internal/fetcher"
	"crypto-exchange-screener-bot/internal/notifier"
	"crypto-exchange-screener-bot/internal/storage"
	"crypto-exchange-screener-bot/internal/telegram"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	analysisengine "crypto-exchange-screener-bot/internal/analysis/engine"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("Не удалось загрузить конфигурацию: %v", err)
	}

	// Выводим информацию о конфигурации
	printHeader("АНАЛИЗ РОСТА/ПАДЕНИЯ КРИПТОВАЛЮТНЫХ ФЬЮЧЕРСОВ")
	fmt.Printf("🔧 Конфигурация:\n")
	fmt.Printf("   Сеть: %s\n", map[bool]string{true: "Testnet 🧪", false: "Mainnet ⚡"}[cfg.UseTestnet])
	fmt.Printf("   Категория: %s фьючерсы\n", cfg.FuturesCategory)
	fmt.Printf("   Интервал анализа: %d секунд\n", cfg.UpdateInterval)
	fmt.Printf("   Периоды анализа: %s\n", formatPeriods(cfg.AnalysisEngine.AnalysisPeriods))
	fmt.Printf("   Порог роста: %.2f%%\n", cfg.Analyzers.GrowthAnalyzer.MinGrowth)
	fmt.Printf("   Порог падения: %.2f%%\n", cfg.Analyzers.FallAnalyzer.MinFall)

	// Создаем EventBus
	eventBusFactory := &events.Factory{}
	eventBus := eventBusFactory.NewEventBusFromConfig(cfg)

	// Создаем хранилище
	storageConfig := &storage.StorageConfig{
		MaxHistoryPerSymbol: 10000,
		MaxSymbols:          1000,
		CleanupInterval:     5 * time.Minute,
		RetentionPeriod:     24 * time.Hour,
	}
	priceStorage := storage.NewInMemoryPriceStorage(storageConfig)

	// Создаем API клиент
	apiClient := bybit.NewBybitClient(cfg)

	// Создаем PriceFetcher
	fetcherFactory := &fetcher.Factory{}
	priceFetcher := fetcherFactory.NewPriceFetcherFromConfig(apiClient, priceStorage, eventBus, cfg)

	// Создаем AnalysisEngine
	engineFactory := &analysisengine.Factory{}
	analysisEngine := engineFactory.NewAnalysisEngineFromConfig(priceStorage, eventBus, cfg)

	// Запускаем компоненты
	if err := priceFetcher.Start(time.Duration(cfg.UpdateInterval) * time.Second); err != nil {
		log.Fatalf("Не удалось запустить PriceFetcher: %v", err)
	}

	if err := analysisEngine.Start(); err != nil {
		log.Fatalf("Не удалось запустить AnalysisEngine: %v", err)
	}

	// Регистрируем стандартных подписчиков
	eventBusFactory.RegisterDefaultSubscribers(eventBus, cfg)

	// Инициализируем Telegram бота если включен
	var telegramNotifier *notifier.TelegramNotifier
	if cfg.TelegramEnabled && cfg.TelegramAPIKey != "" && cfg.TelegramChatID != 0 {
		telegramNotifier = notifier.NewTelegramNotifier(cfg)
		if telegramNotifier != nil {
			// Создаем подписчика для Telegram
			telegramSubscriber := events.NewBaseSubscriber(
				"telegram_notifier",
				[]events.EventType{events.EventSignalDetected},
				func(event events.Event) error {
					// Логируем получение события
					log.Printf("📨 Получено событие для Telegram: %v", event.Type)
					return nil
				},
			)
			eventBus.Subscribe(events.EventSignalDetected, telegramSubscriber)

			// Отправляем тестовое сообщение
			go func() {
				time.Sleep(3 * time.Second)
				// Используем bot из notifier для отправки тестового сообщения
				// Но у TelegramNotifier нет метода GetBot, поэтому создадим отдельно
				telegramBot := telegram.NewTelegramBot(cfg)
				if telegramBot != nil {
					telegramBot.SendTestMessage()
				}
			}()
		}
	}

	fmt.Println("\n✅ Система инициализирована")
	fmt.Println("🚀 Запуск мониторинга...")

	// Статистика
	startTime := time.Now()
	var analysisCount int32 = 0
	var signalCount int32 = 0

	// Подписываемся на события сигналов
	signalSubscriber := events.NewBaseSubscriber(
		"signal_counter",
		[]events.EventType{events.EventSignalDetected},
		func(event events.Event) error {
			atomic.AddInt32(&signalCount, 1)
			return nil
		},
	)
	eventBus.Subscribe(events.EventSignalDetected, signalSubscriber)

	// Подписываемся на события завершения анализа
	analysisSubscriber := events.NewBaseSubscriber(
		"analysis_counter",
		[]events.EventType{events.EventType("analysis_complete")},
		func(event events.Event) error {
			atomic.AddInt32(&analysisCount, 1)
			return nil
		},
	)
	eventBus.Subscribe(events.EventType("analysis_complete"), analysisSubscriber)

	// Горутина для вывода статистики
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		iteration := 1
		for range ticker.C {
			engineStats := analysisEngine.GetStats()
			storageStats := priceStorage.GetStats()

			fmt.Println(strings.Repeat("─", 80))
			fmt.Printf("📊 СТАТИСТИКА (итерация #%d)\n", iteration)
			fmt.Printf("   ⏱️  Время работы: %s\n", formatDuration(time.Since(startTime)))
			fmt.Printf("   🔄 Завершено анализов: %d\n", atomic.LoadInt32(&analysisCount))
			fmt.Printf("   📈 Обнаружено сигналов: %d\n", atomic.LoadInt32(&signalCount))
			fmt.Printf("   💾 Символов в хранилище: %d\n", storageStats.TotalSymbols)
			fmt.Printf("   📊 Точок данных: %d\n", storageStats.TotalDataPoints)
			fmt.Printf("   🧮 Анализаторов: %d\n", engineStats.ActiveAnalyzers)
			fmt.Printf("   🧵 Горутин: %d\n", runtime.NumGoroutine())
			fmt.Printf("   🕐 Текущее время: %s\n", time.Now().Format("15:04:05"))
			fmt.Println(strings.Repeat("─", 80))
			fmt.Println()

			iteration++
		}
	}()

	// Обработка сигналов для graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n🎮 Управление:")
	fmt.Println("   Ctrl+C - Остановить систему")
	fmt.Println()

	// Ожидание сигнала остановки
	<-stopChan

	fmt.Println("\n🛑 Завершение работы...")

	// Останавливаем компоненты
	analysisEngine.Stop()
	priceFetcher.Stop()
	eventBus.Stop()

	// Выводим финальную статистику
	fmt.Printf("\n📊 ФИНАЛЬНАЯ СТАТИСТИКА:\n")
	fmt.Printf("   ⏱️  Время работы: %s\n", formatDuration(time.Since(startTime)))
	fmt.Printf("   🔄 Всего анализов: %d\n", atomic.LoadInt32(&analysisCount))
	fmt.Printf("   📈 Всего сигналов: %d\n", atomic.LoadInt32(&signalCount))

	engineStats := analysisEngine.GetStats()
	fmt.Printf("   🧮 Анализаторов использовано: %d\n", engineStats.ActiveAnalyzers)

	fmt.Println("\n✅ Система завершена корректно")
}

// Вспомогательные функции
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

func formatPeriods(periods []int) string {
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

// parseFloat - вспомогательная функция для парсинга строк в числа
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	return strconv.ParseFloat(s, 64)
}

// package main

// import "honnef.co/go/tools/config"

// func main() {
// 	// Инициализация
// 	cfg := config.Load()
// 	eventBus := events.NewEventBus()

// 	// Создание компонентов
// 	storage := storage.NewTimeSeriesStorage(cfg)
// 	analyzer := analysis.NewAnalysisEngine(cfg, storage, eventBus)
// 	notifier := notification.NewCoordinator(cfg, eventBus)

// 	// Настройка пайплайна
// 	pipeline := pipeline.NewSignalPipeline()
// 	pipeline.AddStage(analysis.NewValidationStage())
// 	pipeline.AddStage(analysis.NewEnrichmentStage(storage))
// 	pipeline.AddStage(filter.NewConfidenceFilter(cfg))
// 	pipeline.AddStage(notification.NewFormattingStage(cfg))

// 	// Подписки
// 	eventBus.Subscribe(events.EventPriceUpdate, analyzer)
// 	eventBus.Subscribe(events.EventSignalDetected, pipeline)
// 	eventBus.Subscribe(events.EventSignalProcessed, notifier)

// 	// Запуск
// 	scheduler := orchestration.NewScheduler(cfg)
// 	scheduler.AddTask(fetcher.UpdatePrices, cfg.UpdateInterval)
// 	scheduler.AddTask(analyzer.RunAnalysis, cfg.AnalysisInterval)

// 	scheduler.Start()
// }
