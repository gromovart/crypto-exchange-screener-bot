package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/manager"
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
	fmt.Println("🚀 ЗАПУСК РАСШИРЕННОЙ ОТЛАДКИ")
	fmt.Println(strings.Repeat("=", 70))

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки конфигурации: %v", err)
	}

	// НАСТРАИВАЕМ ДЛЯ МАКСИМАЛЬНОЙ ЧУВСТВИТЕЛЬНОСТИ
	fmt.Println("\n⚙️  НАСТРОЙКА ДЛЯ ОТЛАДКИ:")

	// Основные настройки
	cfg.DebugMode = true
	cfg.LogLevel = "info"
	cfg.LogToConsole = true
	cfg.LogToFile = false

	// Минимальные пороги
	cfg.UpdateInterval = 20
	cfg.MaxSymbolsToMonitor = 20
	cfg.MaxConcurrentRequests = 3
	cfg.MinVolumeFilter = 0 // Отключаем фильтр объема

	// Анализ - СУПЕР НИЗКИЕ ПОРОГИ
	cfg.AnalysisEngine.UpdateInterval = 20
	cfg.AnalysisEngine.MaxSymbolsPerRun = 20
	cfg.AnalysisEngine.MaxWorkers = 3
	cfg.AnalysisEngine.AnalysisPeriods = []int{1, 5, 15} // Добавляем 1 минуту
	cfg.AnalysisEngine.MinDataPoints = 2

	// Анализаторы - ОЧЕНЬ НИЗКИЕ ПОРОГИ
	cfg.Analyzers.GrowthAnalyzer.Enabled = true
	cfg.Analyzers.GrowthAnalyzer.MinConfidence = 10.0 // Всего 10%!
	cfg.Analyzers.GrowthAnalyzer.MinGrowth = 0.1      // Всего 0.1% роста!

	cfg.Analyzers.FallAnalyzer.Enabled = true
	cfg.Analyzers.FallAnalyzer.MinConfidence = 10.0
	cfg.Analyzers.FallAnalyzer.MinFall = 0.1 // Всего 0.1% падения!

	cfg.Analyzers.ContinuousAnalyzer.Enabled = true

	// Фильтры - ОТКЛЮЧАЕМ
	cfg.SignalFilters.Enabled = false
	cfg.SignalFilters.MinConfidence = 5.0
	cfg.SignalFilters.MaxSignalsPerMin = 100

	// Отключаем Telegram
	cfg.TelegramEnabled = false

	// Выводим настройки
	fmt.Printf("   📊 Конфигурация анализа:\n")
	fmt.Printf("      • Символов: %d\n", cfg.MaxSymbolsToMonitor)
	fmt.Printf("      • Периоды: %v мин\n", cfg.AnalysisEngine.AnalysisPeriods)
	fmt.Printf("      • ПОРОГИ СИГНАЛОВ:\n")
	fmt.Printf("        - Рост: %.2f%% (очень низкий!)\n", cfg.Analyzers.GrowthAnalyzer.MinGrowth)
	fmt.Printf("        - Падение: %.2f%% (очень низкий!)\n", cfg.Analyzers.FallAnalyzer.MinFall)
	fmt.Printf("        - Уверенность: %.0f%%\n", cfg.Analyzers.GrowthAnalyzer.MinConfidence)
	fmt.Printf("      • Фильтр объема: %v\n", cfg.MinVolumeFilter > 0)

	// Создаем менеджер
	fmt.Println("\n🛠️  Создание менеджера данных...")
	dataManager, err := manager.NewDataManager(cfg)
	if err != nil {
		log.Fatalf("❌ Ошибка создания менеджера: %v", err)
	}
	fmt.Println("✅ Менеджер создан")

	// Запускаем сервисы
	fmt.Println("\n🚀 Запуск сервисов...")
	errors := dataManager.StartAllServices()

	if len(errors) > 0 {
		fmt.Println("⚠️  Ошибки запуска:")
		for service, err := range errors {
			fmt.Printf("   ❌ %s: %v\n", service, err)
		}
	} else {
		fmt.Println("✅ Все сервисы запущены")
	}

	// Обработка сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("📈 РАСШИРЕННАЯ ОТЛАДКА")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("⚡ Супер-низкие пороги для тестирования (0.1%)")
	fmt.Println("🔧 Отключены все фильтры")
	fmt.Println("📊 План работы:")
	fmt.Println("   1. Проверка хранилища через 5 секунд")
	fmt.Println("   2. Анализ через 10 секунд")
	fmt.Println("   3. Детальная проверка через 15 секунд")
	fmt.Println("\n🛑 Для остановки нажмите Ctrl+C")
	fmt.Println(strings.Repeat("=", 70))

	// Запускаем тесты
	testChan := make(chan bool, 1)

	go func() {
		// Тест 1: Проверка хранилища
		time.Sleep(5 * time.Second)
		fmt.Println("\n" + strings.Repeat("📊", 25))
		fmt.Println("ТЕСТ 1: ПРОВЕРКА ХРАНИЛИЩА")
		fmt.Println(strings.Repeat("📊", 25))
		runStorageTest(dataManager)

		// Тест 2: Первый анализ
		time.Sleep(5 * time.Second)
		fmt.Println("\n" + strings.Repeat("🧪", 25))
		fmt.Println("ТЕСТ 2: ПЕРВЫЙ АНАЛИЗ")
		fmt.Println(strings.Repeat("🧪", 25))
		runAnalysisTest(dataManager)

		// Тест 3: Детальный анализ
		time.Sleep(5 * time.Second)
		fmt.Println("\n" + strings.Repeat("🔍", 25))
		fmt.Println("ТЕСТ 3: ДЕТАЛЬНЫЙ АНАЛИЗ")
		fmt.Println(strings.Repeat("🔍", 25))
		runDetailedAnalysis(dataManager)

		testChan <- true
	}()

	// Ждем либо завершения тестов, либо сигнала
	select {
	case <-testChan:
		fmt.Println("\n✅ Все тесты завершены")
		fmt.Println("Система продолжает работать в фоновом режиме")
		fmt.Println("Нажмите Ctrl+C для остановки")

		// Ждем сигнал завершения
		<-sigChan

	case <-sigChan:
		fmt.Println("\n🛑 Получен сигнал завершения...")
	}

	// Останавливаем
	fmt.Println("\n⏳ Остановка сервисов...")
	if err := dataManager.Stop(); err != nil {
		fmt.Printf("⚠️ Ошибка остановки: %v\n", err)
	}

	fmt.Println("✅ Программа завершена")
}

// runStorageTest проверяет хранилище
func runStorageTest(dataManager *manager.DataManager) {
	if dataManager == nil {
		return
	}

	storage := dataManager.GetStorage()
	if storage == nil {
		fmt.Println("❌ Хранилище не инициализировано")
		return
	}

	// Получаем символы
	symbols := storage.GetSymbols()
	fmt.Printf("📦 Хранилище:\n")
	fmt.Printf("   • Всего символов: %d\n", len(symbols))

	if len(symbols) == 0 {
		fmt.Println("   ⚠️  Нет символов в хранилище!")
		fmt.Println("   💡 Проверьте API ключи и подключение к Bybit")
		return
	}

	// Показываем первые 10 символов
	showCount := 10
	if len(symbols) < showCount {
		showCount = len(symbols)
	}

	fmt.Printf("   • Первые %d символов:\n", showCount)
	for i := 0; i < showCount; i++ {
		symbol := symbols[i]
		price, ok := storage.GetCurrentPrice(symbol)
		if ok {
			fmt.Printf("      - %s: %.4f\n", symbol, price)
		} else {
			fmt.Printf("      - %s: нет данных\n", symbol)
		}
	}

	// Статистика хранилища
	stats := storage.GetStats()
	fmt.Printf("   • Точки данных: %d\n", stats.TotalDataPoints)
	fmt.Printf("   • Старые данные: %v\n", stats.OldestTimestamp.Format("15:04:05"))
	fmt.Printf("   • Новые данные: %v\n", stats.NewestTimestamp.Format("15:04:05"))
}

// runAnalysisTest выполняет анализ
func runAnalysisTest(dataManager *manager.DataManager) {
	if dataManager == nil {
		return
	}

	fmt.Println("🧪 Запуск анализа...")
	startTime := time.Now()

	results, err := dataManager.RunAnalysis()
	if err != nil {
		fmt.Printf("❌ Ошибка анализа: %v\n", err)
		return
	}

	duration := time.Since(startTime)

	// Статистика
	totalSymbols := len(results)
	totalSignals := 0

	fmt.Printf("📊 Результаты анализа (%v):\n", duration)
	fmt.Printf("   • Проанализировано символов: %d\n", totalSymbols)

	// Собираем все сигналы
	var allSignals []map[string]interface{}

	for symbol, result := range results {
		if len(result.Signals) > 0 {
			totalSignals += len(result.Signals)

			for _, signal := range result.Signals {
				signalInfo := map[string]interface{}{
					"symbol":         symbol,
					"direction":      signal.Direction,
					"change_percent": signal.ChangePercent,
					"confidence":     signal.Confidence,
					"period":         signal.Period,
					"type":           signal.Type,
				}
				allSignals = append(allSignals, signalInfo)
			}
		}
	}

	fmt.Printf("   • Обнаружено сигналов: %d\n", totalSignals)

	if totalSignals > 0 {
		fmt.Println("   🎯 Обнаруженные сигналы:")

		// Сортируем по изменению (по убыванию)
		sort.Slice(allSignals, func(i, j int) bool {
			changeI := allSignals[i]["change_percent"].(float64)
			changeJ := allSignals[j]["change_percent"].(float64)
			return changeI > changeJ
		})

		// Показываем топ 10 сигналов
		showCount := 10
		if len(allSignals) < showCount {
			showCount = len(allSignals)
		}

		for i := 0; i < showCount; i++ {
			sig := allSignals[i]
			icon := "🟢"
			if sig["direction"].(string) == "down" {
				icon = "🔴"
			}

			fmt.Printf("      %s %s: %s %.4f%% (уверенность: %.1f%%, период: %dмин)\n",
				icon,
				sig["symbol"].(string),
				map[string]string{"up": "↑", "down": "↓"}[sig["direction"].(string)],
				sig["change_percent"].(float64),
				sig["confidence"].(float64),
				sig["period"].(int))
		}
	} else {
		fmt.Println("   ⚠️  Сигналы не обнаружены")
		fmt.Println("   🔍 Проверяем возможные проблемы...")
		checkPotentialIssues(dataManager)
	}
}

// runDetailedAnalysis выполняет детальный анализ
func runDetailedAnalysis(dataManager *manager.DataManager) {
	if dataManager == nil {
		return
	}

	fmt.Println("🔍 Детальный анализ системы...")

	// 1. Проверяем анализаторы
	if engine := dataManager.GetAnalysisEngine(); engine != nil {
		analyzers := engine.GetAnalyzers()
		stats := engine.GetStats()

		fmt.Printf("📈 Движок анализа:\n")
		fmt.Printf("   • Анализаторов: %d\n", len(analyzers))
		fmt.Printf("   • Всего анализов: %d\n", stats.TotalAnalyses)
		fmt.Printf("   • Всего сигналов: %d\n", stats.TotalSignals)
		fmt.Printf("   • Анализаторы: %v\n", analyzers)
	}

	// 2. Проверяем хранилище детально
	storage := dataManager.GetStorage()
	if storage != nil {
		symbols := storage.GetSymbols()

		// Проверяем данные для случайных символов
		if len(symbols) > 0 {
			fmt.Printf("📦 Проверка данных для случайных символов:\n")

			checkSymbols := 5
			if len(symbols) < checkSymbols {
				checkSymbols = len(symbols)
			}

			for i := 0; i < checkSymbols; i++ {
				symbol := symbols[i]

				// Получаем историю
				history, err := storage.GetPriceHistory(symbol, 5)
				if err == nil && len(history) > 0 {
					fmt.Printf("   • %s: %d точек данных\n", symbol, len(history))

					// Показываем изменения
					if len(history) >= 2 {
						first := history[0].Price
						last := history[len(history)-1].Price
						change := ((last - first) / first) * 100
						fmt.Printf("      Изменение: %.4f%%\n", change)
					}
				} else {
					fmt.Printf("   • %s: недостаточно данных\n", symbol)
				}
			}
		}
	}

	// 3. Еще один анализ с выводом всех символов
	fmt.Println("\n🧪 Финальный анализ всех символов:")
	results, err := dataManager.RunAnalysis()
	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
		return
	}

	// Группируем по типу сигнала
	growthCount := 0
	fallCount := 0
	var growthSymbols, fallSymbols []string

	for symbol, result := range results {
		for _, signal := range result.Signals {
			if signal.Direction == "up" {
				growthCount++
				growthSymbols = append(growthSymbols, symbol)
			} else {
				fallCount++
				fallSymbols = append(fallSymbols, symbol)
			}
		}
	}

	fmt.Printf("   • Рост: %d символов\n", growthCount)
	if growthCount > 0 {
		fmt.Printf("      Символы: %v\n", growthSymbols)
	}

	fmt.Printf("   • Падение: %d символов\n", fallCount)
	if fallCount > 0 {
		fmt.Printf("      Символы: %v\n", fallSymbols)
	}

	if growthCount == 0 && fallCount == 0 {
		fmt.Println("   ⚠️  АБСОЛЮТНО НИКАКИХ СИГНАЛОВ!")
		fmt.Println("   🚨 Возможные серьезные проблемы:")
		fmt.Println("      1. Анализаторы не работают")
		fmt.Println("      2. Данные не поступают")
		fmt.Println("      3. Очень стабильный рынок (маловероятно)")
		fmt.Println("      4. Ошибки в логике анализа")
	}
}

// checkPotentialIssues проверяет возможные проблемы
func checkPotentialIssues(dataManager *manager.DataManager) {
	fmt.Println("   🔧 Проверка проблем:")

	// 1. Проверяем данные в хранилище
	storage := dataManager.GetStorage()
	if storage != nil {
		symbols := storage.GetSymbols()
		if len(symbols) == 0 {
			fmt.Println("      ❌ Нет символов в хранилище")
			fmt.Println("         • Проверьте API ключи")
			fmt.Println("         • Проверьте подключение к интернету")
			fmt.Println("         • Проверьте статус API Bybit")
			return
		}

		// Проверяем данные для первого символа
		if len(symbols) > 0 {
			symbol := symbols[0]
			history, err := storage.GetPriceHistory(symbol, 3)
			if err != nil || len(history) < 2 {
				fmt.Printf("      ❌ Недостаточно данных для %s\n", symbol)
				fmt.Println("         • Ожидайте больше данных")
				fmt.Println("         • Проверьте обновление цен")
				return
			}

			// Проверяем изменения цены
			first := history[0].Price
			last := history[len(history)-1].Price
			change := ((last - first) / first) * 100

			fmt.Printf("      • Тестовый символ %s: изменение %.4f%%\n", symbol, change)

			if change == 0 {
				fmt.Println("         ⚠️  Цена не меняется")
				fmt.Println("         • Может быть стабильный рынок")
				fmt.Println("         • Или проблемы с данными")
			}
		}
	}

	// 2. Проверяем анализаторы
	if engine := dataManager.GetAnalysisEngine(); engine != nil {
		analyzers := engine.GetAnalyzers()
		if len(analyzers) == 0 {
			fmt.Println("      ❌ Нет зарегистрированных анализаторов")
		} else {
			fmt.Printf("      • Анализаторы: %v\n", analyzers)
		}
	}

	// 3. Советы по настройке
	fmt.Println("   💡 Советы по настройке:")
	fmt.Println("      • Уменьшите пороги до 0.01%")
	fmt.Println("      • Уменьшите уверенность до 1%")
	fmt.Println("      • Проверьте данные вручную через API")
	fmt.Println("      • Добавьте больше символов для мониторинга")
}
