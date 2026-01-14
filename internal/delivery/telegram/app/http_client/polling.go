// internal/delivery/telegram/app/http_client/polling.go
package http_client

import (
	"net/http"
	"strconv"
	"time"
)

// PollingClient клиент для polling запросов с увеличенным таймаутом
type PollingClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewPollingClient создает новый клиент для polling
func NewPollingClient(baseURL string) *PollingClient {
	return &PollingClient{
		httpClient: &http.Client{
			Timeout: 35 * time.Second, // Больше чем timeout=30 в Telegram long-polling
		},
		baseURL: baseURL,
	}
}

// GetUpdates выполняет GET запрос для получения обновлений
func (c *PollingClient) GetUpdates(offset int, timeout int) (*http.Response, error) {
	url := c.baseURL + "getUpdates"
	fullURL := url + "?offset=" + strconv.Itoa(offset) + "&timeout=" + strconv.Itoa(timeout)
	return c.httpClient.Get(fullURL)
}

// SetTimeout устанавливает таймаут для клиента
func (c *PollingClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// GetHTTPClient возвращает HTTP клиент
func (c *PollingClient) GetHTTPClient() *http.Client {
	return c.httpClient
}
