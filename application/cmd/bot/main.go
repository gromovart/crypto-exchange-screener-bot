// application/cmd/bot/main.go
package main

import (
	"crypto-exchange-screener-bot/application/bootstrap"
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

	// === ИСПРАВЛЕНИЕ: Устанавливаем переменную окружения ДО загрузки конфигурации ===
	// Это важно, потому что config.LoadConfig() может использовать os.Getenv()
	os.Setenv("APP_ENV", env)
	logger.Warn("🎯 Установлено окружение: %s", env)

	// 1. Определяем путь к конфигурации
	var configFile string
	if cfgPath != "" {
		// Используем явно указанный путь
		configFile = cfgPath
		logger.Warn("📁 Используется явный конфиг файл: %s", configFile)
	} else {
		// Используем окружение из флага --env
		configFile = filepath.Join("configs", env, ".env")
		logger.Warn("📁 Используется конфиг файл для окружения %s: %s", env, configFile)
	}

	// Проверяем существование файла
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Пробуем fallback на старый путь .env
		if _, err := os.Stat(".env"); err == nil {
			configFile = ".env"
			logger.Warn("⚠️  Используется fallback конфиг: .env (вместо %s)", filepath.Join("configs", env, ".env"))
		} else {
			// Пробуем найти configs/dev/.env как последний fallback
			fallbackPath := filepath.Join("configs", "dev", ".env")
			if _, err := os.Stat(fallbackPath); err == nil {
				configFile = fallbackPath
				logger.Warn("⚠️  Используется fallback конфиг: %s", fallbackPath)
			} else {
				logger.Error("❌ Конфиг файл не найден: %s и .env не найден", configFile)
				os.Exit(1)
			}
		}
	}

	logger.Warn("📁 Используемый конфиг файл: %s", configFile)

	// 2. Загружаем конфигурацию
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		logger.Error("❌ Не удалось загрузить конфигурацию: %v", err)
		os.Exit(1)
	}

	// === ИСПРАВЛЕНИЕ: Устанавливаем окружение в конфигурацию ===
	// Это нужно для последующего использования в приложении
	cfg.Environment = env

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
	logger.Warn("📋 Конфигурация приложения:")
	logger.Warn("   • Окружение: %s", cfg.Environment)
	logger.Warn("   • Биржа: %s %s", strings.ToUpper(cfg.Exchange), cfg.ExchangeType)
	logger.Warn("   • Уровень логирования: %s", cfg.LogLevel)
	logger.Warn("   • Telegram включен: %v", cfg.Telegram.Enabled)
	logger.Warn("   • PostgreSQL: %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	logger.Warn("   • Redis: %s:%d (DB: %d, Pool: %d)", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB, cfg.Redis.PoolSize)

	// Запускаем приложение через Bootstrap
	logger.Info("🚀 Starting Crypto Exchange Screener Bot (Bootstrap Architecture)...")
	runBootstrapMode(cfg, testMode)
}

// runBootstrapMode запускает приложение через Bootstrap
func runBootstrapMode(cfg *config.Config, testMode bool) {
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
		cfg.MonitoringTestMode = true
	} else {
		logger.Info("🚀 ЗАПУСК В РАБОЧЕМ РЕЖИМЕ")
	}

	logger.Info("🚀 Starting Crypto Growth Monitor v%s", version)
	logger.Info("📅 Build time: %s", buildTime)
	logger.Info("⚡ Exchange: %s %s", strings.ToUpper(cfg.Exchange), cfg.ExchangeType)
	logger.Info("📊 Log level: %s", cfg.LogLevel)
	logger.Info("🏗️  Architecture: Bootstrap-based")

	// Валидация конфигурации
	if err := validateConfig(cfg); err != nil {
		logger.Error("❌ Configuration validation failed: %v", err)
		os.Exit(1)
	}

	// Создаем приложение через Bootstrap
	logger.Info("🏗️  Building application via Bootstrap...")

	// Создаем AppBuilder
	builder := bootstrap.NewAppBuilder()

	// Настраиваем через fluent API
	builder = builder.
		WithConfig(cfg).
		WithTestMode(testMode).
		WithTelegramBot(cfg.Telegram.Enabled, cfg.TelegramChatID).
		WithTelegramBotToken(cfg.TelegramBotToken)

	// Собираем приложение
	app, err := builder.Build()
	if err != nil {
		logger.Error("❌ Failed to build application: %v", err)
		os.Exit(1)
	}

	// Инициализируем приложение
	logger.Info("🔧 Initializing application...")
	if err := app.Initialize(); err != nil {
		logger.Error("❌ Failed to initialize application: %v", err)
		os.Exit(1)
	}

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Канал для ошибок запуска
	runErrChan := make(chan error, 1)

	// Запускаем приложение в отдельной горутине
	go func() {
		logger.Info("🚀 Starting application...")
		if err := app.Run(); err != nil {
			runErrChan <- err
		}
	}()

	// Главный цикл ожидания
	logger.Info("✅ Application initialized successfully!")
	logger.Info("🛑 Press Ctrl+C to stop")

	// Ждем сигнала завершения или ошибки
	select {
	case sig := <-sigChan:
		logger.Info("📶 Received signal: %v", sig)
		logger.Info("🛑 Stopping application...")

		// Останавливаем приложение
		if err := app.Stop(); err != nil {
			logger.Error("❌ Error stopping application: %v", err)
		}

		// Даем время на graceful shutdown
		logger.Info("⏳ Waiting for graceful shutdown...")
		time.Sleep(500 * time.Millisecond)

		logger.Info("✅ Application stopped successfully")
		return

	case err := <-runErrChan:
		logger.Error("❌ Application run error: %v", err)

		// Останавливаем приложение при ошибке
		if stopErr := app.Stop(); stopErr != nil {
			logger.Error("❌ Error stopping application after run error: %v", stopErr)
		}

		os.Exit(1)
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
	if cfg.Telegram.Enabled {
		if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
			errors = append(errors, "TG_API_KEY and TG_CHAT_ID are required when Telegram is enabled")
		}
	}

	if len(errors) > 0 {
		errMsg := strings.Join(errors, "; ")
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

func printVersion() {
	fmt.Printf("📈 Crypto Growth Monitor v%s\n", version)
	fmt.Printf("📅 Build: %s\n", buildTime)
	fmt.Printf("🚀 Exchange: Bybit Futures\n")
	fmt.Printf("🏗️  Architecture: Bootstrap-based\n")
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
