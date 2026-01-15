// internal/infrastructure/api/exchanges/bybit/client.go
package bybit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/api"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
)

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
			Timeout: 10 * time.Second, // вынести в отдельную переменную конфигурации
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
				OpenInterest string `json:"openInterest"`
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
			OpenInterest: t.OpenInterest,
			FundingRate:  t.FundingRate,
		})
	}

	return &api.TickerResponse{
		RetCode: tickerResp.RetCode,
		RetMsg:  tickerResp.RetMsg,
		Result: api.TickerList{
			Category: tickerResp.Result.Category,
			List:     tickers,
		},
	}, nil
}

// ============================================
// API ЛИКВИДАЦИЙ
// ============================================

// GetRecentLiquidations получает последние ликвидации
func (c *BybitClient) GetRecentLiquidations(symbol string, limit int) ([]LiquidationData, error) {
	params := url.Values{}
	params.Set("category", "linear")
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("limit", strconv.Itoa(limit))

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/recent-trade", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent trades: %w", err)
	}

	var response RecentTradesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse trades response: %w", err)
	}

	var liquidations []LiquidationData
	for _, item := range response.Result.List {
		// Фильтруем только ликвидации (ExecType == "Liquidation" или "BustTrade")
		if item.ExecType == "Liquidation" || item.ExecType == "BustTrade" {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err != nil {
				logger.Warn("⚠️ Ошибка парсинга цены ликвидации %s: %v", item.Price, err)
				continue
			}

			size, err := strconv.ParseFloat(item.Size, 64)
			if err != nil {
				logger.Warn("⚠️ Ошибка парсинга размера ликвидации %s: %v", item.Size, err)
				continue
			}

			// Парсим время (миллисекунды)
			timestampMs, err := strconv.ParseInt(item.Time, 10, 64)
			if err != nil {
				logger.Warn("⚠️ Ошибка парсинга времени ликвидации %s: %v", item.Time, err)
				continue
			}

			timestamp := time.Unix(timestampMs/1000, 0)

			liquidations = append(liquidations, LiquidationData{
				Symbol:        item.Symbol,
				Side:          item.Side,
				Price:         price,
				Quantity:      size,
				Time:          timestamp,
				IsLiquidation: true,
			})

			logger.Debug("💥 Найдена ликвидация %s: %s %.2f @ $%.2f ($%.0f)",
				item.Symbol, item.Side, size, price, price*size)
		}
	}

	logger.Debug("📊 Получено %d ликвидаций для %s", len(liquidations), symbol)
	return liquidations, nil
}

// GetLiquidationsVolume рассчитывает совокупный объем ликвидаций за период
func (c *BybitClient) GetLiquidationsVolume(symbol string, period time.Duration) (float64, error) {
	// Рассчитываем лимит на основе периода (примерно 1 сделка в секунду)
	estimatedTrades := int(period.Seconds())
	if estimatedTrades > 1000 {
		estimatedTrades = 1000 // Максимальный лимит API
	}

	liquidations, err := c.GetRecentLiquidations(symbol, estimatedTrades)
	if err != nil {
		return 0, fmt.Errorf("failed to get liquidations: %w", err)
	}

	var totalVolume float64
	cutoffTime := time.Now().Add(-period)

	logger.Debug("🔍 Анализ ликвидаций для %s за период %v", symbol, period)

	for _, liq := range liquidations {
		if liq.Time.After(cutoffTime) {
			volume := liq.Price * liq.Quantity // Объем в USD
			totalVolume += volume

			logger.Debug("   + Ликвидация %s: $%.0f @ $%.2f",
				liq.Side, volume, liq.Price)
		}
	}

	logger.Debug("💰 Общий объем ликвидаций %s за %v: $%.0f",
		symbol, period, totalVolume)

	return totalVolume, nil
}

// GetLiquidationsSummary получает сводку по ликвидациям
func (c *BybitClient) GetLiquidationsSummary(symbol string, period time.Duration) (map[string]interface{}, error) {
	logger.Debug("📊 Получение сводки ликвидаций для %s за %v", symbol, period)

	// Получаем данные
	liquidations, err := c.GetRecentLiquidations(symbol, 200) // Максимум 200 записей
	if err != nil {
		return nil, fmt.Errorf("failed to get liquidations: %w", err)
	}

	cutoffTime := time.Now().Add(-period)
	var totalVolume, longLiqVolume, shortLiqVolume float64
	var longCount, shortCount int

	logger.Debug("🔍 Фильтрация %d записей ликвидаций...", len(liquidations))

	for _, liq := range liquidations {
		if liq.Time.After(cutoffTime) {
			volume := liq.Price * liq.Quantity
			totalVolume += volume

			// Определяем тип ликвидации
			// Buy ликвидация = ликвидация длинной позиции (продажа)
			// Sell ликвидация = ликвидация короткой позиции (покупка)
			if liq.Side == "Buy" { // Buy ликвидация = ликвидация длинной позиции
				longLiqVolume += volume
				longCount++
				logger.Debug("   📉 Long liquidation: $%.0f", volume)
			} else if liq.Side == "Sell" { // Sell ликвидация = ликвидация короткой позиции
				shortLiqVolume += volume
				shortCount++
				logger.Debug("   📈 Short liquidation: $%.0f", volume)
			}
		}
	}

	// Рассчитываем соотношения
	longRatio := safeDivide(longLiqVolume, totalVolume)
	shortRatio := safeDivide(shortLiqVolume, totalVolume)

	result := map[string]interface{}{
		"symbol":           symbol,
		"period":           period.String(),
		"total_volume_usd": totalVolume,
		"long_liq_volume":  longLiqVolume,
		"short_liq_volume": shortLiqVolume,
		"long_liq_count":   longCount,
		"short_liq_count":  shortCount,
		"total_liq_count":  longCount + shortCount,
		"long_ratio":       longRatio,
		"short_ratio":      shortRatio,
		"update_time":      time.Now(),
	}

	logger.Debug("✅ Сводка ликвидаций %s:", symbol)
	logger.Debug("   Общий объем: $%.0f", totalVolume)
	logger.Debug("   Long ликвидации: $%.0f (%.1f%%)", longLiqVolume, longRatio)
	logger.Debug("   Short ликвидации: $%.0f (%.1f%%)", shortLiqVolume, shortRatio)
	logger.Debug("   Количество: %d (long: %d, short: %d)",
		longCount+shortCount, longCount, shortCount)

	return result, nil
}

// GetLiquidationsMetrics получает метрики ликвидаций
func (c *BybitClient) GetLiquidationsMetrics(symbol string) (*LiquidationMetrics, error) {
	summary, err := c.GetLiquidationsSummary(symbol, 5*time.Minute) // За последние 5 минут
	if err != nil {
		return nil, err
	}

	metrics := &LiquidationMetrics{
		Symbol:         symbol,
		TotalVolumeUSD: summary["total_volume_usd"].(float64),
		LongLiqVolume:  summary["long_liq_volume"].(float64),
		ShortLiqVolume: summary["short_liq_volume"].(float64),
		LongLiqCount:   summary["long_liq_count"].(int),
		ShortLiqCount:  summary["short_liq_count"].(int),
		UpdateTime:     time.Now(),
	}

	return metrics, nil
}

// GetMultipleLiquidationsMetrics получает метрики ликвидаций для нескольких символов
func (c *BybitClient) GetMultipleLiquidationsMetrics(symbols []string) (map[string]*LiquidationMetrics, error) {
	results := make(map[string]*LiquidationMetrics)

	for _, symbol := range symbols {
		metrics, err := c.GetLiquidationsMetrics(symbol)
		if err != nil {
			log.Printf("⚠️ Не удалось получить ликвидации для %s: %v", symbol, err)
			continue
		}
		results[symbol] = metrics

		// Rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("✅ Получены метрики ликвидаций для %d символов", len(results))
	return results, nil
}

// safeDivide безопасное деление
func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b * 100
}

// ============================================
// OPEN INTEREST API
// ============================================

// GetOpenInterest получает открытый интерес для конкретного символа
func (c *BybitClient) GetOpenInterest(symbol string) (float64, error) {
	return c.GetOpenInterestWithParams(symbol, "", "")
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
	params.Set("intervalTime", interval)

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

// GetOpenInterestForSymbols получает OI для нескольких символов
func (c *BybitClient) GetOpenInterestForSymbols(symbols []string) (map[string]float64, error) {
	result := make(map[string]float64)

	for _, symbol := range symbols {
		oi, err := c.GetOpenInterest(symbol)
		if err != nil {
			logger.Warn("⚠️ Ошибка получения OI для %s: %v", symbol, err)
			continue
		}

		if oi > 0 {
			result[symbol] = oi
			logger.Debug("✅ Получен OI для %s: %.0f", symbol, oi)
		}

		// Rate limiting
		time.Sleep(50 * time.Millisecond)
	}

	return result, nil
}

// ============================================
// ДОПОЛНИТЕЛЬНЫЕ МЕТОДЫ
// ============================================

// GetFundingRate получает ставку фандинга для символа
func (c *BybitClient) GetFundingRate(symbol string) (float64, error) {
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

// TestConnection тестирует подключение к API
func (c *BybitClient) TestConnection() error {
	// Проверяем публичный доступ
	_, err := c.GetTickers("spot")
	if err != nil {
		return fmt.Errorf("tickers API test failed: %w", err)
	}

	log.Printf("✅ BybitClient: подключение успешно")
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
// ТИПЫ ДЛЯ РЕАЛЬНЫХ СДЕЛОК
// ============================================

// TradeData представляет данные о сделке
type TradeData struct {
	Symbol string    `json:"symbol"`
	Side   string    `json:"side"` // "Buy" или "Sell"
	Price  float64   `json:"price"`
	Size   float64   `json:"size"`
	Time   time.Time `json:"time"`
}

// VolumeDelta представляет дельту объемов
type VolumeDelta struct {
	Symbol       string    `json:"symbol"`
	Period       string    `json:"period"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	BuyVolume    float64   `json:"buy_volume"`
	SellVolume   float64   `json:"sell_volume"`
	Delta        float64   `json:"delta"`         // buyVolume - sellVolume
	DeltaPercent float64   `json:"delta_percent"` // Процентное изменение
	TotalTrades  int       `json:"total_trades"`
	UpdateTime   time.Time `json:"update_time"`
}

// ============================================
// МЕТОДЫ ДЛЯ РЕАЛЬНЫХ СДЕЛОК
// ============================================

// GetRecentTrades получает последние сделки
func (c *BybitClient) GetRecentTrades(symbol string, limit int) ([]TradeData, error) {
	params := url.Values{}
	params.Set("category", "linear")
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(limit))

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/recent-trade", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent trades: %w", err)
	}

	var response struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []struct {
				Symbol string `json:"symbol"`
				Side   string `json:"side"` // "Buy" или "Sell"
				Size   string `json:"size"`
				Price  string `json:"price"`
				Time   string `json:"time"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse trades response: %w", err)
	}

	var trades []TradeData
	for _, item := range response.Result.List {
		price, err := strconv.ParseFloat(item.Price, 64)
		if err != nil {
			log.Printf("⚠️ Ошибка парсинга цены сделки %s: %v", item.Price, err)
			continue
		}

		size, err := strconv.ParseFloat(item.Size, 64)
		if err != nil {
			log.Printf("⚠️ Ошибка парсинга размера сделки %s: %v", item.Size, err)
			continue
		}

		// Парсим время (миллисекунды)
		timestampMs, err := strconv.ParseInt(item.Time, 10, 64)
		if err != nil {
			log.Printf("⚠️ Ошибка парсинга времени сделки %s: %v", item.Time, err)
			continue
		}

		timestamp := time.Unix(timestampMs/1000, (timestampMs%1000)*int64(time.Millisecond))

		trades = append(trades, TradeData{
			Symbol: item.Symbol,
			Side:   item.Side,
			Price:  price,
			Size:   size,
			Time:   timestamp,
		})
	}

	logger.Debug("📊 Получено %d сделок для %s", len(trades), symbol)
	return trades, nil
}

// CalculateVolumeDelta рассчитывает дельту объемов за период
func (c *BybitClient) CalculateVolumeDelta(symbol string, period time.Duration) (*VolumeDelta, error) {
	startTime := time.Now().Add(-period)
	endTime := time.Now()

	logger.Debug("🔍 Расчет дельты объемов для %s за период %v", symbol, period)

	// Оцениваем количество сделок (~60 сделок в минуту для активного символа)
	estimatedTrades := int(period.Minutes() * 60)
	if estimatedTrades > 200 {
		estimatedTrades = 200 // Максимальный лимит API
	}
	if estimatedTrades < 10 {
		estimatedTrades = 10 // Минимум 10 сделок
	}

	// Получаем сделки
	trades, err := c.GetRecentTrades(symbol, estimatedTrades)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades for delta calculation: %w", err)
	}

	// Фильтруем сделки по периоду
	var filteredTrades []TradeData
	var buyVolume, sellVolume float64
	var buyCount, sellCount int

	for _, trade := range trades {
		if trade.Time.After(startTime) && trade.Time.Before(endTime) {
			volume := trade.Price * trade.Size
			filteredTrades = append(filteredTrades, trade)

			if trade.Side == "Buy" {
				buyVolume += volume
				buyCount++
			} else if trade.Side == "Sell" {
				sellVolume += volume
				sellCount++
			}
		}
	}

	if len(filteredTrades) == 0 {
		logger.Warn("⚠️ Нет сделок для %s за период %v", symbol, period)
		// Возвращаем нулевую дельту вместо ошибки
		return &VolumeDelta{
			Symbol:       symbol,
			Period:       period.String(),
			StartTime:    startTime,
			EndTime:      endTime,
			BuyVolume:    0,
			SellVolume:   0,
			Delta:        0,
			DeltaPercent: 0,
			TotalTrades:  0,
			UpdateTime:   time.Now(),
		}, nil
	}

	// Рассчитываем дельту
	delta := buyVolume - sellVolume

	// Рассчитываем процент дельты (процентное отношение дельты к общему объему)
	totalVolume := buyVolume + sellVolume
	deltaPercent := 0.0
	if totalVolume > 0 {
		deltaPercent = (delta / totalVolume) * 100
	}

	logger.Debug("📈 Дельта объемов %s:", symbol)
	logger.Debug("   Период: %v - %v", startTime.Format("15:04:05"), endTime.Format("15:04:05"))
	logger.Debug("   Сделки: %d (Buy: %d, Sell: %d)", len(filteredTrades), buyCount, sellCount)
	logger.Debug("   Объемы: Buy $%.0f, Sell $%.0f", buyVolume, sellVolume)
	logger.Debug("   Дельта: $%.0f (%.2f%%)", delta, deltaPercent)

	return &VolumeDelta{
		Symbol:       symbol,
		Period:       period.String(),
		StartTime:    startTime,
		EndTime:      endTime,
		BuyVolume:    buyVolume,
		SellVolume:   sellVolume,
		Delta:        delta,
		DeltaPercent: deltaPercent,
		TotalTrades:  len(filteredTrades),
		UpdateTime:   time.Now(),
	}, nil
}

// GetVolumeDelta получает дельту объемов для символа (с кэшированием)
func (c *BybitClient) GetVolumeDelta(symbol string, period time.Duration) (*VolumeDelta, error) {
	// В реальной реализации можно добавить кэширование
	return c.CalculateVolumeDelta(symbol, period)
}

// GetRealTimeVolumeDelta получает дельту объемов за последние 5 минут
func (c *BybitClient) GetRealTimeVolumeDelta(symbol string) (*VolumeDelta, error) {
	return c.CalculateVolumeDelta(symbol, 5*time.Minute)
}
