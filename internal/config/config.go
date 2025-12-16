// config.go
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

// Config - структура конфигурации приложения
type Config struct {
	// API Keys
	ApiKey           string
	ApiSecret        string
	TestnetApiKey    string
	TestnetApiSecret string
	BaseURL          string
	TestnetBaseURL   string

	// Trading Settings
	UseTestnet     bool
	TradingEnabled bool
	TradingSymbol  string

	// Risk Management
	RiskPercent     float64
	StopLoss        float64
	TakeProfit      float64
	MaxPositionSize float64

	// Monitoring
	UpdateInterval   int
	PriceHistorySize int
	TrackedIntervals []int

	// Alerts
	AlertThreshold float64
	AlertEnabled   bool
	AlertEmail     string

	// Logging
	LogLevel      string
	LogFile       string
	LogMaxSize    int
	LogMaxBackups int

	// HTTP Server
	HttpPort    string
	HttpEnabled bool
	CorsOrigins []string

	// Performance
	MaxConcurrentRequests int
	RequestTimeout        time.Duration
	RateLimitDelay        time.Duration

	InitialDataFetch bool `json:"initial_data_fetch"`
	DataFetchLimit   int  `json:"data_fetch_limit"`

	FuturesCategory string  `json:"futures_category"` // "linear" или "inverse"
	FuturesLeverage float64 `json:"futures_leverage"`

	// Growth Monitoring
	GrowthPeriods   []int   `json:"growth_periods"`   // Периоды роста в минутах
	GrowthThreshold float64 `json:"growth_threshold"` // Порог роста в процентах
	FallThreshold   float64 `json:"fall_threshold"`   // Порог падения в процентах
	CheckContinuity bool    `json:"check_continuity"` // Проверять непрерывность
	MinDataPoints   int     `json:"min_data_points"`  // Минимальное количество точек дан

	// Фильтры символов
	SymbolFilter        string  `json:"symbol_filter"`          // Фильтр символов (например: BTC,ETH,BNB или BTCUSDT,ETHUSDT)
	ExcludeSymbols      string  `json:"exclude_symbols"`        // Символы для исключения
	MaxSymbolsToMonitor int     `json:"max_symbols_to_monitor"` // Максимальное количество символов для мониторинга
	MinVolumeFilter     float64 `json:"min_volume_filter"`      // Минимальный объем для фильтрации

	// Фильтры сигналов
	SignalFilters struct {
		Enabled          bool     `json:"enabled"`             // Включить фильтрацию сигналов
		IncludePatterns  []string `json:"include_patterns"`    // Паттерны для включения (например: BTC*, ETH*)
		ExcludePatterns  []string `json:"exclude_patterns"`    // Паттерны для исключения
		MinConfidence    float64  `json:"min_confidence"`      // Минимальная уверенность сигнала
		MaxSignalsPerMin int      `json:"max_signals_per_min"` // Максимум сигналов в минуту
	} `json:"signal_filters"`
}

// LoadConfig загружает конфигурацию из .env файла
func LoadConfig(envPath string) (*Config, error) {
	// Загружаем .env файл
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: Could not load %s file: %v", envPath, err)
	}

	// Определяем, какую сеть использовать
	useTestnet := getEnvBool("USE_TESTNET", true)

	// Выбираем соответствующие API ключи
	var apiKey, apiSecret, baseURL string
	if useTestnet {
		apiKey = getEnvString("BYBIT_TESTNET_API_KEY", "")
		apiSecret = getEnvString("BYBIT_TESTNET_SECRET_KEY", "")
		baseURL = getEnvString("BYBIT_API_TEST_URL", "https://api-testnet.bybit.com")
	} else {
		apiKey = getEnvString("BYBIT_API_KEY", "")
		apiSecret = getEnvString("BYBIT_SECRET_KEY", "")
		baseURL = getEnvString("BYBIT_API_URL", "https://api.bybit.com")
	}

	// Проверяем обязательные поля
	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("API keys are required. Please set BYBIT_API_KEY and BYBIT_SECRET_KEY in .env file")
	}

	// Парсим интервалы
	intervalsStr := getEnvString("TRACKED_INTERVALS", "1,5,10,15,30,60,120,240,480,720,1440")
	intervals := parseIntervals(intervalsStr)

	config := &Config{
		// API
		ApiKey:           apiKey,
		ApiSecret:        apiSecret,
		TestnetApiKey:    getEnvString("BYBIT_TESTNET_API_KEY", ""),
		TestnetApiSecret: getEnvString("BYBIT_TESTNET_SECRET_KEY", ""),
		BaseURL:          baseURL,
		TestnetBaseURL:   getEnvString("BYBIT_API_TEST_URL", "https://api-testnet.bybit.com"),

		// Trading
		UseTestnet:     useTestnet,
		TradingEnabled: getEnvBool("TRADING_ENABLED", false),
		TradingSymbol:  getEnvString("TRADING_SYMBOL", "BTCUSDT"),

		// Risk
		RiskPercent:     getEnvFloat("RISK_PERCENT", 2.0),
		StopLoss:        getEnvFloat("STOP_LOSS", 5.0),
		TakeProfit:      getEnvFloat("TAKE_PROFIT", 10.0),
		MaxPositionSize: getEnvFloat("MAX_POSITION_SIZE", 0.01),

		// Monitoring
		UpdateInterval:   getEnvInt("UPDATE_INTERVAL", 10),
		PriceHistorySize: getEnvInt("PRICE_HISTORY_SIZE", 8640),
		TrackedIntervals: intervals,

		// Alerts
		AlertThreshold: getEnvFloat("ALERT_THRESHOLD", 5.0),
		AlertEnabled:   getEnvBool("ALERT_ENABLED", true),
		AlertEmail:     getEnvString("ALERT_EMAIL", ""),

		// Logging
		LogLevel:      getEnvString("LOG_LEVEL", "info"),
		LogFile:       getEnvString("LOG_FILE", "logs/bot.log"),
		LogMaxSize:    getEnvInt("LOG_MAX_SIZE", 10),
		LogMaxBackups: getEnvInt("LOG_MAX_BACKUPS", 5),

		// HTTP Server
		HttpPort:    getEnvString("HTTP_PORT", "8080"),
		HttpEnabled: getEnvBool("HTTP_ENABLED", true),
		CorsOrigins: strings.Split(getEnvString("CORS_ORIGINS", "*"), ","),

		// Performance
		MaxConcurrentRequests: getEnvInt("MAX_CONCURRENT_REQUESTS", 10),
		RequestTimeout:        time.Duration(getEnvInt("REQUEST_TIMEOUT", 30)) * time.Second,
		RateLimitDelay:        time.Duration(getEnvInt("RATE_LIMIT_DELAY", 100)) * time.Millisecond,

		InitialDataFetch: getEnvBool("INITIAL_DATA_FETCH", true),
		DataFetchLimit:   getEnvInt("DATA_FETCH_LIMIT", 100),

		// Growth Monitoring
		GrowthPeriods:   parseGrowthPeriods(getEnvString("GROWTH_PERIODS", "5,15,30,60")),
		GrowthThreshold: getEnvFloat("GROWTH_THRESHOLD", 2.0),
		FallThreshold:   getEnvFloat("FALL_THRESHOLD", 2.0),
		CheckContinuity: getEnvBool("CHECK_CONTINUITY", true),
		MinDataPoints:   getEnvInt("MIN_DATA_POINTS", 3),

		// Фильтры символов
		SymbolFilter:        getEnvString("SYMBOL_FILTER", ""),
		ExcludeSymbols:      getEnvString("EXCLUDE_SYMBOLS", ""),
		MaxSymbolsToMonitor: getEnvInt("MAX_SYMBOLS_TO_MONITOR", 0),   // 0 = без ограничений
		MinVolumeFilter:     getEnvFloat("MIN_VOLUME_FILTER", 100000), // $100K по умолчанию

		// Фильтры сигналов
		SignalFilters: struct {
			Enabled          bool     `json:"enabled"`
			IncludePatterns  []string `json:"include_patterns"`
			ExcludePatterns  []string `json:"exclude_patterns"`
			MinConfidence    float64  `json:"min_confidence"`
			MaxSignalsPerMin int      `json:"max_signals_per_min"`
		}{
			Enabled:          getEnvBool("SIGNAL_FILTERS_ENABLED", false),
			IncludePatterns:  parsePatterns(getEnvString("SIGNAL_INCLUDE_PATTERNS", "")),
			ExcludePatterns:  parsePatterns(getEnvString("SIGNAL_EXCLUDE_PATTERNS", "")),
			MinConfidence:    getEnvFloat("MIN_CONFIDENCE", 50.0),
			MaxSignalsPerMin: getEnvInt("MAX_SIGNALS_PER_MIN", 5),
		},
	}

	return config, nil
}

// Вспомогательные функции для парсинга переменных окружения
func getEnvString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func parseIntervals(intervalStr string) []int {
	parts := strings.Split(intervalStr, ",")
	intervals := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if interval, err := strconv.Atoi(part); err == nil && interval > 0 {
			intervals = append(intervals, interval)
		}
	}

	// Если не удалось распарсить, возвращаем стандартные интервалы
	if len(intervals) == 0 {
		return []int{1, 5, 10, 15, 30, 60, 120, 240, 480, 720, 1440}
	}

	return intervals
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	// Проверяем API ключи
	if c.ApiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.ApiSecret == "" {
		return fmt.Errorf("API secret is required")
	}

	// Проверяем длину ключей (Bybit ключи обычно имеют длину 36 символов)
	if len(c.ApiKey) != 36 && len(c.ApiKey) != 32 {
		log.Printf("Warning: API key length (%d) is unusual", len(c.ApiKey))
	}

	// Проверяем допустимость значений
	if c.RiskPercent <= 0 || c.RiskPercent > 100 {
		return fmt.Errorf("risk percent must be between 0 and 100")
	}

	if c.StopLoss <= 0 {
		return fmt.Errorf("stop loss must be positive")
	}

	if c.UpdateInterval < 1 {
		return fmt.Errorf("update interval must be at least 1 second")
	}

	return nil
}

// GetCurrentAPI возвращает текущие API ключи в зависимости от сети
func (c *Config) GetCurrentAPI() (string, string, string) {
	if c.UseTestnet {
		return c.TestnetApiKey, c.TestnetApiSecret, c.TestnetBaseURL
	}
	return c.ApiKey, c.ApiSecret, c.BaseURL
}

// GetIntervalDuration возвращает time.Duration для интервала в минутах
func (c *Config) GetIntervalDuration(minutes int) time.Duration {
	return time.Duration(minutes) * time.Minute
}

// GetAllIntervalDurations возвращает все интервалы в виде time.Duration
func (c *Config) GetAllIntervalDurations() []time.Duration {
	durations := make([]time.Duration, len(c.TrackedIntervals))
	for i, interval := range c.TrackedIntervals {
		durations[i] = c.GetIntervalDuration(interval)
	}
	return durations
}

// Новая функция для парсинга периодов роста
func parseGrowthPeriods(periodsStr string) []int {
	parts := strings.Split(periodsStr, ",")
	periods := make([]int, 0, len(parts))

	supportedPeriods := map[int]bool{
		1: true, 5: true, 10: true, 15: true, 30: true,
		60: true, 120: true, 240: true, 480: true,
		720: true, 1440: true,
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if period, err := strconv.Atoi(part); err == nil {
			if supportedPeriods[period] {
				periods = append(periods, period)
			}
		}
	}

	if len(periods) == 0 {
		periods = []int{5, 15, 30, 60}
	}

	return periods
}
func parsePatterns(patternsStr string) []string {
	if patternsStr == "" {
		return []string{}
	}

	patterns := strings.Split(patternsStr, ",")
	result := make([]string, 0, len(patterns))

	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
