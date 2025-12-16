// internal/api/bybit_client.go
package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"crypto-exchange-screener-bot/internal/config"
)

// BybitClient - клиент для работы с API Bybit
type BybitClient struct {
	httpClient *http.Client
	config     *config.Config
	baseURL    string
	apiKey     string
	apiSecret  string
}

// APIResponse - базовый ответ API Bybit
type APIResponse struct {
	RetCode int         `json:"retCode"`
	RetMsg  string      `json:"retMsg"`
	Result  interface{} `json:"result"`
	Time    int64       `json:"time"`
}

// TickerResponse - ответ для тикеров
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

// NewBybitClient создает новый клиент для работы с API Bybit
func NewBybitClient(cfg *config.Config) *BybitClient {
	// Определяем базовый URL
	baseURL := cfg.BaseURL
	apiKey := cfg.ApiKey
	apiSecret := cfg.ApiSecret

	if cfg.UseTestnet {
		baseURL = cfg.TestnetBaseURL
		apiKey = cfg.TestnetApiKey
		apiSecret = cfg.TestnetApiSecret
	}

	return &BybitClient{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
		},
		config:    cfg,
		baseURL:   baseURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

// generateSignature создает подпись HMAC-SHA256 для приватных запросов
func (c *BybitClient) generateSignature(timestamp, recvWindow, params string) string {
	signString := timestamp + c.apiKey + recvWindow + params

	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(signString))

	return hex.EncodeToString(h.Sum(nil))
}

// sendPublicRequest отправляет публичный запрос к API
func (c *BybitClient) sendPublicRequest(method, endpoint string, params url.Values) ([]byte, error) {
	// Формируем URL
	apiURL := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		apiURL = apiURL + "?" + params.Encode()
	}

	// Создаем запрос
	req, err := http.NewRequest(method, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CryptoExchangeScreenerBot/1.0")

	// Отправляем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Проверяем код ошибки в ответе API
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.RetCode != 0 {
		return nil, fmt.Errorf("API error %d: %s", apiResp.RetCode, apiResp.RetMsg)
	}

	return body, nil
}

// sendPrivateRequest отправляет приватный запрос к API с аутентификацией
func (c *BybitClient) sendPrivateRequest(method, endpoint string, params interface{}) ([]byte, error) {
	// Подготавливаем параметры
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	recvWindow := "5000"

	var paramsStr string
	var bodyData []byte

	if method == http.MethodGet || method == http.MethodDelete {
		// Для GET/DELETE параметры в query string
		if params != nil {
			if p, ok := params.(url.Values); ok {
				paramsStr = p.Encode()
			}
		}
	} else {
		// Для POST/PUT параметры в теле запроса
		if params != nil {
			var err error
			bodyData, err = json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal params: %w", err)
			}
			paramsStr = string(bodyData)
		}
	}

	// Генерируем подпись
	signature := c.generateSignature(timestamp, recvWindow, paramsStr)

	// Формируем URL
	apiURL := c.baseURL + endpoint
	if (method == http.MethodGet || method == http.MethodDelete) && paramsStr != "" {
		apiURL = apiURL + "?" + paramsStr
	}

	// Создаем запрос
	var req *http.Request
	var err error

	if method == http.MethodGet || method == http.MethodDelete {
		req, err = http.NewRequest(method, apiURL, nil)
	} else {
		req, err = http.NewRequest(method, apiURL, bytes.NewBuffer(bodyData))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Добавляем заголовки аутентификации
	req.Header.Set("X-BAPI-API-KEY", c.apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-RECV-WINDOW", recvWindow)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CryptoExchangeScreenerBot/1.0")

	// Отправляем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Проверяем код ошибки в ответе API
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.RetCode != 0 {
		return nil, fmt.Errorf("API error %d: %s", apiResp.RetCode, apiResp.RetMsg)
	}

	return body, nil
}

// GetTickers получает все тикеры для спотового рынка
func (c *BybitClient) GetTickers(category string) (*TickerResponse, error) {
	params := url.Values{}
	params.Set("category", category)

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/tickers", params)
	if err != nil {
		return nil, err
	}

	var tickerResp TickerResponse
	if err := json.Unmarshal(body, &tickerResp); err != nil {
		return nil, fmt.Errorf("failed to parse ticker response: %w", err)
	}

	return &tickerResp, nil
}

// GetKlineData получает свечные данные
func (c *BybitClient) GetKlineData(symbol, category, interval string, limit int) (*KlineResponse, error) {
	params := url.Values{}
	params.Set("category", category)
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	params.Set("limit", strconv.Itoa(limit))

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/kline", params)
	if err != nil {
		return nil, err
	}

	var klineResp KlineResponse
	if err := json.Unmarshal(body, &klineResp); err != nil {
		return nil, fmt.Errorf("failed to parse kline response: %w", err)
	}

	return &klineResp, nil
}

// GetWalletBalance получает баланс кошелька
func (c *BybitClient) GetWalletBalance(accountType string) ([]AccountBalance, error) {
	params := url.Values{}
	params.Set("accountType", accountType)

	body, err := c.sendPrivateRequest(http.MethodGet, "/v5/account/wallet-balance", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result struct {
			List []struct {
				AccountType string           `json:"accountType"`
				Coin        []AccountBalance `json:"coin"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse balance response: %w", err)
	}

	if len(response.Result.List) > 0 {
		return response.Result.List[0].Coin, nil
	}

	return []AccountBalance{}, nil
}

// GetServerTime получает время сервера Bybit
func (c *BybitClient) GetServerTime() (int64, error) {
	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/time", nil)
	if err != nil {
		return 0, err
	}

	var response struct {
		Result struct {
			TimeSecond string `json:"timeSecond"`
			TimeNano   string `json:"timeNano"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to parse server time response: %w", err)
	}

	timeSecond, err := strconv.ParseInt(response.Result.TimeSecond, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timeSecond: %w", err)
	}

	return timeSecond, nil
}

// TestConnection тестирует подключение к API
func (c *BybitClient) TestConnection() error {
	_, err := c.GetServerTime()
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Попробуем получить публичные данные
	_, err = c.GetTickers("spot")
	if err != nil {
		return fmt.Errorf("public API test failed: %w", err)
	}

	return nil
}

// GetPriceChange рассчитывает изменение цены за интервал
func (c *BybitClient) GetPriceChange(symbol string, intervalMinutes int) (float64, error) {
	// Получаем исторические данные
	klineResp, err := c.GetKlineData(symbol, "spot", "1", intervalMinutes+1)
	if err != nil {
		return 0, err
	}

	if len(klineResp.Result.List) < 2 {
		return 0, fmt.Errorf("insufficient data for %s", symbol)
	}

	// Первая свеча (самая старая)
	oldestPrice, err := strconv.ParseFloat(klineResp.Result.List[0][4], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse oldest price: %w", err)
	}

	// Последняя свеча (самая новая)
	newestPrice, err := strconv.ParseFloat(klineResp.Result.List[len(klineResp.Result.List)-1][4], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse newest price: %w", err)
	}

	// Рассчитываем процентное изменение
	changePercent := ((newestPrice - oldestPrice) / oldestPrice) * 100

	return changePercent, nil
}

// GetTopMovers получает топ монет по изменению цены
func (c *BybitClient) GetTopMovers(symbols []string, intervalMinutes int, topN int, ascending bool) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Ограничиваем количество символов для запроса
	if len(symbols) > 50 {
		symbols = symbols[:50]
	}

	// Получаем изменения цен для всех символов
	for _, symbol := range symbols {
		changePercent, err := c.GetPriceChange(symbol, intervalMinutes)
		if err != nil {
			log.Printf("Failed to get price change for %s: %v", symbol, err)
			continue
		}

		results = append(results, map[string]interface{}{
			"symbol":         symbol,
			"change_percent": changePercent,
			"interval_min":   intervalMinutes,
		})
	}

	// Сортируем по изменению цены
	if ascending {
		// По возрастанию (наибольшее падение первое)
		sort.Slice(results, func(i, j int) bool {
			return results[i]["change_percent"].(float64) < results[j]["change_percent"].(float64)
		})
	} else {
		// По убыванию (наибольший рост первым)
		sort.Slice(results, func(i, j int) bool {
			return results[i]["change_percent"].(float64) > results[j]["change_percent"].(float64)
		})
	}

	// Возвращаем топ N
	if topN > len(results) {
		topN = len(results)
	}

	return results[:topN], nil
}
