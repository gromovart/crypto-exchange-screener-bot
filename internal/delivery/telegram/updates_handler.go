// internal/delivery/telegram/updates_handler.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// UpdatesHandler - обработчик обновлений (поддерживает и webhook, и polling)
type UpdatesHandler struct {
	config        *config.Config
	bot           *TelegramBot
	pollingActive bool
	lastUpdateID  int64
	httpClient    *http.Client
	authHandlers  *AuthHandlers // НОВОЕ: обработчики авторизации
}

// NewUpdatesHandler создает новый обработчик обновлений
func NewUpdatesHandler(cfg *config.Config, bot *TelegramBot) *UpdatesHandler {
	return &UpdatesHandler{
		config:       cfg,
		bot:          bot,
		lastUpdateID: 0,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authHandlers: nil, // Без авторизации
	}
}

// NewUpdatesHandlerWithAuth создает обработчик обновлений с поддержкой авторизации
func NewUpdatesHandlerWithAuth(cfg *config.Config, bot *TelegramBot, authHandlers *AuthHandlers) *UpdatesHandler {
	return &UpdatesHandler{
		config:       cfg,
		bot:          bot,
		lastUpdateID: 0,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authHandlers: authHandlers, // С авторизацией
	}
}

// Start запускает обработчик обновлений
func (uh *UpdatesHandler) Start() error {
	if uh.config.HTTPEnabled && uh.config.HTTPPort > 0 {
		// Запуск в режиме webhook
		return uh.startWebhook()
	} else {
		// Запуск в режиме polling
		return uh.startPolling()
	}
}

// startWebhook запускает webhook сервер
func (uh *UpdatesHandler) startWebhook() error {
	log.Println("🌐 Запуск в режиме Webhook...")

	// Создаем webhook сервер
	webhookServer := NewWebhookServer(uh.config, uh.bot)

	// Настраиваем webhook в Telegram
	if err := uh.setWebhook(); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	// Запускаем сервер
	return webhookServer.Start()
}

// startPolling запускает polling (опрос) обновлений
func (uh *UpdatesHandler) startPolling() error {
	log.Println("🔄 Запуск в режиме Polling (локальная разработка)...")

	// Удаляем webhook если был установлен
	if err := uh.deleteWebhook(); err != nil {
		log.Printf("⚠️ Не удалось удалить webhook: %v", err)
	}

	// 🔴 СНАЧАЛА СИНХРОННАЯ ОЧИСТКА, ПОТОМ ЗАПУСК POLLING
	log.Println("🧹 Очистка старых обновлений Telegram...")
	if err := uh.clearPendingUpdates(); err != nil {
		log.Printf("⚠️ Не удалось очистить старые обновления: %v", err)
		// Продолжаем даже при ошибке очистки
	} else {
		log.Println("✅ Старые обновления очищены")
	}

	// 🔴 УБЕДИМСЯ ЧТО offset установлен
	log.Printf("📊 Начальный offset для polling: %d", uh.lastUpdateID)

	// Запускаем polling
	uh.pollingActive = true
	go uh.pollUpdates()

	return nil
}

// clearPendingUpdates очищает все pending updates из очереди Telegram
func (uh *UpdatesHandler) clearPendingUpdates() error {
	// Устанавливаем lastUpdateID в 0 перед очисткой
	uh.lastUpdateID = 0

	// Сначала получаем обновления чтобы узнать текущий offset
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", uh.config.TelegramBotToken)

	params := map[string]interface{}{
		"offset":               0,
		"limit":                1,
		"timeout":              1,
		"allowed_updates":      []string{},
		"drop_pending_updates": false, // Сначала просто получаем
	}

	resp, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	if err != nil {
		return fmt.Errorf("failed to get updates for cleanup: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var response struct {
		OK     bool             `json:"ok"`
		Result []TelegramUpdate `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse cleanup response: %w", err)
	}

	if !response.OK {
		return fmt.Errorf("telegram API error during cleanup: %s", string(body))
	}

	// Находим максимальный update_id
	var maxUpdateID int64 = 0
	if len(response.Result) > 0 {
		for _, update := range response.Result {
			if update.UpdateID > maxUpdateID {
				maxUpdateID = update.UpdateID
			}
		}

		if maxUpdateID > 0 {
			uh.lastUpdateID = maxUpdateID + 1
			log.Printf("✅ Установлен offset после получения обновлений: %d", uh.lastUpdateID)
		}
	}

	// Теперь очищаем с drop_pending_updates
	params["offset"] = uh.lastUpdateID
	params["drop_pending_updates"] = true

	resp2, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	if err != nil {
		return fmt.Errorf("failed to clear pending updates: %w", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	log.Printf("🧹 Telegram API очистка: %s", string(body2))

	return nil
}

// Stop останавливает обработчик
func (uh *UpdatesHandler) Stop() error {
	uh.pollingActive = false
	return nil
}

// pollUpdates опрашивает Telegram API на наличие обновлений
func (uh *UpdatesHandler) pollUpdates() {
	log.Println("🔄 Начало polling обновлений...")

	// Интервал опроса
	pollInterval := 1 * time.Second

	for uh.pollingActive {
		updates, err := uh.getUpdates()
		if err != nil {
			log.Printf("❌ Ошибка получения обновлений: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		// Обрабатываем полученные обновления
		for _, update := range updates {
			uh.processUpdate(update)
			uh.lastUpdateID = update.UpdateID + 1
		}

		// Ждем перед следующим опросом
		time.Sleep(pollInterval)
	}

	log.Println("🛑 Polling остановлен")
}

// getUpdates получает обновления от Telegram API
func (uh *UpdatesHandler) getUpdates() ([]TelegramUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", uh.config.TelegramBotToken)

	log.Printf("🔄 Запрос обновлений с offset: %d", uh.lastUpdateID)

	// Параметры запроса
	params := map[string]interface{}{
		"offset":  uh.lastUpdateID,
		"timeout": 30,
		"limit":   100,
	}

	// Отправляем запрос
	resp, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	if err != nil {
		log.Printf("❌ Ошибка запроса к Telegram API: %v", err)
		return nil, fmt.Errorf("failed to get updates: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Ошибка чтения ответа: %v", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("📥 Ответ от Telegram API (первые 200 символов): %s", string(body[:min(200, len(body))]))

	// Парсим ответ
	var response struct {
		OK     bool             `json:"ok"`
		Result []TelegramUpdate `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("❌ Ошибка парсинга JSON: %v", err)
		log.Printf("📄 Полный ответ: %s", string(body))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		log.Printf("❌ Telegram API вернул ошибку: %s", string(body))
		return nil, fmt.Errorf("telegram API error: %s", string(body))
	}

	log.Printf("✅ Получено обновлений: %d", len(response.Result))
	return response.Result, nil
}

// isOldUpdate проверяет, старое ли обновление
func (uh *UpdatesHandler) isOldUpdate(update TelegramUpdate) bool {
	var messageTime int64

	if update.Message != nil && update.Message.Date > 0 {
		messageTime = update.Message.Date
	} else if update.CallbackQuery != nil && update.CallbackQuery.Message.Date > 0 {
		messageTime = update.CallbackQuery.Message.Date
	} else {
		return false // Не можем определить - обрабатываем
	}

	// Telegram time - Unix timestamp в секундах
	messageTimestamp := time.Unix(messageTime, 0)
	age := time.Since(messageTimestamp)

	// Игнорируем сообщения старше 5 минут
	return age > 5*time.Minute
}

// processUpdate обрабатывает одно обновление
func (uh *UpdatesHandler) processUpdate(update TelegramUpdate) {
	log.Printf("📨 Получено обновление ID: %d", update.UpdateID)

	// 🔴 ПРОВЕРКА: Игнорируем старые сообщения (старше 5 минут)
	if uh.isOldUpdate(update) {
		log.Printf("⏰ Пропускаем старое обновление ID %d (старше 5 минут)", update.UpdateID)
		return
	}

	// Обработка сообщений
	if update.Message != nil && update.Message.Text != "" {
		chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
		text := strings.TrimSpace(update.Message.Text)

		log.Printf("💬 Сообщение от %s: '%s'", chatID, text)

		// Специальная отладка для кнопки "Настройки"
		if text == "⚙️ Настройки" {
			log.Printf("🎯 ОБНАРУЖЕНА КНОПКА 'Настройки'")
			log.Printf("🔍 Сравнение: получено='%s' (байты: %v)", text, []byte(text))
		}

		if strings.HasPrefix(text, "/") {
			// Обработка команд - передаем указатель на update
			uh.handleCommand(text, &update)
		} else {
			// Обработка нажатий кнопок меню
			log.Printf("🔄 Передача в бота: '%s'", text)
			if err := uh.bot.HandleMessage(text, chatID); err != nil {
				log.Printf("❌ Ошибка обработки: %v", err)
			}
		}
	}

	// Обработка callback от inline кнопок
	if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
		chatID := fmt.Sprintf("%d", update.CallbackQuery.Message.Chat.ID)
		callbackData := update.CallbackQuery.Data

		log.Printf("🔄 Callback от chat %s: %s", chatID, callbackData)

		if err := uh.bot.HandleCallback(callbackData, chatID); err != nil {
			log.Printf("❌ Ошибка обработки callback: %v", err)
		}

		// Отвечаем на callback
		uh.answerCallbackQuery(update.CallbackQuery.ID)
	}
}

// handleCommand обрабатывает команды
func (uh *UpdatesHandler) handleCommand(command string, update *TelegramUpdate) {
	log.Printf("⚡ Обработка команды: %s", command)

	// Проверяем, является ли команда командой авторизации
	if uh.isAuthCommand(command) {
		uh.handleAuthCommand(command, update)
		return
	}

	chatID := fmt.Sprintf("%d", uh.getChatIDFromUpdate(update))

	switch command {
	case "/start":
		if command == "/start" {
			if uh.authHandlers != nil && uh.authHandlers.GetAuthMiddleware() != nil {
				// Используем auth middleware для обработки /start
				authMiddleware := uh.authHandlers.GetAuthMiddleware()
				handler := authMiddleware.WithUserContext("start", uh.authHandlers.handleStart)
				if handler != nil {
					if err := handler(update); err != nil {
						log.Printf("❌ Ошибка обработки /start: %v", err)
					}
					return
				}
			} else {
				// Fallback на старую логику
				if err := uh.bot.StartCommandHandler(chatID); err != nil {
					log.Printf("❌ Ошибка обработки /start: %v", err)
				}
				return
			}
		}

		// Проверяем, является ли команда командой авторизации
		if uh.isAuthCommand(command) {
			uh.handleAuthCommand(command, update)
			return
		}

	case "/status":
		if err := uh.bot.SendMessage("📊 Система работает"); err != nil {
			log.Printf("❌ Ошибка отправки статуса: %v", err)
		}
	case "/menu":
		if err := uh.bot.SendMessage("🔘 Меню активировано"); err != nil {
			log.Printf("❌ Ошибка отправки меню: %v", err)
		}
	case "/test":
		if err := uh.bot.SendTestMessage(); err != nil {
			log.Printf("❌ Ошибка отправки тестового сообщения: %v", err)
		}
	default:
		if err := uh.bot.SendMessage(fmt.Sprintf("❓ Неизвестная команда: %s. Используйте /start", command)); err != nil {
			log.Printf("❌ Ошибка отправки ответа: %v", err)
		}
	}
}

// handleAuthCommand обрабатывает команды авторизации
func (uh *UpdatesHandler) handleAuthCommand(command string, update *TelegramUpdate) {
	log.Printf("🔐 Обработка команды авторизации: %s", command)

	// Проверяем, настроены ли обработчики авторизации
	if uh.authHandlers == nil {
		log.Println("⚠️ Обработчики авторизации не настроены")
		// Отправляем сообщение через бота
		uh.bot.SendMessage("🔐 Система авторизации не настроена")
		return
	}

	// Получаем middleware авторизации
	authMiddleware := uh.authHandlers.GetAuthMiddleware()
	if authMiddleware == nil {
		log.Println("⚠️ Middleware авторизации не доступен")
		uh.bot.SendMessage("🔐 Middleware авторизации не доступен")
		return
	}

	// Обрабатываем команду через соответствующий обработчик
	var handler func(update *TelegramUpdate) error

	switch command {
	case "/profile":
		handler = authMiddleware.WithUserContext("profile", uh.authHandlers.handleProfile)
	case "/settings":
		handler = authMiddleware.WithUserContext("settings", uh.authHandlers.handleSettings)
	case "/notifications":
		handler = authMiddleware.WithUserContext("notifications", uh.authHandlers.handleNotifications)
	case "/thresholds":
		handler = authMiddleware.WithUserContext("thresholds", uh.authHandlers.handleThresholds)
	case "/periods":
		handler = authMiddleware.WithUserContext("periods", uh.authHandlers.handlePeriods)
	case "/language":
		handler = authMiddleware.WithUserContext("language", uh.authHandlers.handleLanguage)
	case "/help":
		handler = authMiddleware.WithUserContext("help", uh.authHandlers.handleHelp)
	case "/premium":
		handler = authMiddleware.WithPremiumContext("premium", uh.authHandlers.handlePremium)
	case "/advanced":
		handler = authMiddleware.WithPremiumContext("advanced", uh.authHandlers.handleAdvanced)
	case "/admin":
		handler = authMiddleware.WithAdminContext("admin", uh.authHandlers.handleAdmin)
	case "/stats":
		handler = authMiddleware.WithAdminContext("stats", uh.authHandlers.handleStats)
	case "/users":
		handler = authMiddleware.WithAdminContext("users", uh.authHandlers.handleUsers)
	case "/login":
		handler = authMiddleware.WithUserContext("login", uh.authHandlers.handleLogin)
	case "/logout":
		handler = authMiddleware.WithUserContext("logout", uh.authHandlers.handleLogout)
	default:
		log.Printf("❓ Неизвестная команда авторизации: %s", command)
		uh.bot.SendMessage(fmt.Sprintf("❓ Неизвестная команда авторизации: %s", command))
		return
	}

	// Вызываем обработчик
	if handler != nil {
		if err := handler(update); err != nil {
			log.Printf("❌ Ошибка обработки команды %s: %v", command, err)
		}
	}
}

// getChatIDFromUpdate вспомогательный метод для получения ChatID из обновления
func (uh *UpdatesHandler) getChatIDFromUpdate(update *TelegramUpdate) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID
	}
	return 0
}

// isAuthCommand проверяет, является ли команда командой авторизации
func (uh *UpdatesHandler) isAuthCommand(command string) bool {
	authCommands := []string{
		"/profile",
		"/settings",
		"/notifications",
		"/thresholds",
		"/periods",
		"/language",
		"/premium",
		"/advanced",
		"/admin",
		"/stats",
		"/users",
		"/login",
		"/logout",
		"/help",
	}

	for _, cmd := range authCommands {
		if strings.HasPrefix(command, cmd) {
			return true
		}
	}

	return false
}

// SetAuthHandlers устанавливает обработчики авторизации
func (uh *UpdatesHandler) SetAuthHandlers(authHandlers *AuthHandlers) {
	uh.authHandlers = authHandlers
	log.Println("🔐 Обработчики авторизации установлены для UpdatesHandler")
}

// GetAuthHandlers возвращает обработчики авторизации
func (uh *UpdatesHandler) GetAuthHandlers() *AuthHandlers {
	return uh.authHandlers
}

// HasAuth возвращает true, если авторизация настроена
func (uh *UpdatesHandler) HasAuth() bool {
	return uh.authHandlers != nil
}

// answerCallbackQuery отвечает на callback запрос
func (uh *UpdatesHandler) answerCallbackQuery(callbackID string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", uh.config.TelegramBotToken)

	params := map[string]interface{}{
		"callback_query_id": callbackID,
		"text":              "✅",
		"show_alert":        false,
	}

	_, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	return err
}

// setWebhook настраивает webhook в Telegram
func (uh *UpdatesHandler) setWebhook() error {
	if !uh.config.HTTPEnabled || uh.config.HTTPPort == 0 {
		return fmt.Errorf("HTTP не включен или порт не указан")
	}

	// URL для webhook (нужен публичный URL)
	webhookURL := fmt.Sprintf("https://your-public-url.com:%d/webhook", uh.config.HTTPPort)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", uh.config.TelegramBotToken)

	params := map[string]interface{}{
		"url":             webhookURL,
		"max_connections": 40,
		"allowed_updates": []string{"message", "callback_query"},
	}

	resp, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("🌐 Webhook установлен: %s", string(body))

	return nil
}

// deleteWebhook удаляет webhook
func (uh *UpdatesHandler) deleteWebhook() error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteWebhook", uh.config.TelegramBotToken)

	params := map[string]interface{}{
		"drop_pending_updates": true,
	}

	resp, err := uh.httpClient.Post(url, "application/json", toJSONReader(params))
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("🗑️ Webhook удален: %s", string(body))

	return nil
}

// toJSONReader конвертирует map в io.Reader с JSON
func toJSONReader(data interface{}) io.Reader {
	jsonData, _ := json.Marshal(data)
	return strings.NewReader(string(jsonData))
}
