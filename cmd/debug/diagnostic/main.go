package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/manager"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
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

	// 1. Проверяем конфигурацию
	logger.Debug("\n1️⃣  ПРОВЕРКА КОНФИГУРАЦИИ")
	cfg := createDebugConfig()
	printConfig(cfg)

	// 2. Создаем менеджер
	logger.Debug("\n2️⃣  СОЗДАНИЕ МЕНЕДЖЕРА")
	dataManager, err := manager.NewDataManager(cfg)
	if err != nil {
		log.Fatalf("❌ Ошибка создания менеджера: %v", err)
	}
	logger.Debug("✅ Менеджер создан")

	// 3. Запускаем только хранилище и фетчер
	logger.Debug("\n3️⃣  ЗАПУСК БАЗОВЫХ СЕРВИСОВ")
	startBasicServices(dataManager)

	// 4. Ждем данные
	logger.Debug("\n4️⃣  ОЖИДАНИЕ ДАННЫХ")
	time.Sleep(10 * time.Second)

	// 5. Проверяем данные
	logger.Debug("\n5️⃣  ПРОВЕРКА ДАННЫХ")
	checkData(dataManager)

	// 6. Проверяем анализаторы вручную
	logger.Debug("\n6️⃣  РУЧНАЯ ПРОВЕРКА АНАЛИЗАТОРОВ")
	manualAnalyzerCheck(dataManager)

	// 7. Запускаем полную систему
	logger.Debug("\n7️⃣  ЗАПУСК ПОЛНОЙ СИСТЕМЫ")
	startAllServices(dataManager)

	// 8. Запускаем тестовый анализ
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

		// Проверяем против порогов
		cfg := createDebugConfig()
		if cfg == nil {
			continue
		}

		growthThreshold := cfg.Analyzers.GrowthAnalyzer.MinGrowth
		fallThreshold := cfg.Analyzers.FallAnalyzer.MinFall

		if change > growthThreshold {
			fmt.Printf("         ✅ ДОЛЖЕН БЫТЬ СИГНАЛ РОСТА! (%.6f%% > %.6f%%)\n",
				change, growthThreshold)
		} else if -change > fallThreshold {
			fmt.Printf("         ✅ ДОЛЖЕН БЫТЬ СИГНАЛ ПАДЕНИЯ! (%.6f%% > %.6f%%)\n",
				-change, fallThreshold)
		} else {
			fmt.Printf("         ⚠️  Изменение ниже порогов (рост: %.6f%%, падение: %.6f%%)\n",
				growthThreshold, fallThreshold)
		}

		// Показываем все точки
		fmt.Printf("         📊 Все точки данных:\n")
		for j, point := range history {
			fmt.Printf("           %d. %.6f (%v)\n", j+1, point.Price,
				point.Timestamp.Format("15:04:05"))
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
