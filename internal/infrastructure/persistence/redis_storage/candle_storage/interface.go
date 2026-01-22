// internal/infrastructure/persistence/redis_storage/candle_storage/interface.go
package candle_storage

import (
	"time"
)

// Candle интерфейс для свечи
type Candle interface {
	GetSymbol() string
	GetPeriod() string
	GetOpen() float64
	GetHigh() float64
	GetLow() float64
	GetClose() float64
	GetVolume() float64
	GetVolumeUSD() float64
	GetTrades() int
	GetStartTime() time.Time
	GetEndTime() time.Time
	IsClosed() bool
	IsReal() bool
}

// CandleConfig интерфейс для конфигурации свечей
type CandleConfig interface {
	GetSupportedPeriods() []string
	GetMaxHistory() int
	GetCleanupInterval() time.Duration
}

// CandleStats интерфейс для статистики свечей
type CandleStats interface {
	GetTotalCandles() int
	GetActiveCandles() int
	GetSymbolsCount() int
	GetOldestCandle() time.Time
	GetNewestCandle() time.Time
	GetPeriodsCount() map[string]int
}

// CandleStorage интерфейс хранилища свечей
type CandleStorage interface {
	// Основные операции
	SaveActiveCandle(candle Candle) error
	GetActiveCandle(symbol, period string) (Candle, bool)
	CloseAndArchiveCandle(candle Candle) error
	GetHistory(symbol, period string, limit int) ([]Candle, error)
	GetLatestCandle(symbol, period string) (Candle, bool)
	GetCandle(symbol, period string) (Candle, error)
	CleanupOldCandles(maxAge time.Duration) int
	GetSymbols() []string
	GetPeriodsForSymbol(symbol string) []string
	GetStats() CandleStats
}

// PriceData интерфейс для данных цены (для совместимости)
type PriceData interface {
	GetSymbol() string
	GetPrice() float64
	GetVolume24h() float64
	GetVolumeUSD() float64
	GetTimestamp() time.Time
	GetOpenInterest() float64
	GetFundingRate() float64
	GetChange24h() float64
	GetHigh24h() float64
	GetLow24h() float64
}

// Adapter для конвертации структур в интерфейсы

// GenericCandle базовая реализация Candle
type GenericCandle struct {
	Symbol     string    `json:"symbol"`
	Period     string    `json:"period"`
	Open       float64   `json:"open"`
	High       float64   `json:"high"`
	Low        float64   `json:"low"`
	Close      float64   `json:"close"`
	Volume     float64   `json:"volume"`
	VolumeUSD  float64   `json:"volume_usd"`
	Trades     int       `json:"trades"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	ClosedFlag bool      `json:"is_closed"` // Переименовано
	RealFlag   bool      `json:"is_real"`   // Переименовано
}

func (c *GenericCandle) GetSymbol() string       { return c.Symbol }
func (c *GenericCandle) GetPeriod() string       { return c.Period }
func (c *GenericCandle) GetOpen() float64        { return c.Open }
func (c *GenericCandle) GetHigh() float64        { return c.High }
func (c *GenericCandle) GetLow() float64         { return c.Low }
func (c *GenericCandle) GetClose() float64       { return c.Close }
func (c *GenericCandle) GetVolume() float64      { return c.Volume }
func (c *GenericCandle) GetVolumeUSD() float64   { return c.VolumeUSD }
func (c *GenericCandle) GetTrades() int          { return c.Trades }
func (c *GenericCandle) GetStartTime() time.Time { return c.StartTime }
func (c *GenericCandle) GetEndTime() time.Time   { return c.EndTime }
func (c *GenericCandle) IsClosed() bool          { return c.ClosedFlag }
func (c *GenericCandle) IsReal() bool            { return c.RealFlag }

// GenericCandleConfig базовая реализация CandleConfig
type GenericCandleConfig struct {
	SupportedPeriods []string      `json:"supported_periods"`
	MaxHistory       int           `json:"max_history"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
}

func (c *GenericCandleConfig) GetSupportedPeriods() []string     { return c.SupportedPeriods }
func (c *GenericCandleConfig) GetMaxHistory() int                { return c.MaxHistory }
func (c *GenericCandleConfig) GetCleanupInterval() time.Duration { return c.CleanupInterval }
