// internal/delivery/max/client.go
package max

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// stripMarkdown убирает *bold* → bold, `code` → code
// чтобы Telegram-разметка не отображалась как символы в MAX
var reBold = regexp.MustCompile(`\*([^*]+)\*`)
var reCode = regexp.MustCompile("`([^`]+)`")

func stripMarkdown(s string) string {
	s = reBold.ReplaceAllString(s, "$1")
	s = reCode.ReplaceAllString(s, "$1")
	return s
}

const (
	maxBaseURL    = "https://platform-api.max.ru"
	maxTimeout    = 10 * time.Second
	maxLongPollTO = 40 * time.Second
)

// Client — HTTP-клиент для MAX API (platform-api.max.ru)
type Client struct {
	token         string
	httpClient    *http.Client // для обычных запросов
	pollingClient *http.Client // для long-polling
}

// NewClient создаёт новый MAX API клиент
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: maxTimeout,
		},
		pollingClient: &http.Client{
			Timeout: maxLongPollTO,
		},
	}
}

// addAuth добавляет заголовок Authorization
func (c *Client) addAuth(req *http.Request) {
	req.Header.Set("Authorization", c.token)
}

// doPost выполняет POST-запрос к path с JSON-телом
func (c *Client) doPost(path string, payload interface{}) ([]byte, error) {
	return c.doJSON(http.MethodPost, path, payload)
}

// doPut выполняет PUT-запрос к path с JSON-телом
func (c *Client) doPut(path string, payload interface{}) ([]byte, error) {
	return c.doJSON(http.MethodPut, path, payload)
}

// doPatch выполняет PATCH-запрос к path с JSON-телом
func (c *Client) doPatch(path string, payload interface{}) ([]byte, error) {
	return c.doJSON(http.MethodPatch, path, payload)
}

// doDelete выполняет DELETE-запрос к path
func (c *Client) doDelete(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodDelete, maxBaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("max DELETE %s: %w", path, err)
	}
	c.addAuth(req)
	return c.exec(req, c.httpClient, "DELETE "+path)
}

// doJSON кодирует payload в JSON и выполняет HTTP-запрос
func (c *Client) doJSON(method, path string, payload interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("max %s %s: marshal: %w", method, path, err)
	}
	req, err := http.NewRequest(method, maxBaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("max %s %s: new request: %w", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)
	return c.exec(req, c.httpClient, method+" "+path)
}

// exec выполняет запрос и возвращает тело ответа (2xx) или ошибку
func (c *Client) exec(req *http.Request, hc *http.Client, label string) ([]byte, error) {
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("max %s: http: %w", label, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("max %s: read: %w", label, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("max %s: status %d: %s", label, resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// ───────────────────────────────────────────────
// SendMessage — отправка текстового сообщения
// ───────────────────────────────────────────────

func (c *Client) SendMessage(chatID int64, text string) error {
	_, err := c.SendMessageGetID(chatID, text, nil)
	return err
}

// SendMessageWithKeyboard — отправка сообщения с inline-клавиатурой
func (c *Client) SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error {
	_, err := c.SendMessageGetID(chatID, text, keyboard)
	return err
}

// SendMessageGetID отправляет сообщение и возвращает mid (string)
// keyboard должен быть []interface{} (attachments array) из kb.Keyboard()
func (c *Client) SendMessageGetID(chatID int64, text string, keyboard interface{}) (string, error) {
	payload := map[string]interface{}{
		"text": stripMarkdown(text),
	}
	if keyboard != nil {
		payload["attachments"] = keyboard
	}

	path := "/messages?chat_id=" + strconv.FormatInt(chatID, 10)
	data, err := c.doPost(path, payload)
	if err != nil {
		return "", err
	}

	var r sendMessageResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return "", fmt.Errorf("max SendMessage: unmarshal: %w", err)
	}
	return r.Body.Mid, nil
}

// ───────────────────────────────────────────────
// EditMessageText — редактирование сообщения
// PUT /messages?message_id=<mid>
// ───────────────────────────────────────────────

func (c *Client) EditMessageText(mid, text string, keyboard interface{}) error {
	if mid == "" {
		return fmt.Errorf("max EditMessage: пустой mid")
	}
	payload := map[string]interface{}{
		"text": stripMarkdown(text),
	}
	if keyboard != nil {
		payload["attachments"] = keyboard
	}

	path := "/messages?message_id=" + mid
	_, err := c.doPut(path, payload)
	return err
}

// ───────────────────────────────────────────────
// DeleteMessage — удаление сообщения
// DELETE /messages?message_id=<mid>
// ───────────────────────────────────────────────

func (c *Client) DeleteMessage(mid string) error {
	if mid == "" {
		return fmt.Errorf("max DeleteMessage: пустой mid")
	}
	_, err := c.doDelete("/messages?message_id=" + mid)
	return err
}

// ───────────────────────────────────────────────
// AnswerCallbackQuery — ответ на callback
// POST /answers?callback_id=<id>
// ───────────────────────────────────────────────

func (c *Client) AnswerCallbackQuery(callbackID, notification string) error {
	if callbackID == "" {
		return nil
	}
	payload := map[string]interface{}{
		"notification": notification,
	}
	path := "/answers?callback_id=" + callbackID
	_, err := c.doPost(path, payload)
	return err
}

// ───────────────────────────────────────────────

// ───────────────────────────────────────────────
// GetUpdates — long-polling
// GET /updates?timeout=N&limit=100&marker=M
// ───────────────────────────────────────────────

func (c *Client) GetUpdates(marker int64, timeoutSec int) ([]Update, int64, error) {
	url := maxBaseURL + "/updates" +
		"?timeout=" + strconv.Itoa(timeoutSec) +
		"&limit=100" +
		"&update_types=message_created,message_callback,bot_started" +
		"&marker=" + strconv.FormatInt(marker, 10)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, marker, fmt.Errorf("max getUpdates: new request: %w", err)
	}
	c.addAuth(req)

	body, err := c.exec(req, c.pollingClient, "GET /updates")
	if err != nil {
		return nil, marker, err
	}

	var r getUpdatesResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, marker, fmt.Errorf("max getUpdates: unmarshal: %w", err)
	}
	return r.Updates, r.Marker, nil
}
