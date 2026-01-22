// internal/infrastructure/persistence/redis_storage/candle_storage/structs.go
package candle_storage

import (
	"context"
	"crypto-exchange-screener-bot/internal/core/domain/candle"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCandleStorage - хранилище свечей на Redis
type RedisCandleStorage struct {
	client       *redis.Client
	ctx          context.Context
	prefix       string
	candlePrefix string

	// Конфигурация
	config candle.CandleConfig
}

type CandleData struct {
	Symbol       string    `json:"symbol"`
	Period       string    `json:"period"`
	Open         float64   `json:"open"`
	High         float64   `json:"high"`
	Low          float64   `json:"low"`
	Close        float64   `json:"close"`
	Volume       float64   `json:"volume"`
	VolumeUSD    float64   `json:"volume_usd"`
	Trades       int       `json:"trades"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	IsClosedFlag bool      `json:"is_closed"`
	IsRealFlag   bool      `json:"is_real"`
}

// CandleConfigData конфигурация свечей (реализация интерфейса interfaces.CandleConfig)
type CandleConfigData struct {
	SupportedPeriods []string      `json:"supported_periods"`
	MaxHistory       int           `json:"max_history"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
}

// CandleStatsData статистика свечей (реализация интерфейса interfaces.CandleStats)
type CandleStatsData struct {
	TotalCandles  int            `json:"total_candles"`
	ActiveCandles int            `json:"active_candles"`
	SymbolsCount  int            `json:"symbols_count"`
	OldestCandle  time.Time      `json:"oldest_candle"`
	NewestCandle  time.Time      `json:"newest_candle"`
	PeriodsCount  map[string]int `json:"periods_count"`
}
