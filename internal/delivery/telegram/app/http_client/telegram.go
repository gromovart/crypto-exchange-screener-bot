// internal/delivery/telegram/app/http_client/telegram.go
package http_client

import (
	"bytes"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/pkg/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// SetMyCommands устанавливает меню команд бота
func (c *TelegramClient) SetMyCommands(commands []telegram.BotCommand) error {
	endpoint := "setMyCommands"

	// Проверяем валидность команд
	for _, cmd := range commands {
		if err := validateBotCommand(cmd); err != nil {
			return fmt.Errorf("невалидная команда '%s': %v", cmd.Command, err)
		}
	}

	params := map[string]interface{}{
		"commands":      commands,
		"language_code": "ru", // Русский язык
	}

	logger.Info("📋 Устанавливаем меню команд: %d команд", len(commands))

	// Отправляем запрос
	var response telegram.SetMyCommandsResponse
	if err := c.makeRequest(endpoint, params, &response); err != nil {
		return fmt.Errorf("ошибка установки команд: %v", err)
	}

	if !response.OK {
		return fmt.Errorf("telegram API ошибка: %s", response.Description)
	}

	logger.Info("✅ Меню команд успешно установлено (%d команд)", len(commands))

	// Логируем список команд для отладки (только на уровне debug)
	for _, cmd := range commands {
		logger.Debug("   • %s - %s", cmd.Command, cmd.Description)
	}

	return nil
}

// validateBotCommand проверяет валидность команды бота
func validateBotCommand(cmd telegram.BotCommand) error {
	// Проверяем команду
	if cmd.Command == "" {
		return fmt.Errorf("команда не может быть пустой")
	}

	// Команда должна начинаться с /
	if !strings.HasPrefix(cmd.Command, "/") {
		return fmt.Errorf("команда должна начинаться с '/'")
	}

	// Проверяем длину команды (без /)
	commandBody := cmd.Command[1:]
	if len(commandBody) < 1 || len(commandBody) > 32 {
		return fmt.Errorf("команда должна быть от 1 до 32 символов (без /)")
	}

	// Проверяем допустимые символы
	for _, ch := range commandBody {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_') {
			return fmt.Errorf("команда может содержать только латинские буквы, цифры и подчеркивания")
		}
	}

	// Проверяем описание
	if cmd.Description == "" || len(cmd.Description) > 256 {
		return fmt.Errorf("описание должно быть от 1 до 256 символов")
	}

	return nil
}

// makeRequest отправляет запрос к Telegram API
func (c *TelegramClient) makeRequest(endpoint string, params map[string]interface{}, result interface{}) error {
	// Преобразуем параметры в JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		logger.Error("Ошибка сериализации JSON для %s: %v", endpoint, err)
		return fmt.Errorf("ошибка сериализации JSON: %v", err)
	}

	// Отправляем запрос
	resp, err := c.SendMessage(endpoint, jsonData)
	if err != nil {
		logger.Error("Ошибка отправки запроса %s: %v", endpoint, err)
		return fmt.Errorf("ошибка отправки запроса: %v", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Ошибка чтения ответа от %s: %v", endpoint, err)
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	// Парсим JSON
	if err := json.Unmarshal(body, result); err != nil {
		logger.Error("Ошибка парсинга JSON от %s: %v", endpoint, err)
		return fmt.Errorf("ошибка парсинга JSON: %s, ошибка: %v", string(body), err)
	}

	logger.Debug("Запрос %s успешен", endpoint)
	return nil
}
