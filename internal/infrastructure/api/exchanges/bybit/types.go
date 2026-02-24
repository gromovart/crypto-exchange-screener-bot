// internal/infrastructure/api/exchanges/bybit/types.go
package bybit

import "time"

const (
	CategorySpot    = "spot"
	CategoryLinear  = "linear"  // USDT-M фьючерсы
	CategoryInverse = "inverse" // COIN-M фьючерсы

	OIInterval5Min  = "5min"
	OIInterval15Min = "15min"
	OIInterval30Min = "30min"
	OIInterval1Hour = "1h"
	OIInterval4Hour = "4h"
	OIInterval1Day  = "1d"

	// Ошибки API
	ErrCodeInvalidParams  = 10001
	ErrCodeRateLimit      = 10006
	ErrCodeSymbolNotFound = 30001
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
		Category string `json:"category"`
		List     []struct {
			Symbol       string `json:"symbol"`
			LastPrice    string `json:"lastPrice"`
			Price24hPcnt string `json:"price24hPcnt"`
			Volume24h    string `json:"volume24h"`
			Turnover24h  string `json:"turnover24h"`
		} `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// Или создаем внутреннюю структуру для парсинга
type bybitTickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string `json:"category"`
		List     []struct {
			Symbol       string `json:"symbol"`
			LastPrice    string `json:"lastPrice"`
			Price24hPcnt string `json:"price24hPcnt"`
			Volume24h    string `json:"volume24h"`
			Turnover24h  string `json:"turnover24h"`
		} `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

// OrderLevelV5 — уровень стакана ордеров (V5 API)
type OrderLevelV5 struct {
	Price float64
	Size  float64
}

// OrderBookV5 — стакан ордеров (V5 API)
type OrderBookV5 struct {
	Symbol string
	Bids   []OrderLevelV5
	Asks   []OrderLevelV5
}

// LiquidationData данные о ликвидации
type LiquidationData struct {
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"` // "Buy" или "Sell"
	Price         float64   `json:"price"`
	Quantity      float64   `json:"qty"`
	Time          time.Time `json:"time"`
	IsLiquidation bool      `json:"is_liquidation"`
}

// LiquidationMetrics метрики ликвидаций
type LiquidationMetrics struct {
	Symbol         string    `json:"symbol"`
	TotalVolumeUSD float64   `json:"total_volume_usd"`
	LongLiqVolume  float64   `json:"long_liq_volume"`
	ShortLiqVolume float64   `json:"short_liq_volume"`
	LongLiqCount   int       `json:"long_liq_count"`
	ShortLiqCount  int       `json:"short_liq_count"`
	UpdateTime     time.Time `json:"update_time"`
}

// LiquidationResponse ответ от API ликвидаций
type LiquidationResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			Symbol   string `json:"symbol"`
			Side     string `json:"side"`
			Price    string `json:"price"`
			Size     string `json:"size"`
			Time     string `json:"time"`
			ExecType string `json:"exec_type"` // "Liquidation" для ликвидаций
		} `json:"list"`
	} `json:"result"`
}

// RecentTradesResponse ответ последних сделок
type RecentTradesResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string `json:"category"`
		List     []struct {
			ExecId   string `json:"execId"`
			Symbol   string `json:"symbol"`
			Price    string `json:"price"`
			Size     string `json:"size"`
			Side     string `json:"side"`
			Time     string `json:"time"`
			ExecType string `json:"execType"`
		} `json:"list"`
	} `json:"result"`
}
