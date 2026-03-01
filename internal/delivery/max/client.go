// internal/delivery/max/client.go
package max

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	maxBaseURL    = "https://botapi.max.ru"
	maxTimeout    = 10 * time.Second
	maxLongPollTO = 35 * time.Second
)

// Client HTTP-клиент для MAX Bot API
type Client struct {
	token          string
	httpClient     *http.Client // для обычных запросов
	pollingClient  *http.Client // для long-polling (увеличенный timeout)
}

// NewClient создаёт новый MAX Bot API клиент
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

// url строит URL метода
func (c *Client) url(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", maxBaseURL, c.token, method)
}

// do выполняет POST-запрос с JSON-телом
func (c *Client) do(method string, payload interface{}) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("max %s: marshal: %w", method, err)
	}

	req, err := http.NewRequest(http.MethodPost, c.url(method), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("max %s: new request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("max %s: http: %w", method, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("max %s: read response: %w", method, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("max %s: status %d: %s", method, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// checkOK разбирает базовый ответ и возвращает ошибку при !ok
func checkOK(data []byte, method string) error {
	var r apiResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return fmt.Errorf("max %s: unmarshal: %w", method, err)
	}
	if !r.OK {
		return fmt.Errorf("max %s: api error: %s", method, r.Description)
	}
	return nil
}

// ───────────────────────────────────────
// SendMessage — отправка текстового сообщения (без клавиатуры)
// ───────────────────────────────────────

func (c *Client) SendMessage(chatID int64, text string) error {
	_, err := c.SendMessageGetID(chatID, text, nil)
	return err
}

// ───────────────────────────────────────
// SendMessageWithKeyboard — отправка с inline-клавиатурой
// ───────────────────────────────────────

func (c *Client) SendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) error {
	_, err := c.SendMessageGetID(chatID, text, keyboard)
	return err
}

// SendMessageGetID отправляет сообщение и возвращает message_id
func (c *Client) SendMessageGetID(chatID int64, text string, keyboard interface{}) (int64, error) {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	if keyboard != nil {
		payload["reply_markup"] = keyboard
	}

	data, err := c.do("sendMessage", payload)
	if err != nil {
		return 0, err
	}

	var r sendMessageResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return 0, fmt.Errorf("max sendMessage: unmarshal: %w", err)
	}
	if !r.OK {
		return 0, fmt.Errorf("max sendMessage: api error: %s", r.Description)
	}
	if r.Result != nil {
		return r.Result.MessageID, nil
	}
	return 0, nil
}

// ───────────────────────────────────────
// EditMessageText — редактирование сообщения
// ───────────────────────────────────────

func (c *Client) EditMessageText(chatID, messageID int64, text string, keyboard interface{}) error {
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}
	if keyboard != nil {
		payload["reply_markup"] = keyboard
	}

	data, err := c.do("editMessageText", payload)
	if err != nil {
		return err
	}
	return checkOK(data, "editMessageText")
}

// ───────────────────────────────────────
// DeleteMessage — удаление сообщения
// ───────────────────────────────────────

func (c *Client) DeleteMessage(chatID, messageID int64) error {
	data, err := c.do("deleteMessage", map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	})
	if err != nil {
		return err
	}
	return checkOK(data, "deleteMessage")
}

// ───────────────────────────────────────
// AnswerCallbackQuery — ответ на callback от кнопки
// ───────────────────────────────────────

func (c *Client) AnswerCallbackQuery(callbackQueryID, text string, showAlert bool) error {
	payload := map[string]interface{}{
		"callback_query_id": callbackQueryID,
		"show_alert":        showAlert,
	}
	if text != "" {
		payload["text"] = text
	}

	data, err := c.do("answerCallbackQuery", payload)
	if err != nil {
		return err
	}
	return checkOK(data, "answerCallbackQuery")
}

// ───────────────────────────────────────
// SetMyCommands — установка команд бота в меню
// ───────────────────────────────────────

func (c *Client) SetMyCommands(commands []BotCommand) error {
	data, err := c.do("setMyCommands", map[string]interface{}{
		"commands": commands,
	})
	if err != nil {
		return err
	}
	return checkOK(data, "setMyCommands")
}

// ───────────────────────────────────────
// GetUpdates — long-polling получение обновлений
// ───────────────────────────────────────

func (c *Client) GetUpdates(offset int64, timeoutSec int) ([]Update, error) {
	url := c.url("getUpdates") +
		"?offset=" + strconv.FormatInt(offset, 10) +
		"&timeout=" + strconv.Itoa(timeoutSec)

	resp, err := c.pollingClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("max getUpdates: http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("max getUpdates: read: %w", err)
	}

	var r getUpdatesResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("max getUpdates: unmarshal: %w", err)
	}
	if !r.OK {
		return nil, fmt.Errorf("max getUpdates: api error")
	}
	return r.Result, nil
}
