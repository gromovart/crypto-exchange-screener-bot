// application/main.go
package main

import (
	bootstrap "crypto-exchange-screener-bot/application/bootstrap"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"flag"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Парсим аргументы командной строки
	var (
		env      string
		cfgPath  string
		logLevel string
	)

	flag.StringVar(&env, "env", "dev", "Environment (dev/prod)")
	flag.StringVar(&cfgPath, "config", "", "Path to config file (overrides env)")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug/info/warn/error)")
	flag.Parse()

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
			log.Printf("⚠️  Using fallback config: .env (instead of %s)", filepath.Join("configs", env, ".env"))
		} else {
			log.Fatalf("❌ Config file not found: %s and .env not found", configFile)
		}
	}

	log.Printf("🎯 Environment: %s", env)
	log.Printf("📁 Config file: %s", configFile)

	// 2. Загружаем конфигурацию
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatal("❌ Failed to load config:", err)
	}

	// Переопределяем уровень логирования, если указан в аргументах
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}

	// Выводим информацию о конфигурации
	cfg.PrintSummary()

	// 3. Строим приложение с опциями
	app, err := bootstrap.NewAppBuilder().
		WithConfig(cfg).
		WithOption(bootstrap.WithConsoleLogging(cfg.LogLevel)).
		WithOption(bootstrap.WithTelegramBot(cfg.TelegramEnabled, cfg.TelegramChatID)).
		Build()
	if err != nil {
		log.Fatal("❌ Failed to build application:", err)
	}

	// 4. Устанавливаем обработку завершения
	defer app.Cleanup()

	// 5. Запускаем
	log.Println("🚀 Starting Crypto Exchange Screener Bot...")
	if err := app.Run(); err != nil {
		app.Cleanup()
		log.Fatal("❌ Failed to run application:", err)
	}

	log.Println("👋 Application stopped gracefully")
	os.Exit(0)
}
