// internal/delivery/telegram/app/http_client/telegram.go
package http_client

import (
	"bytes"
	"net/http"
	"time"
)

// TelegramClient клиент для работы с Telegram API
type TelegramClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewTelegramClient создает новый клиент Telegram
func NewTelegramClient(baseURL string) *TelegramClient {
	return &TelegramClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// SendMessage отправляет сообщение через Telegram API
func (c *TelegramClient) SendMessage(endpoint string, payload []byte) (*http.Response, error) {
	url := c.baseURL + endpoint
	return c.httpClient.Post(url, "application/json", bytes.NewBuffer(payload))
}

// Get выполняет GET запрос
func (c *TelegramClient) Get(endpoint string) (*http.Response, error) {
	url := c.baseURL + endpoint
	return c.httpClient.Get(url)
}

// SetTimeout устанавливает таймаут для клиента
func (c *TelegramClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// GetBaseURL возвращает базовый URL
func (c *TelegramClient) GetBaseURL() string {
	return c.baseURL
}

// GetHTTPClient возвращает HTTP клиент
func (c *TelegramClient) GetHTTPClient() *http.Client {
	return c.httpClient
}
