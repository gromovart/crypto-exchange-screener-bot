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
	buildTime = "неизвестно"
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

	flag.StringVar(&env, "env", "dev", "Окружение (dev/prod)")
	flag.StringVar(&cfgPath, "config", "", "Путь к файлу конфигурации (переопределяет env)")
	flag.StringVar(&logLevel, "log-level", "", "Уровень логирования: debug, info, warn, error (переопределяет .env)")
	flag.BoolVar(&testMode, "test", false, "Тестовый режим (без приветственных сообщений)")
	flag.BoolVar(&showHelp, "help", false, "Показать справку")
	flag.BoolVar(&showVersion, "version", false, "Показать версию")
	flag.Parse()

	if showVersion {
		printVersion()
		return
	}

	if showHelp {
		printHelp()
		return
	}

	// Устанавливаем переменную окружения ДО загрузки конфигурации
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
				logger.Error("❌ Файл конфигурации не найден: %s и .env не найден", configFile)
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

	// Устанавливаем окружение в конфигурацию
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
	logger.Info("🚀 Запуск Crypto Exchange Screener Bot (архитектура Bootstrap)...")
	runBootstrapMode(cfg, testMode)
}

// runBootstrapMode запускает приложение через Bootstrap
func runBootstrapMode(cfg *config.Config, testMode bool) {
	// Инициализация логгера
	logPath := cfg.LogFile
	if logPath == "" {
		logPath = "logs/growth_monitor.log"
	}

	// Создаем директорию для логов, если её нет
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("❌ Не удалось создать директорию логов %s: %v\n", logDir, err)
		os.Exit(1)
	}

	if err := logger.InitGlobal(logPath, cfg.LogLevel, true); err != nil {
		// Если не удалось инициализировать логгер с файлом, используем консольный
		fmt.Printf("❌ Не удалось инициализировать файловый логгер: %v. Переход на консольный...\n", err)
		if err := logger.InitGlobal("", cfg.LogLevel, true); err != nil {
			fmt.Printf("❌ Не удалось инициализировать консольный логгер: %v\n", err)
			os.Exit(1)
		}
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

	logger.Info("🚀 Запуск Crypto Growth Monitor v%s", version)
	logger.Info("📅 Время сборки: %s", buildTime)
	logger.Info("⚡ Биржа: %s %s", strings.ToUpper(cfg.Exchange), cfg.ExchangeType)
	logger.Info("📊 Уровень логирования: %s", cfg.LogLevel)
	logger.Info("🏗️  Архитектура: Bootstrap-based")

	// Валидация конфигурации
	if err := validateConfig(cfg); err != nil {
		logger.Error("❌ Валидация конфигурации не пройдена: %v", err)
		os.Exit(1)
	}

	// Создаем приложение через Bootstrap
	logger.Info("🏗️  Сборка приложения через Bootstrap...")

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
		logger.Error("❌ Не удалось собрать приложение: %v", err)
		os.Exit(1)
	}

	// Инициализируем приложение
	logger.Info("🔧 Инициализация приложения...")
	if err := app.Initialize(); err != nil {
		logger.Error("❌ Не удалось инициализировать приложение: %v", err)
		os.Exit(1)
	}

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Канал для ошибок запуска
	runErrChan := make(chan error, 1)

	// Запускаем приложение в отдельной горутине
	go func() {
		logger.Info("🚀 Запуск приложения...")
		if err := app.Run(); err != nil {
			runErrChan <- err
		}
	}()

	// Ждем некоторое время для проверки запуска
	time.Sleep(2 * time.Second)

	// Проверяем, запущено ли приложение
	if !app.IsRunning() {
		logger.Error("❌ Не удалось запустить приложение")
		if err := app.Stop(); err != nil {
			logger.Error("❌ Ошибка остановки приложения: %v", err)
		}
		os.Exit(1)
	}

	// Главный цикл ожидания
	logger.Info("✅ Приложение успешно инициализировано!")
	logger.Info("🛑 Нажмите Ctrl+C для остановки")

	// Ждем сигнала завершения или ошибки
	select {
	case sig := <-sigChan:
		logger.Info("📶 Получен сигнал: %v", sig)
		logger.Info("🛑 Остановка приложения...")

		// Останавливаем приложение
		if err := app.Stop(); err != nil {
			logger.Error("❌ Ошибка остановки приложения: %v", err)
		}

		// Даем время на graceful shutdown
		logger.Info("⏳ Ожидание graceful shutdown...")
		time.Sleep(1 * time.Second)

		logger.Info("✅ Приложение успешно остановлено")
		return

	case err := <-runErrChan:
		logger.Error("❌ Ошибка запуска приложения: %v", err)

		// Останавливаем приложение при ошибке
		if stopErr := app.Stop(); stopErr != nil {
			logger.Error("❌ Ошибка остановки приложения после ошибки запуска: %v", stopErr)
		}

		os.Exit(1)
	}
}

// validateConfig валидация конфигурации
func validateConfig(cfg *config.Config) error {
	var errors []string

	// Проверка окружения
	if cfg.Environment != "dev" && cfg.Environment != "prod" && cfg.Environment != "test" {
		errors = append(errors, fmt.Sprintf("Недопустимое окружение: %s (должно быть dev, prod или test)", cfg.Environment))
	}

	// Проверка биржи
	validExchanges := map[string]bool{"bybit": true, "binance": true}
	if !validExchanges[strings.ToLower(cfg.Exchange)] {
		errors = append(errors, fmt.Sprintf("Недопустимая биржа: %s (должно быть bybit или binance)", cfg.Exchange))
	}

	// Проверка API ключей
	if cfg.Exchange == "bybit" {
		if cfg.ApiKey == "" || cfg.ApiSecret == "" {
			errors = append(errors, "BYBIT_API_KEY и BYBIT_SECRET_KEY требуются для Bybit")
		}
	} else if cfg.Exchange == "binance" {
		if cfg.ApiKey == "" || cfg.ApiSecret == "" {
			errors = append(errors, "BINANCE_API_KEY и BINANCE_API_SECRET требуются для Binance")
		}
	}

	// Проверка Telegram если включен
	if cfg.Telegram.Enabled {
		if cfg.TelegramBotToken == "" {
			errors = append(errors, "TG_API_KEY требуется когда Telegram включен")
		}
		// TelegramChatID может быть пустым для ботов без чата по умолчанию
	}

	// Проверка базы данных
	if cfg.Database.Host == "" || cfg.Database.Name == "" {
		errors = append(errors, "Хост и имя базы данных требуются")
	}

	// Проверка Redis
	if cfg.Redis.Host == "" {
		errors = append(errors, "Хост Redis требуется")
	}

	// Проверка уровня логирования
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(cfg.LogLevel)] {
		errors = append(errors, fmt.Sprintf("Недопустимый уровень логирования: %s (должно быть debug, info, warn или error)", cfg.LogLevel))
	}

	if len(errors) > 0 {
		errMsg := strings.Join(errors, "; ")
		return fmt.Errorf("Ошибки конфигурации: %s", errMsg)
	}

	return nil
}

func printVersion() {
	fmt.Printf("📈 Crypto Exchange Screener Bot v%s\n", version)
	fmt.Printf("📅 Сборка: %s\n", buildTime)
	fmt.Printf("🚀 Архитектура: Bootstrap-based\n")
	fmt.Printf("⚡ Поддержка: Bybit, Binance\n")
	fmt.Println()
	fmt.Println("📊 Функции:")
	fmt.Println("  • Мониторинг цен в реальном времени")
	fmt.Println("  • Сигналы технического анализа")
	fmt.Println("  • Уведомления в Telegram")
	fmt.Println("  • Поддержка нескольких бирж")
	fmt.Println("  • Архитектура Bootstrap")
}

func printHelp() {
	fmt.Println("📈 Crypto Exchange Screener Bot")
	fmt.Println("Бот для мониторинга и анализа криптовалют в реальном времени с уведомлениями в Telegram")
	fmt.Println()
	fmt.Println("Использование: bot [опции]")
	fmt.Println()
	fmt.Println("Опции:")
	fmt.Println("  --env string       Окружение (dev/prod) (по умолчанию: dev)")
	fmt.Println("  --config string    Путь к файлу конфигурации (переопределяет env)")
	fmt.Println("  --log-level string Уровень логирования: debug, info, warn, error (переопределяет .env)")
	fmt.Println("  --test             Тестовый режим (без приветственных сообщений, dry run)")
	fmt.Println("  --version          Показать информацию о версии")
	fmt.Println("  --help             Показать это справочное сообщение")
	fmt.Println()
	fmt.Println("Переменные окружения (через .env файл):")
	fmt.Println("  APP_ENV            Окружение приложения (dev/prod)")
	fmt.Println("  EXCHANGE           Используемая биржа (bybit/binance)")
	fmt.Println("  API_KEY            API ключ биржи")
	fmt.Println("  API_SECRET         Секретный ключ API биржи")
	fmt.Println("  TG_API_KEY         Токен API Telegram бота")
	fmt.Println("  TG_CHAT_ID         ID чата Telegram для уведомлений")
	fmt.Println("  DATABASE_URL       Строка подключения PostgreSQL")
	fmt.Println("  REDIS_URL          Строка подключения Redis")
	fmt.Println("  LOG_LEVEL          Уровень логирования")
	fmt.Println("  LOG_FILE           Путь к файлу логов")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  go run application/cmd/bot/main.go --env=dev --log-level=info")
	fmt.Println("  go run application/cmd/bot/main.go --env=prod --test")
	fmt.Println("  go run application/cmd/bot/main.go --config=configs/dev/.env")
	fmt.Println("  go run application/cmd/bot/main.go --help")
	fmt.Println("  go run application/cmd/bot/main.go --version")
	fmt.Println()
	fmt.Println("Структура директорий:")
	fmt.Println("  configs/dev/.env   Конфигурация для разработки")
	fmt.Println("  configs/prod/.env  Конфигурация для продакшена")
	fmt.Println("  logs/              Файлы логов")
	fmt.Println("  application/       Код приложения")
	fmt.Println("  internal/          Внутренние пакеты")
	fmt.Println("  pkg/               Публичные пакеты")
}
