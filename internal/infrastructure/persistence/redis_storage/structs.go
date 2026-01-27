// internal/infrastructure/persistence/redis_storage/structs.go
package redis_storage

import (
	"time"
)

// SymbolMetrics содержит все метрики символа
type SymbolMetrics struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Volume24h     float64   `json:"volume_24h"`
	VolumeUSD     float64   `json:"volume_usd"`
	OpenInterest  float64   `json:"open_interest"`
	FundingRate   float64   `json:"funding_rate"`
	Change24h     float64   `json:"change_24h"`
	High24h       float64   `json:"high_24h"`
	Low24h        float64   `json:"low_24h"`
	OIChange24h   float64   `json:"oi_change_24h"`
	FundingChange float64   `json:"funding_change"`
	Timestamp     time.Time `json:"timestamp"`
}

// SymbolStats статистика по символу
type SymbolStats struct {
	Symbol         string    `json:"symbol"`
	DataPoints     int       `json:"data_points"`
	FirstTimestamp time.Time `json:"first_timestamp"`
	LastTimestamp  time.Time `json:"last_timestamp"`
	CurrentPrice   float64   `json:"current_price"`
	AvgVolume24h   float64   `json:"avg_volume_24h"`
	AvgVolumeUSD   float64   `json:"avg_volume_usd"`
	PriceChange24h float64   `json:"price_change_24h"`
	OpenInterest   float64   `json:"open_interest"`
	OIChange24h    float64   `json:"oi_change_24h"`
	FundingRate    float64   `json:"funding_rate"`
	FundingChange  float64   `json:"funding_change"`
	High24h        float64   `json:"high_24h"`
	Low24h         float64   `json:"low_24h"`
}

// SymbolVolume символ с объемом
type SymbolVolume struct {
	Symbol    string  `json:"symbol"`
	Volume    float64 `json:"volume"`
	VolumeUSD float64 `json:"volume_usd,omitempty"`
}

// StorageConfig конфигурация хранилища
type StorageConfig struct {
	MaxHistoryPerSymbol int
	MaxSymbols          int
	CleanupInterval     time.Duration
	RetentionPeriod     time.Duration
	EnableCompression   bool
	EnablePersistence   bool
	PersistencePath     string
}

// StorageStats статистика хранилища
type StorageStats struct {
	TotalSymbols        int           `json:"total_symbols"`
	TotalDataPoints     int64         `json:"total_data_points"`
	MemoryUsageBytes    int64         `json:"memory_usage_bytes"`
	OldestTimestamp     time.Time     `json:"oldest_timestamp"`
	NewestTimestamp     time.Time     `json:"newest_timestamp"`
	UpdateRatePerSecond float64       `json:"update_rate_per_second"`
	StorageType         string        `json:"storage_type"`
	MaxHistoryPerSymbol int           `json:"max_history_per_symbol"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	SymbolsWithOI       int           `json:"symbols_with_oi"`
	SymbolsWithFunding  int           `json:"symbols_with_funding"`
}

// PriceData представляет точку данных цены
type PriceData struct {
	Symbol       string                 `json:"symbol"`
	Price        float64                `json:"price"`
	Volume24h    float64                `json:"volume_24h"`
	VolumeUSD    float64                `json:"volume_usd"`
	Timestamp    time.Time              `json:"timestamp"`
	OpenInterest float64                `json:"open_interest"`
	FundingRate  float64                `json:"funding_rate"`
	Change24h    float64                `json:"change_24h"`
	High24h      float64                `json:"high_24h"`
	Low24h       float64                `json:"low_24h"`
	Category     string                 `json:"category,omitempty"`
	Basis        float64                `json:"basis,omitempty"`
	Liquidation  float64                `json:"liquidation,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type PriceDataPoint struct {
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
	Volume    float64   `json:"volume"`
}

type GrowthAnalysis struct {
	Symbol        string           `json:"symbol"`
	Period        int              `json:"period"`
	DataPoints    []PriceDataPoint `json:"data_points"`
	IsGrowing     bool             `json:"is_growing"`
	IsFalling     bool             `json:"is_falling"`
	GrowthPercent float64          `json:"growth_percent"`
	FallPercent   float64          `json:"fall_percent"`
	MinPrice      float64          `json:"min_price"`
	MaxPrice      float64          `json:"max_price"`
	Volatility    float64          `json:"volatility"`
}

// PriceSnapshot текущий снапшот цены
type PriceSnapshot struct {
	Symbol       string    `json:"symbol"`
	Price        float64   `json:"price"`
	Volume24h    float64   `json:"volume_24h"`
	VolumeUSD    float64   `json:"volume_usd"`
	Timestamp    time.Time `json:"timestamp"`
	OpenInterest float64   `json:"open_interest"`
	FundingRate  float64   `json:"funding_rate"`
	Change24h    float64   `json:"change_24h"`
	High24h      float64   `json:"high_24h"`
	Low24h       float64   `json:"low_24h"`
}

// CandleConfig - конфигурация построителя
type CandleConfig struct {
	SupportedPeriods []string      // Поддерживаемые периоды
	MaxHistory       int           // Максимальная история свечей
	CleanupInterval  time.Duration // Интервал очистки
	AutoBuild        bool          // Автоматическое построение
}

// Candle - свеча (OHLCV)
type Candle struct {
	Symbol       string
	Period       string // "5m", "15m", "30m", "1h", "4h", "1d"
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       float64 // Объем в базовой валюте
	VolumeUSD    float64 // Объем в USD
	Trades       int     // Количество сделок
	StartTime    time.Time
	EndTime      time.Time
	IsClosedFlag bool // Закрыта ли свеча
	IsRealFlag   bool // Реальные данные или построенные
}
