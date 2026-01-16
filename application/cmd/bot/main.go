// application/cmd/bot/main.go
package main

import (
	"crypto-exchange-screener-bot/application/bootstrap"
	manager "crypto-exchange-screener-bot/application/services/orchestrator"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
)

func main() {
	// Парсим аргументы командной строки
	var (
		env         string
		cfgPath     string
		logLevel    string
		testMode    bool
		showHelp    bool
		showVersion bool
		mode        string // Режим запуска: "simple" или "full"
	)

	flag.StringVar(&env, "env", "dev", "Environment (dev/prod)")
	flag.StringVar(&cfgPath, "config", "", "Path to config file (overrides env)")
	flag.StringVar(&logLevel, "log-level", "", "Log level: debug, info, warn, error (overrides .env)")
	flag.BoolVar(&testMode, "test", false, "Test mode (no welcome messages)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.StringVar(&mode, "mode", "simple", "Run mode: simple (basic app) or full (with DataManager)")
	flag.Parse()

	if showVersion {
		printVersion()
		return
	}

	if showHelp {
		printHelp()
		return
	}

	// 1. Определяем путь к конфигурации
	var configFile string
	if cfgPath != "" {
		// Используем явно указанный путь
		configFile = cfgPath
	} else {
		// Используем окружение по умолчанию
		configFile = filepath.Join("configs", env, ".env")
	}

	// Проверяем существование файла
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Пробуем fallback на старый путь .env
		if _, err := os.Stat(".env"); err == nil {
			configFile = ".env"
			logger.Warn("⚠️  Using fallback config: .env (instead of %s)", filepath.Join("configs", env, ".env"))
		} else {
			logger.Error("❌ Config file not found: %s and .env not found", configFile)
		}
	}

	logger.Warn("🎯 Environment: %s", env)
	logger.Warn("📁 Config file: %s", configFile)
	logger.Warn("🔧 Run mode: %s", mode)

	// 2. Загружаем конфигурацию
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatal("❌ Failed to load config:", err)
	}

	// Переопределяем уровень логирования, если указан в аргументах
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}

	// Определяем тестовый режим
	if !testMode {
		// Проверяем переменную окружения как резервный вариант
		testMode = strings.ToLower(os.Getenv("TEST_MODE")) == "true"
	}

	// Выводим информацию о конфигурации
	cfg.PrintSummary()

	// Выбираем режим запуска
	switch mode {
	case "full":
		// Запуск полной версии с DataManager
		logger.Info("🚀 Starting Crypto Exchange Screener Bot (FULL MODE)...")
		logger.Warn("🧪 Test mode: %v", testMode)
		runFullMode(cfg, testMode)
	case "simple":
		fallthrough
	default:
		// Запуск простой версии с bootstrap
		logger.Info("🚀 Starting Crypto Exchange Screener Bot (SIMPLE MODE)...")
		runSimpleMode(cfg)
	}
}

// runSimpleMode запускает простое приложение через bootstrap
func runSimpleMode(cfg *config.Config) {
	// Канал для сигналов остановки
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Строим приложение с опциями
	app, err := bootstrap.NewAppBuilder().
		WithConfig(cfg).
		WithOption(bootstrap.WithConsoleLogging(cfg.LogLevel)).
		WithOption(bootstrap.WithTelegramBot(cfg.TelegramEnabled, cfg.TelegramChatID)).
		Build()
	if err != nil {
		log.Fatal("❌ Failed to build application:", err)
	}

	// Запускаем приложение в горутине
	errChan := make(chan error, 1)
	go func() {
		if err := app.Run(); err != nil {
			errChan <- err
		}
	}()

	// Ждем либо сигнала остановки, либо ошибки
	log.Println("🛑 Press Ctrl+C to stop")

	select {
	case sig := <-stopChan:
		log.Printf("📶 Received signal: %v", sig)
	case err := <-errChan:
		log.Printf("❌ Application error: %v", err)
	}

	// Останавливаем приложение
	log.Println("🛑 Stopping application...")
	app.Cleanup()
	log.Println("👋 Application stopped gracefully")
}

// runFullMode запускает полную версию с DataManager
func runFullMode(cfg *config.Config, testMode bool) {
	// Для полного режима нужны дополнительные импорты
	// Динамически загружаем пакеты для избежания циклических зависимостей
	runFullModeImpl(cfg, testMode)
}

// runFullModeImpl реализация полного режима
func runFullModeImpl(cfg *config.Config, testMode bool) {
	// Импорты внутри функции чтобы избежать циклических зависимостей

	// Инициализация логгера
	logPath := cfg.LogFile
	if logPath == "" {
		logPath = "logs/growth_monitor.log"
	}

	if err := logger.InitGlobal(logPath, cfg.LogLevel, true); err != nil {
		log.Fatalf("❌ Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Логируем режим запуска
	if testMode {
		logger.Info("🧪 ЗАПУСК В ТЕСТОВОМ РЕЖИМЕ")
		logger.Info("• Приветственные сообщения Telegram отключены")
		logger.Info("• Реальные уведомления не отправляются")
	} else {
		logger.Info("🚀 ЗАПУСК В РАБОЧЕМ РЕЖИМЕ")
	}

	logger.Info("🚀 Starting Crypto Growth Monitor v%s", version)
	logger.Info("📅 Build time: %s", buildTime)
	logger.Info("⚡ Exchange: %s %s", strings.ToUpper(cfg.Exchange), cfg.ExchangeType)
	logger.Info("📊 Log level: %s", cfg.LogLevel)

	// Логируем режим
	if testMode {
		logger.Info("🧪 РЕЖИМ: Тестовый (без отправки Telegram уведомлений)")
	}

	// Валидация конфигурации
	if err := validateConfig(cfg); err != nil {
		logger.Error("❌ Configuration validation failed: %v", err)
		os.Exit(1)
	}

	// Логирование конфигурации
	logConfig(cfg, testMode)

	// Создание менеджера данных с передачей тестового режима
	logger.Info("🛠️ Creating data manager (test mode: %v)...", testMode)

	// Используем правильный конструктор с testMode
	dataManager, err := manager.NewDataManager(cfg, testMode)
	if err != nil {
		logger.Error("❌ Failed to create data manager: %v", err)
		os.Exit(1)
	}

	// Проверяем инициализацию
	if !dataManager.IsInitialized() {
		logger.Error("❌ DataManager не инициализирован корректно")
		os.Exit(1)
	}

	// Проверка CounterAnalyzer
	engine := dataManager.GetAnalysisEngine()
	if engine != nil {
		analyzers := engine.GetAnalyzers()
		logger.Info("🔍 Зарегистрированные анализаторы:")

		for i, name := range analyzers {
			logger.Info("   %d. %s", i+1, name)
		}

		// Проверяем наличие CounterAnalyzer
		hasCounter := false
		for _, name := range analyzers {
			if strings.Contains(strings.ToLower(name), "counter") {
				hasCounter = true
				break
			}
		}

		if hasCounter {
			logger.Info("✅ CounterAnalyzer активен!")
		} else if cfg.IsCounterAnalyzerEnabled() {
			logger.Warn("⚠️ CounterAnalyzer включен в конфиге, но не найден в движке")
			logger.Warn("⚠️ Проверьте настройки COUNTER_ANALYZER_ENABLED")
		}
	}

	// Graceful shutdown
	// Глобальная обработка Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	graceful := true
	go func() {
		for sig := range sigChan {
			if graceful {
				fmt.Printf("\n📶 Получен сигнал: %v (начинаем graceful shutdown)\n", sig)
				fmt.Println("🛑 Нажмите Ctrl+C еще раз для принудительного выхода")
				graceful = false

				// Запускаем graceful shutdown
				go func() {
					time.Sleep(5 * time.Second)
					fmt.Println("⏰ Таймаут graceful shutdown, выход...")
					os.Exit(0)
				}()
			} else {
				fmt.Printf("\n📶 Получен второй сигнал: %v (принудительный выход)\n", sig)
				os.Exit(1)
			}
		}
	}()

	errChan := make(chan error, 1)

	// Запуск системы
	go func() {
		logger.Info("🚦 Starting system services...")
		if err := startSystem(dataManager, cfg, testMode); err != nil {
			errChan <- err
		}
	}()

	// Таймер для статуса
	statusTicker := time.NewTicker(1 * time.Minute)
	defer statusTicker.Stop()

	// Главный цикл
	logger.Info("✅ System started successfully!")

	// Получаем список символов
	symbolList := cfg.GetSymbolList()
	symbolCount := len(symbolList)
	if symbolCount == 0 {
		logger.Info("🎯 Monitoring ALL symbols with volume > %.0f USDT", cfg.MinVolumeFilter)
	} else {
		logger.Info("🎯 Monitoring %d symbols", symbolCount)
		if symbolCount <= 15 {
			logger.Info("📋 Symbols: %v", symbolList)
		}
	}

	logger.Info("🛑 Press Ctrl+C to stop")

	startTime := time.Now()

	for {
		select {
		case sig := <-sigChan:
			logger.Info("📶 Received signal: %v", sig)
			shutdown(dataManager, startTime)
			return

		case err := <-errChan:
			logger.Error("❌ System error: %v", err)
			shutdown(dataManager, startTime)
			os.Exit(1)

		case <-statusTicker.C:
			logStatus(dataManager, startTime)
		}
	}
}

// Вспомогательные функции для полного режима
func validateConfig(cfg *config.Config) error {
	var errors []string

	// Проверка API ключей
	if cfg.Exchange == "bybit" {
		if cfg.ApiKey == "" || cfg.ApiSecret == "" {
			errors = append(errors, "BYBIT_API_KEY and BYBIT_SECRET_KEY are required for Bybit")
		}
	} else if cfg.Exchange == "binance" {
		if cfg.ApiKey == "" || cfg.ApiSecret == "" {
			errors = append(errors, "BINANCE_API_KEY and BINANCE_API_SECRET are required for Binance")
		}
	}

	// Проверка Telegram если включен
	if cfg.TelegramEnabled {
		if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
			errors = append(errors, "TG_API_KEY and TG_CHAT_ID are required when Telegram is enabled")
		}

		// Проверка порта для вебхука
		if cfg.HTTPPort == 0 {
			cfg.HTTPPort = 8080
		}
	}

	// Проверка CounterAnalyzer если включен
	if cfg.IsCounterAnalyzerEnabled() {
		// Используем геттер для получения базового периода
		if cfg.GetCounterBasePeriodMinutes() <= 0 {
			errors = append(errors, "COUNTER_BASE_PERIOD_MINUTES must be positive")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

func logConfig(cfg *config.Config, testMode bool) {
	logger.Info("📝 Configuration loaded:")
	logger.Info("   • Exchange: %s %s", strings.ToUpper(cfg.Exchange), cfg.ExchangeType)
	logger.Info("   • Test Mode: %v", testMode)

	// Символы
	symbols := cfg.GetSymbolList()
	if len(symbols) > 0 {
		logger.Info("   • Monitoring %d symbols", len(symbols))
		if len(symbols) <= 10 {
			logger.Info("   • Symbols: %v", symbols)
		} else {
			logger.Info("   • First 10 symbols: %v", symbols[:10])
		}
	} else {
		logger.Info("   • Monitoring: ALL symbols (filtered by volume)")
	}

	// Анализ
	logger.Info("   • Analysis: every %d seconds", cfg.AnalysisEngine.UpdateInterval)
	logger.Info("   • Periods: %v minutes", cfg.AnalysisEngine.AnalysisPeriods)

	// Counter Analyzer
	if cfg.IsCounterAnalyzerEnabled() {
		logger.Info("   📊 Counter Analyzer: ENABLED")
		logger.Info("      • Period: %s", cfg.GetCounterAnalysisPeriod())
		logger.Info("      • Base period: %d minutes", cfg.GetCounterBasePeriodMinutes())
		logger.Info("      • Growth threshold: %.2f%%", cfg.GetCounterGrowthThreshold())
		logger.Info("      • Fall threshold: %.2f%%", cfg.GetCounterFallThreshold())
		logger.Info("      • Track growth: %v", cfg.GetCounterTrackGrowth())
		logger.Info("      • Track fall: %v", cfg.GetCounterTrackFall())
		logger.Info("      • Notification enabled: %v", cfg.GetCounterNotificationEnabled())
	} else {
		logger.Info("   📊 Counter Analyzer: DISABLED")
	}

	// Уведомления
	logger.Info("   📱 Telegram: %v", cfg.TelegramEnabled)
	if cfg.TelegramEnabled {
		logger.Info("   📨 Notify: growth=%v, fall=%v",
			cfg.TelegramNotifyGrowth, cfg.TelegramNotifyFall)
		if !testMode {
			logger.Info("   🌐 Webhook порт: %d", cfg.HTTPPort)
		}
	}
}

func startSystem(dataManager *manager.DataManager, cfg *config.Config, testMode bool) error {
	// Проверяем Telegram бота перед запуском
	if bot := dataManager.GetTelegramBot(); bot != nil {
		logger.Info("🤖 Telegram bot initialized")
	} else if cfg.TelegramEnabled {
		logger.Warn("⚠️ Telegram включен в конфигурации, но бот не создан")
	}

	// Запускаем все сервисы через DataManager
	logger.Info("🚀 Starting all services...")
	errors := dataManager.StartAllServices()
	if len(errors) > 0 {
		for service, err := range errors {
			logger.Warn("❌ Failed to start %s: %v", service, err)
		}
		return fmt.Errorf("failed to start one or more services")
	}

	// Проверка работоспособности
	time.Sleep(3 * time.Second)
	if !checkSystemHealth(dataManager) {
		return fmt.Errorf("system health check failed")
	}

	logger.Info("🎯 System is running and monitoring for growth signals")
	return nil
}

func checkSystemHealth(dataManager *manager.DataManager) bool {
	storage := dataManager.GetStorage()
	if storage == nil {
		logger.Error("❌ Storage not initialized")
		return false
	}

	symbols := storage.GetSymbols()
	logger.Info("✅ System health check passed")
	logger.Info("📦 Storage initialized with %d symbols", len(symbols))

	// Получаем статус всех сервисов
	servicesInfo := dataManager.GetServicesInfo()
	logger.Info("🔧 Статус сервисов:")

	allRunning := true
	for name, info := range servicesInfo {
		status := "❌"
		if info.State == manager.StateRunning {
			status = "✅"
		} else {
			allRunning = false
		}
		logger.Info("   • %s: %s %s", name, status, info.State)
	}

	return allRunning
}

func shutdown(dataManager *manager.DataManager, startTime time.Time) {
	logger.Info("🛑 Shutting down system...")

	shutdownStart := time.Now()

	if err := dataManager.Stop(); err != nil {
		logger.Error("❌ Error during shutdown: %v", err)
	} else {
		logger.Info("✅ System stopped cleanly")
	}

	uptime := time.Since(startTime).Round(time.Second)
	shutdownTime := time.Since(shutdownStart).Round(time.Millisecond)

	logger.Info("📊 Session summary:")
	logger.Info("   • Uptime: %v", uptime)
	logger.Info("   • Shutdown time: %v", shutdownTime)
}

func logStatus(dataManager *manager.DataManager, startTime time.Time) {
	storage := dataManager.GetStorage()
	symbolCount := 0
	if storage != nil {
		symbolCount = len(storage.GetSymbols())
	}

	stats := map[string]string{
		"Uptime":         time.Since(startTime).Round(time.Second).String(),
		"Symbols Loaded": fmt.Sprintf("%d", symbolCount),
	}

	if engine := dataManager.GetAnalysisEngine(); engine != nil {
		engineStats := engine.GetStats()
		stats["Total Analyses"] = fmt.Sprintf("%d", engineStats.TotalAnalyses)
		stats["Signals Found"] = fmt.Sprintf("%d", engineStats.TotalSignals)

		// Используем поле LastRunTime из EngineStats
		if !engineStats.LastRunTime.IsZero() {
			stats["Last Analysis"] = time.Since(engineStats.LastRunTime).Round(time.Second).String() + " ago"
		}
	}

	logger.Status(stats)
}

func printVersion() {
	fmt.Printf("📈 Crypto Growth Monitor v%s\n", version)
	fmt.Printf("📅 Build: %s\n", buildTime)
	fmt.Printf("🚀 Exchange: Bybit Futures\n")
}

func printHelp() {
	fmt.Println("📈 Crypto Growth Monitor")
	fmt.Println("Usage: growth-monitor [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --config string    Path to configuration file (default: .env)")
	fmt.Println("  --log-level string Log level: debug, info, warn, error")
	fmt.Println("  --test             Test mode (no welcome messages)")
	fmt.Println("  --version          Show version information")
	fmt.Println("  --help             Show this help message")
	fmt.Println("  --env string       Environment (dev/prod) (default: dev)")
	fmt.Println("  --mode string      Run mode: simple (basic app) or full (with DataManager) (default: simple)")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  TEST_MODE=true     Enable test mode (same as --test)")
	fmt.Println()
	fmt.Println("Configuration (.env file):")
	fmt.Println("  Required: API_KEY, API_SECRET (or BYBIT_API_KEY/BYBIT_SECRET_KEY)")
	fmt.Println("  Optional: SYMBOL_FILTER, MIN_VOLUME_FILTER, etc.")
	fmt.Println("  Telegram: TG_API_KEY, TG_CHAT_ID, TELEGRAM_ENABLED=true")
	fmt.Println("  Counter: COUNTER_ANALYZER_ENABLED=true, COUNTER_ANALYSIS_PERIOD=15m")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/bot/main.go --env=dev --log-level=info")
	fmt.Println("  go run cmd/bot/main.go --mode=full --test")
	fmt.Println("  go run cmd/bot/main.go --config=configs/dev/.env --mode=full")
	fmt.Println("  go run cmd/bot/main.go --help")
}
