// internal/delivery/telegram/app/bot/middlewares/auth.go
package middlewares

import (
	"fmt"
	"strconv"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	subscription_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/subscription"
	"crypto-exchange-screener-bot/pkg/logger"
)

// AuthMiddleware - middleware для проверки авторизации
type AuthMiddleware struct {
	userService         *users.Service
	subscriptionService *subscription.Service
	subscriptionRepo    subscription_repo.SubscriptionRepository
}

// NewAuthMiddleware создает новый middleware аутентификации
func NewAuthMiddleware(
	userService *users.Service,
	subscriptionService *subscription.Service,
	subscriptionRepo subscription_repo.SubscriptionRepository,
) *AuthMiddleware {
	return &AuthMiddleware{
		userService:         userService,
		subscriptionService: subscriptionService,
		subscriptionRepo:    subscriptionRepo,
	}
}

// ProcessUpdate обрабатывает обновление и создает handlers.HandlerParams
func (m *AuthMiddleware) ProcessUpdate(update *telegram.TelegramUpdate) (handlers.HandlerParams, error) {
	// ЗАЩИТА ОТ NIL: проверяем userService
	if m.userService == nil {
		logger.Warn("❌ ProcessUpdate: userService is nil! Cannot process update")
		return handlers.HandlerParams{}, fmt.Errorf("сервис пользователей временно недоступен")
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

		if update.CallbackQuery.Message != nil {
			chatID = update.CallbackQuery.Message.Chat.ID
			logger.Info("🔍 ProcessUpdate: Callback from user %d, chat %d (from Message), data: %s", userID, chatID, data)
		} else {
			chatID = userID
			logger.Warn("⚠️ ProcessUpdate: No Message in callback, using userID as chatID: %d, data: %s", chatID, data)
		}
	} else if update.PreCheckoutQuery != nil && update.PreCheckoutQuery.From.ID > 0 {
		userID = update.PreCheckoutQuery.From.ID
		username = update.PreCheckoutQuery.From.Username
		firstName = update.PreCheckoutQuery.From.FirstName
		lastName = update.PreCheckoutQuery.From.LastName
		chatID = userID

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
		return handlers.HandlerParams{}, fmt.Errorf("не удалось получить информацию о пользователе")
	}

	// Получаем или создаем пользователя
	user, err := m.userService.GetOrCreateUser(userID, username, firstName, lastName)
	if err != nil {
		logger.Error("❌ ProcessUpdate: Ошибка получения пользователя %d: %v", userID, err)
		return handlers.HandlerParams{}, fmt.Errorf("ошибка получения пользователя: %w", err)
	}

	logger.Info("✅ ProcessUpdate: User found/created: ID=%d, TelegramID=%d, ChatID=%s",
		user.ID, user.TelegramID, user.ChatID)

	// Проверяем активность пользователя
	if !user.IsActive {
		logger.Warn("❌ ProcessUpdate: User %d is not active", user.ID)
		return handlers.HandlerParams{}, fmt.Errorf("аккаунт деактивирован")
	}

	// ⭐ ПРОВЕРКА ПОДПИСКИ (опционально, можно раскомментировать)
	// if err := m.ensureSubscription(user.ID); err != nil {
	//     return handlers.HandlerParams{}, err
	// }

	// Добавляем ChatID если его нет
	if user.ChatID == "" {
		user.ChatID = strconv.FormatInt(chatID, 10)
		if err := m.userService.UpdateUser(user); err != nil {
			logger.Warn("⚠️ ProcessUpdate: Не удалось обновить ChatID для user %d: %v", user.ID, err)
		} else {
			logger.Info("📝 ProcessUpdate: Updated ChatID for user %d: %s", user.ID, user.ChatID)
		}
	}

	return handlers.HandlerParams{
		User:     user,
		ChatID:   chatID,
		Text:     text,
		Data:     data,
		UpdateID: updateID,
	}, nil
}

// RequireAuth создает обертку для хэндлера с проверкой авторизации
func (m *AuthMiddleware) RequireAuth(handler handlers.Handler) handlers.Handler {
	return &authWrapper{
		handler:      handler,
		userService:  m.userService,
		requireAuth:  true,
		requiredRole: "",
	}
}

// RequireRole создает обертку для хэндлера с проверкой роли
func (m *AuthMiddleware) RequireRole(requiredRole string, handler handlers.Handler) handlers.Handler {
	return &authWrapper{
		handler:      handler,
		userService:  m.userService,
		requireAuth:  true,
		requiredRole: requiredRole,
	}
}

// RequireAdmin создает обертку для хэндлера с проверкой администратора
func (m *AuthMiddleware) RequireAdmin(handler handlers.Handler) handlers.Handler {
	return m.RequireRole(models.RoleAdmin, handler)
}

// RequirePremium создает обертку для хэндлера с проверкой премиум статуса
func (m *AuthMiddleware) RequirePremium(handler handlers.Handler) handlers.Handler {
	return &authWrapper{
		handler:        handler,
		userService:    m.userService,
		requireAuth:    true,
		requirePremium: true,
	}
}

// authWrapper обертка для хэндлера с проверкой авторизации
type authWrapper struct {
	handler        handlers.Handler
	userService    *users.Service
	requireAuth    bool
	requiredRole   string
	requirePremium bool
}

// Execute выполняет проверку авторизации и вызывает оригинальный хэндлер
func (w *authWrapper) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if w.requireAuth && params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("требуется авторизация")
	}

	if w.requiredRole != "" && params.User != nil {
		if !w.hasRequiredRole(params.User, w.requiredRole) {
			return handlers.HandlerResult{}, fmt.Errorf("недостаточно прав. Требуется роль: %s", w.requiredRole)
		}
	}

	if w.requirePremium && params.User != nil {
		if !w.isPremiumUser(params.User) {
			return handlers.HandlerResult{}, fmt.Errorf("эта функция доступна только премиум пользователям")
		}
	}

	return w.handler.Execute(params)
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
func (w *authWrapper) GetType() handlers.HandlerType {
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
