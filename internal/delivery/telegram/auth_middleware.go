// internal/delivery/telegram/auth_middleware.go
package telegram

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// TelegramUserInfo - информация о пользователе из Telegram
type TelegramUserInfo struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// TelegramCallbackQuery - callback запрос
type TelegramCallbackQuery struct {
	ID      string           `json:"id"`
	From    TelegramUserInfo `json:"from"`
	Message TelegramMessage  `json:"message"`
	Data    string           `json:"data"`
}

// AuthMiddleware - middleware для проверки авторизации
type AuthMiddleware struct {
	userService *users.Service
	botToken    string
	httpClient  *http.Client
	baseURL     string
}

// NewAuthMiddleware создает новый middleware аутентификации
func NewAuthMiddleware(userService *users.Service, botToken string) *AuthMiddleware {
	return &AuthMiddleware{
		userService: userService,
		botToken:    botToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		baseURL:     fmt.Sprintf("https://api.telegram.org/bot%s/", botToken),
	}
}

// RequireAuth проверяет, авторизован ли пользователь
func (m *AuthMiddleware) RequireAuth(handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return func(update *TelegramUpdate) error {
		// Получаем информацию о пользователе из Telegram обновления
		var userID int64
		var username, firstName, lastName string
		var chatID int64

		if update.Message != nil {
			userID = update.Message.From.ID
			username = update.Message.From.Username
			firstName = update.Message.From.FirstName
			lastName = update.Message.From.LastName
			chatID = update.Message.Chat.ID
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
			username = update.CallbackQuery.From.Username
			firstName = update.CallbackQuery.From.FirstName
			lastName = update.CallbackQuery.From.LastName
			chatID = update.CallbackQuery.Message.Chat.ID
		} else {
			return fmt.Errorf("не удалось получить информацию о пользователе")
		}

		// Получаем или создаем пользователя
		user, err := m.userService.GetOrCreateUser(userID, username, firstName, lastName)
		if err != nil {
			log.Printf("❌ Ошибка получения пользователя: %v", err)
			return m.sendAuthError(chatID, "Ошибка авторизации. Попробуйте позже.")
		}

		// Проверяем активность пользователя
		if !user.IsActive {
			return m.sendAuthError(chatID, "Ваш аккаунт деактивирован. Обратитесь к администратору.")
		}

		// Добавляем ChatID если его нет
		if user.ChatID == "" {
			user.ChatID = strconv.FormatInt(chatID, 10)
			if err := m.userService.UpdateUser(user); err != nil {
				log.Printf("⚠️ Не удалось обновить ChatID: %v", err)
			}
		}

		// Вызываем обработчик с пользователем
		return handler(user, update)
	}
}

// RequireRole проверяет роль пользователя
func (m *AuthMiddleware) RequireRole(requiredRole string, handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return m.RequireAuth(func(user *models.User, update *TelegramUpdate) error {
		if !m.hasRequiredRole(user, requiredRole) {
			chatID := m.getChatID(update)
			return m.sendAuthError(chatID, fmt.Sprintf("Недостаточно прав. Требуется роль: %s", requiredRole))
		}
		return handler(user, update)
	})
}

// RequireAdmin требует роль администратора
func (m *AuthMiddleware) RequireAdmin(handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return m.RequireRole(models.RoleAdmin, handler)
}

// RequirePremium требует премиум статус
func (m *AuthMiddleware) RequirePremium(handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return m.RequireAuth(func(user *models.User, update *TelegramUpdate) error {
		if !m.isPremiumUser(user) {
			chatID := m.getChatID(update)
			return m.sendAuthError(chatID, "Эта функция доступна только премиум пользователям")
		}
		return handler(user, update)
	})
}

// CheckDailyLimit проверяет дневной лимит пользователя
func (m *AuthMiddleware) CheckDailyLimit(handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return m.RequireAuth(func(user *models.User, update *TelegramUpdate) error {
		if user.HasReachedDailyLimit() {
			chatID := m.getChatID(update)
			message := fmt.Sprintf("Вы достигли дневного лимита сигналов: %d/%d\n\nЛимит сбросится в 00:00 UTC",
				user.SignalsToday, user.MaxSignalsPerDay)
			return m.sendMessage(chatID, message, nil)
		}
		return handler(user, update)
	})
}

// WithUserContext добавляет пользователя в контекст команды
func (m *AuthMiddleware) WithUserContext(command string, handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return func(update *TelegramUpdate) error {
		return m.RequireAuth(handler)(update)
	}
}

// WithRoleContext добавляет проверку роли для команды
func (m *AuthMiddleware) WithRoleContext(command, requiredRole string, handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return func(update *TelegramUpdate) error {
		return m.RequireRole(requiredRole, handler)(update)
	}
}

// WithAdminContext добавляет проверку администратора для команды
func (m *AuthMiddleware) WithAdminContext(command string, handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return m.WithRoleContext(command, models.RoleAdmin, handler)
}

// WithPremiumContext добавляет проверку премиум статуса
func (m *AuthMiddleware) WithPremiumContext(command string, handler func(user *models.User, update *TelegramUpdate) error) func(update *TelegramUpdate) error {
	return func(update *TelegramUpdate) error {
		return m.RequirePremium(handler)(update)
	}
}

// RegisterProtectedCommands регистрирует защищенные команды
func (m *AuthMiddleware) RegisterProtectedCommands(handlers *AuthHandlers, router func(command string, handler func(update *TelegramUpdate) error)) {
	// Команды, требующие авторизации
	router("profile", m.WithUserContext("profile", handlers.handleProfile))
	router("settings", m.WithUserContext("settings", handlers.handleSettings))
	router("notifications", m.WithUserContext("notifications", handlers.handleNotifications))
	router("thresholds", m.WithUserContext("thresholds", handlers.handleThresholds))
	router("periods", m.WithUserContext("periods", handlers.handlePeriods))
	router("language", m.WithUserContext("language", handlers.handleLanguage))

	// Команды, требующие премиум статуса
	router("premium", m.WithPremiumContext("premium", handlers.handlePremium))
	router("advanced", m.WithPremiumContext("advanced", handlers.handleAdvanced))

	// Команды для администраторов
	router("admin", m.WithAdminContext("admin", handlers.handleAdmin))
	router("stats", m.WithAdminContext("stats", handlers.handleStats))
	router("users", m.WithAdminContext("users", handlers.handleUsers))
}

// Вспомогательные функции

// sendAuthError отправляет сообщение об ошибке авторизации
func (m *AuthMiddleware) sendAuthError(chatID int64, message string) error {
	fullMessage := "🔐 *Ошибка авторизации*\n\n" + message

	// Создаем инлайн клавиатуру
	keyboard := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔑 Войти", CallbackData: "auth_login"},
			},
		},
	}

	return m.sendMessage(chatID, fullMessage, &keyboard)
}

// sendMessage отправляет сообщение пользователю через Telegram API
func (m *AuthMiddleware) sendMessage(chatID int64, text string, replyMarkup interface{}) error {
	url := fmt.Sprintf("%ssendMessage", m.baseURL)

	// Подготавливаем запрос
	request := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	if replyMarkup != nil {
		request["reply_markup"] = replyMarkup
	}

	// Конвертируем в JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Отправляем запрос
	resp, err := m.httpClient.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}

	log.Printf("📤 Message sent to chat %d: %s", chatID, text[:min(50, len(text))])
	return nil
}

// getChatID получает Chat ID из обновления
func (m *AuthMiddleware) getChatID(update *TelegramUpdate) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID
	}
	return 0
}

// hasRequiredRole проверяет, есть ли у пользователя требуемая роль
func (m *AuthMiddleware) hasRequiredRole(user *models.User, requiredRole string) bool {
	switch requiredRole {
	case models.RoleAdmin:
		return user.IsAdmin()
	case models.RolePremium:
		return user.IsPremium()
	case models.RoleUser:
		return true
	default:
		return user.Role == requiredRole
	}
}

// isPremiumUser проверяет, является ли пользователь премиум
func (m *AuthMiddleware) isPremiumUser(user *models.User) bool {
	return user.IsPremium() || user.Role == models.RoleAdmin
}

// getUserInfo возвращает информацию о пользователе для логов
func (m *AuthMiddleware) getUserInfo(update *TelegramUpdate) (int64, string, string) {
	if update.Message != nil {
		return update.Message.From.ID, update.Message.From.Username, update.Message.From.FirstName
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID, update.CallbackQuery.From.Username, update.CallbackQuery.From.FirstName
	}
	return 0, "", ""
}

// CreateAuthInlineKeyboard создает инлайн клавиатуру для авторизации
func (m *AuthMiddleware) CreateAuthInlineKeyboard() InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔑 Войти", CallbackData: "auth_login"},
				{Text: "📋 Профиль", CallbackData: "auth_profile"},
			},
			{
				{Text: "⚙️ Настройки", CallbackData: "auth_settings"},
				{Text: "🔔 Уведомления", CallbackData: "auth_notifications"},
			},
		},
	}
}

// CreateAdminInlineKeyboard создает инлайн клавиатуру для администратора
func (m *AuthMiddleware) CreateAdminInlineKeyboard() InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "👥 Пользователи", CallbackData: "admin_users"},
				{Text: "📊 Статистика", CallbackData: "admin_stats"},
			},
			{
				{Text: "⚙️ Настройки системы", CallbackData: "admin_system"},
				{Text: "🔄 Логи", CallbackData: "admin_logs"},
			},
		},
	}
}

// CreatePremiumInlineKeyboard создает инлайн клавиатуру для премиум пользователей
func (m *AuthMiddleware) CreatePremiumInlineKeyboard() InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🚀 Расширенные сигналы", CallbackData: "premium_advanced"},
				{Text: "📈 Детальная аналитика", CallbackData: "premium_analytics"},
			},
			{
				{Text: "⏱️ Приоритетная очередь", CallbackData: "premium_priority"},
				{Text: "🔔 Кастомные уведомления", CallbackData: "premium_notifications"},
			},
		},
	}
}

// answerCallbackQuery отвечает на callback запрос
func (m *AuthMiddleware) answerCallbackQuery(callbackID string, text string, showAlert bool) error {
	url := fmt.Sprintf("%sanswerCallbackQuery", m.baseURL)

	request := map[string]interface{}{
		"callback_query_id": callbackID,
		"text":              text,
		"show_alert":        showAlert,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal callback answer: %w", err)
	}

	resp, err := m.httpClient.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to answer callback query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}

	return nil
}

// editMessageText редактирует текст сообщения
func (m *AuthMiddleware) editMessageText(chatID, messageID int64, text string, replyMarkup interface{}) error {
	url := fmt.Sprintf("%seditMessageText", m.baseURL)

	request := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	if replyMarkup != nil {
		request["reply_markup"] = replyMarkup
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal edit request: %w", err)
	}

	resp, err := m.httpClient.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}

	return nil
}

// LogUserActivity логирует активность пользователя
func (m *AuthMiddleware) LogUserActivity(user *models.User, activityType, description string) {
	log.Printf("👤 Activity: user_id=%d, username=%s, type=%s, description=%s",
		user.ID, user.Username, activityType, description)
}
