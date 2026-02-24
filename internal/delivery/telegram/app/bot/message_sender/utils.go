// internal/delivery/telegram/app/bot/message_sender/utils.go
package message_sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

// sendTelegramRequestGetMsgID отправляет запрос к Telegram API и возвращает message_id ответа
func (ms *MessageSenderImpl) sendTelegramRequestGetMsgID(method string, request map[string]interface{}) (int64, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := ms.baseURL + method
	resp, err := ms.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to send request to %s: %w", method, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var telegramResp struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description,omitempty"`
		Result      struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if !telegramResp.OK {
		if telegramResp.ErrorCode == 429 {
			retryAfter := 5
			var retryResp struct {
				Parameters struct {
					RetryAfter int `json:"retry_after"`
				} `json:"parameters"`
			}
			if json.Unmarshal(body, &retryResp) == nil && retryResp.Parameters.RetryAfter > 0 {
				retryAfter = retryResp.Parameters.RetryAfter
			}
			log.Printf("⚠️ Telegram API rate limit, waiting %d seconds", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
			return ms.sendTelegramRequestGetMsgID(method, request)
		}
		return 0, fmt.Errorf("telegram API error %d: %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	return telegramResp.Result.MessageID, nil
}

// sendTelegramRequest отправляет запрос к Telegram API
func (ms *MessageSenderImpl) sendTelegramRequest(method string, request map[string]interface{}) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := ms.baseURL + method
	resp, err := ms.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", method, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var telegramResp struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description,omitempty"`
	}

	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !telegramResp.OK {
		// Обработка ошибки 429 (Too Many Requests)
		if telegramResp.ErrorCode == 429 {
			retryAfter := 5 // секунд по умолчанию
			var retryResp struct {
				Parameters struct {
					RetryAfter int `json:"retry_after"`
				} `json:"parameters"`
			}
			if json.Unmarshal(body, &retryResp) == nil && retryResp.Parameters.RetryAfter > 0 {
				retryAfter = retryResp.Parameters.RetryAfter
			}
			log.Printf("⚠️ Telegram API rate limit, waiting %d seconds", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
			// Пробуем снова
			return ms.sendTelegramRequest(method, request)
		}
		return fmt.Errorf("telegram API error %d: %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	return nil
}

// GetMessageHash создает хэш для проверки дубликатов
func GetMessageHash(chatID int64, text string, keyboard interface{}) string {
	// Простой хэш для проверки дубликатов
	keyboardStr := ""
	if keyboard != nil {
		keyboardBytes, _ := json.Marshal(keyboard)
		keyboardStr = string(keyboardBytes)
	}
	return fmt.Sprintf("%d:%s:%s", chatID, text, keyboardStr)
}

// ParseChatID преобразует строковый chat ID в int64
func ParseChatID(chatID string) int64 {
	if chatID == "" {
		return 0
	}

	// Убираем "@" если есть
	cleanID := strings.TrimPrefix(chatID, "@")

	var result int64
	fmt.Sscanf(cleanID, "%d", &result)
	return result
}

// min возвращает минимум из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// EscapeMarkdownV2 экранирует специальные символы для MarkdownV2
func EscapeMarkdownV2(text string) string {
	// Специальные символы в MarkdownV2
	specialChars := []string{
		"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!",
	}

	result := text
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}
