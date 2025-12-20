package main

import (
	"crypto-exchange-screener-bot/internal/config"
	"crypto-exchange-screener-bot/internal/manager"
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

	// Запуск
	runBot(cfg)
}

func runBot(cfg *config.Config) {
	logger.Info("🚀 Starting Crypto Growth Monitor v%s", version)
	logger.Info("📅 Build time: %s", buildTime)
	logger.Info("⚡ Exchange: %s %s", strings.ToUpper(cfg.Exchange), cfg.ExchangeType)
	logger.Info("📊 Log level: %s", cfg.LogLevel)

	// Валидация конфигурации
	if err := validateConfig(cfg); err != nil {
		logger.Error("❌ Configuration validation failed: %v", err)
		os.Exit(1)
	}

	// Логирование конфигурации
	logConfig(cfg)

	// Создание менеджера данных
	logger.Info("🛠️ Creating data manager...")
	dataManager, err := manager.NewDataManager(cfg)
	if err != nil {
		logger.Error("❌ Failed to create data manager: %v", err)
		os.Exit(1)
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	errChan := make(chan error, 1)

	// Запуск системы
	go func() {
		logger.Info("🚦 Starting system services...")
		if err := startSystem(dataManager); err != nil {
			errChan <- err
		}
	}()

	// Таймер для статуса
	statusTicker := time.NewTicker(1 * time.Minute)
	defer statusTicker.Stop()

	// Главный цикл
	logger.Info("✅ System started successfully!")
	logger.Info("🎯 Monitoring %d+ symbols", len(cfg.GetSymbolList()))
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

func startSystem(dataManager *manager.DataManager) error {
	errors := dataManager.StartAllServices()
	if len(errors) > 0 {
		for service, err := range errors {
			logger.Error("❌ Failed to start %s: %v", service, err)
		}
		return fmt.Errorf("failed to start one or more services")
	}

	// Проверка работоспособности
	time.Sleep(5 * time.Second)
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

	// Проверка Telegram
	if cfg.TelegramEnabled {
		if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
			errors = append(errors, "TG_API_KEY and TG_CHAT_ID are required when Telegram is enabled")
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

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

func logConfig(cfg *config.Config) {
	logger.Info("📝 Configuration loaded:")
	logger.Info("   • Exchange: %s %s", strings.ToUpper(cfg.Exchange), cfg.ExchangeType)

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

	// Уведомления
	logger.Info("   📱 Telegram: %v", cfg.TelegramEnabled)
	if cfg.TelegramEnabled {
		logger.Info("   📨 Notify: growth=%v, fall=%v",
			cfg.TelegramNotifyGrowth, cfg.TelegramNotifyFall)
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

	if len(symbols) > 0 {
		// Показываем несколько символов с ценами
		sampleCount := min(5, len(symbols))
		for i := 0; i < sampleCount; i++ {
			if price, ok := storage.GetCurrentPrice(symbols[i]); ok {
				logger.Debug("   • %s: %.4f", symbols[i], price)
			}
		}
	}

	return true
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
	fmt.Println("  --version          Show version information")
	fmt.Println("  --help             Show this help message")
	fmt.Println()
	fmt.Println("Configuration (.env file):")
	fmt.Println("  Required: BYBIT_API_KEY, BYBIT_SECRET_KEY")
	fmt.Println("  Optional: SYMBOL_FILTER, MIN_VOLUME_FILTER, etc.")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  growth-monitor --config=.env --log-level=info")
	fmt.Println("  growth-monitor --help")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
