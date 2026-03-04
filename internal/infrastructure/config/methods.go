// internal/infrastructure/config/methods.go
package config

import (
	"fmt"
	"log"
	"strings"
)

// ============================================
// МЕТОДЫ КОНФИГУРАЦИИ
// ============================================

// GetDatabaseConfig возвращает конфигурацию базы данных
func (c *Config) GetDatabaseConfig() DatabaseConfig {
	return c.Database
}

// GetPostgresDSN возвращает DSN для подключения к PostgreSQL
func (c *Config) GetPostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// PrintSummary выводит сводку конфигурации
func (c *Config) PrintSummary() {
	log.Printf("📋 Конфигурация приложения:")
	log.Printf("   • Окружение: %s", c.Environment)
	log.Printf("   • Биржа: %s %s", strings.ToUpper(c.Exchange), c.ExchangeType)
	log.Printf("   • Уровень логирования: %s", c.Logging.Level)
	log.Printf("   • Telegram режим: %s", c.TelegramMode)
	log.Printf("   • Telegram включен: %v", c.Telegram.Enabled)

	// Настройки пользователей по умолчанию
	log.Printf("   • Настройки по умолчанию:")
	log.Printf("     - Порог роста: %.1f%%", c.UserDefaults.MinGrowthThreshold)
	log.Printf("     - Порог падения: %.1f%%", c.UserDefaults.MinFallThreshold)
	log.Printf("     - Язык: %s", c.UserDefaults.Language)
	log.Printf("     - Часовой пояс: %s", c.UserDefaults.Timezone)

	// База данных
	log.Printf("   • PostgreSQL: %s:%d/%s", c.Database.Host, c.Database.Port, c.Database.Name)
	log.Printf("   • Redis: %s:%d (DB: %d, Pool: %d)",
		c.Redis.Host, c.Redis.Port, c.Redis.DB, c.Redis.PoolSize)

	if c.IsWebhookMode() {
		log.Printf("   • Webhook URL: %s", c.GetWebhookURL())
		log.Printf("   • Webhook порт: %d", c.Webhook.Port)
		log.Printf("   • TLS: %v", c.Webhook.UseTLS)
	} else {
		log.Printf("   • Polling timeout: %d сек", c.Polling.Timeout)
		log.Printf("   • Polling retry: %d сек", c.Polling.RetryInterval)
	}

	if c.Telegram.Enabled {
		token := c.Telegram.BotToken
		if len(token) > 10 {
			token = token[:10] + "..." + token[len(token)-10:]
		}
		log.Printf("   • Telegram Token: %s", token)
		log.Printf("   • Telegram Chat ID: %s", c.Telegram.ChatID)
	}

	log.Printf("   • Counter Analyzer включен: %v", c.AnalyzerConfigs.CounterAnalyzer.Enabled)
	log.Printf("   • HTTP сервер: %v (порт: %d)", c.Logging.HTTPEnabled, c.Logging.HTTPPort)
	log.Printf("   • Макс. символов: %d", c.MaxSymbolsToMonitor)
	log.Printf("   • Интервал обновления: %d сек", c.UpdateInterval)

	// Выводим информацию об анализаторах
	log.Printf("   • Анализаторы:")
	log.Printf("     - Growth: %v (порог: %.2f%%)",
		c.AnalyzerConfigs.GrowthAnalyzer.Enabled,
		c.AnalyzerConfigs.GrowthAnalyzer.MinGrowth)
	log.Printf("     - Fall: %v (порог: %.2f%%)",
		c.AnalyzerConfigs.FallAnalyzer.Enabled,
		c.AnalyzerConfigs.FallAnalyzer.MinFall)
	log.Printf("     - Counter: %v (период: %s)",
		c.AnalyzerConfigs.CounterAnalyzer.Enabled,
		c.GetCounterAnalysisPeriod())
}

// ============================================
// АНАЛИЗАТОРЫ
// ============================================

// IsCounterAnalyzerEnabled проверяет, включен ли анализатор счетчика
func (c *Config) IsCounterAnalyzerEnabled() bool {
	return c.AnalyzerConfigs.CounterAnalyzer.Enabled
}

// GetSymbolList возвращает список символов для мониторинга
func (c *Config) GetSymbolList() []string {
	if c.SymbolFilter == "" || c.SymbolFilter == "all" {
		return []string{} // Пустой список означает "все символы"
	}

	var symbols []string
	parts := strings.Split(c.SymbolFilter, ",")

	for _, part := range parts {
		symbol := strings.TrimSpace(part)
		if symbol != "" {
			// Если символ не содержит USDT, добавляем его
			if !strings.HasSuffix(strings.ToUpper(symbol), "USDT") {
				symbol = strings.ToUpper(symbol) + "USDT"
			}
			symbols = append(symbols, symbol)
		}
	}

	return symbols
}

// ShouldExcludeSymbol проверяет, нужно ли исключить символ
func (c *Config) ShouldExcludeSymbol(symbol string) bool {
	if c.ExcludeSymbols == "" {
		return false
	}

	excludeList := strings.Split(c.ExcludeSymbols, ",")
	for _, exclude := range excludeList {
		if strings.TrimSpace(strings.ToUpper(exclude)) == strings.ToUpper(symbol) {
			return true
		}
	}
	return false
}

// GetCounterBasePeriodMinutes получает базовый период CounterAnalyzer
func (c *Config) GetCounterBasePeriodMinutes() int {
	if settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings; settings != nil {
		if minutes, ok := settings["base_period_minutes"].(int); ok {
			return minutes
		}
	}
	return 1
}

// GetCounterAnalysisPeriod получает период анализа CounterAnalyzer
func (c *Config) GetCounterAnalysisPeriod() string {
	if settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings; settings != nil {
		if period, ok := settings["analysis_period"].(string); ok {
			return period
		}
	}
	return "15m"
}

// GetCounterGrowthThreshold получает порог роста CounterAnalyzer
func (c *Config) GetCounterGrowthThreshold() float64 {
	if settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings; settings != nil {
		if threshold, ok := settings["growth_threshold"].(float64); ok {
			return threshold
		}
	}
	return 0.1
}

// GetCounterFallThreshold получает порог падения CounterAnalyzer
func (c *Config) GetCounterFallThreshold() float64 {
	if settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings; settings != nil {
		if threshold, ok := settings["fall_threshold"].(float64); ok {
			return threshold
		}
	}
	return 0.1
}

// GetCounterNotificationEnabled проверяет, включены ли уведомления CounterAnalyzer
func (c *Config) GetCounterNotificationEnabled() bool {
	if settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings; settings != nil {
		if enabled, ok := settings["notification_enabled"].(bool); ok {
			return enabled
		}
	}
	return true
}

// GetCounterTrackGrowth проверяет, отслеживается ли рост
func (c *Config) GetCounterTrackGrowth() bool {
	if settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings; settings != nil {
		if track, ok := settings["track_growth"].(bool); ok {
			return track
		}
	}
	return true
}

// GetCounterTrackFall проверяет, отслеживается ли падение
func (c *Config) GetCounterTrackFall() bool {
	if settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings; settings != nil {
		if track, ok := settings["track_fall"].(bool); ok {
			return track
		}
	}
	return true
}

// GetCounterNotificationThreshold получает порог уведомлений
func (c *Config) GetCounterNotificationThreshold() int {
	if settings := c.AnalyzerConfigs.CounterAnalyzer.CustomSettings; settings != nil {
		if threshold, ok := settings["notification_threshold"].(int); ok {
			return threshold
		}
	}
	return 1
}

// GetGrowthContinuityThreshold получает порог непрерывности для GrowthAnalyzer
func (c *Config) GetGrowthContinuityThreshold() float64 {
	if settings := c.AnalyzerConfigs.GrowthAnalyzer.CustomSettings; settings != nil {
		if threshold, ok := settings["continuity_threshold"].(float64); ok {
			return threshold
		}
	}
	return 0.7
}

// GetFallContinuityThreshold получает порог непрерывности для FallAnalyzer
func (c *Config) GetFallContinuityThreshold() float64 {
	if settings := c.AnalyzerConfigs.FallAnalyzer.CustomSettings; settings != nil {
		if threshold, ok := settings["continuity_threshold"].(float64); ok {
			return threshold
		}
	}
	return 0.7
}

// GetContinuousAnalyzerMinPoints получает минимальное количество точек для ContinuousAnalyzer
func (c *Config) GetContinuousAnalyzerMinPoints() int {
	if settings := c.AnalyzerConfigs.ContinuousAnalyzer.CustomSettings; settings != nil {
		if points, ok := settings["min_continuous_points"].(int); ok {
			return points
		}
	}
	return 3
}

// GetEnabledAnalyzers возвращает список включенных анализаторов
func (c *Config) GetEnabledAnalyzers() []string {
	var enabled []string

	if c.AnalyzerConfigs.GrowthAnalyzer.Enabled {
		enabled = append(enabled, "growth_analyzer")
	}
	if c.AnalyzerConfigs.FallAnalyzer.Enabled {
		enabled = append(enabled, "fall_analyzer")
	}
	if c.AnalyzerConfigs.ContinuousAnalyzer.Enabled {
		enabled = append(enabled, "continuous_analyzer")
	}
	if c.AnalyzerConfigs.VolumeAnalyzer.Enabled {
		enabled = append(enabled, "volume_analyzer")
	}
	if c.AnalyzerConfigs.OpenInterestAnalyzer.Enabled {
		enabled = append(enabled, "open_interest_analyzer")
	}
	if c.AnalyzerConfigs.CounterAnalyzer.Enabled {
		enabled = append(enabled, "counter_analyzer")
	}

	return enabled
}

// ============================================
// TELEGRAM
// ============================================

// IsWebhookMode проверяет, используется ли режим вебхука
func (c *Config) IsWebhookMode() bool {
	return strings.ToLower(c.TelegramMode) == "webhook"
}

// IsPollingMode проверяет, используется ли режим polling
func (c *Config) IsPollingMode() bool {
	mode := strings.ToLower(c.TelegramMode)
	return mode == "polling" || mode == "" // по умолчанию polling
}

// GetWebhookURL возвращает URL вебхука
func (c *Config) GetWebhookURL() string {
	scheme := "https"
	if !c.Webhook.UseTLS {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, c.Webhook.Domain, c.Webhook.Port, c.Webhook.Path)
}

// ============================================
// REDIS
// ============================================

// GetRedisAddress возвращает адрес Redis
func (c *Config) GetRedisAddress() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// GetRedisPassword возвращает пароль Redis
func (c *Config) GetRedisPassword() string {
	return c.Redis.Password
}

// GetRedisDB возвращает номер базы данных Redis
func (c *Config) GetRedisDB() int {
	return c.Redis.DB
}

// GetRedisPoolSize возвращает размер пула соединений Redis
func (c *Config) GetRedisPoolSize() int {
	return c.Redis.PoolSize
}

// GetRedisMinIdleConns возвращает минимальное количество idle соединений Redis
func (c *Config) GetRedisMinIdleConns() int {
	return c.Redis.MinIdleConns
}

// ============================================
// ОБЩИЕ
// ============================================

// IsDev возвращает true если текущее окружение — разработка
func (c *Config) IsDev() bool {
	return c.Environment == "dev"
}
