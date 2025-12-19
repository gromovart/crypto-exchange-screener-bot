package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Выбор биржи
	Exchange     string `json:"exchange"`
	ExchangeType string `json:"exchange_type"`

	// API ключи
	ApiKey    string `json:"api_key"`
	ApiSecret string `json:"api_secret"`
	BaseURL   string `json:"base_url"`

	// Bybit специфичные
	FuturesCategory string `json:"futures_category"`

	// Символы и фильтрация
	SymbolFilter        string  `json:"symbol_filter"`
	ExcludeSymbols      string  `json:"exclude_symbols"`
	MaxSymbolsToMonitor int     `json:"max_symbols_to_monitor"`
	MinVolumeFilter     float64 `json:"min_volume_filter"`

	// Движок анализа
	AnalysisEngine struct {
		UpdateInterval   int     `json:"update_interval"`
		AnalysisPeriods  []int   `json:"analysis_periods"`
		MaxSymbolsPerRun int     `json:"max_symbols_per_run"`
		SignalThreshold  float64 `json:"signal_threshold"`
		RetentionPeriod  int     `json:"retention_period"`
		EnableCache      bool    `json:"enable_cache"`
		EnableParallel   bool    `json:"enable_parallel"`
		MaxWorkers       int     `json:"max_workers"`
	} `json:"analysis_engine"`

	// Анализаторы
	Analyzers struct {
		GrowthAnalyzer struct {
			Enabled             bool    `json:"enabled"`
			MinConfidence       float64 `json:"min_confidence"`
			MinGrowth           float64 `json:"min_growth"`
			ContinuityThreshold float64 `json:"continuity_threshold"`
		} `json:"growth_analyzer"`
		FallAnalyzer struct {
			Enabled             bool    `json:"enabled"`
			MinConfidence       float64 `json:"min_confidence"`
			MinFall             float64 `json:"min_fall"`
			ContinuityThreshold float64 `json:"continuity_threshold"`
		} `json:"fall_analyzer"`
		ContinuousAnalyzer struct {
			Enabled             bool `json:"enabled"`
			MinContinuousPoints int  `json:"min_continuous_points"`
		} `json:"continuous_analyzer"`
	} `json:"analyzers"`

	// Шина событий
	EventBus struct {
		BufferSize    int  `json:"buffer_size"`
		WorkerCount   int  `json:"worker_count"`
		EnableMetrics bool `json:"enable_metrics"`
		EnableLogging bool `json:"enable_logging"`
	} `json:"event_bus"`

	// Фильтры сигналов
	SignalFilters struct {
		Enabled          bool     `json:"enabled"`
		MinConfidence    float64  `json:"min_confidence"`
		MaxSignalsPerMin int      `json:"max_signals_per_min"`
		IncludePatterns  []string `json:"include_patterns"`
		ExcludePatterns  []string `json:"exclude_patterns"`
	} `json:"signal_filters"`

	// Настройки отображения
	Display struct {
		Mode               string `json:"mode"`
		MaxSignalsPerBatch int    `json:"max_signals_per_batch"`
		MinConfidence      int    `json:"min_confidence"`
		DisplayGrowth      bool   `json:"display_growth"`
		DisplayFall        bool   `json:"display_fall"`
		DisplayPeriods     []int  `json:"display_periods"`
		UseColors          bool   `json:"use_colors"`
	} `json:"display"`

	// Telegram
	TelegramEnabled         bool    `json:"telegram_enabled"`
	TelegramBotToken        string  `json:"telegram_bot_token"`
	TelegramChatID          string  `json:"telegram_chat_id"`
	TelegramNotifyGrowth    bool    `json:"telegram_notify_growth"`
	TelegramNotifyFall      bool    `json:"telegram_notify_fall"`
	TelegramGrowthThreshold float64 `json:"telegram_growth_threshold"`
	TelegramFallThreshold   float64 `json:"telegram_fall_threshold"`
	MessageFormat           string  `json:"message_format"`
	Include24hStats         bool    `json:"include_24h_stats"`

	// Производительность и логирование
	LogLevel    string `json:"log_level"`
	LogFile     string `json:"log_file"`
	HTTPPort    int    `json:"http_port"`
	HTTPEnabled bool   `json:"http_enabled"`

	// Устаревшие настройки (для обратной совместимости)
	UpdateInterval  int     `json:"update_interval"`
	CheckContinuity bool    `json:"check_continuity"`
	MinDataPoints   int     `json:"min_data_points"`
	GrowthThreshold float64 `json:"growth_threshold"`
	FallThreshold   float64 `json:"fall_threshold"`
	GrowthPeriods   []int   `json:"growth_periods"`
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

	// API ключи
	if cfg.Exchange == "bybit" {
		cfg.ApiKey = getEnv("BYBIT_API_KEY", "")
		cfg.ApiSecret = getEnv("BYBIT_SECRET_KEY", "")
		cfg.BaseURL = getEnv("BYBIT_API_URL", "https://api.bybit.com")
		cfg.FuturesCategory = getEnv("FUTURES_CATEGORY", "linear")
	} else if cfg.Exchange == "binance" {
		cfg.ApiKey = getEnv("BINANCE_API_KEY", "")
		cfg.ApiSecret = getEnv("BINANCE_API_SECRET", "")
		cfg.BaseURL = "https://api.binance.com"
	}

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

	// Производительность и логирование
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.LogFile = getEnv("LOG_FILE", "logs/growth.log")
	cfg.HTTPPort = getEnvInt("HTTP_PORT", 8080)
	cfg.HTTPEnabled = getEnvBool("HTTP_ENABLED", false)

	// Устаревшие настройки (для обратной совместимости)
	cfg.UpdateInterval = getEnvInt("UPDATE_INTERVAL", 5)
	cfg.CheckContinuity = getEnvBool("CHECK_CONTINUITY", false)
	cfg.MinDataPoints = getEnvInt("MIN_DATA_POINTS", 2)
	cfg.GrowthThreshold = getEnvFloat("GROWTH_THRESHOLD", 0.05)
	cfg.FallThreshold = getEnvFloat("FALL_THRESHOLD", 0.05)
	cfg.GrowthPeriods = parseIntList(getEnv("GROWTH_PERIODS", "5,15,30"))

	return cfg, nil
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
