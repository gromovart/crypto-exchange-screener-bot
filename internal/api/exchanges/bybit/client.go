// internal/api/bybit/bybit_client.go
package bybit

import (
	"bytes"
	"crypto-exchange-screener-bot/internal/api"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// BybitClient - клиент для работы с API Bybit
type BybitClient struct {
	httpClient *http.Client
	config     *config.Config
	baseURL    string
	apiKey     string
	apiSecret  string
	category   string // "spot", "linear", "inverse"
}

// NewBybitClient создает новый клиент для работы с API Bybit
func NewBybitClient(cfg *config.Config) *BybitClient {
	// Определяем базовый URL
	baseURL := cfg.BaseURL
	apiKey := cfg.ApiKey
	apiSecret := cfg.ApiSecret

	// Определяем категорию по умолчанию (фьючерсы)
	category := CategoryLinear
	if cfg.FuturesCategory != "" {
		category = cfg.FuturesCategory
	} else {
		// Если в конфиге не указана категория, используем linear
		category = "linear"
	}

	return &BybitClient{
		httpClient: &http.Client{
			Timeout: time.Duration(30) * time.Second,
		},
		config:    cfg,
		baseURL:   baseURL,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		category:  category,
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

// GetTickers получает все тикеры для указанной категории
func (c *BybitClient) GetTickers(category string) (*api.TickerResponse, error) {
	params := url.Values{}

	// Если категория пустая, используем линейные фьючерсы
	if category == "" {
		category = "linear"
	}

	params.Set("category", category)

	body, err := c.sendPublicRequest(http.MethodGet, "/v5/market/tickers", params)
	if err != nil {
		return nil, err
	}

	var tickerResp bybitTickerResponse
	if err := json.Unmarshal(body, &tickerResp); err != nil {
		return nil, fmt.Errorf("failed to parse ticker response: %w", err)
	}

	// Преобразуем в общую структуру api.TickerResponse
	return c.convertToApiTickerResponse(&tickerResp), nil
}

// Вспомогательный метод для преобразования
func (c *BybitClient) convertToApiTickerResponse(bybitResp *bybitTickerResponse) *api.TickerResponse {
	var tickers []api.Ticker

	for _, t := range bybitResp.Result.List {
		tickers = append(tickers, api.Ticker{
			Symbol:       t.Symbol,
			LastPrice:    t.LastPrice,
			Volume24h:    t.Volume24h,
			Price24hPcnt: t.Price24hPcnt,
			Turnover24h:  t.Turnover24h,
		})
	}

	return &api.TickerResponse{
		RetCode: bybitResp.RetCode,
		RetMsg:  bybitResp.RetMsg,
		Result: api.TickerList{
			List: tickers,
		},
	}
}

// GetInstrumentsInfo получает информацию об инструментах фьючерсов
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
		return nil, err
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

// Category возвращает текущую категорию клиента
func (c *BybitClient) Category() string {
	// Возвращаем поле category клиента, а не из конфига
	if c.category != "" {
		return c.category
	}
	// Если category пустая, возвращаем linear по умолчанию
	return "linear"
}

// GetRecentKlinesForPeriod получает свечи для анализа периода роста
func (c *BybitClient) GetRecentKlinesForPeriod(symbol string, periodMinutes int) ([][]string, error) {
	// Определяем интервал свечей в зависимости от периода
	var interval string
	var limit int

	switch {
	case periodMinutes <= 5:
		interval = "1" // 1-минутные свечи
		limit = periodMinutes
	case periodMinutes <= 30:
		interval = "5" // 5-минутные свечи
		limit = periodMinutes / 5
	case periodMinutes <= 240:
		interval = "15" // 15-минутные свечи
		limit = periodMinutes / 15
	case periodMinutes <= 1440:
		interval = "60" // 1-часовые свечи
		limit = periodMinutes / 60
	default:
		interval = "D" // Дневные свечи
		limit = periodMinutes / 1440
	}

	// Минимальное количество свечей
	if limit < 2 {
		limit = 2
	}

	// Добавляем буфер на случай если некоторые данные отсутствуют
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

	// Сортируем по времени (от старых к новым)
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

// FindGrowthSignals ищет сигналы роста/падения
func (c *BybitClient) FindGrowthSignals(symbols []string, periodMinutes int,
	growthThreshold, fallThreshold float64, checkContinuity bool) ([]types.GrowthSignal, error) {

	var signals []types.GrowthSignal

	// Получаем тикеры для всех символов одним запросом
	tickerResp, err := c.GetTickers("linear")
	if err != nil {
		return nil, err
	}

	// Создаем мапу объемов для быстрого поиска
	volumeMap := make(map[string]float64)
	for _, ticker := range tickerResp.Result.List {
		volume, err := strconv.ParseFloat(ticker.Turnover24h, 64)
		if err == nil {
			volumeMap[ticker.Symbol] = volume
		}
	}

	for _, symbol := range symbols {
		// Фильтруем по объему (минимум $100,000)
		if volume, ok := volumeMap[symbol]; !ok || volume < 100000 {
			continue
		}

		analysis, err := c.AnalyzeGrowth(symbol, periodMinutes, checkContinuity)
		if err != nil {
			continue // Пропускаем символы с недостаточными данными
		}

		var signal *types.GrowthSignal

		// Проверяем рост
		if analysis.IsGrowing && analysis.GrowthPercent >= growthThreshold {
			signal = &types.GrowthSignal{
				Symbol:        symbol,
				PeriodMinutes: periodMinutes,
				GrowthPercent: analysis.GrowthPercent,
				FallPercent:   0,
				IsContinuous:  true,
				DataPoints:    len(analysis.DataPoints),
				StartPrice:    analysis.MinPrice,
				EndPrice:      analysis.MaxPrice,
				Direction:     "growth",
				Confidence:    c.calculateConfidence(analysis),
				Timestamp:     time.Now(),
			}
		}

		// Проверяем падение
		if analysis.IsFalling && analysis.FallPercent >= fallThreshold {
			signal = &types.GrowthSignal{
				Symbol:        symbol,
				PeriodMinutes: periodMinutes,
				GrowthPercent: 0,
				FallPercent:   analysis.FallPercent,
				IsContinuous:  true,
				DataPoints:    len(analysis.DataPoints),
				StartPrice:    analysis.MaxPrice,
				EndPrice:      analysis.MinPrice,
				Direction:     "fall",
				Confidence:    c.calculateConfidence(analysis),
				Timestamp:     time.Now(),
			}
		}

		if signal != nil {
			signals = append(signals, *signal)
		}
	}

	// Сортируем по проценту изменения
	sort.Slice(signals, func(i, j int) bool {
		changeI := signals[i].GrowthPercent + signals[i].FallPercent
		changeJ := signals[j].GrowthPercent + signals[j].FallPercent
		return math.Abs(changeI) > math.Abs(changeJ)
	})

	return signals, nil
}

// calculateConfidence рассчитывает уверенность в сигнале
func (c *BybitClient) calculateConfidence(analysis *types.GrowthAnalysis) float64 {
	confidence := 0.0

	// Более высокий процент изменения = более высокая уверенность
	changePercent := math.Abs(analysis.GrowthPercent)
	confidence += math.Min(changePercent*2, 40) // Максимум 40% за изменение

	// Непрерывность добавляет уверенности
	if (analysis.IsGrowing && analysis.GrowthPercent > 0) ||
		(analysis.IsFalling && analysis.FallPercent > 0) {
		confidence += 30
	}

	// Объем добавляет уверенности
	avgVolume := 0.0
	for _, point := range analysis.DataPoints {
		avgVolume += point.Volume
	}
	avgVolume /= float64(len(analysis.DataPoints))

	if avgVolume > 1000000 { // > $1M объема
		confidence += 15
	} else if avgVolume > 100000 { // > $100K объема
		confidence += 10
	} else {
		confidence += 5
	}

	// Количество точек данных
	dataPointConfidence := float64(len(analysis.DataPoints)) * 1.5
	confidence += math.Min(dataPointConfidence, 15)

	return math.Min(confidence, 100.0)
}

// GetKlineDataWithInterval получает свечные данные с указанным интервалом
func (c *BybitClient) GetKlineDataWithInterval(symbol, category, interval string, limit int) (*KlineResponse, error) {
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

// GetSymbolVolume получает объем для нескольких символов за один запрос
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

func NewBybitClientSimple(apiKey, apiSecret, baseURL, category string, timeout time.Duration) *BybitClient {
	return &BybitClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		category:  category,
	}
}
