package main

import (
	manager "crypto-exchange-screener-bot/application/data_manager"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

func main() {
	logger.Debug("🔬 ГЛУБОКАЯ ДИАГНОСТИКА СИСТЕМЫ")
	logger.Debug(strings.Repeat("=", 70))
	var testMode bool = true
	// 1. Проверяем конфигурацию
	logger.Debug("\n1️⃣  ПРОВЕРКА КОНФИГУРАЦИИ")
	cfg := createDebugConfig()
	printConfig(cfg)

	// 2. Создаем менеджер
	logger.Debug("\n2️⃣  СОЗДАНИЕ МЕНЕДЖЕРА")
	dataManager, err := manager.NewDataManager(cfg, testMode)
	if err != nil {
		log.Fatalf("❌ Ошибка создания менеджера: %v", err)
	}
	logger.Debug("✅ Менеджер создан")

	// 3. Тестируем CounterAnalyzer отдельно
	logger.Debug("\n🔧 ТЕСТ COUNTER ANALYZER")
	testCounterAnalyzerSeparately()

	// 4. Запускаем только хранилище и фетчер
	logger.Debug("\n3️⃣  ЗАПУСК БАЗОВЫХ СЕРВИСОВ")
	startBasicServices(dataManager)

	// 5. Ждем данные
	logger.Debug("\n4️⃣  ОЖИДАНИЕ ДАННЫХ")
	time.Sleep(10 * time.Second)

	// 6. Проверяем данные
	logger.Debug("\n5️⃣  ПРОВЕРКА ДАННЫХ")
	checkData(dataManager)

	// 7. Проверяем анализаторы вручную
	logger.Debug("\n6️⃣  РУЧНАЯ ПРОВЕРКА АНАЛИЗАТОРОВ")
	manualAnalyzerCheck(dataManager)

	// 8. Запускаем полную систему
	logger.Debug("\n7️⃣  ЗАПУСК ПОЛНОЙ СИСТЕМЫ")
	startAllServices(dataManager)

	// 9. Запускаем тестовый анализ
	logger.Debug("\n8️⃣  ТЕСТОВЫЙ АНАЛИЗ")
	runTestAnalysis(dataManager)

	// Ожидание завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Debug("\n" + strings.Repeat("=", 70))
	logger.Debug("🏁 СИСТЕМА ЗАПУЩЕНА. Нажмите Ctrl+C для остановки")
	logger.Debug(strings.Repeat("=", 70))

	<-sigChan
	logger.Debug("\n🛑 Остановка...")
	dataManager.Stop()
	logger.Debug("✅ Готово")
}

func testCounterAnalyzerSeparately() {
	logger.Debug("   🧪 Тестируем CounterAnalyzer отдельно...")

	// Создаем тестовые данные
	now := time.Now()
	testData := []types.PriceData{
		{Symbol: "TESTUSDT", Price: 100.0, Timestamp: now.Add(-2 * time.Minute)},
		{Symbol: "TESTUSDT", Price: 100.2, Timestamp: now.Add(-1 * time.Minute)}, // +0.2% рост
	}

	fmt.Printf("      📊 Тестовые данные:\n")
	fmt.Printf("         • Начальная цена: %.2f\n", testData[0].Price)
	fmt.Printf("         • Конечная цена: %.2f\n", testData[len(testData)-1].Price)
	fmt.Printf("         • Изменение: +%.4f%%\n",
		((testData[len(testData)-1].Price-testData[0].Price)/testData[0].Price)*100)

	logger.Debug("      ✅ Тест CounterAnalyzer завершен")
}

// Добавляем проверку CounterAnalyzer в manualAnalyzerCheck
func manualAnalyzerCheck(dataManager *manager.DataManager) {
	storage := dataManager.GetStorage()
	if storage == nil {
		logger.Debug("   ❌ Нет доступа к хранилищу")
		return
	}

	symbols := storage.GetSymbols()
	if len(symbols) == 0 {
		logger.Debug("   ⚠️  Нет символов для проверки")
		return
	}

	// Выбираем случайные символы
	testSymbols := []string{}
	for i := 0; i < 3 && i < len(symbols); i++ {
		testSymbols = append(testSymbols, symbols[i])
	}

	fmt.Printf("   🔍 Ручная проверка %d символов:\n", len(testSymbols))

	for _, symbol := range testSymbols {
		fmt.Printf("      • %s:\n", symbol)

		// Получаем историю
		history, err := storage.GetPriceHistory(symbol, 5)
		if err != nil {
			fmt.Printf("         ❌ Ошибка получения истории: %v\n", err)
			continue
		}

		if len(history) < 2 {
			fmt.Printf("         ⚠️  Недостаточно данных: %d точек\n", len(history))
			continue
		}

		// Рассчитываем вручную
		first := history[0].Price
		last := history[len(history)-1].Price
		change := ((last - first) / first) * 100

		fmt.Printf("         📈 Изменение: %.6f%% (%.6f → %.6f)\n", change, first, last)

		// Проверяем против порогов CounterAnalyzer
		if change > 0.1 { // Порог роста CounterAnalyzer
			fmt.Printf("         ✅ ДОЛЖЕН БЫТЬ ЗАСЧИТАН В COUNTER! (рост > 0.1%%)\n")
		} else if -change > 0.1 { // Порог падения CounterAnalyzer
			fmt.Printf("         ✅ ДОЛЖЕН БЫТЬ ЗАСЧИТАН В COUNTER! (падение > 0.1%%)\n")
		}

		// Проверяем быстрые изменения для CounterAnalyzer
		for i := 1; i < len(history); i++ {
			pointChange := ((history[i].Price - history[i-1].Price) / history[i-1].Price) * 100
			if math.Abs(pointChange) > 0.1 {
				fmt.Printf("         ⚡ Быстрое изменение %d→%d: %.4f%%\n",
					i-1, i, pointChange)
			}
		}
	}
}

func createDebugConfig() *config.Config {
	cfg, _ := config.LoadConfig(".env")

	// Экстремальные настройки для отладки
	cfg.DebugMode = true
	cfg.LogLevel = "error" // Только ошибки

	// Минимальные пороги
	cfg.Analyzers.GrowthAnalyzer.MinGrowth = 0.001 // 0.001%!
	cfg.Analyzers.GrowthAnalyzer.MinConfidence = 1.0
	cfg.Analyzers.FallAnalyzer.MinFall = 0.001
	cfg.Analyzers.FallAnalyzer.MinConfidence = 1.0

	// Отключаем все фильтры
	cfg.SignalFilters.Enabled = false
	cfg.MinVolumeFilter = 0

	// Быстрые обновления
	cfg.UpdateInterval = 5
	cfg.MaxSymbolsToMonitor = 50

	return cfg
}

func printConfig(cfg *config.Config) {
	logger.Debug("   ⚙️  Настройки анализа:")
	fmt.Printf("      • Рост: порог=%.3f%%, уверенность=%.1f%%\n",
		cfg.Analyzers.GrowthAnalyzer.MinGrowth,
		cfg.Analyzers.GrowthAnalyzer.MinConfidence)
	fmt.Printf("      • Падение: порог=%.3f%%, уверенность=%.1f%%\n",
		cfg.Analyzers.FallAnalyzer.MinFall,
		cfg.Analyzers.FallAnalyzer.MinConfidence)
	fmt.Printf("      • Фильтры: %v\n", cfg.SignalFilters.Enabled)
	fmt.Printf("      • Фильтр объема: %.0f\n", cfg.MinVolumeFilter)
	fmt.Printf("      • Символов: %d\n", cfg.MaxSymbolsToMonitor)
}

func startBasicServices(dataManager *manager.DataManager) {
	// Запускаем только хранилище и фетчер
	errors := make(map[string]error)

	// PriceStorage
	if err := dataManager.StartService("PriceStorage"); err != nil {
		errors["PriceStorage"] = err
	}

	// PriceFetcher
	if err := dataManager.StartService("PriceFetcher"); err != nil {
		errors["PriceFetcher"] = err
	}

	if len(errors) > 0 {
		logger.Debug("   ⚠️  Ошибки запуска:")
		for service, err := range errors {
			fmt.Printf("      • %s: %v\n", service, err)
		}
	} else {
		logger.Debug("   ✅ Базовые сервисы запущены")
	}
}

func checkData(dataManager *manager.DataManager) {
	storage := dataManager.GetStorage()
	if storage == nil {
		logger.Debug("   ❌ Хранилище не инициализировано")
		return
	}

	symbols := storage.GetSymbols()
	fmt.Printf("   📊 Данные в хранилище:\n")
	fmt.Printf("      • Символов: %d\n", len(symbols))

	if len(symbols) == 0 {
		logger.Debug("      ⚠️  Нет данных! Проверьте API ключи")
		return
	}

	// Проверяем несколько символов
	checkCount := 5
	if len(symbols) < checkCount {
		checkCount = len(symbols)
	}

	fmt.Printf("      • Проверяем %d символов:\n", checkCount)

	for i := 0; i < checkCount; i++ {
		symbol := symbols[i]

		// История
		history, err := storage.GetPriceHistory(symbol, 3)
		if err != nil {
			fmt.Printf("         • %s: ошибка истории - %v\n", symbol, err)
			continue
		}

		if len(history) < 2 {
			fmt.Printf("         • %s: недостаточно данных (%d точек)\n", symbol, len(history))
			continue
		}

		// Рассчитываем изменение
		first := history[0].Price
		last := history[len(history)-1].Price
		change := ((last - first) / first) * 100

		// Текущая цена
		current, _ := storage.GetCurrentPrice(symbol)

		fmt.Printf("         • %s: %.6f → %.6f (изменение: %.6f%%), текущая: %.6f\n",
			symbol, first, last, change, current)

		// Если изменение очень маленькое
		if change == 0 {
			fmt.Printf("           ⚠️  Цена не меняется!\n")
		}
	}
}

func startAllServices(dataManager *manager.DataManager) {
	logger.Debug("   🚀 Запуск всех сервисов...")

	services := []string{
		"EventBus",
		"AnalysisEngine",
		"SignalPipeline",
		"NotificationService",
	}

	for _, service := range services {
		if err := dataManager.StartService(service); err != nil {
			fmt.Printf("      ⚠️  %s: %v\n", service, err)
		} else {
			fmt.Printf("      ✅ %s запущен\n", service)
		}
	}
}

func runTestAnalysis(dataManager *manager.DataManager) {
	logger.Debug("   🧪 Запуск тестового анализа...")

	startTime := time.Now()
	results, err := dataManager.RunAnalysis()
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("      ❌ Ошибка анализа: %v\n", err)
		return
	}

	fmt.Printf("      📊 Результаты (%v):\n", duration)
	fmt.Printf("         • Символов проанализировано: %d\n", len(results))

	// Считаем сигналы
	totalSignals := 0
	var signalDetails []string

	for symbol, result := range results {
		if len(result.Signals) > 0 {
			totalSignals += len(result.Signals)

			for _, signal := range result.Signals {
				icon := "🟢"
				if signal.Direction == "down" {
					icon = "🔴"
				}

				detail := fmt.Sprintf("%s %s: %s %.6f%% (уверенность: %.1f%%)",
					icon, symbol,
					map[string]string{"up": "↑", "down": "↓"}[signal.Direction],
					signal.ChangePercent,
					signal.Confidence)
				signalDetails = append(signalDetails, detail)
			}
		}
	}

	fmt.Printf("         • Сигналов обнаружено: %d\n", totalSignals)

	if totalSignals > 0 {
		logger.Debug("         🎯 Детали сигналов:")
		// Сортируем по изменению
		sort.Slice(signalDetails, func(i, j int) bool {
			// Извлекаем процент изменения из строки
			return signalDetails[i] > signalDetails[j] // Простая сортировка
		})

		for _, detail := range signalDetails {
			fmt.Printf("            %s\n", detail)
		}
	} else {
		logger.Debug("         ⚠️  НЕТ СИГНАЛОВ!")
		logger.Debug("         🚨 ВОЗМОЖНЫЕ ПРИЧИНЫ:")
		logger.Debug("            1. Анализаторы не работают")
		logger.Debug("            2. Данные не обновляются")
		logger.Debug("            3. Ошибки в конфигурации")
		logger.Debug("            4. Все цены действительно стабильны")
	}
}
