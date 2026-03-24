// internal/infrastructure/config/types.go
package config

import "time"

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
// ОСНОВНАЯ КОНФИГУРАЦИЯ ПРИЛОЖЕНИЯ
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
	// MAX РЕЖИМ РАБОТЫ
	// ======================
	MAXMode string `mapstructure:"MAX_MODE"` // "polling" или "webhook"

	// ======================
	// TELEGRAM РЕЖИМ РАБОТЫ
	// ======================
	TelegramMode string `mapstructure:"TELEGRAM_MODE"` // "polling" или "webhook"

	// ======================
	// ВЕБХУК КОНФИГУРАЦИЯ (TELEGRAM)
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
	// ВЕБХУК КОНФИГУРАЦИЯ (MAX)
	// ======================
	MAXWebhook struct {
		Domain      string `mapstructure:"MAX_WEBHOOK_DOMAIN"`
		Port        int    `mapstructure:"MAX_WEBHOOK_PORT"`
		Path        string `mapstructure:"MAX_WEBHOOK_PATH"`
		SecretToken string `mapstructure:"MAX_WEBHOOK_SECRET_TOKEN"`
		UseTLS      bool   `mapstructure:"MAX_WEBHOOK_USE_TLS"`
		TLSCertPath string `mapstructure:"MAX_WEBHOOK_TLS_CERT_PATH"`
		TLSKeyPath  string `mapstructure:"MAX_WEBHOOK_TLS_KEY_PATH"`
		MaxBodySize int64  `mapstructure:"MAX_WEBHOOK_MAX_BODY_SIZE"`
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

	// =============================
	// Т-БАНК ЭКВАЙРИНГ
	// =============================
	TBank struct {
		Enabled       bool   `mapstructure:"TBANK_ENABLED"`
		TerminalKey   string `mapstructure:"TBANK_TERMINAL_KEY"`
		Password      string `mapstructure:"TBANK_PASSWORD"`
		NotifyURL     string `mapstructure:"TBANK_NOTIFY_URL"`       // URL для уведомлений от Т-Банк
		NotifyPort    int    `mapstructure:"TBANK_NOTIFY_PORT"`      // порт для сервера уведомлений
		SuccessURL    string `mapstructure:"TBANK_SUCCESS_URL"`      // Telegram: редирект при успехе
		FailURL       string `mapstructure:"TBANK_FAIL_URL"`         // Telegram: редирект при ошибке
		MaxSuccessURL string `mapstructure:"MAX_TBANK_SUCCESS_URL"`  // MAX: редирект при успехе
		MaxFailURL    string `mapstructure:"MAX_TBANK_FAIL_URL"`     // MAX: редирект при ошибке
	}

	// ============================
	// AUTH OTP SERVER
	// ============================
	Auth struct {
		Enabled  bool          `mapstructure:"AUTH_ENABLED"`
		Port     int           `mapstructure:"AUTH_PORT"`
		Secret   string        `mapstructure:"AUTH_INTERNAL_SECRET"`
		OTPTTLSec int          `mapstructure:"AUTH_OTP_TTL_SEC"` // TTL кода в секундах (default 300)
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
