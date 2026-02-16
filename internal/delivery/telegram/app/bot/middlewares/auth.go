// internal/delivery/telegram/app/bot/middlewares/auth.go
package middlewares

import (
	"fmt"
	"strconv"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Handler интерфейс хэндлера (совместимый с handlers.Handler)
type Handler interface {
	Execute(params interface{}) (interface{}, error)
	GetName() string
	GetCommand() string
	GetType() string
}

// HandlerParams параметры для хэндлеров (совместимые)
type HandlerParams struct {
	User     *models.User
	ChatID   int64
	Text     string // текст сообщения
	Data     string // для callback данных
	UpdateID string // ID обновления
}

// AuthMiddleware - middleware для проверки авторизации
type AuthMiddleware struct {
	userService *users.Service
}

// NewAuthMiddleware создает новый middleware аутентификации
func NewAuthMiddleware(userService *users.Service) *AuthMiddleware {
	return &AuthMiddleware{
		userService: userService,
	}
}

// ProcessUpdate обрабатывает обновление и создает HandlerParams
func (m *AuthMiddleware) ProcessUpdate(update *telegram.TelegramUpdate) (HandlerParams, error) {
	// ЗАЩИТА ОТ NIL: проверяем userService
	if m.userService == nil {
		logger.Warn("❌ ProcessUpdate: userService is nil! Cannot process update")
		return HandlerParams{}, fmt.Errorf("сервис пользователей временно недоступен")
	}

	var userID int64
	var username, firstName, lastName string
	var chatID int64
	var text, data string
	var updateID string

	updateID = strconv.Itoa(update.UpdateID)

	// Извлекаем данные из обновления
	if update.Message != nil && update.Message.From.ID > 0 {
		userID = update.Message.From.ID
		username = update.Message.From.Username
		firstName = update.Message.From.FirstName
		lastName = update.Message.From.LastName
		chatID = update.Message.Chat.ID
		text = update.Message.Text

		// Проверяем successful_payment
		if update.Message.SuccessfulPayment != nil {
			// ⭐ ИСПРАВЛЕНО: правильный формат для successful_payment
			// Формат: successful_payment:{payment_id}:{payload}:{amount}:{currency}:{user_id}:{charge_id}
			data = fmt.Sprintf("successful_payment:%s:%s:%d:%s:%d:%s",
				update.Message.SuccessfulPayment.TelegramPaymentChargeID,
				update.Message.SuccessfulPayment.InvoicePayload,
				update.Message.SuccessfulPayment.TotalAmount,
				update.Message.SuccessfulPayment.Currency,
				userID,
				update.Message.SuccessfulPayment.ProviderPaymentChargeID)

			logger.Info("🔍 ProcessUpdate: SuccessfulPayment from user %d, amount: %d %s, payload: %s, data: %s",
				userID, update.Message.SuccessfulPayment.TotalAmount,
				update.Message.SuccessfulPayment.Currency, update.Message.SuccessfulPayment.InvoicePayload, data)
		} else {
			logger.Info("🔍 ProcessUpdate: Message from user %d, chat %d, text: %s", userID, chatID, text)
		}
	} else if update.CallbackQuery != nil && update.CallbackQuery.From.ID > 0 {
		userID = update.CallbackQuery.From.ID
		username = update.CallbackQuery.From.Username
		firstName = update.CallbackQuery.From.FirstName
		lastName = update.CallbackQuery.From.LastName
		data = update.CallbackQuery.Data

		// Для callback пытаемся получить chatID из Message
		if update.CallbackQuery.Message != nil {
			chatID = update.CallbackQuery.Message.Chat.ID
			logger.Info("🔍 ProcessUpdate: Callback from user %d, chat %d (from Message), data: %s", userID, chatID, data)
		} else {
			// Если нет Message, используем userID как chatID (для приватных чатов)
			chatID = userID
			logger.Warn("⚠️ ProcessUpdate: No Message in callback, using userID as chatID: %d, data: %s", chatID, data)
		}
	} else if update.PreCheckoutQuery != nil && update.PreCheckoutQuery.From.ID > 0 {
		// Обработка pre_checkout_query
		userID = update.PreCheckoutQuery.From.ID
		username = update.PreCheckoutQuery.From.Username
		firstName = update.PreCheckoutQuery.From.FirstName
		lastName = update.PreCheckoutQuery.From.LastName
		chatID = userID // Для pre_checkout_query используем userID как chatID

		// Формируем данные для передачи в обработчик
		// Формат: pre_checkout_query:{query_id}:{payload}:{amount}:{currency}:{user_id}
		data = fmt.Sprintf("pre_checkout_query:%s:%s:%d:%s:%d",
			update.PreCheckoutQuery.ID,
			update.PreCheckoutQuery.InvoicePayload,
			update.PreCheckoutQuery.TotalAmount,
			update.PreCheckoutQuery.Currency,
			userID)

		logger.Info("🔍 ProcessUpdate: PreCheckoutQuery from user %d, amount: %d %s, payload: %s, data: %s",
			userID, update.PreCheckoutQuery.TotalAmount,
			update.PreCheckoutQuery.Currency, update.PreCheckoutQuery.InvoicePayload, data)
	} else {
		logger.Warn("❌ ProcessUpdate: Не удалось получить информацию о пользователе")
		return HandlerParams{}, fmt.Errorf("не удалось получить информацию о пользователе")
	}

	// Получаем или создаем пользователя
	user, err := m.userService.GetOrCreateUser(userID, username, firstName, lastName)
	if err != nil {
		logger.Error("❌ ProcessUpdate: Ошибка получения пользователя %d: %v", userID, err)
		return HandlerParams{}, fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	logger.Info("✅ ProcessUpdate: User found/created: ID=%d, TelegramID=%d, ChatID=%s",
		user.ID, user.TelegramID, user.ChatID)

	// Проверяем активность пользователя
	if !user.IsActive {
		logger.Warn("❌ ProcessUpdate: User %d is not active", user.ID)
		return HandlerParams{}, fmt.Errorf("аккаунт деактивирован")
	}

	// Добавляем ChatID если его нет
	if user.ChatID == "" {
		user.ChatID = strconv.FormatInt(chatID, 10)
		if err := m.userService.UpdateUser(user); err != nil {
			logger.Warn("⚠️ ProcessUpdate: Не удалось обновить ChatID для user %d: %v", user.ID, err)
		} else {
			logger.Info("📝 ProcessUpdate: Updated ChatID for user %d: %s", user.ID, user.ChatID)
		}
	}

	return HandlerParams{
		User:     user,
		ChatID:   chatID,
		Text:     text,
		Data:     data,
		UpdateID: updateID,
	}, nil
}

// RequireAuth создает обертку для хэндлера с проверкой авторизации
func (m *AuthMiddleware) RequireAuth(handler Handler) Handler {
	return &authWrapper{
		handler:      handler,
		userService:  m.userService,
		requireAuth:  true,
		requiredRole: "",
	}
}

// RequireRole создает обертку для хэндлера с проверкой роли
func (m *AuthMiddleware) RequireRole(requiredRole string, handler Handler) Handler {
	return &authWrapper{
		handler:      handler,
		userService:  m.userService,
		requireAuth:  true,
		requiredRole: requiredRole,
	}
}

// RequireAdmin создает обертку для хэндлера с проверкой администратора
func (m *AuthMiddleware) RequireAdmin(handler Handler) Handler {
	return m.RequireRole(models.RoleAdmin, handler)
}

// RequirePremium создает обертку для хэндлера с проверкой премиум статуса
func (m *AuthMiddleware) RequirePremium(handler Handler) Handler {
	return &authWrapper{
		handler:        handler,
		userService:    m.userService,
		requireAuth:    true,
		requirePremium: true,
	}
}

// authWrapper обертка для хэндлера с проверкой авторизации
type authWrapper struct {
	handler        Handler
	userService    *users.Service
	requireAuth    bool
	requiredRole   string
	requirePremium bool
}

// Execute выполняет проверку авторизации и вызывает оригинальный хэндлер
func (w *authWrapper) Execute(params interface{}) (interface{}, error) {
	handlerParams, ok := params.(HandlerParams)
	if !ok {
		return nil, fmt.Errorf("неверный тип параметров")
	}

	if w.requireAuth && handlerParams.User == nil {
		return nil, fmt.Errorf("требуется авторизация")
	}

	if w.requiredRole != "" && handlerParams.User != nil {
		if !w.hasRequiredRole(handlerParams.User, w.requiredRole) {
			return nil, fmt.Errorf("недостаточно прав. Требуется роль: %s", w.requiredRole)
		}
	}

	if w.requirePremium && handlerParams.User != nil {
		if !w.isPremiumUser(handlerParams.User) {
			return nil, fmt.Errorf("эта функция доступна только премиум пользователям")
		}
	}

	return w.handler.Execute(handlerParams)
}

// GetName возвращает имя обернутого хэндлера
func (w *authWrapper) GetName() string {
	return "auth_wrapper_" + w.handler.GetName()
}

// GetCommand возвращает команду обернутого хэндлера
func (w *authWrapper) GetCommand() string {
	return w.handler.GetCommand()
}

// GetType возвращает тип обернутого хэндлера
func (w *authWrapper) GetType() string {
	return w.handler.GetType()
}

// hasRequiredRole проверяет, есть ли у пользователя требуемая роль
func (w *authWrapper) hasRequiredRole(user *models.User, requiredRole string) bool {
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
func (w *authWrapper) isPremiumUser(user *models.User) bool {
	return user.IsPremium() || user.Role == models.RoleAdmin
}
