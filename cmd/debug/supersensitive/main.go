// cmd/bot/debug_super_sensitive.go (исправленная версия)
package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/manager"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"
)

func main() {
	// Инициализируем логгер
	if err := logger.InitGlobal("logs/debug_super_sensitive.log", "debug", true); err != nil {
		log.Fatalf("❌ Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	var testMode bool = true

	logger.Debug("🚀 ЗАПУСК СУПЕР-ЧУВСТВИТЕЛЬНОЙ ОТЛАДКИ")
	logger.Debug(strings.Repeat("=", 70))
	logger.Debug("⚡ ЭКСТРЕМАЛЬНЫЕ НАСТРОЙКИ: пороги 0.01%, уверенность 1%")
	logger.Debug(strings.Repeat("=", 70))

	// Используем загрузку из .env файла вместо ручного создания
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Printf("⚠️  Config file not found, using default values: %v", err)
	}

	// Переопределяем настройки для супер-чувствительного режима
	cfg.DebugMode = true
	cfg.LogLevel = "debug"
	cfg.UpdateInterval = 10
	cfg.MaxSymbolsToMonitor = 50
	cfg.MinVolumeFilter = 0

	// Анализ - ЭКСТРЕМАЛЬНО НИЗКИЕ ПОРОГИ
	cfg.AnalysisEngine.UpdateInterval = 10
	cfg.AnalysisEngine.AnalysisPeriods = []int{1, 2, 5} // Очень короткие периоды
	cfg.AnalysisEngine.MaxSymbolsPerRun = 50
	cfg.AnalysisEngine.EnableParallel = true
	cfg.AnalysisEngine.MaxWorkers = 5
	cfg.AnalysisEngine.MinDataPoints = 2
	cfg.AnalysisEngine.SignalThreshold = 0.01
	cfg.AnalysisEngine.EnableCache = false
	cfg.AnalysisEngine.RetentionPeriod = 1

	// Анализаторы - СУПЕР ЧУВСТВИТЕЛЬНЫЕ
	cfg.Analyzers.GrowthAnalyzer.Enabled = true
	cfg.Analyzers.GrowthAnalyzer.MinConfidence = 1.0 // Всего 1%!
	cfg.Analyzers.GrowthAnalyzer.MinGrowth = 0.01    // Всего 0.01%!
	cfg.Analyzers.GrowthAnalyzer.ContinuityThreshold = 0.5

	cfg.Analyzers.FallAnalyzer.Enabled = true
	cfg.Analyzers.FallAnalyzer.MinConfidence = 1.0
	cfg.Analyzers.FallAnalyzer.MinFall = 0.01
	cfg.Analyzers.FallAnalyzer.ContinuityThreshold = 0.5

	cfg.Analyzers.ContinuousAnalyzer.Enabled = true
	cfg.Analyzers.ContinuousAnalyzer.MinContinuousPoints = 2

	// Фильтры - ВЫКЛЮЧЕНЫ
	cfg.SignalFilters.Enabled = false
	cfg.SignalFilters.MinConfidence = 0.5
	cfg.SignalFilters.MaxSignalsPerMin = 1000
	cfg.SignalFilters.IncludePatterns = []string{}
	cfg.SignalFilters.ExcludePatterns = []string{}

	// Уведомления - ВЫКЛЮЧЕНЫ
	cfg.TelegramEnabled = false
	cfg.TelegramNotifyGrowth = false
	cfg.TelegramNotifyFall = false

	// Логирование
	cfg.LogToConsole = true
	cfg.LogToFile = true

	// Выводим сумасшедшие настройки
	logger.Debug("\n⚡ НАСТРОЙКИ (СУПЕР ЧУВСТВИТЕЛЬНЫЕ):")
	fmt.Printf("   • Порог роста: %.3f%%\n", cfg.Analyzers.GrowthAnalyzer.MinGrowth)
	fmt.Printf("   • Порог падения: %.3f%%\n", cfg.Analyzers.FallAnalyzer.MinFall)
	fmt.Printf("   • Уверенность: %.1f%%\n", cfg.Analyzers.GrowthAnalyzer.MinConfidence)
	fmt.Printf("   • Периоды: %v мин\n", cfg.AnalysisEngine.AnalysisPeriods)
	fmt.Printf("   • Символов: %d\n", cfg.MaxSymbolsToMonitor)
	fmt.Printf("   • Фильтры: %v\n", cfg.SignalFilters.Enabled)

	// Создаем менеджер
	logger.Debug("\n🛠️  Создание менеджера...")
	dataManager, err := manager.NewDataManager(cfg, testMode)
	if err != nil {
		log.Fatalf("❌ Ошибка: %v", err)
	}
	logger.Debug("✅ Менеджер создан")

	// Запускаем
	logger.Debug("\n🚀 Запуск...")
	dataManager.StartAllServices()

	// Сигналы
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Debug("\n" + strings.Repeat("=", 70))
	logger.Debug("🏃 ЗАПУЩЕНО! Ожидание сигналов...")
	logger.Debug("   Пороги настолько низкие, что должны обнаружить ЛЮБОЕ движение")
	logger.Debug("   Даже 0.01% изменение цены будет обнаружено!")
	logger.Debug(strings.Repeat("=", 70))

	// Быстрый тест через 15 секунд
	go func() {
		time.Sleep(15 * time.Second)

		logger.Debug("\n" + strings.Repeat("⚡", 30))
		logger.Debug("СУПЕР-ЧУВСТВИТЕЛЬНЫЙ ТЕСТ")
		logger.Debug(strings.Repeat("⚡", 30))

		// Анализ
		results, err := dataManager.RunAnalysis()
		if err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
			return
		}

		// Считаем сигналы
		totalSignals := 0
		for _, result := range results {
			totalSignals += len(result.Signals)
		}

		fmt.Printf("📊 Результаты:\n")
		fmt.Printf("   • Символов проанализировано: %d\n", len(results))
		fmt.Printf("   • Сигналов обнаружено: %d\n", totalSignals)

		if totalSignals == 0 {
			logger.Debug("   ⚠️  НОЛЬ СИГНАЛОВ ДАЖЕ С ПОРОГОМ 0.01%!")
			logger.Debug("   🚨 СИСТЕМА НЕ РАБОТАЕТ ПРАВИЛЬНО!")
			logger.Debug("   🔧 Проверьте:")
			logger.Debug("      - API подключение")
			logger.Debug("      - Данные в хранилище")
			logger.Debug("      - Работу анализаторов")
		} else {
			logger.Debug("   ✅ Система работает! Обнаружены сигналы")
			fmt.Printf("   🎯 Количество сигналов: %d\n", totalSignals)

			// Показываем первые 5 сигналов
			count := 0
			for _, result := range results {
				for _, signal := range result.Signals {
					if count < 5 {
						icon := "🟢"
						if signal.Direction == "down" {
							icon = "🔴"
						}
						fmt.Printf("      %s %s: %.4f%%\n",
							icon, signal.Symbol, signal.ChangePercent)
						count++
					}
				}
			}
		}
	}()

	// Ждем
	<-sigChan
	logger.Debug("\n🛑 Остановка...")
	dataManager.Stop()
	logger.Debug("✅ Готово")
}
