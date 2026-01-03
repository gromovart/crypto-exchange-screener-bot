// internal/infrastructure/api/exchanges/binance/client.go
package binance

import (
	"crypto-exchange-screener-bot/internal/infrastructure/api"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// BinanceClient - клиент для API Binance
type BinanceClient struct {
	config     *config.Config
	httpClient *http.Client
	baseURL    string
	futuresURL string
	category   string
}

// BinanceTickerResponse - ответ от Binance API для тикеров
type BinanceTickerResponse struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty"`
	BidPrice           string `json:"bidPrice"`
	BidQty             string `json:"bidQty"`
	AskPrice           string `json:"askPrice"`
	AskQty             string `json:"askQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	FirstID            int64  `json:"firstId"`
	LastID             int64  `json:"lastId"`
	Count              int64  `json:"count"`
}

// BinanceFuturesTickerResponse - ответ от Binance Futures API
type BinanceFuturesTickerResponse struct {
	Symbol              string `json:"symbol"`
	PriceChange         string `json:"priceChange"`
	PriceChangePercent  string `json:"priceChangePercent"`
	WeightedAvgPrice    string `json:"weightedAvgPrice"`
	PrevClosePrice      string `json:"prevClosePrice"`
	LastPrice           string `json:"lastPrice"`
	LastQty             string `json:"lastQty"`
	OpenPrice           string `json:"openPrice"`
	HighPrice           string `json:"highPrice"`
	LowPrice            string `json:"lowPrice"`
	Volume              string `json:"volume"`
	QuoteVolume         string `json:"quoteVolume"`
	OpenTime            int64  `json:"openTime"`
	CloseTime           int64  `json:"closeTime"`
	FirstID             int64  `json:"firstId"`
	LastID              int64  `json:"lastId"`
	Count               int64  `json:"count"`
	BidPrice            string `json:"bidPrice"`
	BidQty              string `json:"bidQty"`
	AskPrice            string `json:"askPrice"`
	AskQty              string `json:"askQty"`
	Underlying          string `json:"underlying"`
	UnderlyingType      string `json:"underlyingType"`
	UnderlyingIndex     string `json:"underlyingIndex"`
	UnderlyingIndexType string `json:"underlyingIndexType"`
	MarginAsset         string `json:"marginAsset"`
	ContractType        string `json:"contractType"`
	DeliveryDate        int64  `json:"deliveryDate"`
	OnboardDate         int64  `json:"onboardDate"`
}

// NewBinanceClient создает нового клиента для Binance
func NewBinanceClient(cfg *config.Config) *BinanceClient {
	baseURL := "https://api.binance.com"
	futuresURL := "https://fapi.binance.com"

	return &BinanceClient{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		futuresURL: futuresURL,
		category:   cfg.FuturesCategory,
	}
}

// GetTickers получает тикеры с Binance
func (c *BinanceClient) GetTickers(category string) (*api.TickerResponse, error) {
	var url string
	var response []byte
	var err error

	switch category {
	case "spot":
		url = c.baseURL + "/api/v3/ticker/24hr"
		response, err = c.makeRequest(url)
		if err != nil {
			return nil, err
		}
		return c.parseSpotResponse(response)
	case "futures", "linear":
		url = c.futuresURL + "/fapi/v1/ticker/24hr"
		response, err = c.makeRequest(url)
		if err != nil {
			return nil, err
		}
		return c.parseFuturesResponse(response)
	default:
		return nil, fmt.Errorf("unsupported category: %s", category)
	}
}

// parseSpotResponse парсит ответ от Spot API
func (c *BinanceClient) parseSpotResponse(response []byte) (*api.TickerResponse, error) {
	var binanceTickers []BinanceTickerResponse
	if err := json.Unmarshal(response, &binanceTickers); err != nil {
		return nil, fmt.Errorf("failed to parse binance response: %w", err)
	}

	var tickers []api.Ticker
	for _, ticker := range binanceTickers {
		// Фильтруем только USDT пары
		if len(ticker.Symbol) >= 4 && ticker.Symbol[len(ticker.Symbol)-4:] == "USDT" {
			volume, _ := strconv.ParseFloat(ticker.Volume, 64)
			tickers = append(tickers, api.Ticker{
				Symbol:       ticker.Symbol,
				LastPrice:    ticker.LastPrice,
				Volume24h:    fmt.Sprintf("%.2f", volume),
				Price24hPcnt: ticker.PriceChangePercent,
			})
		}
	}

	return &api.TickerResponse{
		RetCode: 0,
		RetMsg:  "OK",
		Result: api.TickerList{
			List: tickers,
		},
	}, nil
}

// parseFuturesResponse парсит ответ от Futures API
func (c *BinanceClient) parseFuturesResponse(response []byte) (*api.TickerResponse, error) {
	var binanceTickers []BinanceFuturesTickerResponse
	if err := json.Unmarshal(response, &binanceTickers); err != nil {
		return nil, fmt.Errorf("failed to parse binance futures response: %w", err)
	}

	var tickers []api.Ticker
	for _, ticker := range binanceTickers {
		// Фильтруем только USDT perpetual контракты
		if ticker.ContractType == "PERPETUAL" &&
			len(ticker.Symbol) >= 4 &&
			ticker.Symbol[len(ticker.Symbol)-4:] == "USDT" {
			volume, _ := strconv.ParseFloat(ticker.Volume, 64)
			tickers = append(tickers, api.Ticker{
				Symbol:       ticker.Symbol,
				LastPrice:    ticker.LastPrice,
				Volume24h:    fmt.Sprintf("%.2f", volume),
				Price24hPcnt: ticker.PriceChangePercent,
			})
		}
	}

	return &api.TickerResponse{
		RetCode: 0,
		RetMsg:  "OK",
		Result: api.TickerList{
			List: tickers,
		},
	}, nil
}

// makeRequest выполняет HTTP запрос
func (c *BinanceClient) makeRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CryptoExchangeScreenerBot/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// Category возвращает категорию торгов
func (c *BinanceClient) Category() string {
	return c.category
}
