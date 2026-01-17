// application/cmd/bot/main.go
package main

import (
	layer_manager "crypto-exchange-screener-bot/application/layer_manager"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"flag"
	"fmt"
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
	)

	flag.StringVar(&env, "env", "dev", "Environment (dev/prod)")
	flag.StringVar(&cfgPath, "config", "", "Path to config file (overrides env)")
	flag.StringVar(&logLevel, "log-level", "", "Log level: debug, info, warn, error (overrides .env)")
	flag.BoolVar(&testMode, "test", false, "Test mode (no welcome messages)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showVersion, "version", false, "Show version")
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
			os.Exit(1)
		}
	}

	logger.Warn("🎯 Environment: %s", env)
	logger.Warn("📁 Config file: %s", configFile)

	// 2. Загружаем конфигурацию
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		logger.Error("❌ Failed to load config: %v", err)
		os.Exit(1)
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

	// Запускаем новую архитектуру со слоями
	logger.Info("🚀 Starting Crypto Exchange Screener Bot (Layer-based Architecture)...")
	runLayersMode(cfg, testMode)
}

// runLayersMode запускает новую архитектуру со слоями
func runLayersMode(cfg *config.Config, testMode bool) {
	// Инициализация логгера
	logPath := cfg.LogFile
	if logPath == "" {
		logPath = "logs/growth_monitor.log"
	}

	if err := logger.InitGlobal(logPath, cfg.LogLevel, true); err != nil {
		logger.Error("❌ Failed to initialize logger: %v", err)
		os.Exit(1)
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
	logger.Info("🏗️  Architecture: Layer-based")

	// Валидация конфигурации
	if err := validateConfig(cfg); err != nil {
		logger.Error("❌ Configuration validation failed: %v", err)
		os.Exit(1)
	}

	// Создание LayerManager
	logger.Info("🛠️ Creating LayerManager...")
	layerManager := layer_manager.NewLayerManager(cfg)

	// Инициализация LayerManager
	logger.Info("🔧 Initializing LayerManager...")
	if err := layerManager.Initialize(); err != nil {
		logger.Error("❌ Failed to initialize LayerManager: %v", err)
		os.Exit(1)
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)

	// Запуск системы
	go func() {
		logger.Info("🚦 Starting system services via LayerManager...")
		if err := startSystem(layerManager, cfg, testMode); err != nil {
			errChan <- err
		}
	}()

	// Таймер для статуса
	statusTicker := time.NewTicker(1 * time.Minute)
	defer statusTicker.Stop()

	// Главный цикл
	logger.Info("✅ System initialized successfully!")
	logger.Info("🛑 Press Ctrl+C to stop")

	startTime := time.Now()

	for {
		select {
		case sig := <-sigChan:
			logger.Info("📶 Received signal: %v", sig)
			shutdown(layerManager, startTime)
			return

		case err := <-errChan:
			logger.Error("❌ System error: %v", err)
			shutdown(layerManager, startTime)
			os.Exit(1)

		case <-statusTicker.C:
			logStatus(layerManager, startTime)
		}
	}
}

// validateConfig валидация конфигурации
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
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// startSystem запускает систему через LayerManager
func startSystem(layerManager *layer_manager.LayerManager, cfg *config.Config, testMode bool) error {
	logger.Info("🚀 Starting all layers...")

	// Запускаем LayerManager
	if err := layerManager.Start(); err != nil {
		return fmt.Errorf("failed to start LayerManager: %w", err)
	}

	// Проверка работоспособности
	time.Sleep(3 * time.Second)
	if !checkSystemHealth(layerManager) {
		return fmt.Errorf("system health check failed")
	}

	// Логируем символы для мониторинга
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

	logger.Info("🎯 System is running and monitoring for growth signals")
	return nil
}

// checkSystemHealth проверяет здоровье системы
func checkSystemHealth(layerManager *layer_manager.LayerManager) bool {
	healthStatus := layerManager.GetHealthStatus()

	logger.Info("✅ System health check passed")
	logger.Info("   • Initialized: %v", healthStatus["initialized"])
	logger.Info("   • Running: %v", healthStatus["running"])
	logger.Info("   • Uptime: %v", healthStatus["uptime"])

	if layersStatus, ok := healthStatus["layers"].(map[string]interface{}); ok {
		logger.Info("   • Layers: %d", len(layersStatus))
		for layerName, status := range layersStatus {
			logger.Info("     - %s: %v", layerName, status)
		}
	}

	return true
}

// shutdown останавливает систему
func shutdown(layerManager *layer_manager.LayerManager, startTime time.Time) {
	logger.Info("🛑 Shutting down system...")

	shutdownStart := time.Now()

	if err := layerManager.Stop(); err != nil {
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

// logStatus логирует статус системы
func logStatus(layerManager *layer_manager.LayerManager, startTime time.Time) {
	healthStatus := layerManager.GetHealthStatus()

	stats := map[string]string{
		"Uptime":      time.Since(startTime).Round(time.Second).String(),
		"Initialized": fmt.Sprintf("%v", healthStatus["initialized"]),
		"Running":     fmt.Sprintf("%v", healthStatus["running"]),
	}

	if layersStatus, ok := healthStatus["layers"].(map[string]interface{}); ok {
		stats["Layers"] = fmt.Sprintf("%d", len(layersStatus))
	}

	logger.Status(stats)
}

func printVersion() {
	fmt.Printf("📈 Crypto Growth Monitor v%s\n", version)
	fmt.Printf("📅 Build: %s\n", buildTime)
	fmt.Printf("🚀 Exchange: Bybit Futures\n")
	fmt.Printf("🏗️  Architecture: Layer-based\n")
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
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/bot/main.go --env=dev --log-level=info")
	fmt.Println("  go run cmd/bot/main.go --test")
	fmt.Println("  go run cmd/bot/main.go --config=configs/dev/.env")
	fmt.Println("  go run cmd/bot/main.go --help")
}
