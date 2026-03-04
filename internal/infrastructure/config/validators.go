// internal/infrastructure/config/validators.go
package config

import (
	"fmt"
	"strings"
)

// ============================================
// ВАЛИДАЦИЯ КОНФИГУРАЦИИ
// ============================================

// validate проверяет обязательные параметры конфигурации
func (c *Config) validate() error {
	var validationErrors []string

	// Проверка API ключей
	if c.Exchange == "bybit" {
		if c.ApiKey == "" {
			validationErrors = append(validationErrors, "BYBIT_API_KEY is required")
		}
		if c.ApiSecret == "" {
			validationErrors = append(validationErrors, "BYBIT_SECRET_KEY is required")
		}
	} else if c.Exchange == "binance" {
		if c.ApiKey == "" {
			validationErrors = append(validationErrors, "BINANCE_API_KEY is required")
		}
		if c.ApiSecret == "" {
			validationErrors = append(validationErrors, "BINANCE_API_SECRET is required")
		}
	}

	// Проверка Telegram если включен
	if c.Telegram.Enabled {
		if c.Telegram.BotToken == "" {
			validationErrors = append(validationErrors, "TG_API_KEY is required when Telegram is enabled")
		}
		if c.Telegram.ChatID == "" {
			validationErrors = append(validationErrors, "TG_CHAT_ID is required when Telegram is enabled")
		}
	}

	// Проверка MAX если включен
	if c.MAX.Enabled {
		if c.MAX.BotToken == "" {
			validationErrors = append(validationErrors, "MAX_BOT_TOKEN is required when MAX is enabled")
		}
		// MAX_CHAT_ID опционален — нужен только для широковещательных сигналов
		// Если не задан, сигналы не будут отправляться в чат, но интерактивный бот работает
	}

	// Проверка Counter Analyzer если включен
	if c.AnalyzerConfigs.CounterAnalyzer.Enabled {
		settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings
		if settings != nil {
			if basePeriod, ok := settings["base_period_minutes"].(int); ok && basePeriod <= 0 {
				validationErrors = append(validationErrors, "COUNTER_BASE_PERIOD_MINUTES must be positive")
			}
			if period, ok := settings["analysis_period"].(string); ok && !isValidPeriod(period) {
				validationErrors = append(validationErrors, "COUNTER_ANALYSIS_PERIOD must be one of: 5m, 15m, 30m, 1h, 4h, 1d")
			}
		}
	}

	// Проверка настроек базы данных
	if c.Database.Host == "" {
		validationErrors = append(validationErrors, "DB_HOST is required")
	}
	if c.Database.Port <= 0 {
		validationErrors = append(validationErrors, "DB_PORT must be positive")
	}
	if c.Database.User == "" {
		validationErrors = append(validationErrors, "DB_USER is required")
	}
	if c.Database.Password == "" {
		validationErrors = append(validationErrors, "DB_PASSWORD is required")
	}
	if c.Database.Name == "" {
		validationErrors = append(validationErrors, "DB_NAME is required")
	}

	// Валидация режима Telegram
	mode := strings.ToLower(c.TelegramMode)
	if mode != "polling" && mode != "webhook" {
		validationErrors = append(validationErrors, "TELEGRAM_MODE должен быть 'polling' или 'webhook'")
	}

	// Валидация вебхуков если используется webhook режим
	if mode == "webhook" {
		if c.Webhook.Domain == "" {
			validationErrors = append(validationErrors, "WEBHOOK_DOMAIN обязателен для webhook режима")
		}
		if c.Webhook.Port <= 0 || c.Webhook.Port > 65535 {
			validationErrors = append(validationErrors, "WEBHOOK_PORT должен быть в диапазоне 1-65535")
		}
		if c.Webhook.UseTLS {
			if c.Webhook.TLSCertPath == "" {
				validationErrors = append(validationErrors, "WEBHOOK_TLS_CERT_PATH обязателен при использовании TLS")
			}
			if c.Webhook.TLSKeyPath == "" {
				validationErrors = append(validationErrors, "WEBHOOK_TLS_KEY_PATH обязателен при использовании TLS")
			}
		}
	}

	if len(validationErrors) > 0 {
		errMsg := strings.Join(validationErrors, "; ")
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

// Validate проверяет конфигурацию (публичный метод)
func (c *Config) Validate() error {
	// Используем встроенную валидацию
	return c.validate()
}
