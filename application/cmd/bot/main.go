// cmd/bot/main.go (исправленная версия с DI)
package main

import (
	manager "crypto-exchange-screener-bot/application/services/orchestrator"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
)

func main() {
	// Парсинг флагов командной строки
	var (
		configPath  = flag.String("config", ".env", "Path to configuration file")
		logLevel    = flag.String("log-level", "", "Log level: debug, info, warn, error (overrides .env)")
		testMode    = flag.Bool("test", false, "Test mode (no welcome messages)")
		showHelp    = flag.Bool("help", false, "Show help")
		showVersion = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("📈 Crypto Growth Monitor v%s\n", version)
		fmt.Printf("📅 Build: %s\n", buildTime)
		fmt.Printf("🚀 Exchange: Bybit Futures\n")
		return
	}

	if *showHelp {
		printHelp()
		return
	}

	// Загрузка конфигурации
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Переопределение уровня логирования из флага
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	// Определяем тестовый режим
	testModeEnabled := *testMode
	if !testModeEnabled {
		// Проверяем переменную окружения как резервный вариант
		testModeEnabled = strings.ToLower(os.Getenv("TEST_MODE")) == "true"
	}

	// Инициализация логгера
	logPath := cfg.LogFile
	if logPath == "" {
		logPath = "logs/growth_monitor.log"
	}

	if err := logger.InitGlobal(logPath, cfg.LogLevel, true); err != nil {
		fmt.Printf("❌ Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Логируем режим запуска
	if testModeEnabled {
		logger.Info("🧪 ЗАПУСК В ТЕСТОВОМ РЕЖИМЕ")
		logger.Info("• Приветственные сообщения Telegram отключены")
		logger.Info("• Реальные уведомления не отправляются")
	} else {
		logger.Info("🚀 ЗАПУСК В РАБОЧЕМ РЕЖИМЕ")
	}

	// Запуск
	runBot(cfg, testModeEnabled)
}

func runBot(cfg *config.Config, testMode bool) {
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

	// 🔴 ВАЖНО: Используем правильный конструктор с testMode
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

	// ПРОВЕРКА: CounterAnalyzer активен ли?
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

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

func startSystem(dataManager *manager.DataManager, cfg *config.Config, testMode bool) error {
	// Проверяем Telegram бота перед запуском
	if bot := dataManager.GetTelegramBot(); bot != nil {
		botTestMode := bot.IsTestMode()
		logger.Info("🤖 Telegram bot initialized (test mode: %v)", botTestMode)

		if testMode && !botTestMode {
			logger.Warn("⚠️ Запущен в тестовом режиме, но Telegram bot не в тестовом режиме")
		}
	} else if cfg.TelegramEnabled {
		logger.Warn("⚠️ Telegram включен в конфигурации, но бот не создан")
	}

	// Запускаем все сервисы через DataManager
	logger.Info("🚀 Starting all services...")
	errors := dataManager.StartAllServices()
	if len(errors) > 0 {
		for service, err := range errors {
			logger.Error("❌ Failed to start %s: %v", service, err)
		}
		return fmt.Errorf("failed to start one or more services")
	}

	// Проверяем запущен ли WebhookServer
	if webhookServer := dataManager.GetWebhookServer(); webhookServer != nil {
		logger.Info("✅ Telegram webhook server ready")

		// Проверяем порт
		if cfg.HTTPPort == 0 {
			logger.Warn("⚠️ HTTP_PORT не указан в конфигурации, будет использован порт 8080")
		} else {
			logger.Info("🌐 Webhook порт: %d", cfg.HTTPPort)
		}
	}

	// Проверка работоспособности
	time.Sleep(3 * time.Second)
	if !checkSystemHealth(dataManager) {
		return fmt.Errorf("system health check failed")
	}

	logger.Info("🎯 System is running and monitoring for growth signals")
	return nil
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

	// Проверка анализаторов
	if cfg.Analyzers.GrowthAnalyzer.MinGrowth < 0.1 {
		logger.Warn("⚠️  MinGrowth (%.2f%%) is very low, may generate many signals",
			cfg.Analyzers.GrowthAnalyzer.MinGrowth)
	}

	if cfg.Analyzers.FallAnalyzer.MinFall < 0.1 {
		logger.Warn("⚠️  MinFall (%.2f%%) is very low, may generate many signals",
			cfg.Analyzers.FallAnalyzer.MinFall)
	}

	// Проверка CounterAnalyzer если включен
	if cfg.IsCounterAnalyzerEnabled() {
		if cfg.CounterAnalyzer.BasePeriodMinutes <= 0 {
			errors = append(errors, "COUNTER_BASE_PERIOD_MINUTES must be positive")
		}

		validPeriods := map[string]bool{"5m": true, "15m": true, "30m": true, "1h": true, "4h": true, "1d": true}
		if !validPeriods[cfg.CounterAnalyzer.DefaultPeriod] {
			errors = append(errors, "COUNTER_DEFAULT_PERIOD must be one of: 5m, 15m, 30m, 1h, 4h, 1d")
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

	// Пороги
	logger.Info("   ⚡ Growth detection: >%.2f%% (confidence: >%.0f%%)",
		cfg.Analyzers.GrowthAnalyzer.MinGrowth,
		cfg.Analyzers.GrowthAnalyzer.MinConfidence)

	logger.Info("   📉 Fall detection: >%.2f%% (confidence: >%.0f%%)",
		cfg.Analyzers.FallAnalyzer.MinFall,
		cfg.Analyzers.FallAnalyzer.MinConfidence)

	// Фильтры
	logger.Info("   🛡️  Volume filter: >%.0f USDT", cfg.MinVolumeFilter)
	logger.Info("   🚦 Signal filters: %v", cfg.SignalFilters.Enabled)

	// Counter Analyzer
	if cfg.IsCounterAnalyzerEnabled() {
		logger.Info("   📊 Counter Analyzer: ENABLED")
		logger.Info("      • Period: %s", cfg.CounterAnalyzer.DefaultPeriod)
		logger.Info("      • Base period: %d minutes", cfg.CounterAnalyzer.BasePeriodMinutes)
		logger.Info("      • Growth threshold: %.2f%%", cfg.CounterAnalyzer.GrowthThreshold)
		logger.Info("      • Fall threshold: %.2f%%", cfg.CounterAnalyzer.FallThreshold)
		logger.Info("      • Notify: %v", cfg.CounterAnalyzer.NotifyOnSignal)
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

	if len(symbols) > 0 {
		// Показываем несколько символов с ценами
		sampleCount := min(5, len(symbols))
		logger.Info("   📊 Sample prices:")
		for i := 0; i < sampleCount; i++ {
			if price, ok := storage.GetCurrentPrice(symbols[i]); ok {
				logger.Info("      • %s: %.4f", symbols[i], price)
			}
		}
	}

	return allRunning
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

	// Используем обновленный метод Status
	logger.Status(stats)
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
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  TEST_MODE=true     Enable test mode (same as --test)")
	fmt.Println()
	fmt.Println("Configuration (.env file):")
	fmt.Println("  Required: BYBIT_API_KEY, BYBIT_SECRET_KEY")
	fmt.Println("  Optional: SYMBOL_FILTER, MIN_VOLUME_FILTER, etc.")
	fmt.Println("  Telegram: TG_API_KEY, TG_CHAT_ID, TELEGRAM_ENABLED=true")
	fmt.Println("  Counter: COUNTER_ANALYZER_ENABLED=true, COUNTER_DEFAULT_PERIOD=15m")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  growth-monitor --config=.env --log-level=info")
	fmt.Println("  growth-monitor --test (test mode, no Telegram messages)")
	fmt.Println("  TEST_MODE=true growth-monitor (test mode via env)")
	fmt.Println("  growth-monitor --help")
}
