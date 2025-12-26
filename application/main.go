// application/main.go
package main

import (
	bootstrap "crypto-exchange-screener-bot/application/bootstrap"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"log"
	"os"
)

func main() {
	// 1. Загружаем конфигурацию
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 2. Строим приложение с опциями
	app, err := bootstrap.NewAppBuilder().
		WithConfig(cfg).
		WithOption(bootstrap.WithConsoleLogging(cfg.LogLevel)).
		WithOption(bootstrap.WithTelegramBot(cfg.TelegramEnabled, cfg.TelegramChatID)).
		Build()
	if err != nil {
		log.Fatal("Failed to build application:", err)
	}

	// 3. Устанавливаем обработку завершения
	defer app.Cleanup()

	// 4. Запускаем
	if err := app.Run(); err != nil {
		app.Cleanup()
		log.Fatal("Failed to run application:", err)
	}

	os.Exit(0)
}
