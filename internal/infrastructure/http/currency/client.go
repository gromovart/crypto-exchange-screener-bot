// internal/infrastructure/http/currency/client.go
package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"
)

const (
	cbrURL      = "https://www.cbr-xml-daily.ru/daily_json.js"
	cacheTTL    = 1 * time.Hour
	FallbackRate = 100.0 // резервный курс если ЦБ недоступен
)

type cbrResponse struct {
	Valute map[string]struct {
		Value   float64 `json:"Value"`
		Nominal int     `json:"Nominal"`
	} `json:"Valute"`
}

// Client получает и кэширует курс USD/RUB от ЦБ РФ
type Client struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cachedRate float64
	cachedAt   time.Time
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		cachedRate: FallbackRate,
	}
}

// GetUSDRUB возвращает текущий курс USD/RUB с кэшем на 1 час
func (c *Client) GetUSDRUB(ctx context.Context) float64 {
	c.mu.RLock()
	if c.cachedRate > 0 && time.Since(c.cachedAt) < cacheTTL {
		rate := c.cachedRate
		c.mu.RUnlock()
		return rate
	}
	c.mu.RUnlock()

	rate, err := c.fetchUSDRUB(ctx)
	if err != nil {
		logger.Warn("⚠️ Не удалось получить курс USD/RUB от ЦБ РФ: %v, используем резервный %.2f", err, FallbackRate)
		return FallbackRate
	}

	c.mu.Lock()
	c.cachedRate = rate
	c.cachedAt = time.Now()
	c.mu.Unlock()

	logger.Info("💱 Курс USD/RUB обновлён: %.2f ₽", rate)
	return rate
}

func (c *Client) fetchUSDRUB(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cbrURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("ошибка запроса к ЦБ РФ: %w", err)
	}
	defer resp.Body.Close()

	var data cbrResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("ошибка разбора ответа ЦБ РФ: %w", err)
	}

	usd, ok := data.Valute["USD"]
	if !ok {
		return 0, fmt.Errorf("USD не найден в ответе ЦБ РФ")
	}
	if usd.Nominal == 0 {
		return 0, fmt.Errorf("номинал USD равен нулю")
	}

	return usd.Value / float64(usd.Nominal), nil
}
