// internal/types/api/common.go
package api

import (
	"crypto_exchange_screener_bot/internal/types/common"
	"time"
)

// Candle - свеча данных
type Candle struct {
	Open      float64          `json:"open"`
	High      float64          `json:"high"`
	Low       float64          `json:"low"`
	Close     float64          `json:"close"`
	Volume    float64          `json:"volume"`
	Timestamp time.Time        `json:"timestamp"`
	Symbol    common.Symbol    `json:"symbol"`
	Timeframe common.Timeframe `json:"timeframe"`
}

// MarketData - рыночные данные
type MarketData struct {
	Symbol    common.Symbol `json:"symbol"`
	Price     float64       `json:"price"`
	Volume24h float64       `json:"volume_24h"`
	Timestamp time.Time     `json:"timestamp"`
}

// RequestOptions - опции запроса
type RequestOptions struct {
	Limit     int       `json:"limit,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
}
