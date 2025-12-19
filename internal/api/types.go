// internal/api/types.go
package api

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
}

// TickerList - список тикеров
type TickerList struct {
	List []Ticker `json:"list"`
}

// TickerResponse - ответ от API с тикерами
type TickerResponse struct {
	RetCode int        `json:"retCode"`
	RetMsg  string     `json:"retMsg"`
	Result  TickerList `json:"result"`
}
