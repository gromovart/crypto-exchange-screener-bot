// internal/infrastructure/api/types.go
package api

import (
	"fmt"
	"strconv"
)

// ExchangeClient интерфейс для клиентов бирж
type ExchangeClient interface {
	GetTickers(category string) (*TickerResponse, error)
	Category() string
}

// Ticker - тикер
type Ticker struct {
	Symbol       string `json:"symbol"`
	LastPrice    string `json:"lastPrice"`
	Volume24h    string `json:"volume24h"`
	Price24hPcnt string `json:"price24hPcnt,omitempty"`
	Turnover24h  string `json:"turnover24h,omitempty"`
	OpenInterest string `json:"openInterest,omitempty"` // ✅ Убедитесь, что это поле есть
	FundingRate  string `json:"fundingRate,omitempty"`
	High24h      string `json:"high24h"`
	Low24h       string `json:"low24h"`
}

// TickerList - список тикеров
type TickerList struct {
	Category string   `json:"category,omitempty"` // ✅ Добавляем это поле!
	List     []Ticker `json:"list"`
}

// TickerResponse - ответ от API с тикерами
type TickerResponse struct {
	RetCode int        `json:"retCode"`
	RetMsg  string     `json:"retMsg"`
	Result  TickerList `json:"result"`
}

// GetOpenInterest возвращает Open Interest как строку
func (t *Ticker) GetOpenInterest() (string, bool) {
	if t.OpenInterest == "" {
		return "", false
	}
	return t.OpenInterest, true
}

// GetOpenInterestFloat возвращает Open Interest как float64
func (t *Ticker) GetOpenInterestFloat() (float64, error) {
	if t.OpenInterest == "" {
		return 0, fmt.Errorf("open interest not available")
	}
	return strconv.ParseFloat(t.OpenInterest, 64)
}
