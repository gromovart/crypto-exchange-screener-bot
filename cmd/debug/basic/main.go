// cmd/bot/debug_main.go
package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/manager"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	logger.Debug("🚀 Запуск в режиме отладки...")
	logger.Debug("📁 Загрузка конфигурации из .env файла...")

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки конфигурации: %v", err)
	}

	// ПРЕВРАЩАЕМ КОНФИГ В РЕЖИМ ОТЛАДКИ
	logger.Debug("\n⚙️  НАСТРОЙКА РЕЖИМА ОТЛАДКИ:")

	// Общие настройки отладки
	cfg.DebugMode = true
	cfg.LogLevel = "debug"
	cfg.LogToConsole = true
	cfg.LogToFile = true
	cfg.LogFile = "logs/debug.log"

	// Уменьшаем нагрузку для отладки
	cfg.UpdateInterval = 30              // Раз в 30 секунд вместо 10
	cfg.MaxSymbolsToMonitor = 10         // Только 10 символов
	cfg.MaxConcurrentRequests = 3        // Меньше параллельных запросов
	cfg.RateLimitDelay = 1 * time.Second // Задержка 1 секунда

	// Настройки анализа для отладки
	cfg.AnalysisEngine.UpdateInterval = 30            // Анализ каждые 30 секунд
	cfg.AnalysisEngine.MaxSymbolsPerRun = 10          // Только 10 символов за раз
	cfg.AnalysisEngine.MaxWorkers = 2                 // Только 2 потока
	cfg.AnalysisEngine.EnableParallel = false         // Отключаем параллелизм для простоты
	cfg.AnalysisEngine.AnalysisPeriods = []int{5, 15} // Только 2 периода

	// Отключаем фильтры для отладки
	cfg.SignalFilters.Enabled = false
	cfg.SignalFilters.MinConfidence = 30.0
	cfg.SignalFilters.MaxSignalsPerMin = 10

	// Настройки анализаторов
	cfg.Analyzers.GrowthAnalyzer.Enabled = true
	cfg.Analyzers.GrowthAnalyzer.MinConfidence = 50.0
	cfg.Analyzers.GrowthAnalyzer.MinGrowth = 1.0 // Более низкий порог для отладки

	cfg.Analyzers.FallAnalyzer.Enabled = true
	cfg.Analyzers.FallAnalyzer.MinConfidence = 50.0
	cfg.Analyzers.FallAnalyzer.MinFall = 1.0

	// Отключаем Telegram для отладки
	cfg.TelegramEnabled = false

	// Выводим конфигурацию отладки
	fmt.Printf("   Режим: отладка\n")
	fmt.Printf("   Логирование: %s (файл: %s)\n", cfg.LogLevel, cfg.LogFile)
	fmt.Printf("   Интервал обновления: %d сек\n", cfg.UpdateInterval)
	fmt.Printf("   Символов для мониторинга: %d\n", cfg.MaxSymbolsToMonitor)
	fmt.Printf("   Периоды анализа: %v минут\n", cfg.AnalysisEngine.AnalysisPeriods)
	fmt.Printf("   Порог роста: %.1f%%\n", cfg.Analyzers.GrowthAnalyzer.MinGrowth)
	fmt.Printf("   Порог падения: %.1f%%\n", cfg.Analyzers.FallAnalyzer.MinFall)
	fmt.Printf("   Telegram: %v\n", cfg.TelegramEnabled)

	logger.Debug("\n🛠️  Создание менеджера данных...")

	var testMode bool = true

	// Создаем менеджер данных
	dataManager, err := manager.NewDataManager(cfg, testMode)
	if err != nil {
		log.Fatalf("❌ Ошибка создания менеджера данных: %v", err)
	}

	logger.Debug("✅ Менеджер данных создан")

	// Запускаем сервисы
	logger.Debug("\n🚀 Запуск сервисов...")
	startTime := time.Now()
	errors := dataManager.StartAllServices()

	if len(errors) > 0 {
		logger.Debug("⚠️  Ошибки при запуске сервисов:")
		for service, err := range errors {
			fmt.Printf("   ❌ %s: %v\n", service, err)
		}
	}

	fmt.Printf("✅ Все сервисы запущены за %v\n", time.Since(startTime))

	// Добавляем консольный подписчик для отладки
	dataManager.AddConsoleSubscriber()

	// Создаем отладчик
	debugger := NewDebugger(dataManager, cfg)
	debugger.Start()

	// Запускаем тестовый анализ через 10 секунд
	go func() {
		logger.Debug("\n⏳ Ожидание 10 секунд для сбора данных...")
		time.Sleep(10 * time.Second)

		logger.Debug("\n🧪 ТЕСТОВЫЙ АНАЛИЗ:")
		debugger.TestAnalysis()

		// Показываем статистику каждые 30 секунд
		ticker := time.NewTicker(30 * time.Second)
		for {
			select {
			case <-ticker.C:
				debugger.PrintStats()
			}
		}
	}()

	// Ожидаем сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Debug("\n" + strings.Repeat("=", 70))
	logger.Debug("📊 СИСТЕМА ОТЛАДКИ ЗАПУЩЕНА")
	logger.Debug(strings.Repeat("=", 70))
	logger.Debug("📈 Мониторинг роста/падения криптовалют")
	logger.Debug("⚡ Режим: ОТЛАДКА (упрощенная конфигурация)")
	logger.Debug("📁 Логи: debug.log")
	logger.Debug("\n📋 КОНФИГУРАЦИЯ ОТЛАДКИ:")
	fmt.Printf("   • Символов: %d (из вашего списка)\n", cfg.MaxSymbolsToMonitor)
	fmt.Printf("   • Интервал обновления: %d сек\n", cfg.UpdateInterval)
	fmt.Printf("   • Анализ каждые: %d сек\n", cfg.AnalysisEngine.UpdateInterval)
	fmt.Printf("   • Порог сигнала: рост %.1f%% / падение %.1f%%\n",
		cfg.Analyzers.GrowthAnalyzer.MinGrowth,
		cfg.Analyzers.FallAnalyzer.MinFall)
	fmt.Printf("   • Фильтры: %v\n", cfg.SignalFilters.Enabled)
	logger.Debug("\n⏰ Первый анализ через 10 секунд...")
	logger.Debug("📊 Статистика каждые 30 секунд")
	logger.Debug("\n🛑 Для остановки нажмите Ctrl+C")
	logger.Debug(strings.Repeat("=", 70))

	// Ждем сигнал завершения
	<-sigChan
	logger.Debug("\n🛑 Получен сигнал завершения...")

	// Останавливаем отладчик
	debugger.Stop()

	// Останавливаем менеджер
	logger.Debug("⏳ Остановка сервисов...")
	stopTime := time.Now()
	if err := dataManager.Stop(); err != nil {
		log.Printf("⚠️ Ошибка при остановке: %v", err)
	}

	fmt.Printf("✅ Все сервисы остановлены за %v\n", time.Since(stopTime))
	logger.Debug("🎯 Программа завершена")
}

// Debugger - отладчик системы
type Debugger struct {
	dataManager *manager.DataManager
	config      *config.Config
	running     bool
	stopChan    chan struct{}
	statsCount  int
}

// NewDebugger создает новый отладчик
func NewDebugger(dataManager *manager.DataManager, cfg *config.Config) *Debugger {
	return &Debugger{
		dataManager: dataManager,
		config:      cfg,
		stopChan:    make(chan struct{}),
		running:     false,
		statsCount:  0,
	}
}

// Start запускает отладчик
func (d *Debugger) Start() {
	if d.running {
		return
	}

	d.running = true
	logger.Debug("🔧 Отладчик активирован")
}

// Stop останавливает отладчик
func (d *Debugger) Stop() {
	if !d.running {
		return
	}

	d.running = false
	close(d.stopChan)
	logger.Debug("🔧 Отладчик деактивирован")
}

// PrintStats выводит статистику
func (d *Debugger) PrintStats() {
	if d.dataManager == nil {
		return
	}

	d.statsCount++

	fmt.Printf("\n%s СТАТИСТИКА #%d %s\n",
		strings.Repeat("─", 20), d.statsCount, strings.Repeat("─", 20))

	// Время работы
	uptime := time.Since(d.dataManager.GetSystemStats().LastUpdated).Round(time.Second)
	fmt.Printf("⏱️  Время работы: %v\n", uptime)

	// Статистика хранилища
	storage := d.dataManager.GetStorage()
	if storage != nil {
		symbols := storage.GetSymbols()
		stats := storage.GetStats()

		fmt.Printf("📊 Хранилище:\n")
		fmt.Printf("   • Символов: %d\n", len(symbols))
		fmt.Printf("   • Точки данных: %d\n", stats.TotalDataPoints)

		if len(symbols) > 0 {
			fmt.Printf("   • Примеры: ")
			count := 5
			if len(symbols) < count {
				count = len(symbols)
			}
			fmt.Printf("%v\n", symbols[:count])
		}
	}

	// Статистика анализа
	analysisEngine := d.dataManager.GetAnalysisEngine()
	if analysisEngine != nil {
		analyzerStats := analysisEngine.GetStats()
		fmt.Printf("📈 Анализ:\n")
		fmt.Printf("   • Всего анализов: %d\n", analyzerStats.TotalAnalyses)
		fmt.Printf("   • Всего сигналов: %d\n", analyzerStats.TotalSignals)
		fmt.Printf("   • Анализаторов: %d\n", len(analysisEngine.GetAnalyzers()))
	}

	// Информация о сервисах
	services := d.dataManager.GetServicesInfo()
	running := 0
	total := len(services)

	for _, info := range services {
		if info.State == manager.StateRunning {
			running++
		}
	}

	fmt.Printf("🛠️  Сервисы: %d/%d активны\n", running, total)

	fmt.Printf("%s\n", strings.Repeat("─", 60))
}

// TestAnalysis запускает тестовый анализ
func (d *Debugger) TestAnalysis() {
	if d.dataManager == nil {
		logger.Debug("❌ Менеджер данных не инициализирован")
		return
	}

	logger.Debug("🧪 Запуск ручного анализа...")
	startTime := time.Now()

	results, err := d.dataManager.RunAnalysis()
	if err != nil {
		fmt.Printf("❌ Ошибка анализа: %v\n", err)
		return
	}

	duration := time.Since(startTime)

	// Считаем статистику
	totalSymbols := len(results)
	totalSignals := 0
	growthSignals := 0
	fallSignals := 0

	for symbol, result := range results {
		signals := len(result.Signals)
		totalSignals += signals

		if signals > 0 {
			fmt.Printf("   📈 %s: %d сигналов\n", symbol, signals)
			for _, signal := range result.Signals {
				if signal.Direction == "up" {
					growthSignals++
				} else {
					fallSignals++
				}
			}
		}
	}

	fmt.Printf("\n📊 РЕЗУЛЬТАТЫ АНАЛИЗА:\n")
	fmt.Printf("   • Время выполнения: %v\n", duration)
	fmt.Printf("   • Проанализировано символов: %d\n", totalSymbols)
	fmt.Printf("   • Обнаружено сигналов: %d\n", totalSignals)
	fmt.Printf("   • Рост: %d | Падение: %d\n", growthSignals, fallSignals)

	if totalSignals > 0 {
		logger.Debug("\n🎯 ТОП СИГНАЛЫ:")
		// Собираем все сигналы
		var allSignals []interface{}
		for _, result := range results {
			for _, signal := range result.Signals {
				allSignals = append(allSignals, signal)
			}
		}

		// Показываем первые 5
		count := 5
		if len(allSignals) < count {
			count = len(allSignals)
		}

		for i := 0; i < count; i++ {
			if sig, ok := allSignals[i].(map[string]interface{}); ok {
				symbol := sig["symbol"].(string)
				dir := sig["direction"].(string)
				change := sig["change_percent"].(float64)
				conf := sig["confidence"].(float64)

				icon := "🟢"
				if dir == "down" {
					icon = "🔴"
				}

				fmt.Printf("   %s %s: %.2f%% (уверенность: %.0f%%)\n",
					icon, symbol, change, conf)
			}
		}
	}
}
