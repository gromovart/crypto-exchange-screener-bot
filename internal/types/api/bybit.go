// internal/types/api/bybit.go
package api

import (
	"crypto_exchange_screener_bot/internal/types/common"
	"time"
)

// BybitCandle - свеча от Bybit
type BybitCandle struct {
	Open      float64          `json:"open"`
	High      float64          `json:"high"`
	Low       float64          `json:"low"`
	Close     float64          `json:"close"`
	Volume    float64          `json:"volume"`
	Timestamp time.Time        `json:"timestamp"`
	Symbol    common.Symbol    `json:"symbol"`
	Interval  common.Timeframe `json:"interval"`
}

// BybitTicker - тикер от Bybit
type BybitTicker struct {
	Symbol                common.Symbol `json:"symbol"`
	LastPrice             string        `json:"last_price"`
	Price24hPercentChange string        `json:"price_24h_percent_change"`
	Volume24h             string        `json:"volume_24h"`
	Turnover24h           string        `json:"turnover_24h"`
}

// BybitResponse - ответ от Bybit API
type BybitResponse struct {
	RetCode int         `json:"ret_code"`
	RetMsg  string      `json:"ret_msg"`
	Result  interface{} `json:"result"`
	Time    int64       `json:"time"`
}

const (
	CategorySpot    = "spot"
	CategoryLinear  = "linear"  // USDT-M фьючерсы
	CategoryInverse = "inverse" // COIN-M фьючерсы
)

// APIResponse - базовый ответ API Bybit
type APIResponse struct {
	RetCode int         `json:"retCode"`
	RetMsg  string      `json:"retMsg"`
	Result  interface{} `json:"result"`
	Time    int64       `json:"time"`
}

// KlineResponse - ответ для свечных данных
type KlineResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string     `json:"category"`
		Symbol   string     `json:"symbol"`
		List     [][]string `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// OrderBookResponse - ответ для стакана заявок
type OrderBookResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		S []struct {
			Price string `json:"price"`
			Size  string `json:"size"`
		} `json:"s"`
		B []struct {
			Price string `json:"price"`
			Size  string `json:"size"`
		} `json:"b"`
		Ts int64 `json:"ts"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// AccountBalance - баланс аккаунта
type AccountBalance struct {
	Coin             string `json:"coin"`
	Equity           string `json:"equity"`
	WalletBalance    string `json:"walletBalance"`
	PositionMM       string `json:"positionMM"`
	AvailableBalance string `json:"availableBalance"`
}

// OrderResponse - ответ на создание ордера
type OrderResponse struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// InstrumentInfo - информация об инструменте фьючерса
type InstrumentInfo struct {
	Symbol          string `json:"symbol"`
	ContractType    string `json:"contractType"`
	Status          string `json:"status"`
	BaseCoin        string `json:"baseCoin"`
	QuoteCoin       string `json:"quoteCoin"`
	LaunchTime      string `json:"launchTime"`
	DeliveryTime    string `json:"deliveryTime"`
	DeliveryFeeRate string `json:"deliveryFeeRate"`
	PriceScale      string `json:"priceScale"`
	LeverageFilter  struct {
		MinLeverage  string `json:"minLeverage"`
		MaxLeverage  string `json:"maxLeverage"`
		LeverageStep string `json:"leverageStep"`
	} `json:"leverageFilter"`
	PriceFilter struct {
		MinPrice string `json:"minPrice"`
		MaxPrice string `json:"maxPrice"`
		TickSize string `json:"tickSize"`
	} `json:"priceFilter"`
	LotSizeFilter struct {
		MaxOrderQty         string `json:"maxOrderQty"`
		MinOrderQty         string `json:"minOrderQty"`
		QtyStep             string `json:"qtyStep"`
		PostOnlyMaxOrderQty string `json:"postOnlyMaxOrderQty"`
	} `json:"lotSizeFilter"`
}

// TickerResponse сохраняем только для внутреннего использования
// Используем алиас или переименуем
type TickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string          `json:"category"`
		List     []common.Ticker `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// Или создаем внутреннюю структуру для парсинга
type BybitTickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string          `json:"category"`
		List     []common.Ticker `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}
