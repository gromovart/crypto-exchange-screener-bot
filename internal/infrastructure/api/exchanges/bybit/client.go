// internal/infrastructure/api/exchanges/bybit/client.go
package bybit

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
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/api"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
)

// ============================================
// BYBIT CLIENT
// ============================================

// BybitClient - клиент для работы с API Bybit
type BybitClient struct {
	httpClient  *http.Client
	config      *config.Config
	baseURL     string
	apiKey      string
	apiSecret   string
	category    string
	lastRequest time.Time
	rateLimit   time.Duration
}

// OIConfig настройки для получения Open Interest
type OIConfig struct {
	DefaultCategory string        `json:"default_category"`
	DefaultInterval string        `json:"default_interval"`
	CacheTTL        time.Duration `json:"cache_ttl"`
	RetryCount      int           `json:"retry_count"`
}

// NewOIConfig создает конфигурацию по умолчанию
func NewOIConfig() OIConfig {
	return OIConfig{
		DefaultCategory: CategoryLinear,
		DefaultInterval: OIInterval5Min,
		CacheTTL:        5 * time.Minute,
		RetryCount:      3,
	}
}

// NewBybitClient создает новый клиент для работы с API Bybit
func NewBybitClient(cfg *config.Config) *BybitClient {
	// Определяем базовый URL
	baseURL := cfg.BaseURL
	apiKey := cfg.ApiKey
	apiSecret := cfg.ApiSecret

	// Определяем категорию по умолчанию
	category := cfg.FuturesCategory
	if category == "" {
		category = CategoryLinear
	}

	// Настраиваем rate limiting
	rateLimit := cfg.RateLimitDelay
	if rateLimit <= 0 {
		rateLimit = 100 * time.Millisecond
	}

	return &BybitClient{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.HTTPPort) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        cfg.MaxConcurrentRequests,
				MaxIdleConnsPerHost: cfg.MaxConcurrentRequests,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		config:      cfg,
		baseURL:     baseURL,
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		category:    category,
		rateLimit:   rateLimit,
		lastRequest: time.Now().Add(-rateLimit),
	}
}

// ============================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// ============================================

// waitForRateLimit ждет, если нужно соблюдать rate limit
func (c *BybitClient) waitForRateLimit() {
	elapsed := time.Since(c.lastRequest)
	if elapsed < c.rateLimit {
		sleepTime := c.rateLimit - elapsed
		time.Sleep(sleepTime)
	}
	c.lastRequest = time.Now()
}

// generateSignature создает подпись HMAC-SHA256
func (c *BybitClient) generateSignature(timestamp, recvWindow, params string) string {
	signString := timestamp + c.apiKey + recvWindow + params

	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(signString))

	return hex.EncodeToString(h.Sum(nil))
}

// sendPublicRequest отправляет публичный запрос
func (c *BybitClient) sendPublicRequest(method, endpoint string, params url.Values) ([]byte, error) {
	c.waitForRateLimit()

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

// sendPrivateRequest отправляет приватный запрос
func (c *BybitClient) sendPrivateRequest(method, endpoint string, params interface{}) ([]byte, error) {
	c.waitForRateLimit()

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
	var reqErr error

	if method == http.MethodGet || method == http.MethodDelete {
		req, reqErr = http.NewRequest(method, apiURL, nil)
	} else {
		req, reqErr = http.NewRequest(method, apiURL, bytes.NewBuffer(bodyData))
	}

	if reqErr != nil {
		return nil, fmt.Errorf("failed to create request: %w", reqErr)
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

// ============================================
// ОСНОВНЫЕ API МЕТОДЫ
// ============================================

// GetTickers получает все тикеры для указанной категории
func (c *BybitClient) GetTickers(category string) (*api.TickerResponse, error) {
	if category == "" {
		category = c.category
	}

	params := url.Values{}
	params.Set("category", category)

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/tickers", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get tickers: %w", err)
	}

	var tickerResp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string `json:"category"`
			List     []struct {
				Symbol       string `json:"symbol"`
				LastPrice    string `json:"lastPrice"`
				Volume24h    string `json:"volume24h"`
				Price24hPcnt string `json:"price24hPcnt"`
				Turnover24h  string `json:"turnover24h"`
				High24h      string `json:"high24h"`
				Low24h       string `json:"low24h"`
				OpenInterest string `json:"openInterest"` // ✅ Обязательно парсим это поле
				FundingRate  string `json:"fundingRate"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &tickerResp); err != nil {
		return nil, fmt.Errorf("failed to parse ticker response: %w", err)
	}

	// Преобразуем в общую структуру api.TickerResponse
	var tickers []api.Ticker
	for _, t := range tickerResp.Result.List {
		tickers = append(tickers, api.Ticker{
			Symbol:       t.Symbol,
			LastPrice:    t.LastPrice,
			Volume24h:    t.Volume24h,
			Price24hPcnt: t.Price24hPcnt,
			Turnover24h:  t.Turnover24h,
			High24h:      t.High24h,
			Low24h:       t.Low24h,
			OpenInterest: t.OpenInterest, // ✅ Сохраняем Open Interest
			FundingRate:  t.FundingRate,
		})

		// Отладочный лог для OI
		if t.OpenInterest != "" && t.OpenInterest != "0" {
			oi, _ := strconv.ParseFloat(t.OpenInterest, 64)
			log.Printf("📊 BybitClient.GetTickers: %s OI = %.0f", t.Symbol, oi)
		}
	}

	return &api.TickerResponse{
		RetCode: tickerResp.RetCode,
		RetMsg:  tickerResp.RetMsg,
		Result: api.TickerList{
			Category: tickerResp.Result.Category, // ✅ Теперь Category будет установлен
			List:     tickers,
		},
	}, nil
}

// GetInstrumentsInfo получает информацию об инструментах
func (c *BybitClient) GetInstrumentsInfo(category, symbol, status string) ([]InstrumentInfo, error) {
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if status != "" {
		params.Set("status", status)
	}

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/instruments-info", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get instruments info: %w", err)
	}

	var response struct {
		Result struct {
			List []InstrumentInfo `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse instruments info response: %w", err)
	}

	return response.Result.List, nil
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
		return nil, fmt.Errorf("failed to get kline data: %w", err)
	}

	var klineResp KlineResponse
	if err := json.Unmarshal(body, &klineResp); err != nil {
		return nil, fmt.Errorf("failed to parse kline response: %w", err)
	}

	return &klineResp, nil
}

// GetKlineDataWithInterval получает свечные данные с указанным интервалом
func (c *BybitClient) GetKlineDataWithInterval(symbol, category, interval string, limit int) (*KlineResponse, error) {
	return c.GetKlineData(symbol, category, interval, limit)
}

// ============================================
// OPEN INTEREST API
// ============================================

// GetOpenInterest получает открытый интерес для конкретного символа
func (c *BybitClient) GetOpenInterest(symbol string) (float64, error) {
	return c.GetOpenInterestWithParams(symbol, "", "")
}

// GetOpenInterestForSymbolsBatch получает OI для нескольких символов (оптимизировано)
func (c *BybitClient) GetOpenInterestForSymbolsBatch(symbols []string) (map[string]float64, error) {
	result := make(map[string]float64)

	if len(symbols) == 0 {
		return result, nil
	}

	// Сначала пытаемся получить из тикеров
	tickers, err := c.GetTickers(c.category)
	if err != nil {
		log.Printf("⚠️ Не удалось получить тикеры: %v", err)
		// Продолжаем с индивидуальными запросами
	} else {
		// Создаем карту для быстрого поиска
		tickerMap := make(map[string]api.Ticker)
		for _, ticker := range tickers.Result.List {
			tickerMap[ticker.Symbol] = ticker
		}

		// Ищем OI в тикерах
		for _, symbol := range symbols {
			if ticker, exists := tickerMap[symbol]; exists {
				if oi, err := ticker.GetOpenInterestFloat(); err == nil && oi > 0 {
					result[symbol] = oi
					log.Printf("✅ Получен OI из тикеров для %s: %.0f", symbol, oi)
					continue
				}
			}
		}
	}

	// Для символов, где OI не нашли в тикерах
	remainingSymbols := make([]string, 0)
	for _, symbol := range symbols {
		if _, found := result[symbol]; !found {
			remainingSymbols = append(remainingSymbols, symbol)
		}
	}

	// Делаем индивидуальные запросы для оставшихся символов
	for _, symbol := range remainingSymbols {
		c.waitForRateLimit()

		oi, err := c.GetOpenInterestWithParams(symbol, c.category, "5min")
		if err != nil {
			log.Printf("⚠️ Ошибка получения OI для %s: %v", symbol, err)
			continue
		}

		if oi > 0 {
			result[symbol] = oi
			log.Printf("✅ Получен OI через API для %s: %.0f", symbol, oi)
		}

		time.Sleep(20 * time.Millisecond)
	}

	log.Printf("📊 Итого получено OI для %d из %d символов", len(result), len(symbols))
	return result, nil
}

// tryGetOpenInterestWithDifferentCategories пробует получить OI с разными категориями
func (c *BybitClient) tryGetOpenInterestWithDifferentCategories(symbol string) (float64, error) {
	// Пробуем разные категории
	categories := []string{"linear", "inverse", "spot"}

	// ⚠️ Правильные интервалы для Bybit API
	intervals := []string{"5min", "15min", "30min", "1h", "4h", "1d"}

	for _, category := range categories {
		for _, interval := range intervals {
			oi, err := c.GetOpenInterestWithParams(symbol, category, interval)
			if err == nil && oi > 0 {
				log.Printf("🔍 BybitClient: найден OI для %s в категории %s интервал %s: %.0f",
					symbol, category, interval, oi)
				return oi, nil
			}

			time.Sleep(20 * time.Millisecond)
		}
	}

	return 0, fmt.Errorf("не удалось получить OI для %s ни в одной категории/интервале", symbol)
}

// IsOIAvailable проверяет доступность Open Interest API
func (c *BybitClient) IsOIAvailable() (bool, error) {
	// Пробуем получить OI для BTCUSDT (самый ликвидный символ)
	_, err := c.GetOpenInterest("BTCUSDT")
	if err != nil {
		// Проверяем тип ошибки
		if strings.Contains(err.Error(), "params error") ||
			strings.Contains(err.Error(), "10001") ||
			strings.Contains(err.Error(), "interval") {
			log.Println("⚠️  BybitClient: OI API требует исправления параметров")
			return false, err
		}

		if strings.Contains(err.Error(), "rate limit") ||
			strings.Contains(err.Error(), "10006") {
			log.Println("⚠️  BybitClient: OI API ограничено rate limit")
			return true, nil // API доступно, но с ограничениями
		}

		// Другие ошибки
		return false, err
	}

	return true, nil
}

// GetOpenInterestWithParams получает открытый интерес с указанием параметров
func (c *BybitClient) GetOpenInterestWithParams(symbol, category, interval string) (float64, error) {
	if symbol == "" {
		return 0, fmt.Errorf("symbol is required for open interest API")
	}

	if category == "" {
		category = "linear"
	}
	if interval == "" {
		interval = "5min"
	}

	endpoint := "/v5/market/open-interest"
	params := url.Values{}
	params.Set("category", category)
	params.Set("symbol", symbol)
	params.Set("intervalTime", interval) // ⚠️ Правильное имя параметра для Bybit V5!

	body, err := c.sendPublicRequest(http.MethodGet, endpoint, params)
	if err != nil {
		return 0, fmt.Errorf("failed to get open interest for %s: %w", symbol, err)
	}

	var response struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Symbol       string `json:"symbol"`
				OpenInterest string `json:"openInterest"`
				Timestamp    string `json:"timestamp"`
			} `json:"list"`
		} `json:"result"`
		RetExtInfo map[string]interface{} `json:"retExtInfo"`
		Time       int64                  `json:"time"`
	}

	// Парсим ответ
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to parse open interest response: %w", err)
	}

	if response.RetCode != 0 {
		return 0, fmt.Errorf("bybit API error %d: %s", response.RetCode, response.RetMsg)
	}

	// Проверяем, что есть данные в массиве
	if len(response.Result.List) == 0 || response.Result.List[0].OpenInterest == "" {
		return 0, nil
	}

	// Берем первый элемент (самый свежий)
	oi, err := strconv.ParseFloat(response.Result.List[0].OpenInterest, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse open interest value: %w", err)
	}

	return oi, nil
}

// GetCurrentOpenInterestFromTickers получает текущий OI из данных тикеров
func (c *BybitClient) GetCurrentOpenInterestFromTickers(symbol string) (float64, error) {
	// Получаем все тикеры
	tickers, err := c.GetTickers(c.category)
	if err != nil {
		return 0, err
	}

	// Ищем нужный символ
	for _, ticker := range tickers.Result.List {
		if ticker.Symbol == symbol {
			// Парсим Open Interest из тикеров
			if openInterestStr, ok := ticker.GetOpenInterest(); ok && openInterestStr != "" {
				oi, err := strconv.ParseFloat(openInterestStr, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse open interest from ticker: %w", err)
				}
				return oi, nil
			}
		}
	}

	return 0, fmt.Errorf("open interest not found for %s in tickers", symbol)
}

// GetOpenInterestForSymbols получает OI для нескольких символов (исправленная версия)
func (c *BybitClient) GetOpenInterestForSymbols(symbols []string) (map[string]float64, error) {
	// Ограничиваем количество символов для одного запроса
	maxSymbols := 10 // Bybit может иметь ограничения
	if len(symbols) > maxSymbols {
		// Делим на группы
		allResults := make(map[string]float64)

		for i := 0; i < len(symbols); i += maxSymbols {
			end := i + maxSymbols
			if end > len(symbols) {
				end = len(symbols)
			}

			batch := symbols[i:end]
			batchResults, err := c.GetOpenInterestForSymbolsBatch(batch)
			if err != nil {
				log.Printf("⚠️ Ошибка получения OI для batch %d-%d: %v", i, end, err)
			}

			// Объединяем результаты
			for symbol, oi := range batchResults {
				allResults[symbol] = oi
			}

			// Задержка между группами
			if end < len(symbols) {
				time.Sleep(500 * time.Millisecond)
			}
		}

		return allResults, nil
	}

	// Для небольшого количества символов
	return c.GetOpenInterestForSymbolsBatch(symbols)
}

// GetOpenInterestHistory получает историю OI
func (c *BybitClient) GetOpenInterestHistory(symbol, interval string, limit int) ([]OIDataPoint, error) {
	endpoint := "/v5/market/open-interest"
	params := url.Values{}
	params.Set("category", "linear")
	params.Set("symbol", symbol)
	params.Set("intervalTime", interval)

	if limit > 0 && limit <= 200 { // Bybit максимум 200
		params.Set("limit", strconv.Itoa(limit))
	}

	body, err := c.sendPublicRequest(http.MethodGet, endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get open interest history: %w", err)
	}

	var response struct {
		Result struct {
			List []struct {
				Symbol       string `json:"symbol"`
				OpenInterest string `json:"openInterest"`
				Timestamp    string `json:"timestamp"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse open interest history: %w", err)
	}

	var dataPoints []OIDataPoint
	for _, item := range response.Result.List {
		oi, _ := strconv.ParseFloat(item.OpenInterest, 64)
		timestamp, _ := strconv.ParseInt(item.Timestamp, 10, 64)

		dataPoints = append(dataPoints, OIDataPoint{
			Symbol:       item.Symbol,
			OpenInterest: oi,
			Timestamp:    time.Unix(timestamp/1000, 0),
		})
	}

	return dataPoints, nil
}

// OIDataPoint структура для хранения OI с временной меткой
type OIDataPoint struct {
	Symbol       string    `json:"symbol"`
	OpenInterest float64   `json:"openInterest"`
	Timestamp    time.Time `json:"timestamp"`
}

// ============================================
// АККАУНТ И БАЛАНС
// ============================================

// GetWalletBalance получает баланс кошелька
func (c *BybitClient) GetWalletBalance(accountType string) ([]AccountBalance, error) {
	params := url.Values{}
	params.Set("accountType", accountType)

	body, err := c.sendPrivateRequest(http.MethodGet, "/v5/account/wallet-balance", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balance: %w", err)
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

// ============================================
// СИСТЕМНЫЕ МЕТОДЫ
// ============================================

// GetServerTime получает время сервера Bybit
func (c *BybitClient) GetServerTime() (int64, error) {
	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/time", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get server time: %w", err)
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
	// Проверяем публичный доступ
	_, err := c.GetServerTime()
	if err != nil {
		return fmt.Errorf("public API test failed: %w", err)
	}

	// Проверяем тикеры
	tickers, err := c.GetTickers("spot")
	if err != nil {
		return fmt.Errorf("tickers API test failed: %w", err)
	}

	log.Printf("✅ BybitClient: подключение успешно, получено %d тикеров", len(tickers.Result.List))
	return nil
}

// Category возвращает текущую категорию клиента
func (c *BybitClient) Category() string {
	if c.category != "" {
		return c.category
	}
	return CategoryLinear
}

// ============================================
// МЕТОДЫ ДЛЯ АНАЛИЗА
// ============================================

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

// GetRecentKlinesForPeriod получает свечи для анализа периода роста
func (c *BybitClient) GetRecentKlinesForPeriod(symbol string, periodMinutes int) ([][]string, error) {
	// Определяем интервал свечей в зависимости от периода
	var interval string
	var limit int

	switch {
	case periodMinutes <= 5:
		interval = "1"
		limit = periodMinutes
	case periodMinutes <= 30:
		interval = "5"
		limit = periodMinutes / 5
	case periodMinutes <= 240:
		interval = "15"
		limit = periodMinutes / 15
	case periodMinutes <= 1440:
		interval = "60"
		limit = periodMinutes / 60
	default:
		interval = "D"
		limit = periodMinutes / 1440
	}

	// Минимальное количество свечей
	if limit < 2 {
		limit = 2
	}

	limit = limit + 2

	resp, err := c.GetKlineDataWithInterval(symbol, "linear", interval, limit)
	if err != nil {
		return nil, err
	}

	return resp.Result.List, nil
}

// AnalyzeGrowth анализирует рост/падение за период
func (c *BybitClient) AnalyzeGrowth(symbol string, periodMinutes int, checkContinuity bool) (*types.GrowthAnalysis, error) {
	klines, err := c.GetRecentKlinesForPeriod(symbol, periodMinutes)
	if err != nil {
		return nil, err
	}

	if len(klines) < 2 {
		return nil, fmt.Errorf("insufficient data for growth analysis")
	}

	var dataPoints []types.PriceDataPoint

	// Парсим данные из свечей
	for _, kline := range klines {
		if len(kline) >= 5 {
			closePrice, err := strconv.ParseFloat(kline[4], 64)
			if err != nil {
				continue
			}

			timestampMs, err := strconv.ParseInt(kline[0], 10, 64)
			if err != nil {
				continue
			}

			volume, _ := strconv.ParseFloat(kline[5], 64)

			dataPoints = append(dataPoints, types.PriceDataPoint{
				Price:     closePrice,
				Timestamp: time.Unix(timestampMs/1000, 0),
				Volume:    volume,
			})
		}
	}

	if len(dataPoints) < 2 {
		return nil, fmt.Errorf("not enough valid data points")
	}

	// Анализируем рост/падение
	return c.analyzeGrowthData(symbol, periodMinutes, dataPoints, checkContinuity)
}

// analyzeGrowthData анализирует данные на рост/падение
func (c *BybitClient) analyzeGrowthData(symbol string, periodMinutes int, dataPoints []types.PriceDataPoint, checkContinuity bool) (*types.GrowthAnalysis, error) {
	analysis := &types.GrowthAnalysis{
		Symbol:     symbol,
		Period:     periodMinutes,
		DataPoints: dataPoints,
	}

	// Сортируем по времени
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	// Рассчитываем базовые метрики
	startPrice := dataPoints[0].Price
	endPrice := dataPoints[len(dataPoints)-1].Price

	// Процент изменения
	analysis.GrowthPercent = ((endPrice - startPrice) / startPrice) * 100
	analysis.FallPercent = -analysis.GrowthPercent

	// Находим min/max
	minPrice := startPrice
	maxPrice := startPrice
	for _, point := range dataPoints {
		if point.Price < minPrice {
			minPrice = point.Price
		}
		if point.Price > maxPrice {
			maxPrice = point.Price
		}
	}
	analysis.MinPrice = minPrice
	analysis.MaxPrice = maxPrice

	// Волатильность
	analysis.Volatility = ((maxPrice - minPrice) / startPrice) * 100

	// Проверяем непрерывный рост
	if checkContinuity {
		analysis.IsGrowing = c.checkContinuousGrowth(dataPoints)
		analysis.IsFalling = c.checkContinuousFall(dataPoints)
	} else {
		// Просто проверяем общее изменение
		analysis.IsGrowing = analysis.GrowthPercent > 0
		analysis.IsFalling = analysis.GrowthPercent < 0
	}

	return analysis, nil
}

// checkContinuousGrowth проверяет непрерывный рост
func (c *BybitClient) checkContinuousGrowth(dataPoints []types.PriceDataPoint) bool {
	for i := 1; i < len(dataPoints); i++ {
		if dataPoints[i].Price <= dataPoints[i-1].Price {
			return false
		}
	}
	return true
}

// checkContinuousFall проверяет непрерывное падение
func (c *BybitClient) checkContinuousFall(dataPoints []types.PriceDataPoint) bool {
	for i := 1; i < len(dataPoints); i++ {
		if dataPoints[i].Price >= dataPoints[i-1].Price {
			return false
		}
	}
	return true
}

// ============================================
// ДОПОЛНИТЕЛЬНЫЕ МЕТОДЫ
// ============================================

// Get24hVolume получает 24-часовой объем для символа
func (c *BybitClient) Get24hVolume(symbol string) (float64, error) {
	tickers, err := c.GetTickers(c.category)
	if err != nil {
		return 0, err
	}

	for _, ticker := range tickers.Result.List {
		if ticker.Symbol == symbol {
			volume, err := strconv.ParseFloat(ticker.Turnover24h, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse volume: %w", err)
			}
			return volume, nil
		}
	}

	return 0, fmt.Errorf("symbol %s not found", symbol)
}

// GetSymbolVolume получает объем для нескольких символов
func (c *BybitClient) GetSymbolVolume(symbols []string) (map[string]float64, error) {
	params := url.Values{}
	params.Set("category", c.category)

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/tickers", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result struct {
			List []struct {
				Symbol      string `json:"symbol"`
				Turnover24h string `json:"turnover24h"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse volume response: %w", err)
	}

	volumes := make(map[string]float64)
	for _, symbol := range symbols {
		for _, ticker := range response.Result.List {
			if ticker.Symbol == symbol && ticker.Turnover24h != "" {
				volume, err := strconv.ParseFloat(ticker.Turnover24h, 64)
				if err == nil {
					volumes[symbol] = volume
				}
				break
			}
		}
	}

	return volumes, nil
}

// GetFundingRate получает ставку фандинга для символа
func (c *BybitClient) GetFundingRate(symbol string) (float64, error) {
	// Получаем тикеры, включая funding rate
	params := url.Values{}
	params.Set("category", c.category)

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/tickers", params)
	if err != nil {
		return 0, err
	}

	var response struct {
		Result struct {
			List []struct {
				Symbol      string `json:"symbol"`
				FundingRate string `json:"fundingRate"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to parse tickers response: %w", err)
	}

	for _, ticker := range response.Result.List {
		if ticker.Symbol == symbol && ticker.FundingRate != "" {
			rate, err := strconv.ParseFloat(ticker.FundingRate, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse funding rate: %w", err)
			}
			return rate, nil
		}
	}

	return 0, fmt.Errorf("funding rate not found for %s", symbol)
}

// GetFundingRates получает ставки фандинга для нескольких символов
func (c *BybitClient) GetFundingRates(symbols []string) (map[string]float64, error) {
	params := url.Values{}
	params.Set("category", c.category)

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/tickers", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result struct {
			List []struct {
				Symbol      string `json:"symbol"`
				FundingRate string `json:"fundingRate"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse tickers response: %w", err)
	}

	rates := make(map[string]float64)

	for _, symbol := range symbols {
		for _, ticker := range response.Result.List {
			if ticker.Symbol == symbol && ticker.FundingRate != "" {
				rate, err := strconv.ParseFloat(ticker.FundingRate, 64)
				if err == nil {
					rates[symbol] = rate
				}
				break
			}
		}
	}

	return rates, nil
}

// ============================================
// ПРОСТЫЕ КОНСТРУКТОРЫ
// ============================================

// NewBybitClientSimple создает простой клиент
func NewBybitClientSimple(apiKey, apiSecret, baseURL, category string, timeout time.Duration) *BybitClient {
	return &BybitClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		category:  category,
		rateLimit: 100 * time.Millisecond,
	}
}
