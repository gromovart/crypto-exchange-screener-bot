// internal/types/common.go
package common

import (
	"sync"
	"time"
)

// Symbol - торговая пара
type Symbol string

// Exchange - биржа
type Exchange string

// Timeframe - таймфрейм
type Timeframe string

// Константы бирж
const (
	ExchangeBinance Exchange = "binance"
	ExchangeBybit   Exchange = "bybit"
)

// Константы таймфреймов
const (
	Timeframe1m  Timeframe = "1m"
	Timeframe5m  Timeframe = "5m"
	Timeframe15m Timeframe = "15m"
	Timeframe30m Timeframe = "30m"
	Timeframe1h  Timeframe = "1h"
	Timeframe4h  Timeframe = "4h"
	Timeframe1d  Timeframe = "1d"
)

// PriceData - данные о цене
type PriceData struct {
	Symbol    Symbol    `json:"symbol"`
	Exchange  Exchange  `json:"exchange"`
	Timeframe Timeframe `json:"timeframe"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Volume24h float64   `json:"volume_24h,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open,omitempty"`
	High      float64   `json:"high,omitempty"`
	Low       float64   `json:"low,omitempty"`
	Close     float64   `json:"close,omitempty"`
}

// Config - базовая конфигурация
type Config struct {
	Symbols    []Symbol    `json:"symbols"`
	Timeframes []Timeframe `json:"timeframes"`
	Exchanges  []Exchange  `json:"exchanges"`
}

// ExchangeClient интерфейс для клиентов бирж
type ExchangeClient interface {
	GetTickers(category string) (*TickerResponse, error)
	Category() string
}

// Ticker - тикер
type Ticker struct {
	Symbol       Symbol `json:"symbol"`
	LastPrice    string `json:"lastPrice"`
	Volume24h    string `json:"volume24h"`
	Price24hPcnt string `json:"price24hPcnt,omitempty"`
	Turnover24h  string `json:"turnover24h,omitempty"`
}

// TickerList - список тикеров
type TickerList struct {
	Category string   `json:"category"`
	List     []Ticker `json:"list"`
}

// TickerResponse - ответ от API с тикерами
type TickerResponse struct {
	RetCode int        `json:"retCode"`
	RetMsg  string     `json:"retMsg"`
	Result  TickerList `json:"result"`
}

// RateLimiter - ограничитель частоты запросов
type RateLimiter struct {
	Mu       sync.Mutex
	LastSent map[string]time.Time
	MinDelay time.Duration
}

// CanSend проверяет, можно ли отправить сообщение
func (rl *RateLimiter) CanSend(key string) bool {
	rl.Mu.Lock()
	defer rl.Mu.Unlock()

	now := time.Now()
	if last, exists := rl.LastSent[key]; exists {
		if now.Sub(last) < rl.MinDelay {
			return false
		}
	}
	rl.LastSent[key] = now
	return true
}

// NewRateLimiter создает новый ограничитель частоты
func NewRateLimiter(minDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		LastSent: make(map[string]time.Time),
		MinDelay: minDelay,
	}
}
