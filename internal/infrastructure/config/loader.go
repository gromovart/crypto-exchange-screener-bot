// internal/infrastructure/config/loader.go
package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
)

// ============================================
// ЗАГРУЗКА КОНФИГУРАЦИИ
// ============================================

// LoadConfig загружает конфигурацию из .env файла
func LoadConfig(path string) (*Config, error) {
	if err := godotenv.Load(path); err != nil {
		fmt.Printf("⚠️  Config file not found, using environment variables\n")
	}

	cfg := &Config{}

	// ======================
	// ОСНОВНЫЕ НАСТРОЙКИ
	// ======================
	cfg.Environment = getEnv("ENVIRONMENT", "production")
	cfg.Version = getEnv("VERSION", "1.0.0")

	// ======================
	// НАСТРОЙКИ ПОЛЬЗОВАТЕЛЕЙ ПО УМОЛЧАНИЮ
	// ======================
	cfg.UserDefaults.MinGrowthThreshold = getEnvFloat("COUNTER_GROWTH_THRESHOLD", 2.0)
	cfg.UserDefaults.MinFallThreshold = getEnvFloat("COUNTER_FALL_THRESHOLD", 2.0)
	cfg.UserDefaults.Language = getEnv("DEFAULT_LANGUAGE", "ru")
	cfg.UserDefaults.Timezone = getEnv("DEFAULT_TIMEZONE", "Europe/Moscow")

	// ======================
	// БАЗА ДАННЫХ
	// ======================
	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnvInt("DB_PORT", 5432)
	cfg.Database.User = getEnv("DB_USER", "")
	cfg.Database.Password = getEnv("DB_PASSWORD", "")
	cfg.Database.Name = getEnv("DB_NAME", "")
	cfg.Database.SSLMode = getEnv("DB_SSLMODE", "disable")
	cfg.Database.MaxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	cfg.Database.MaxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", 10)
	cfg.Database.MaxConnLifetime = getEnvDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute)
	cfg.Database.MaxConnIdleTime = getEnvDuration("DB_MAX_CONN_IDLE_TIME", 10*time.Minute)
	cfg.Database.MigrationsPath = getEnv("DB_MIGRATIONS_PATH", "./persistence/postgres/migrations")
	cfg.Database.EnableAutoMigrate = getEnvBool("DB_ENABLE_AUTO_MIGRATE", true)
	cfg.Database.Enabled = getEnvBool("DB_ENABLED", true)

	// ======================
	// REDIS
	// ======================
	cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Redis.Port = getEnvInt("REDIS_PORT", 6379)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvInt("REDIS_DB", 0)
	cfg.Redis.PoolSize = getEnvInt("REDIS_POOL_SIZE", 10)
	cfg.Redis.MinIdleConns = getEnvInt("REDIS_MIN_IDLE_CONNS", 5)
	cfg.Redis.MaxRetries = getEnvInt("REDIS_MAX_RETRIES", 3)
	cfg.Redis.MinRetryBackoff = getEnvDuration("REDIS_MIN_RETRY_BACKOFF", 8*time.Millisecond)
	cfg.Redis.MaxRetryBackoff = getEnvDuration("REDIS_MAX_RETRY_BACKOFF", 512*time.Millisecond)
	cfg.Redis.DialTimeout = getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second)
	cfg.Redis.ReadTimeout = getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second)
	cfg.Redis.WriteTimeout = getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second)
	cfg.Redis.PoolTimeout = getEnvDuration("REDIS_POOL_TIMEOUT", 4*time.Second)
	cfg.Redis.IdleTimeout = getEnvDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute)
	cfg.Redis.MaxConnAge = getEnvDuration("REDIS_MAX_CONN_AGE", 0)
	cfg.Redis.DefaultTTL = getEnvDuration("REDIS_DEFAULT_TTL", 1*time.Hour)
	cfg.Redis.UseTLS = getEnvBool("REDIS_USE_TLS", false)
	cfg.Redis.Enabled = getEnvBool("REDIS_ENABLED", true)

	// ======================
	// БИРЖА И API КЛЮЧИ
	// ======================
	cfg.Exchange = getEnv("EXCHANGE", "bybit")
	cfg.ExchangeType = getEnv("EXCHANGE_TYPE", "futures")

	// API ключи (универсальный формат)
	cfg.ApiKey = getEnv("API_KEY", "")
	cfg.ApiSecret = getEnv("API_SECRET", "")
	cfg.BaseURL = getEnv("BASE_URL", "")

	// Обратная совместимость
	if cfg.Exchange == "bybit" {
		if cfg.ApiKey == "" {
			cfg.ApiKey = getEnv("BYBIT_API_KEY", "")
		}
		if cfg.ApiSecret == "" {
			cfg.ApiSecret = getEnv("BYBIT_SECRET_KEY", "")
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = getEnv("BYBIT_API_URL", "https://api.bybit.com")
		}
		cfg.FuturesCategory = getEnv("FUTURES_CATEGORY", "linear")
	} else if cfg.Exchange == "binance" {
		if cfg.ApiKey == "" {
			cfg.ApiKey = getEnv("BINANCE_API_KEY", "")
		}
		if cfg.ApiSecret == "" {
			cfg.ApiSecret = getEnv("BINANCE_API_SECRET", "")
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.binance.com"
		}
	}

	// Сохраняем для обратной совместимости
	cfg.BybitApiKey = cfg.ApiKey
	cfg.BybitSecretKey = cfg.ApiSecret
	cfg.BybitApiUrl = cfg.BaseURL
	cfg.BinanceApiKey = cfg.ApiKey
	cfg.BinanceApiSecret = cfg.ApiSecret

	// ======================
	// СИМВОЛЫ И ФИЛЬТРАЦИЯ
	// ======================
	cfg.SymbolFilter = getEnv("SYMBOL_FILTER", "")
	cfg.ExcludeSymbols = getEnv("EXCLUDE_SYMBOLS", "")
	cfg.MaxSymbolsToMonitor = getEnvInt("MAX_SYMBOLS_TO_MONITOR", 50)
	cfg.MinVolumeFilter = getEnvFloat("MIN_VOLUME_FILTER", 100000)
	cfg.UpdateInterval = getEnvInt("UPDATE_INTERVAL", 30)

	// ======================
	// ДВИЖОК АНАЛИЗА
	// ======================
	cfg.AnalysisEngine.UpdateInterval = getEnvInt("ANALYSIS_UPDATE_INTERVAL", 30)
	cfg.AnalysisEngine.AnalysisPeriods = parseIntList(getEnv("ANALYSIS_PERIODS", "1,5,15,30"))
	cfg.AnalysisEngine.MaxSymbolsPerRun = getEnvInt("ANALYSIS_MAX_SYMBOLS_PER_RUN", 50)
	cfg.AnalysisEngine.SignalThreshold = getEnvFloat("ANALYSIS_SIGNAL_THRESHOLD", 2.0)
	cfg.AnalysisEngine.RetentionPeriod = getEnvInt("ANALYSIS_RETENTION_PERIOD", 24)
	cfg.AnalysisEngine.EnableCache = getEnvBool("ANALYSIS_ENABLE_CACHE", true)
	cfg.AnalysisEngine.EnableParallel = getEnvBool("ANALYSIS_ENABLE_PARALLEL", true)
	cfg.AnalysisEngine.MaxWorkers = getEnvInt("ANALYSIS_MAX_WORKERS", 5)
	cfg.AnalysisEngine.MinDataPoints = getEnvInt("ANALYSIS_MIN_DATA_POINTS", 3)

	// ======================
	// АНАЛИЗАТОРЫ
	// ======================
	cfg.AnalyzerConfigs = AnalyzerConfigs{
		GrowthAnalyzer: AnalyzerConfig{
			Enabled:       getEnvBool("GROWTH_ANALYZER_ENABLED", true),
			MinConfidence: getEnvFloat("GROWTH_ANALYZER_MIN_CONFIDENCE", 60.0),
			MinGrowth:     getEnvFloat("GROWTH_ANALYZER_MIN_GROWTH", 2.0),
			CustomSettings: map[string]interface{}{
				"continuity_threshold": getEnvFloat("GROWTH_ANALYZER_CONTINUITY_THRESHOLD", 0.7),
				"volume_weight":        0.2,
			},
		},
		FallAnalyzer: AnalyzerConfig{
			Enabled:       getEnvBool("FALL_ANALYZER_ENABLED", true),
			MinConfidence: getEnvFloat("FALL_ANALYZER_MIN_CONFIDENCE", 60.0),
			MinFall:       getEnvFloat("FALL_ANALYZER_MIN_FALL", 2.0),
			CustomSettings: map[string]interface{}{
				"continuity_threshold": getEnvFloat("FALL_ANALYZER_CONTINUITY_THRESHOLD", 0.7),
				"volume_weight":        0.2,
			},
		},
		ContinuousAnalyzer: AnalyzerConfig{
			Enabled: getEnvBool("CONTINUOUS_ANALYZER_ENABLED", true),
			CustomSettings: map[string]interface{}{
				"min_continuous_points": getEnvInt("CONTINUOUS_ANALYZER_MIN_POINTS", 3),
			},
		},
		VolumeAnalyzer: AnalyzerConfig{
			Enabled:       getEnvBool("VOLUME_ANALYZER_ENABLED", true),
			MinConfidence: getEnvFloat("VOLUME_ANALYZER_MIN_CONFIDENCE", 30.0),
			CustomSettings: map[string]interface{}{
				"min_volume": getEnvFloat("VOLUME_ANALYZER_MIN_VOLUME", 100000.0),
			},
		},
		OpenInterestAnalyzer: AnalyzerConfig{
			Enabled:       getEnvBool("OPEN_INTEREST_ANALYZER_ENABLED", false),
			MinConfidence: getEnvFloat("OPEN_INTEREST_MIN_CONFIDENCE", 50.0),
			CustomSettings: map[string]interface{}{
				"min_price_change":     getEnvFloat("OPEN_INTEREST_MIN_PRICE_CHANGE", 1.0),
				"min_price_fall":       getEnvFloat("OPEN_INTEREST_MIN_PRICE_FALL", 1.0),
				"min_oi_change":        getEnvFloat("OPEN_INTEREST_MIN_OI_CHANGE", 5.0),
				"extreme_oi_threshold": getEnvFloat("OPEN_INTEREST_EXTREME_THRESHOLD", 1.5),
				"analyzer_weight":      getEnvFloat("OPEN_INTEREST_ANALYZER_WEIGHT", 0.6),
				"notify_enabled":       getEnvBool("OPEN_INTEREST_NOTIFY_ENABLED", true),
			},
		},
		CounterAnalyzer: AnalyzerConfig{
			Enabled: getEnvBool("COUNTER_ANALYZER_ENABLED", true),
			CustomSettings: map[string]interface{}{
				"base_period_minutes":    getEnvInt("COUNTER_BASE_PERIOD_MINUTES", 1),
				"analysis_period":        getEnv("COUNTER_ANALYSIS_PERIOD", "15m"),
				"growth_threshold":       getEnvFloat("COUNTER_GROWTH_THRESHOLD", 0.1),
				"fall_threshold":         getEnvFloat("COUNTER_FALL_THRESHOLD", 0.1),
				"track_growth":           getEnvBool("COUNTER_TRACK_GROWTH", true),
				"track_fall":             getEnvBool("COUNTER_TRACK_FALL", true),
				"notify_on_signal":       getEnvBool("COUNTER_NOTIFY_ON_SIGNAL", true),
				"notification_threshold": getEnvInt("COUNTER_NOTIFICATION_THRESHOLD", 1),
				"chart_provider":         getEnv("COUNTER_CHART_PROVIDER", "coinglass"),
				"notification_enabled":   getEnvBool("COUNTER_NOTIFICATION_ENABLED", true),
				"max_signals_5m":         getEnvInt("COUNTER_MAX_SIGNALS_5MIN", 5),
				"max_signals_15m":        getEnvInt("COUNTER_MAX_SIGNALS_15MIN", 8),
				"max_signals_30m":        getEnvInt("COUNTER_MAX_SIGNALS_30MIN", 10),
				"max_signals_1h":         getEnvInt("COUNTER_MAX_SIGNALS_1HOUR", 12),
				"max_signals_4h":         getEnvInt("COUNTER_MAX_SIGNALS_4HOURS", 15),
				"max_signals_1d":         getEnvInt("COUNTER_MAX_SIGNALS_1DAY", 20),
			},
		},
	}

	// ======================
	// ШИНА СОБЫТИЙ
	// ======================
	cfg.EventBus.BufferSize = getEnvInt("EVENT_BUS_BUFFER_SIZE", 1000)
	cfg.EventBus.WorkerCount = getEnvInt("EVENT_BUS_WORKER_COUNT", 5)
	cfg.EventBus.EnableMetrics = getEnvBool("EVENT_BUS_ENABLE_METRICS", true)
	cfg.EventBus.EnableLogging = getEnvBool("EVENT_BUS_ENABLE_LOGGING", true)

	// ======================
	// ФИЛЬТРЫ СИГНАЛОВ
	// ======================
	cfg.SignalFilters.Enabled = getEnvBool("SIGNAL_FILTERS_ENABLED", true)
	cfg.SignalFilters.MinConfidence = getEnvFloat("MIN_CONFIDENCE", 50.0)
	cfg.SignalFilters.MaxSignalsPerMin = getEnvInt("MAX_SIGNALS_PER_MIN", 5)
	cfg.SignalFilters.IncludePatterns = parsePatterns(getEnv("SIGNAL_INCLUDE_PATTERNS", ""))
	cfg.SignalFilters.ExcludePatterns = parsePatterns(getEnv("SIGNAL_EXCLUDE_PATTERNS", ""))

	// ======================
	// НАСТРОЙКИ ОТОБРАЖЕНИЯ
	// ======================
	cfg.Display.Mode = getEnv("DISPLAY_MODE", "compact")
	cfg.Display.MaxSignalsPerBatch = getEnvInt("MAX_SIGNALS_PER_BATCH", 10)
	cfg.Display.MinConfidence = getEnvInt("MIN_CONFIDENCE_DISPLAY", 30)
	cfg.Display.DisplayGrowth = getEnvBool("DISPLAY_GROWTH", true)
	cfg.Display.DisplayFall = getEnvBool("DISPLAY_FALL", true)
	cfg.Display.DisplayPeriods = parseIntList(getEnv("DISPLAY_PERIODS", "5,15,30"))
	cfg.Display.UseColors = getEnvBool("USE_COLORS", true)

	// ======================
	// TELEGRAM УВЕДОМЛЕНИЯ
	// ======================
	cfg.Telegram.Enabled = getEnvBool("TELEGRAM_ENABLED", false)
	cfg.Telegram.BotToken = getEnv("TG_API_KEY", "")
	cfg.Telegram.BotUsername = getEnv("TG_BOT_USERNAME", "")
	cfg.Telegram.ChatID = getEnv("TG_CHAT_ID", "")
	cfg.Telegram.NotifyGrowth = getEnvBool("TELEGRAM_NOTIFY_GROWTH", true)
	cfg.Telegram.NotifyFall = getEnvBool("TELEGRAM_NOTIFY_FALL", true)
	cfg.Telegram.GrowthThreshold = getEnvFloat("TELEGRAM_GROWTH_THRESHOLD", 0.5)
	cfg.Telegram.FallThreshold = getEnvFloat("TELEGRAM_FALL_THRESHOLD", 0.5)
	cfg.Telegram.MessageFormat = getEnv("MESSAGE_FORMAT", "compact")
	cfg.Telegram.Include24hStats = getEnvBool("INCLUDE_24H_STATS", false)

	// ======================
	// MAX МЕССЕНДЖЕР
	// ======================
	cfg.MAX.Enabled = getEnvBool("MAX_ENABLED", false)
	cfg.MAX.BotToken = getEnv("MAX_BOT_TOKEN", "")
	cfg.MAX.ChatID = getEnvInt64("MAX_CHAT_ID", 0)

	// ======================
	// TELEGRAM РЕЖИМ РАБОТЫ
	// ======================
	cfg.TelegramMode = getEnv("TELEGRAM_MODE", "polling")

	// ======================
	// ВЕБХУК КОНФИГУРАЦИЯ
	// ======================
	cfg.Webhook.Domain = getEnv("WEBHOOK_DOMAIN", "localhost")
	cfg.Webhook.Port = getEnvInt("WEBHOOK_PORT", 8443)
	cfg.Webhook.Path = getEnv("WEBHOOK_PATH", "/webhook")
	cfg.Webhook.SecretToken = getEnv("WEBHOOK_SECRET_TOKEN", "")
	cfg.Webhook.UseTLS = getEnvBool("WEBHOOK_USE_TLS", true)
	cfg.Webhook.TLSCertPath = getEnv("WEBHOOK_TLS_CERT_PATH", "")
	cfg.Webhook.TLSKeyPath = getEnv("WEBHOOK_TLS_KEY_PATH", "")
	cfg.Webhook.MaxBodySize = getEnvInt64("WEBHOOK_MAX_BODY_SIZE", 1024*1024) // 1MB

	// ======================
	// POLLING КОНФИГУРАЦИЯ
	// ======================
	cfg.Polling.Timeout = getEnvInt("POLLING_TIMEOUT", 30)
	cfg.Polling.Limit = getEnvInt("POLLING_LIMIT", 100)
	cfg.Polling.RetryInterval = getEnvInt("POLLING_RETRY_INTERVAL", 5)

	// =============================
	// TELEGRAM STARS КОНФИГУРАЦИЯ
	// =============================
	cfg.TelegramStars.ProviderToken = getEnv("TELEGRAM_STARS_PROVIDER_TOKEN", "")
	cfg.TelegramStars.BotUsername = getEnv("TELEGRAM_STARS_BOT_USERNAME", "")

	// =============================
	// Т-БАНК ЭКВАЙРИНГ
	// =============================
	cfg.TBank.Enabled = getEnvBool("TBANK_ENABLED", false)
	cfg.TBank.TerminalKey = getEnv("TBANK_TERMINAL_KEY", "")
	cfg.TBank.Password = getEnv("TBANK_PASSWORD", "")
	cfg.TBank.NotifyURL = getEnv("TBANK_NOTIFY_URL", "")
	cfg.TBank.NotifyPort = getEnvInt("TBANK_NOTIFY_PORT", 8082)
	cfg.TBank.SuccessURL = getEnv("TBANK_SUCCESS_URL", "")
	cfg.TBank.FailURL = getEnv("TBANK_FAIL_URL", "")
	cfg.TBank.MaxSuccessURL = getEnv("MAX_TBANK_SUCCESS_URL", "")
	cfg.TBank.MaxFailURL = getEnv("MAX_TBANK_FAIL_URL", "")

	// ============================
	// AUTH OTP SERVER
	// ============================
	cfg.Auth.Enabled   = getEnvBool("AUTH_ENABLED", false)
	cfg.Auth.Port      = getEnvInt("AUTH_PORT", 8081)
	cfg.Auth.Secret    = getEnv("AUTH_INTERNAL_SECRET", "")
	cfg.Auth.OTPTTLSec = getEnvInt("AUTH_OTP_TTL_SEC", 300)

	// ======================
	// ДОПОЛНИТЕЛЬНЫЙ МОНИТОРИНГ
	// ======================
	cfg.Monitoring.ChatID = getEnv("MONITORING_CHAT_ID", "")
	cfg.Monitoring.Enabled = getEnvBool("MONITORING_ENABLED", false)
	cfg.Monitoring.NotifyGrowth = getEnvBool("MONITORING_NOTIFY_GROWTH", true)
	cfg.Monitoring.NotifyFall = getEnvBool("MONITORING_NOTIFY_FALL", true)
	cfg.Monitoring.TestMode = getEnvBool("MONITORING_TEST_MODE", false)

	// ======================
	// ПОДПИСКИ И БИЛЛИНГ
	// ======================
	cfg.Subscriptions.Enabled = getEnvBool("ENABLE_SUBSCRIPTIONS", false)
	cfg.Subscriptions.StripeSecretKey = getEnv("STRIPE_SECRET_KEY", "")
	cfg.Subscriptions.StripeWebhookKey = getEnv("STRIPE_WEBHOOK_SECRET", "")
	cfg.Subscriptions.DefaultTrialDays = getEnvInt("DEFAULT_TRIAL_DAYS", 7)
	cfg.Subscriptions.EnableAutoRenewal = getEnvBool("ENABLE_AUTO_RENEWAL", true)

	// ======================
	// ЛОГИРОВАНИЕ И МОНИТОРИНГ
	// ======================
	cfg.Logging.Level = getEnv("LOG_LEVEL", "info")
	cfg.Logging.File = getEnv("LOG_FILE", "logs/growth_monitor.log")
	cfg.Logging.ToConsole = getEnvBool("LOG_TO_CONSOLE", true)
	cfg.Logging.ToFile = getEnvBool("LOG_TO_FILE", true)
	cfg.Logging.DebugMode = getEnvBool("DEBUG_MODE", false)
	cfg.Logging.HTTPEnabled = getEnvBool("HTTP_ENABLED", false)
	cfg.Logging.HTTPPort = getEnvInt("HTTP_PORT", 8080)

	// ======================
	// ПРОИЗВОДИТЕЛЬНОСТЬ
	// ======================
	cfg.Performance.RateLimitDelay = getEnvDuration("RATE_LIMIT_DELAY", 100*time.Millisecond)
	cfg.Performance.MaxConcurrentRequests = getEnvInt("MAX_CONCURRENT_REQUESTS", 10)

	// ======================
	// ОБРАТНАЯ СОВМЕСТИМОСТЬ
	// ======================
	// Назначаем старые поля из новых структур для обратной совместимости
	cfg.TelegramBotToken = cfg.Telegram.BotToken
	cfg.TelegramChatID = cfg.Telegram.ChatID
	cfg.TelegramNotifyGrowth = cfg.Telegram.NotifyGrowth
	cfg.TelegramNotifyFall = cfg.Telegram.NotifyFall
	cfg.TelegramGrowthThreshold = cfg.Telegram.GrowthThreshold
	cfg.TelegramFallThreshold = cfg.Telegram.FallThreshold
	cfg.MessageFormat = cfg.Telegram.MessageFormat
	cfg.Include24hStats = cfg.Telegram.Include24hStats

	cfg.MonitoringChatID = cfg.Monitoring.ChatID
	cfg.MonitoringEnabled = cfg.Monitoring.Enabled
	cfg.MonitoringNotifyGrowth = cfg.Monitoring.NotifyGrowth
	cfg.MonitoringNotifyFall = cfg.Monitoring.NotifyFall
	cfg.MonitoringTestMode = cfg.Monitoring.TestMode

	cfg.LogLevel = cfg.Logging.Level
	cfg.LogFile = cfg.Logging.File
	cfg.LogToConsole = cfg.Logging.ToConsole
	cfg.LogToFile = cfg.Logging.ToFile
	cfg.DebugMode = cfg.Logging.DebugMode
	cfg.HTTPEnabled = cfg.Logging.HTTPEnabled
	cfg.HTTPPort = cfg.Logging.HTTPPort

	cfg.RateLimitDelay = cfg.Performance.RateLimitDelay
	cfg.MaxConcurrentRequests = cfg.Performance.MaxConcurrentRequests

	// ======================
	// ВАЛИДАЦИЯ КОНФИГУРАЦИИ
	// ======================
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}
