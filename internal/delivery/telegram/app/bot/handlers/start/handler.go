// internal/delivery/telegram/app/bot/handlers/start/handler.go
package start

import (
	"fmt"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// startHandlerImpl реализация StartHandler
type startHandlerImpl struct {
	*base.BaseHandler
}

// NewHandler создает новый хэндлер команды /start
func NewHandler() handlers.Handler {
	return &startHandlerImpl{
		BaseHandler: &base.BaseHandler{
			Name:    "start_handler",
			Command: "start",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /start
func (h *startHandlerImpl) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Логируем полученную команду для отладки
	logger.Debug("Обработка /start: текст='%s', data='%s'", params.Text, params.Data)

	// Проверяем есть ли параметры после /start
	text := strings.TrimSpace(params.Text)

	// Если текст начинается с /start, обрабатываем параметры
	if strings.HasPrefix(text, "/start ") {
		payload := strings.TrimSpace(text[len("/start"):])
		return h.handleStartWithPayload(params.User, payload)
	}

	// Если есть данные в params.Data (из роутера)
	if params.Data != "" && strings.HasPrefix(params.Data, "pay_") {
		return h.handleStartWithPayload(params.User, params.Data)
	}

	// Стандартное приветствие без параметров
	return h.handleStandardStart(params.User)
}

// handleStartWithPayload обрабатывает /start с параметрами
func (h *startHandlerImpl) handleStartWithPayload(user *models.User, payload string) (handlers.HandlerResult, error) {
	logger.Info("Обработка /start с payload: %s для пользователя %d", payload, user.ID)

	// Проверяем формат платежного payload: pay_{user_id}_{plan_id}
	if strings.HasPrefix(payload, "pay_") {
		return h.handlePaymentStart(user, payload)
	}

	// Другие типы payload можно добавить здесь
	// Например: ref_{referral_code}, promo_{promo_code} и т.д.

	// Если payload не распознан, показываем стандартное приветствие с уведомлением
	message := h.formatWelcomeMessage(user)
	message += "\n\n⚠️ *Неизвестный параметр:* `" + payload + "`"

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: h.createWelcomeKeyboard(),
		Metadata: map[string]interface{}{
			"user_id":   user.ID,
			"payload":   payload,
			"timestamp": time.Now(),
		},
	}, nil
}

// handlePaymentStart обрабатывает платежный payload
func (h *startHandlerImpl) handlePaymentStart(user *models.User, payload string) (handlers.HandlerResult, error) {
	// Извлекаем параметры: pay_{user_id}_{plan_id}
	parts := strings.Split(payload, "_")
	if len(parts) != 3 {
		// Неверный формат - возвращаем стандартное приветствие без сообщения об оплате
		logger.Warn("Неверный формат платежного payload: %s", payload)
		message := h.formatWelcomeMessage(user)
		message += "\n\n⚠️ *Неверный формат платежной ссылки*"
		return handlers.HandlerResult{
			Message:  message,
			Keyboard: h.createWelcomeKeyboard(),
			Metadata: map[string]interface{}{
				"user_id":   user.ID,
				"payload":   payload,
				"timestamp": time.Now(),
			},
		}, nil
	}

	userIDStr := parts[1]
	planID := parts[2]

	// Проверяем что user_id совпадает с текущим пользователем
	userID, err := h.parseUserID(userIDStr)
	if err != nil {
		logger.Warn("Неверный user_id в payload: %s", userIDStr)
		message := h.formatWelcomeMessage(user)
		message += "\n\n⚠️ *Ошибка в платежной ссылке*"
		return handlers.HandlerResult{
			Message:  message,
			Keyboard: h.createWelcomeKeyboard(),
			Metadata: map[string]interface{}{
				"user_id":   user.ID,
				"payload":   payload,
				"timestamp": time.Now(),
			},
		}, nil
	}

	if userID != user.ID {
		logger.Warn("UserID в payload (%d) не совпадает с текущим пользователем (%d)", userID, user.ID)
		message := h.formatWelcomeMessage(user)
		message += "\n\n⚠️ *Ссылка предназначена для другого пользователя*"
		return handlers.HandlerResult{
			Message:  message,
			Keyboard: h.createWelcomeKeyboard(),
			Metadata: map[string]interface{}{
				"user_id":   user.ID,
				"payload":   payload,
				"timestamp": time.Now(),
			},
		}, nil
	}

	logger.Info("Начало процесса оплаты: пользователь=%d, план=%s", user.ID, planID)

	// Показываем ТОЛЬКО сообщение о начале оплаты, НЕ стандартное приветствие
	message := "💳 *Начинаем процесс оплаты*\n\n"
	message += fmt.Sprintf("План: *%s*\n", h.getPlanName(planID))
	message += "Для продолжения оплаты используйте команду /buy\n\n"
	message += "Или нажмите кнопку ниже:"

	// Создаем клавиатуру с кнопкой для оплаты
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "💳 Перейти к оплате", "callback_data": constants.PaymentConstants.CommandBuy},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id":         user.ID,
			"plan_id":         planID,
			"payment_status":  "pending", // Ожидание оплаты
			"payment_started": true,
			"timestamp":       time.Now(),
		},
	}, nil
}

// handleStandardStart стандартное приветствие без параметров
func (h *startHandlerImpl) handleStandardStart(user *models.User) (handlers.HandlerResult, error) {
	message := h.formatWelcomeMessage(user)
	keyboard := h.createWelcomeKeyboard()

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id":    user.ID,
			"first_name": user.FirstName,
			"timestamp":  time.Now(),
		},
	}, nil
}

// parseUserID парсит user_id из строки
func (h *startHandlerImpl) parseUserID(userIDStr string) (int, error) {
	// Пытаемся распарсить как число
	var userID int
	_, err := fmt.Sscanf(userIDStr, "%d", &userID)
	if err != nil {
		return 0, fmt.Errorf("не удалось распарсить user_id: %w", err)
	}
	return userID, nil
}

// getPlanName возвращает читаемое название плана по ID
func (h *startHandlerImpl) getPlanName(planID string) string {
	plans := map[string]string{
		"basic":      "📱 Basic",
		"pro":        "🚀 Pro",
		"enterprise": "🏢 Enterprise",
	}
	if name, exists := plans[planID]; exists {
		return name
	}
	return "Неизвестный план"
}

// formatWelcomeMessage форматирует приветственное сообщение
func (h *startHandlerImpl) formatWelcomeMessage(user *models.User) string {
	firstName := user.FirstName
	if firstName == "" {
		firstName = "Гость"
	}

	username := user.Username
	if username == "" {
		username = "не указан"
	} else {
		username = "@" + username
	}

	return fmt.Sprintf(
		"👋 *Добро пожаловать, %s!*\n\n"+
			"🚀 *Crypto Exchange Screener Bot*\n\n"+
			"✅ Ваш аккаунт: %s\n"+
			"👤 Имя: %s\n"+
			"⭐ Роль: %s\n"+
			"📅 Дата регистрации: %s\n\n"+
			"Бот анализирует рынок криптовалют и отправляет уведомления о сильных движениях.\n\n"+
			"Используйте меню ниже для управления ботом:",
		firstName,
		username,
		firstName,
		h.GetRoleDisplay(user.Role),
		user.CreatedAt.Format("02.01.2006"),
	)
}

// createWelcomeKeyboard создает клавиатуру для приветствия
func (h *startHandlerImpl) createWelcomeKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.MenuButtonTexts.Profile, "callback_data": constants.CallbackProfileMain},
				{"text": constants.ButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
			},
			{
				{"text": constants.MenuButtonTexts.Notifications, "callback_data": constants.CallbackNotificationsMenu},
				{"text": constants.ButtonTexts.Help, "callback_data": constants.CallbackHelp},
			},
		},
	}
}
