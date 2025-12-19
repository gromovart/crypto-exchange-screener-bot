// cmd/bot/debug_super_sensitive.go
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
)

func main() {
	logger.Debug("🚀 ЗАПУСК СУПЕР-ЧУВСТВИТЕЛЬНОЙ ОТЛАДКИ")
	logger.Debug(strings.Repeat("=", 70))
	logger.Debug("⚡ ЭКСТРЕМАЛЬНЫЕ НАСТРОЙКИ: пороги 0.01%, уверенность 1%")
	logger.Debug(strings.Repeat("=", 70))

	// Создаем минимальную конфигурацию
	cfg := &config.Config{
		// API (можно тестовые)
		ApiKey:    os.Getenv("BYBIT_API_KEY"),
		ApiSecret: os.Getenv("BYBIT_SECRET_KEY"),
		BaseURL:   "https://api.bybit.com",

		// Отладка
		DebugMode:      true,
		LogLevel:       "error", // Только ошибки
		LogToConsole:   true,
		LogToFile:      false,

		// Основные
		UpdateInterval:        10,
		MaxSymbolsToMonitor:   50,
		MaxConcurrentRequests: 5,
		MinVolumeFilter:       0,

		// Анализ - ЭКСТРЕМАЛЬНО НИЗКИЕ ПОРОГИ
		AnalysisEngine: struct {
			UpdateInterval   int           `json:"update_interval"`
			AnalysisPeriods  []int         `json:"analysis_periods"`
			MinVolumeFilter  float64       `json:"min_volume_filter"`
			MaxSymbolsPerRun int           `json:"max_symbols_per_run"`
			EnableParallel   bool          `json:"enable_parallel"`
			MaxWorkers       int           `json:"max_workers"`
			SignalThreshold  float64       `json:"signal_threshold"`
			RetentionPeriod  time.Duration `json:"retention_period"`
			EnableCache      bool          `json:"enable_cache"`
			MinDataPoints    int           `json:"min_data_points"`
		}{
			UpdateInterval:   10,
			AnalysisPeriods:  []int{1, 2, 5}, // Очень короткие периоды
			MaxSymbolsPerRun: 50,
			EnableParallel:   true,
			MaxWorkers:       5,
			MinDataPoints:    2,
		},

		// Анализаторы - СУПЕР ЧУВСТВИТЕЛЬНЫЕ
		Analyzers: struct {
			GrowthAnalyzer struct {
				Enabled             bool    `json:"enabled"`
				MinConfidence       float64 `json:"min_confidence"`
				MinGrowth           float64 `json:"min_growth"`
				ContinuityThreshold float64 `json:"continuity_threshold"`
			}{
				Enabled:             true,
				MinConfidence:       1.0, // Всего 1%!
				MinGrowth:           0.01, // Всего 0.01%!
			},
			FallAnalyzer struct {
				Enabled             bool    `json:"enabled"`
				MinConfidence       float64 `json:"min_confidence"`
				MinFall             float64 `json:"min_fall"`
				ContinuityThreshold float64 `json:"continuity_threshold"`
			}{
				Enabled:       true,
				MinConfidence: 1.0,
				MinFall:       0.01,
			},
			ContinuousAnalyzer struct {
				Enabled             bool `json:"enabled"`
				MinContinuousPoints int  `json:"min_continuous_points"`
			}{
				Enabled: true,
			},
		},

		// Фильтры - ВЫКЛЮЧЕНЫ
		SignalFilters: struct {
			Enabled          bool     `json:"enabled"`
			IncludePatterns  []string `json:"include_patterns"`
			ExcludePatterns  []string `json:"exclude_patterns"`
			MinConfidence    float64  `json:"min_confidence"`
			MaxSignalsPerMin int      `json:"max_signals_per_min"`
		}{
			Enabled:       false,
			MinConfidence: 0.5,
		},

		TelegramEnabled: false,
	}

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
	dataManager, err := manager.NewDataManager(cfg)
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