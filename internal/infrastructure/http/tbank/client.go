// internal/infrastructure/http/tbank/client.go
package tbank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const apiBaseURL = "https://securepay.tinkoff.ru/v2"

// Client HTTP-клиент для работы с API Т-Банк Эквайринг
type Client struct {
	terminalKey string
	password    string
	httpClient  *http.Client
}

// NewClient создает новый клиент Т-Банк
func NewClient(terminalKey, password string) *Client {
	return &Client{
		terminalKey: terminalKey,
		password:    password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TerminalKey возвращает TerminalKey клиента
func (c *Client) TerminalKey() string {
	return c.terminalKey
}

// Init инициирует платёж и возвращает PaymentURL для редиректа
func (c *Client) Init(ctx context.Context, req InitRequest) (*InitResponse, error) {
	req.TerminalKey = c.terminalKey

	// Параметры для подписи (только корневые поля, без вложенных объектов)
	params := map[string]string{
		"TerminalKey": req.TerminalKey,
		"Amount":      strconv.FormatInt(req.Amount, 10),
		"OrderId":     req.OrderId,
	}
	if req.Description != "" {
		params["Description"] = req.Description
	}
	if req.NotificationURL != "" {
		params["NotificationURL"] = req.NotificationURL
	}
	if req.SuccessURL != "" {
		params["SuccessURL"] = req.SuccessURL
	}
	if req.FailURL != "" {
		params["FailURL"] = req.FailURL
	}
	if req.PayType != "" {
		params["PayType"] = req.PayType
	}
	if req.Language != "" {
		params["Language"] = req.Language
	}
	if req.RedirectDueDate != "" {
		params["RedirectDueDate"] = req.RedirectDueDate
	}

	req.Token = GenerateToken(params, c.password)

	var resp InitResponse
	if err := c.post(ctx, "/Init", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// post выполняет POST-запрос к API Т-Банк
func (c *Client) post(ctx context.Context, path string, reqBody interface{}, resp interface{}) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBaseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ошибка HTTP запроса к Т-Банк %s: %w", path, err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if err := json.Unmarshal(respBody, resp); err != nil {
		return fmt.Errorf("ошибка десериализации ответа %s: %w (тело: %s)", path, err, string(respBody))
	}
	return nil
}
