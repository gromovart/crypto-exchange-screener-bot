// internal/infrastructure/config/config.go
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

// CounterAnalyzerConfig - конфигурация анализатора счетчика
type CounterAnalyzerConfig struct {
	Enabled               bool    `mapstructure:"COUNTER_ANALYZER_ENABLED"`
	BasePeriodMinutes     int     `mapstructure:"COUNTER_BASE_PERIOD_MINUTES"`
	DefaultPeriod         string  `mapstructure:"COUNTER_DEFAULT_PERIOD"`
	MaxSignals5Min        int     `mapstructure:"COUNTER_MAX_SIGNALS_5MIN"`
	MaxSignals15Min       int     `mapstructure:"COUNTER_MAX_SIGNALS_15MIN"`
	MaxSignals30Min       int     `mapstructure:"COUNTER_MAX_SIGNALS_30MIN"`
	MaxSignals1Hour       int     `mapstructure:"COUNTER_MAX_SIGNALS_1HOUR"`
	MaxSignals4Hours      int     `mapstructure:"COUNTER_MAX_SIGNALS_4HOURS"`
	MaxSignals1Day        int     `mapstructure:"COUNTER_MAX_SIGNALS_1DAY"`
	GrowthThreshold       float64 `mapstructure:"COUNTER_GROWTH_THRESHOLD"`
	FallThreshold         float64 `mapstructure:"COUNTER_FALL_THRESHOLD"`
	TrackGrowth           bool    `mapstructure:"COUNTER_TRACK_GROWTH"`
	TrackFall             bool    `mapstructure:"COUNTER_TRACK_FALL"`
	NotifyOnSignal        bool    `mapstructure:"COUNTER_NOTIFY_ON_SIGNAL"`
	NotificationThreshold int     `mapstructure:"COUNTER_NOTIFICATION_THRESHOLD"`
	NotificationEnabled   bool    `mapstructure:"COUNTER_NOTIFICATION_ENABLED"`
	ChartProvider         string  `mapstructure:"COUNTER_CHART_PROVIDER"`
	AnalysisPeriod        string  `mapstructure:"COUNTER_ANALYSIS_PERIOD"`
}

// Config - основная структура конфигурации
type Config struct {
	Environment string

	// Выбор биржи
	Exchange     string `mapstructure:"EXCHANGE"`
	ExchangeType string `mapstructure:"EXCHANGE_TYPE"`

	// API ключи
	ApiKey    string `mapstructure:"API_KEY"`
	ApiSecret string `mapstructure:"API_SECRET"`
	BaseURL   string `mapstructure:"BASE_URL"`

	// Bybit специфичные
	BybitApiKey     string `mapstructure:"BYBIT_API_KEY"`
	BybitSecretKey  string `mapstructure:"BYBIT_SECRET_KEY"`
	BybitApiUrl     string `mapstructure:"BYBIT_API_URL"`
	FuturesCategory string `mapstructure:"FUTURES_CATEGORY"`

	// Binance специфичные
	BinanceApiKey    string `mapstructure:"BINANCE_API_KEY"`
	BinanceApiSecret string `mapstructure:"BINANCE_API_SECRET"`

	// Символы и фильтрация
	SymbolFilter        string  `mapstructure:"SYMBOL_FILTER"`
	ExcludeSymbols      string  `mapstructure:"EXCLUDE_SYMBOLS"`
	MaxSymbolsToMonitor int     `mapstructure:"MAX_SYMBOLS_TO_MONITOR"`
	MinVolumeFilter     float64 `mapstructure:"MIN_VOLUME_FILTER"`

	// Движок анализа
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

	// Анализаторы
	Analyzers struct {
		GrowthAnalyzer struct {
			Enabled             bool    `mapstructure:"GROWTH_ANALYZER_ENABLED"`
			MinConfidence       float64 `mapstructure:"GROWTH_ANALYZER_MIN_CONFIDENCE"`
			MinGrowth           float64 `mapstructure:"GROWTH_ANALYZER_MIN_GROWTH"`
			ContinuityThreshold float64 `mapstructure:"GROWTH_ANALYZER_CONTINUITY_THRESHOLD"`
		} `mapstructure:",squash"`
		FallAnalyzer struct {
			Enabled             bool    `mapstructure:"FALL_ANALYZER_ENABLED"`
			MinConfidence       float64 `mapstructure:"FALL_ANALYZER_MIN_CONFIDENCE"`
			MinFall             float64 `mapstructure:"FALL_ANALYZER_MIN_FALL"`
			ContinuityThreshold float64 `mapstructure:"FALL_ANALYZER_CONTINUITY_THRESHOLD"`
		} `mapstructure:",squash"`
		ContinuousAnalyzer struct {
			Enabled             bool `mapstructure:"CONTINUOUS_ANALYZER_ENABLED"`
			MinContinuousPoints int  `mapstructure:"CONTINUOUS_ANALYZER_MIN_POINTS"`
		} `mapstructure:",squash"`
	} `mapstructure:",squash"`

	// Шина событий
	EventBus struct {
		BufferSize    int  `mapstructure:"EVENT_BUS_BUFFER_SIZE"`
		WorkerCount   int  `mapstructure:"EVENT_BUS_WORKER_COUNT"`
		EnableMetrics bool `mapstructure:"EVENT_BUS_ENABLE_METRICS"`
		EnableLogging bool `mapstructure:"EVENT_BUS_ENABLE_LOGGING"`
	} `mapstructure:",squash"`

	// Фильтры сигналов
	SignalFilters struct {
		Enabled          bool     `mapstructure:"SIGNAL_FILTERS_ENABLED"`
		MinConfidence    float64  `mapstructure:"MIN_CONFIDENCE"`
		MaxSignalsPerMin int      `mapstructure:"MAX_SIGNALS_PER_MIN"`
		IncludePatterns  []string `mapstructure:"SIGNAL_INCLUDE_PATTERNS"`
		ExcludePatterns  []string `mapstructure:"SIGNAL_EXCLUDE_PATTERNS"`
	} `mapstructure:",squash"`

	// Настройки отображения
	Display struct {
		Mode               string `mapstructure:"DISPLAY_MODE"`
		MaxSignalsPerBatch int    `mapstructure:"MAX_SIGNALS_PER_BATCH"`
		MinConfidence      int    `mapstructure:"MIN_CONFIDENCE_DISPLAY"`
		DisplayGrowth      bool   `mapstructure:"DISPLAY_GROWTH"`
		DisplayFall        bool   `mapstructure:"DISPLAY_FALL"`
		DisplayPeriods     []int  `mapstructure:"DISPLAY_PERIODS"`
		UseColors          bool   `mapstructure:"USE_COLORS"`
	} `mapstructure:",squash"`

	// Telegram
	TelegramEnabled         bool    `mapstructure:"TELEGRAM_ENABLED"`
	TelegramBotToken        string  `mapstructure:"TG_API_KEY"`
	TelegramChatID          string  `mapstructure:"TG_CHAT_ID"`
	TelegramNotifyGrowth    bool    `mapstructure:"TELEGRAM_NOTIFY_GROWTH"`
	TelegramNotifyFall      bool    `mapstructure:"TELEGRAM_NOTIFY_FALL"`
	TelegramGrowthThreshold float64 `mapstructure:"TELEGRAM_GROWTH_THRESHOLD"`
	TelegramFallThreshold   float64 `mapstructure:"TELEGRAM_FALL_THRESHOLD"`
	MessageFormat           string  `mapstructure:"MESSAGE_FORMAT"`
	Include24hStats         bool    `mapstructure:"INCLUDE_24H_STATS"`

	EnableSubscriptions bool   `mapstructure:"ENABLE_SUBSCRIPTIONS"`
	StripeSecretKey     string `mapstructure:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret string `mapstructure:"STRIPE_WEBHOOK_SECRET"`
	DefaultTrialDays    int    `mapstructure:"DEFAULT_TRIAL_DAYS"`
	EnableAutoRenewal   bool   `mapstructure:"ENABLE_AUTO_RENEWAL"`

	// НОВЫЕ поля для мониторинга
	MonitoringChatID       string `mapstructure:"MONITORING_CHAT_ID"`
	MonitoringEnabled      bool   `mapstructure:"MONITORING_ENABLED"`
	MonitoringNotifyGrowth bool   `mapstructure:"MONITORING_NOTIFY_GROWTH"`
	MonitoringNotifyFall   bool   `mapstructure:"MONITORING_NOTIFY_FALL"`
	MonitoringTestMode     bool   `mapstructure:"MONITORING_TEST_MODE"`

	// Производительность и логирование
	LogLevel     string `mapstructure:"LOG_LEVEL"`
	LogFile      string `mapstructure:"LOG_FILE"`
	HTTPPort     int    `mapstructure:"HTTP_PORT"`
	HTTPEnabled  bool   `mapstructure:"HTTP_ENABLED"`
	DebugMode    bool   `mapstructure:"DEBUG_MODE,omitempty"`
	LogToConsole bool   `mapstructure:"LOG_TO_CONSOLE,omitempty"`
	LogToFile    bool   `mapstructure:"LOG_TO_FILE,omitempty"`

	// Устаревшие настройки (для обратной совместимости)
	UpdateInterval  int     `mapstructure:"UPDATE_INTERVAL"`
	CheckContinuity bool    `mapstructure:"CHECK_CONTINUITY"`
	MinDataPoints   int     `mapstructure:"MIN_DATA_POINTS"`
	GrowthThreshold float64 `mapstructure:"GROWTH_THRESHOLD"`
	FallThreshold   float64 `mapstructure:"FALL_THRESHOLD"`
	GrowthPeriods   []int   `mapstructure:"GROWTH_PERIODS"`

	// Конфигурация Rate Limiting
	RateLimitDelay        time.Duration `mapstructure:"RATE_LIMIT_DELAY,omitempty"`
	MaxConcurrentRequests int           `mapstructure:"MAX_CONCURRENT_REQUESTS,omitempty"`

	// CounterAnalyzer - анализатор счетчика
	CounterAnalyzer CounterAnalyzerConfig `mapstructure:",squash"`
}

// LoadConfig загружает конфигурацию из .env файла
func LoadConfig(path string) (*Config, error) {
	if err := godotenv.Load(path); err != nil {
		// Пробуем загрузить из переменных окружения, если файла нет
		fmt.Printf("⚠️  Config file not found, using environment variables\n")
	}

	cfg := &Config{}

	// Выбор биржи
	cfg.Exchange = getEnv("EXCHANGE", "bybit")
	cfg.ExchangeType = getEnv("EXCHANGE_TYPE", "futures")

	// API ключи (поддерживаем старые и новые форматы)
	if cfg.Exchange == "bybit" {
		// Сначала пробуем новые названия переменных
		cfg.ApiKey = getEnv("API_KEY", "")
		cfg.ApiSecret = getEnv("API_SECRET", "")

		// Если не нашли в новых, пробуем старые
		if cfg.ApiKey == "" {
			cfg.ApiKey = getEnv("BYBIT_API_KEY", "")
		}
		if cfg.ApiSecret == "" {
			cfg.ApiSecret = getEnv("BYBIT_SECRET_KEY", "")
		}

		cfg.BaseURL = getEnv("BYBIT_API_URL", "https://api.bybit.com")
		cfg.FuturesCategory = getEnv("FUTURES_CATEGORY", "linear")
	} else if cfg.Exchange == "binance" {
		cfg.ApiKey = getEnv("BINANCE_API_KEY", "")
		cfg.ApiSecret = getEnv("BINANCE_API_SECRET", "")
		cfg.BaseURL = "https://api.binance.com"
	}

	// Сохраняем для обратной совместимости
	cfg.BybitApiKey = cfg.ApiKey
	cfg.BybitSecretKey = cfg.ApiSecret
	cfg.BybitApiUrl = cfg.BaseURL
	cfg.BinanceApiKey = cfg.ApiKey
	cfg.BinanceApiSecret = cfg.ApiSecret

	// Символы и фильтрация
	cfg.SymbolFilter = getEnv("SYMBOL_FILTER", "BTC,ETH,USDT")
	cfg.ExcludeSymbols = getEnv("EXCLUDE_SYMBOLS", "")
	cfg.MaxSymbolsToMonitor = getEnvInt("MAX_SYMBOLS_TO_MONITOR", 50)
	cfg.MinVolumeFilter = getEnvFloat("MIN_VOLUME_FILTER", 100000)

	// Движок анализа
	cfg.AnalysisEngine.UpdateInterval = getEnvInt("ANALYSIS_UPDATE_INTERVAL", 10)
	cfg.AnalysisEngine.AnalysisPeriods = parseIntList(getEnv("ANALYSIS_PERIODS", "5,15,30,60"))
	cfg.AnalysisEngine.MaxSymbolsPerRun = getEnvInt("ANALYSIS_MAX_SYMBOLS_PER_RUN", 50)
	cfg.AnalysisEngine.SignalThreshold = getEnvFloat("ANALYSIS_SIGNAL_THRESHOLD", 2.0)
	cfg.AnalysisEngine.RetentionPeriod = getEnvInt("ANALYSIS_RETENTION_PERIOD", 24)
	cfg.AnalysisEngine.EnableCache = getEnvBool("ANALYSIS_ENABLE_CACHE", true)
	cfg.AnalysisEngine.EnableParallel = getEnvBool("ANALYSIS_ENABLE_PARALLEL", true)
	cfg.AnalysisEngine.MaxWorkers = getEnvInt("ANALYSIS_MAX_WORKERS", 5)
	cfg.AnalysisEngine.MinDataPoints = getEnvInt("ANALYSIS_MIN_DATA_POINTS", 3)

	// Анализаторы
	cfg.Analyzers.GrowthAnalyzer.Enabled = getEnvBool("GROWTH_ANALYZER_ENABLED", true)
	cfg.Analyzers.GrowthAnalyzer.MinConfidence = getEnvFloat("GROWTH_ANALYZER_MIN_CONFIDENCE", 60.0)
	cfg.Analyzers.GrowthAnalyzer.MinGrowth = getEnvFloat("GROWTH_ANALYZER_MIN_GROWTH", 2.0)
	cfg.Analyzers.GrowthAnalyzer.ContinuityThreshold = getEnvFloat("GROWTH_ANALYZER_CONTINUITY_THRESHOLD", 0.7)

	cfg.Analyzers.FallAnalyzer.Enabled = getEnvBool("FALL_ANALYZER_ENABLED", true)
	cfg.Analyzers.FallAnalyzer.MinConfidence = getEnvFloat("FALL_ANALYZER_MIN_CONFIDENCE", 60.0)
	cfg.Analyzers.FallAnalyzer.MinFall = getEnvFloat("FALL_ANALYZER_MIN_FALL", 2.0)
	cfg.Analyzers.FallAnalyzer.ContinuityThreshold = getEnvFloat("FALL_ANALYZER_CONTINUITY_THRESHOLD", 0.7)

	cfg.Analyzers.ContinuousAnalyzer.Enabled = getEnvBool("CONTINUOUS_ANALYZER_ENABLED", true)
	cfg.Analyzers.ContinuousAnalyzer.MinContinuousPoints = getEnvInt("CONTINUOUS_ANALYZER_MIN_POINTS", 3)

	// Шина событий
	cfg.EventBus.BufferSize = getEnvInt("EVENT_BUS_BUFFER_SIZE", 1000)
	cfg.EventBus.WorkerCount = getEnvInt("EVENT_BUS_WORKER_COUNT", 10)
	cfg.EventBus.EnableMetrics = getEnvBool("EVENT_BUS_ENABLE_METRICS", true)
	cfg.EventBus.EnableLogging = getEnvBool("EVENT_BUS_ENABLE_LOGGING", true)

	// Фильтры сигналов
	cfg.SignalFilters.Enabled = getEnvBool("SIGNAL_FILTERS_ENABLED", false)
	cfg.SignalFilters.MinConfidence = getEnvFloat("MIN_CONFIDENCE", 50.0)
	cfg.SignalFilters.MaxSignalsPerMin = getEnvInt("MAX_SIGNALS_PER_MIN", 5)
	cfg.SignalFilters.IncludePatterns = parsePatterns(getEnv("SIGNAL_INCLUDE_PATTERNS", ""))
	cfg.SignalFilters.ExcludePatterns = parsePatterns(getEnv("SIGNAL_EXCLUDE_PATTERNS", ""))

	// Настройки отображения
	cfg.Display.Mode = getEnv("DISPLAY_MODE", "compact")
	cfg.Display.MaxSignalsPerBatch = getEnvInt("MAX_SIGNALS_PER_BATCH", 10)
	cfg.Display.MinConfidence = getEnvInt("MIN_CONFIDENCE_DISPLAY", 30)
	cfg.Display.DisplayGrowth = getEnvBool("DISPLAY_GROWTH", true)
	cfg.Display.DisplayFall = getEnvBool("DISPLAY_FALL", true)
	cfg.Display.DisplayPeriods = parseIntList(getEnv("DISPLAY_PERIODS", "5,15,30"))
	cfg.Display.UseColors = getEnvBool("USE_COLORS", true)

	// Telegram
	cfg.TelegramEnabled = getEnvBool("TELEGRAM_ENABLED", false)
	cfg.TelegramBotToken = getEnv("TG_API_KEY", "")
	cfg.TelegramChatID = getEnv("TG_CHAT_ID", "")
	cfg.TelegramNotifyGrowth = getEnvBool("TELEGRAM_NOTIFY_GROWTH", true)
	cfg.TelegramNotifyFall = getEnvBool("TELEGRAM_NOTIFY_FALL", true)
	cfg.TelegramGrowthThreshold = getEnvFloat("TELEGRAM_GROWTH_THRESHOLD", 0.5)
	cfg.TelegramFallThreshold = getEnvFloat("TELEGRAM_FALL_THRESHOLD", 0.5)
	cfg.MessageFormat = getEnv("MESSAGE_FORMAT", "compact")
	cfg.Include24hStats = getEnvBool("INCLUDE_24H_STATS", false)
	cfg.MonitoringChatID = getEnv("MONITORING_CHAT_ID", "")
	cfg.MonitoringEnabled = getEnvBool("MONITORING_ENABLED", false)
	cfg.MonitoringNotifyGrowth = getEnvBool("MONITORING_NOTIFY_GROWTH", true)
	cfg.MonitoringNotifyFall = getEnvBool("MONITORING_NOTIFY_FALL", true)

	// Производительность и логирование
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.LogFile = getEnv("LOG_FILE", "logs/growth.log")
	cfg.HTTPPort = getEnvInt("HTTP_PORT", 8080)
	cfg.HTTPEnabled = getEnvBool("HTTP_ENABLED", false)
	cfg.DebugMode = getEnvBool("DEBUG_MODE", false)
	cfg.LogToConsole = getEnvBool("LOG_TO_CONSOLE", true)
	cfg.LogToFile = getEnvBool("LOG_TO_FILE", true)

	// Устаревшие настройки (для обратной совместимости)
	cfg.UpdateInterval = getEnvInt("UPDATE_INTERVAL", 5)
	cfg.CheckContinuity = getEnvBool("CHECK_CONTINUITY", false)
	cfg.MinDataPoints = getEnvInt("MIN_DATA_POINTS", 2)
	cfg.GrowthThreshold = getEnvFloat("GROWTH_THRESHOLD", 0.05)
	cfg.FallThreshold = getEnvFloat("FALL_THRESHOLD", 0.05)
	cfg.GrowthPeriods = parseIntList(getEnv("GROWTH_PERIODS", "5,15,30"))

	// Rate limiting настройки
	cfg.RateLimitDelay = getEnvDuration("RATE_LIMIT_DELAY", 100*time.Millisecond)
	cfg.MaxConcurrentRequests = getEnvInt("MAX_CONCURRENT_REQUESTS", 10)

	// Counter Analyzer настройки
	cfg.CounterAnalyzer.Enabled = getEnvBool("COUNTER_ANALYZER_ENABLED", true)
	cfg.CounterAnalyzer.BasePeriodMinutes = getEnvInt("COUNTER_BASE_PERIOD_MINUTES", 1)
	cfg.CounterAnalyzer.DefaultPeriod = getEnv("COUNTER_DEFAULT_PERIOD", "15m")
	cfg.CounterAnalyzer.MaxSignals5Min = getEnvInt("COUNTER_MAX_SIGNALS_5MIN", 5)
	cfg.CounterAnalyzer.MaxSignals15Min = getEnvInt("COUNTER_MAX_SIGNALS_15MIN", 8)
	cfg.CounterAnalyzer.MaxSignals30Min = getEnvInt("COUNTER_MAX_SIGNALS_30MIN", 10)
	cfg.CounterAnalyzer.MaxSignals1Hour = getEnvInt("COUNTER_MAX_SIGNALS_1HOUR", 12)
	cfg.CounterAnalyzer.MaxSignals4Hours = getEnvInt("COUNTER_MAX_SIGNALS_4HOURS", 15)
	cfg.CounterAnalyzer.MaxSignals1Day = getEnvInt("COUNTER_MAX_SIGNALS_1DAY", 20)
	cfg.CounterAnalyzer.GrowthThreshold = getEnvFloat("COUNTER_GROWTH_THRESHOLD", 0.1)
	cfg.CounterAnalyzer.FallThreshold = getEnvFloat("COUNTER_FALL_THRESHOLD", 0.1)
	cfg.CounterAnalyzer.TrackGrowth = getEnvBool("COUNTER_TRACK_GROWTH", true)
	cfg.CounterAnalyzer.TrackFall = getEnvBool("COUNTER_TRACK_FALL", true)
	cfg.CounterAnalyzer.NotifyOnSignal = getEnvBool("COUNTER_NOTIFY_ON_SIGNAL", true)
	cfg.CounterAnalyzer.NotificationThreshold = getEnvInt("COUNTER_NOTIFICATION_THRESHOLD", 1)
	cfg.CounterAnalyzer.ChartProvider = getEnv("COUNTER_CHART_PROVIDER", "coinglass")
	cfg.CounterAnalyzer.NotificationEnabled = getEnvBool("COUNTER_NOTIFICATION_ENABLED", true)
	cfg.CounterAnalyzer.AnalysisPeriod = getEnv("COUNTER_ANALYSIS_PERIOD", "15m")

	// Проверка обязательных параметров
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// validate проверяет обязательные параметры конфигурации
func (c *Config) validate() error {
	var errors []string

	// Проверка API ключей
	if c.Exchange == "bybit" {
		if c.ApiKey == "" {
			errors = append(errors, "BYBIT_API_KEY is required")
		}
		if c.ApiSecret == "" {
			errors = append(errors, "BYBIT_SECRET_KEY is required")
		}
	} else if c.Exchange == "binance" {
		if c.ApiKey == "" {
			errors = append(errors, "BINANCE_API_KEY is required")
		}
		if c.ApiSecret == "" {
			errors = append(errors, "BINANCE_API_SECRET is required")
		}
	}

	// Проверка Telegram если включен
	if c.TelegramEnabled {
		if c.TelegramBotToken == "" {
			errors = append(errors, "TG_API_KEY is required when Telegram is enabled")
		}
		if c.TelegramChatID == "" {
			errors = append(errors, "TG_CHAT_ID is required when Telegram is enabled")
		}
	}

	// Проверка Counter Analyzer
	if c.CounterAnalyzer.Enabled {
		if c.CounterAnalyzer.BasePeriodMinutes <= 0 {
			errors = append(errors, "COUNTER_BASE_PERIOD_MINUTES must be positive")
		}
		if !isValidPeriod(c.CounterAnalyzer.DefaultPeriod) {
			errors = append(errors, "COUNTER_DEFAULT_PERIOD must be one of: 5m, 15m, 30m, 1h, 4h, 1d")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

// isValidPeriod проверяет валидность периода
func isValidPeriod(period string) bool {
	validPeriods := map[string]bool{
		"5m":  true,
		"15m": true,
		"30m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
	}
	return validPeriods[period]
}

// GetCounterConfig возвращает конфигурацию для анализатора счетчика
func (c *Config) GetCounterConfig() map[string]interface{} {
	return map[string]interface{}{
		"base_period_minutes":    c.CounterAnalyzer.BasePeriodMinutes,
		"analysis_period":        c.CounterAnalyzer.DefaultPeriod,
		"growth_threshold":       c.CounterAnalyzer.GrowthThreshold,
		"fall_threshold":         c.CounterAnalyzer.FallThreshold,
		"track_growth":           c.CounterAnalyzer.TrackGrowth,
		"track_fall":             c.CounterAnalyzer.TrackFall,
		"notify_on_signal":       c.CounterAnalyzer.NotifyOnSignal,
		"notification_threshold": c.CounterAnalyzer.NotificationThreshold,
		"chart_provider":         c.CounterAnalyzer.ChartProvider,
		"max_signals_5m":         c.CounterAnalyzer.MaxSignals5Min,
		"max_signals_15m":        c.CounterAnalyzer.MaxSignals15Min,
		"max_signals_30m":        c.CounterAnalyzer.MaxSignals30Min,
		"max_signals_1h":         c.CounterAnalyzer.MaxSignals1Hour,
		"max_signals_4h":         c.CounterAnalyzer.MaxSignals4Hours,
		"max_signals_1d":         c.CounterAnalyzer.MaxSignals1Day,
	}
}

// IsCounterAnalyzerEnabled проверяет, включен ли анализатор счетчика
func (c *Config) IsCounterAnalyzerEnabled() bool {
	return c.CounterAnalyzer.Enabled
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

// Вспомогательные функции
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

// PrintSummary выводит краткую информацию о конфигурации (без чувствительных данных)
func (c *Config) PrintSummary() {
	log.Printf("📋 Конфигурация приложения:")
	log.Printf("   Окружение: %s", c.Exchange)
	log.Printf("   Уровень логирования: %s", c.LogLevel)
	log.Printf("   Telegram включен: %v", c.TelegramEnabled)
	if c.TelegramEnabled {
		token := c.TelegramBotToken
		if len(token) > 10 {
			token = token[:10] + "..." + token[len(token)-10:]
		}
		log.Printf("   Telegram Token: %s", token)
		log.Printf("   Telegram Chat ID: %s", c.TelegramChatID)
	}
	log.Printf("   Counter Analyzer включен: %v", c.CounterAnalyzer.Enabled)
	log.Printf("   HTTP сервер включен: %v (порт: %d)", c.HTTPEnabled, c.HTTPPort)
	log.Printf("   Макс. символов для мониторинга: %d", c.MaxSymbolsToMonitor)
	log.Printf("   Интервал обновления: %d секунд", c.AnalysisEngine.UpdateInterval)

	// Выводим информацию об анализаторах
	log.Printf("   Анализаторы:")
	log.Printf("     - Growth Analyzer: %v", c.Analyzers.GrowthAnalyzer.Enabled)
	log.Printf("     - Fall Analyzer: %v", c.Analyzers.FallAnalyzer.Enabled)
	log.Printf("     - Continuous Analyzer: %v", c.Analyzers.ContinuousAnalyzer.Enabled)

	// Выводим информацию о Counter Analyzer
	if c.CounterAnalyzer.Enabled {
		log.Printf("   Counter Analyzer настройки:")
		log.Printf("     - Базовый период: %d минут", c.CounterAnalyzer.BasePeriodMinutes)
		log.Printf("     - Период анализа: %s", c.CounterAnalyzer.AnalysisPeriod)
		log.Printf("     - Порог роста: %.2f%%", c.CounterAnalyzer.GrowthThreshold)
		log.Printf("     - Порог падения: %.2f%%", c.CounterAnalyzer.FallThreshold)
		log.Printf("     - Уведомления: %v", c.CounterAnalyzer.NotificationEnabled)
	}
}
