// /internal/infrastructure/config/config.go
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ============================================
// КОНФИГУРАЦИЯ БАЗЫ ДАННЫХ
// ============================================

// DatabaseConfig - конфигурация базы данных
type DatabaseConfig struct {
	// Основные параметры подключения
	Host     string `mapstructure:"DB_HOST"`
	Port     int    `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Name     string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"DB_SSLMODE"`

	// Новое поле для включения/отключения БД
	Enabled bool `mapstructure:"DB_ENABLED"`

	// Настройки пула соединений
	MaxOpenConns    int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	MaxConnLifetime time.Duration `mapstructure:"DB_MAX_CONN_LIFETIME"`
	MaxConnIdleTime time.Duration `mapstructure:"DB_MAX_CONN_IDLE_TIME"`

	// Настройки миграций
	MigrationsPath    string `mapstructure:"DB_MIGRATIONS_PATH"`
	EnableAutoMigrate bool   `mapstructure:"DB_ENABLE_AUTO_MIGRATE"`
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	// Основные настройки подключения
	Host     string `mapstructure:"REDIS_HOST"`     // localhost
	Port     int    `mapstructure:"REDIS_PORT"`     // 6379
	Password string `mapstructure:"REDIS_PASSWORD"` // пустой или пароль
	DB       int    `mapstructure:"REDIS_DB"`       // 0

	// Новое поле для включения/отключения Redis
	Enabled bool `mapstructure:"REDIS_ENABLED"`

	// Настройки пула соединений
	PoolSize        int           `mapstructure:"REDIS_POOL_SIZE"`         // 10
	MinIdleConns    int           `mapstructure:"REDIS_MIN_IDLE_CONNS"`    // 5
	MaxRetries      int           `mapstructure:"REDIS_MAX_RETRIES"`       // 3
	MinRetryBackoff time.Duration `mapstructure:"REDIS_MIN_RETRY_BACKOFF"` // 8ms
	MaxRetryBackoff time.Duration `mapstructure:"REDIS_MAX_RETRY_BACKOFF"` // 512ms
	DialTimeout     time.Duration `mapstructure:"REDIS_DIAL_TIMEOUT"`      // 5s
	ReadTimeout     time.Duration `mapstructure:"REDIS_READ_TIMEOUT"`      // 3s
	WriteTimeout    time.Duration `mapstructure:"REDIS_WRITE_TIMEOUT"`     // 3s
	PoolTimeout     time.Duration `mapstructure:"REDIS_POOL_TIMEOUT"`      // 4s
	IdleTimeout     time.Duration `mapstructure:"REDIS_IDLE_TIMEOUT"`      // 5m
	MaxConnAge      time.Duration `mapstructure:"REDIS_MAX_CONN_AGE"`      // 0 (без ограничения)

	// Настройки кэширования
	DefaultTTL time.Duration `mapstructure:"REDIS_DEFAULT_TTL"` // 1h

	// Флаги
	UseTLS bool `mapstructure:"REDIS_USE_TLS"` // false
}

// ============================================
// КОНФИГУРАЦИЯ АНАЛИЗАТОРОВ
// ============================================

// AnalyzerConfig - конфигурация анализатора
type AnalyzerConfig struct {
	Enabled        bool                   `mapstructure:"ENABLED"`
	MinConfidence  float64                `mapstructure:"MIN_CONFIDENCE"`
	MinGrowth      float64                `mapstructure:"MIN_GROWTH"`
	MinFall        float64                `mapstructure:"MIN_FALL"`
	CustomSettings map[string]interface{} `mapstructure:"CUSTOM_SETTINGS,omitempty"`
}

// AnalyzerConfigs - конфигурация всех анализаторов
type AnalyzerConfigs struct {
	GrowthAnalyzer       AnalyzerConfig `mapstructure:"GROWTH_ANALYZER"`
	FallAnalyzer         AnalyzerConfig `mapstructure:"FALL_ANALYZER"`
	ContinuousAnalyzer   AnalyzerConfig `mapstructure:"CONTINUOUS_ANALYZER"`
	VolumeAnalyzer       AnalyzerConfig `mapstructure:"VOLUME_ANALYZER"`
	OpenInterestAnalyzer AnalyzerConfig `mapstructure:"OPEN_INTEREST_ANALYZER"`
	CounterAnalyzer      AnalyzerConfig `mapstructure:"COUNTER_ANALYZER"`
}

// UserDefaultsConfig - настройки пользователей по умолчанию
type UserDefaultsConfig struct {
	MinGrowthThreshold float64 `mapstructure:"COUNTER_GROWTH_THRESHOLD"`
	MinFallThreshold   float64 `mapstructure:"COUNTER_FALL_THRESHOLD"`
	Language           string  `mapstructure:"DEFAULT_LANGUAGE"`
	Timezone           string  `mapstructure:"DEFAULT_TIMEZONE"`
}

// ============================================
// ОСНОВНАЯ КОНФИГУРАЦИЯ ПРИЛОЖЕНИЯ (добавлено DatabaseConfig)
// ============================================

// Config - основная структура конфигурации
type Config struct {
	// ======================
	// ОСНОВНЫЕ НАСТРОЙКИ
	// ======================
	Environment string `mapstructure:"ENVIRONMENT"`
	Version     string `mapstructure:"VERSION"`

	// ======================
	// БАЗА ДАННЫХ
	// ======================
	Database DatabaseConfig `mapstructure:"DATABASE"`

	// Redis конфигурация Redis
	Redis RedisConfig `mapstructure:",squash"`

	// ======================
	// БИРЖА И API КЛЮЧИ
	// ======================
	Exchange     string `mapstructure:"EXCHANGE"`
	ExchangeType string `mapstructure:"EXCHANGE_TYPE"`

	// API ключи (общий формат)
	ApiKey    string `mapstructure:"API_KEY"`
	ApiSecret string `mapstructure:"API_SECRET"`
	BaseURL   string `mapstructure:"BASE_URL"`

	// Bybit специфичные (для обратной совместимости)
	BybitApiKey     string `mapstructure:"BYBIT_API_KEY"`
	BybitSecretKey  string `mapstructure:"BYBIT_SECRET_KEY"`
	BybitApiUrl     string `mapstructure:"BYBIT_API_URL"`
	FuturesCategory string `mapstructure:"FUTURES_CATEGORY"`

	// Binance специфичные (для обратной совместимости)
	BinanceApiKey    string `mapstructure:"BINANCE_API_KEY"`
	BinanceApiSecret string `mapstructure:"BINANCE_API_SECRET"`

	// ======================
	// СИМВОЛЫ И ФИЛЬТРАЦИЯ
	// ======================
	SymbolFilter        string  `mapstructure:"SYMBOL_FILTER"`
	ExcludeSymbols      string  `mapstructure:"EXCLUDE_SYMBOLS"`
	MaxSymbolsToMonitor int     `mapstructure:"MAX_SYMBOLS_TO_MONITOR"`
	MinVolumeFilter     float64 `mapstructure:"MIN_VOLUME_FILTER"`
	UpdateInterval      int     `mapstructure:"UPDATE_INTERVAL"` // Интервал обновления данных

	// ======================
	// ДВИЖОК АНАЛИЗА
	// ======================
	AnalysisEngine struct {
		UpdateInterval   int     `mapstructure:"ANALYSIS_UPDATE_INTERVAL"`
		AnalysisPeriods  []int   `mapstructure:"ANALYSIS_PERIODS"`
		MaxSymbolsPerRun int     `mapstructure:"ANALYSIS_MAX_SYMBOLS_PER_RUN"`
		SignalThreshold  float64 `mapstructure:"ANALYSIS_SIGNAL_THRESHOLD"`
		RetentionPeriod  int     `mapstructure:"ANALYSIS_RETENTION_PERIOD"`
		EnableCache      bool    `mapstructure:"ANALYSIS_ENABLE_CACHE"`
		EnableParallel   bool    `mapstructure:"ANALYSIS_ENABLE_PARALLEL"`
		MaxWorkers       int     `mapstructure:"ANALYSIS_MAX_WORKERS"`
		MinDataPoints    int     `mapstructure:"ANALYSIS_MIN_DATA_POINTS"`
	} `mapstructure:",squash"`

	// ======================
	// АНАЛИЗАТОРЫ
	// ======================
	AnalyzerConfigs AnalyzerConfigs `mapstructure:"ANALYZERS"`

	// ======================
	// ШИНА СОБЫТИЙ
	// ======================
	EventBus struct {
		BufferSize    int  `mapstructure:"EVENT_BUS_BUFFER_SIZE"`
		WorkerCount   int  `mapstructure:"EVENT_BUS_WORKER_COUNT"`
		EnableMetrics bool `mapstructure:"EVENT_BUS_ENABLE_METRICS"`
		EnableLogging bool `mapstructure:"EVENT_BUS_ENABLE_LOGGING"`
	} `mapstructure:",squash"`

	// ======================
	// ФИЛЬТРЫ СИГНАЛОВ
	// ======================
	SignalFilters struct {
		Enabled          bool     `mapstructure:"SIGNAL_FILTERS_ENABLED"`
		MinConfidence    float64  `mapstructure:"MIN_CONFIDENCE"`
		MaxSignalsPerMin int      `mapstructure:"MAX_SIGNALS_PER_MIN"`
		IncludePatterns  []string `mapstructure:"SIGNAL_INCLUDE_PATTERNS"`
		ExcludePatterns  []string `mapstructure:"SIGNAL_EXCLUDE_PATTERNS"`
	} `mapstructure:",squash"`

	// ======================
	// НАСТРОЙКИ ОТОБРАЖЕНИЯ
	// ======================
	Display struct {
		Mode               string `mapstructure:"DISPLAY_MODE"`
		MaxSignalsPerBatch int    `mapstructure:"MAX_SIGNALS_PER_BATCH"`
		MinConfidence      int    `mapstructure:"MIN_CONFIDENCE_DISPLAY"`
		DisplayGrowth      bool   `mapstructure:"DISPLAY_GROWTH"`
		DisplayFall        bool   `mapstructure:"DISPLAY_FALL"`
		DisplayPeriods     []int  `mapstructure:"DISPLAY_PERIODS"`
		UseColors          bool   `mapstructure:"USE_COLORS"`
	} `mapstructure:",squash"`

	// ======================
	// TELEGRAM УВЕДОМЛЕНИЯ
	// ======================
	Telegram struct {
		Enabled         bool    `mapstructure:"TELEGRAM_ENABLED"`
		BotToken        string  `mapstructure:"TG_API_KEY"`
		BotUsername     string  `mapstructure:"TG_BOT_USERNAME"`
		ChatID          string  `mapstructure:"TG_CHAT_ID"`
		NotifyGrowth    bool    `mapstructure:"TELEGRAM_NOTIFY_GROWTH"`
		NotifyFall      bool    `mapstructure:"TELEGRAM_NOTIFY_FALL"`
		GrowthThreshold float64 `mapstructure:"TELEGRAM_GROWTH_THRESHOLD"`
		FallThreshold   float64 `mapstructure:"TELEGRAM_FALL_THRESHOLD"`
		MessageFormat   string  `mapstructure:"MESSAGE_FORMAT"`
		Include24hStats bool    `mapstructure:"INCLUDE_24H_STATS"`
	} `mapstructure:",squash"`

	// ======================
	// MAX МЕССЕНДЖЕР
	// ======================
	MAX struct {
		Enabled  bool   `mapstructure:"MAX_ENABLED"`
		BotToken string `mapstructure:"MAX_BOT_TOKEN"`
		ChatID   int64  `mapstructure:"MAX_CHAT_ID"`
	} `mapstructure:",squash"`

	// ======================
	// TELEGRAM РЕЖИМ РАБОТЫ
	// ======================
	TelegramMode string `mapstructure:"TELEGRAM_MODE"` // "polling" или "webhook"

	// ======================
	// ВЕБХУК КОНФИГУРАЦИЯ
	// ======================
	Webhook struct {
		Domain      string `mapstructure:"WEBHOOK_DOMAIN"`
		Port        int    `mapstructure:"WEBHOOK_PORT"`
		Path        string `mapstructure:"WEBHOOK_PATH"`
		SecretToken string `mapstructure:"WEBHOOK_SECRET_TOKEN"`
		UseTLS      bool   `mapstructure:"WEBHOOK_USE_TLS"`
		TLSCertPath string `mapstructure:"WEBHOOK_TLS_CERT_PATH"`
		TLSKeyPath  string `mapstructure:"WEBHOOK_TLS_KEY_PATH"`
		MaxBodySize int64  `mapstructure:"WEBHOOK_MAX_BODY_SIZE"`
	} `mapstructure:",squash"`

	// ======================
	// POLLING КОНФИГУРАЦИЯ
	// ======================
	Polling struct {
		Timeout       int `mapstructure:"POLLING_TIMEOUT"`        // timeout в секундах
		Limit         int `mapstructure:"POLLING_LIMIT"`          // лимит обновлений
		RetryInterval int `mapstructure:"POLLING_RETRY_INTERVAL"` // интервал переподключения
	} `mapstructure:",squash"`

	// =============================
	// TELEGRAM STARS КОНФИГУРАЦИЯ
	// =============================
	TelegramStars struct {
		ProviderToken string `mapstructure:"TELEGRAM_STARS_PROVIDER_TOKEN"`
		BotUsername   string `mapstructure:"TELEGRAM_STARS_BOT_USERNAME"`
	}

	// ======================
	// ДОПОЛНИТЕЛЬНЫЙ МОНИТОРИНГ
	// ======================
	Monitoring struct {
		ChatID       string `mapstructure:"MONITORING_CHAT_ID"`
		Enabled      bool   `mapstructure:"MONITORING_ENABLED"`
		NotifyGrowth bool   `mapstructure:"MONITORING_NOTIFY_GROWTH"`
		NotifyFall   bool   `mapstructure:"MONITORING_NOTIFY_FALL"`
		TestMode     bool   `mapstructure:"MONITORING_TEST_MODE"`
	} `mapstructure:",squash"`

	// ======================
	// ПОДПИСКИ И БИЛЛИНГ
	// ======================
	Subscriptions struct {
		Enabled           bool   `mapstructure:"ENABLE_SUBSCRIPTIONS"`
		StripeSecretKey   string `mapstructure:"STRIPE_SECRET_KEY"`
		StripeWebhookKey  string `mapstructure:"STRIPE_WEBHOOK_SECRET"`
		DefaultTrialDays  int    `mapstructure:"DEFAULT_TRIAL_DAYS"`
		EnableAutoRenewal bool   `mapstructure:"ENABLE_AUTO_RENEWAL"`
	} `mapstructure:",squash"`

	// ======================
	// ЛОГИРОВАНИЕ И МОНИТОРИНГ
	// ======================
	Logging struct {
		Level       string `mapstructure:"LOG_LEVEL"`
		File        string `mapstructure:"LOG_FILE"`
		ToConsole   bool   `mapstructure:"LOG_TO_CONSOLE,omitempty"`
		ToFile      bool   `mapstructure:"LOG_TO_FILE,omitempty"`
		DebugMode   bool   `mapstructure:"DEBUG_MODE,omitempty"`
		HTTPEnabled bool   `mapstructure:"HTTP_ENABLED"`
		HTTPPort    int    `mapstructure:"HTTP_PORT"`
	} `mapstructure:",squash"`

	// ======================
	// ПРОИЗВОДИТЕЛЬНОСТЬ
	// ======================
	Performance struct {
		RateLimitDelay        time.Duration `mapstructure:"RATE_LIMIT_DELAY,omitempty"`
		MaxConcurrentRequests int           `mapstructure:"MAX_CONCURRENT_REQUESTS,omitempty"`
	} `mapstructure:",squash"`

	// ======================
	// НАСТРОЙКИ ПОЛЬЗОВАТЕЛЕЙ ПО УМОЛЧАНИЮ
	// ======================
	UserDefaults UserDefaultsConfig `mapstructure:",squash"`

	// ======================
	// ДЛЯ ОБРАТНОЙ СОВМЕСТИМОСТИ
	// ======================

	// Старые поля Telegram (для совместимости)
	TelegramBotToken        string  `mapstructure:"-"`
	TelegramChatID          string  `mapstructure:"-"`
	TelegramNotifyGrowth    bool    `mapstructure:"-"`
	TelegramNotifyFall      bool    `mapstructure:"-"`
	TelegramGrowthThreshold float64 `mapstructure:"-"`
	TelegramFallThreshold   float64 `mapstructure:"-"`
	MessageFormat           string  `mapstructure:"-"`
	Include24hStats         bool    `mapstructure:"-"`

	// Старые поля мониторинга
	MonitoringChatID       string `mapstructure:"-"`
	MonitoringEnabled      bool   `mapstructure:"-"`
	MonitoringNotifyGrowth bool   `mapstructure:"-"`
	MonitoringNotifyFall   bool   `mapstructure:"-"`
	MonitoringTestMode     bool   `mapstructure:"-"`

	// Старые поля производительности
	LogLevel              string        `mapstructure:"-"`
	LogFile               string        `mapstructure:"-"`
	HTTPPort              int           `mapstructure:"-"`
	HTTPEnabled           bool          `mapstructure:"-"`
	DebugMode             bool          `mapstructure:"-"`
	LogToConsole          bool          `mapstructure:"-"`
	LogToFile             bool          `mapstructure:"-"`
	RateLimitDelay        time.Duration `mapstructure:"-"`
	MaxConcurrentRequests int           `mapstructure:"-"`
}

// ============================================
// ЗАГРУЗКА КОНФИГУРАЦИИ (обновленная)
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

// ============================================
// ВАЛИДАЦИЯ (обновленная)
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
		if c.MAX.ChatID == 0 {
			validationErrors = append(validationErrors, "MAX_CHAT_ID is required when MAX is enabled")
		}
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

// ============================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ (добавлены новые)
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

// PrintSummary обновлен для отображения настроек БД
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
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func parseIntList(value string) []int {
	var result []int
	if value == "" {
		return result
	}

	parts := strings.Split(value, ",")
	for _, part := range parts {
		if part == "" {
			continue
		}
		if intValue, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			result = append(result, intValue)
		}
	}
	return result
}

func parsePatterns(value string) []string {
	if value == "" {
		return []string{}
	}

	patterns := strings.Split(value, ",")
	var result []string
	for _, pattern := range patterns {
		if trimmed := strings.TrimSpace(pattern); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func isValidPeriod(period string) bool {
	validPeriods := map[string]bool{
		"1m":  true,
		"5m":  true,
		"15m": true,
		"30m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
	}
	return validPeriods[period]
}

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

// Добавляю вспомогательные методы:
func (c *Config) IsWebhookMode() bool {
	return strings.ToLower(c.TelegramMode) == "webhook"
}

func (c *Config) IsPollingMode() bool {
	mode := strings.ToLower(c.TelegramMode)
	return mode == "polling" || mode == "" // по умолчанию polling
}

func (c *Config) GetWebhookURL() string {
	scheme := "https"
	if !c.Webhook.UseTLS {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, c.Webhook.Domain, c.Webhook.Port, c.Webhook.Path)
}

// Validate проверяет конфигурацию
func (c *Config) Validate() error {
	// Используем встроенную валидацию
	return c.validate()
}

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

// IsDev возвращает true если текущее окружение — разработка
func (c *Config) IsDev() bool {
	return c.Environment == "dev"
}
