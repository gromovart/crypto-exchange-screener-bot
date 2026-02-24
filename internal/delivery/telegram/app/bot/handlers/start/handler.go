// internal/delivery/telegram/app/bot/handlers/start/handler.go
package start

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/middlewares"
	trading_session "crypto-exchange-screener-bot/internal/delivery/telegram/services/trading_session"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// startHandlerImpl реализация StartHandler
type startHandlerImpl struct {
	*base.BaseHandler
	subscriptionMiddleware *middlewares.SubscriptionMiddleware
	tradingSessionService  trading_session.Service
}

// NewHandler создает новый хэндлер команды /start
func NewHandler(subscriptionMiddleware *middlewares.SubscriptionMiddleware, tradingSessionSvc trading_session.Service) handlers.Handler {
	return &startHandlerImpl{
		BaseHandler: &base.BaseHandler{
			Name:    "start_handler",
			Command: "start",
			Type:    handlers.TypeCommand,
		},
		subscriptionMiddleware: subscriptionMiddleware,
		tradingSessionService:  tradingSessionSvc,
	}
}

// Execute выполняет обработку команды /start
func (h *startHandlerImpl) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
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
		result, err := h.handlePaymentStart(user, payload)
		if err != nil {
			logger.Warn("Ошибка обработки платежного payload %s: %v", payload, err)
			message := "⚠️ *Ошибка обработки платежной ссылки*\n\n"
			message += "Пожалуйста, используйте команду /buy для выбора плана оплаты."

			return handlers.HandlerResult{
				Message:  message,
				Keyboard: h.createBuyKeyboard(),
				Metadata: map[string]interface{}{
					"user_id":   user.ID,
					"payload":   payload,
					"error":     err.Error(),
					"timestamp": time.Now(),
				},
			}, nil
		}
		return result, nil
	}

	// Если payload не распознан, показываем стандартное приветствие с уведомлением
	message := "⚠️ *Неизвестный параметр:* `" + payload + "`\n\n"
	message += "Используйте команду /help для получения списка доступных команд."

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: h.createBuyKeyboard(),
		Metadata: map[string]interface{}{
			"user_id":   user.ID,
			"payload":   payload,
			"timestamp": time.Now(),
		},
	}, nil
}

// handlePaymentStart обрабатывает платежный payload
func (h *startHandlerImpl) handlePaymentStart(user *models.User, payload string) (handlers.HandlerResult, error) {
	logger.Info("Обработка платежного payload: %s для пользователя %d", payload, user.ID)

	// Извлекаем параметры: pay_{user_id}_{plan_id}
	parts := strings.Split(payload, "_")
	if len(parts) != 3 {
		logger.Warn("Неверный формат платежного payload: %s", payload)
		return handlers.HandlerResult{
			Message: "⚠️ *Неверный формат платежной ссылки*\n\n" +
				"Пожалуйста, используйте команду /buy для выбора плана оплаты.",
			Keyboard: h.createBuyKeyboard(),
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
		return handlers.HandlerResult{
			Message: "⚠️ *Ошибка в платежной ссылке*\n\n" +
				"Пожалуйста, используйте команду /buy для выбора плана оплаты.",
			Keyboard: h.createBuyKeyboard(),
			Metadata: map[string]interface{}{
				"user_id":   user.ID,
				"payload":   payload,
				"timestamp": time.Now(),
			},
		}, nil
	}

	if userID != user.ID {
		logger.Warn("UserID в payload (%d) не совпадает с текущим пользователем (%d)", userID, user.ID)
		return handlers.HandlerResult{
			Message: "⚠️ *Ссылка предназначена для другого пользователя*\n\n" +
				"Пожалуйста, используйте команду /buy для выбора плана оплаты.",
			Keyboard: h.createBuyKeyboard(),
			Metadata: map[string]interface{}{
				"user_id":   user.ID,
				"payload":   payload,
				"timestamp": time.Now(),
			},
		}, nil
	}

	logger.Info("Начало процесса оплаты: пользователь=%d, план=%s", user.ID, planID)

	// Показываем сообщение о начале оплаты
	message := "💳 *Начинаем процесс оплаты*\n\n"
	message += fmt.Sprintf("План: *%s*\n", h.getPlanName(planID))
	message += "Для продолжения оплаты используйте команду /buy\n\n"
	message += "Или нажмите кнопку ниже:"

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
			"payment_status":  "pending",
			"payment_started": true,
			"timestamp":       time.Now(),
		},
	}, nil
}

// handleStandardStart стандартное приветствие без параметров
func (h *startHandlerImpl) handleStandardStart(user *models.User) (handlers.HandlerResult, error) {
	ctx := context.Background()

	// Получаем активную подписку пользователя
	subscription, err := h.subscriptionMiddleware.GetSubscriptionService().GetActiveSubscription(ctx, user.ID)

	var subscriptionStatus string

	if err == nil && subscription != nil {
		// Есть активная подписка
		if subscription.PlanCode == "free" {
			// Бесплатный период - показываем таймер
			remaining := subscription.CurrentPeriodEnd.Sub(time.Now())
			hours := int(remaining.Hours())
			minutes := int(remaining.Minutes()) % 60

			var timeLeft string
			if hours > 0 {
				timeLeft = fmt.Sprintf("%dч %dмин", hours, minutes)
			} else {
				timeLeft = fmt.Sprintf("%dмин", minutes)
			}

			subscriptionStatus = fmt.Sprintf(
				"🎁 *Бесплатный период*\n"+
					"   • Осталось: *%s*\n"+
					"   • Действует до: *%s*",
				timeLeft,
				subscription.CurrentPeriodEnd.Format("02.01.2006 15:04"))
		} else {
			// Платная подписка - показываем дату окончания
			// Определяем название плана
			planName := subscription.PlanName
			if planName == "" {
				switch subscription.PlanCode {
				case "basic":
					planName = "📱 Доступ на 1 месяц"
				case "pro":
					planName = "🚀 Доступ на 3 месяца"
				case "enterprise":
					planName = "🏢 Доступ на 12 месяцев"
				case "test":
					planName = "🧪 Тестовый доступ"
				default:
					planName = subscription.PlanCode
				}
			}

			subscriptionStatus = fmt.Sprintf(
				"✅ *Подписка активна*\n"+
					"   • План: *%s*\n"+
					"   • Действует до: *%s*",
				planName,
				subscription.CurrentPeriodEnd.Format("02.01.2006 15:04"))
		}
	} else {
		// Нет активной подписки
		subscriptionStatus = "❌ *Нет активной подписки*\n" +
			"   • Используйте /buy для покупки"
	}

	message := fmt.Sprintf(
		"👋 *Добро пожаловать, %s!*\n"+
			"🚀 *Crypto Exchange Screener Bot*\n\n"+
			"✅ @%s  •  👤 %s  •  📅 %s\n"+
			"⭐ Роль: %s\n\n"+
			"━━━ 🎁 ПОДПИСКА ━━━\n"+
			"%s\n"+
			"━━━ 📊 О БОТЕ ━━━\n"+
			"▫️ Биржа: *Bybit*  •  Обновление: *10-20 сек*\n"+
			"▫️ Символы: фьючерсы USDT\n"+
			"▫️ Сигналы: рост / падение / объёмы / OI\n\n"+
			"📚 [Полная документация](https://teletype.in/@gromovart/pj2UIVlmr55)\n"+
			"✉️ Поддержка: support@gromovart.ru\n\n"+
			"━━━ ⚠️ ВАЖНОЕ ПРЕДУПРЕЖДЕНИЕ ━━━\n\n"+
			"▫️ *Рыночные риски* — рынок криптовалют высоко волатилен, торговля связана с риском потери капитала\n\n"+
			"▫️ *Информационный характер* — сигналы не являются руководством к действию (Buy/Sell)\n\n"+
			"▫️ *Ограниченная интерпретация* — бот даёт базовый анализ, используйте несколько источников\n\n"+
			"▫️ *Ответственность* — все решения о сделках вы принимаете самостоятельно\n\n"+
			"▫️ *Задержка данных* — цена в момент сигнала может отличаться от терминала биржи\n\n"+
			"▫️ *Временной лаг* — пока вы анализируете сигнал, цена может существенно измениться\n\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
			"Используйте меню ниже для управления ботом:",
		user.FirstName,
		user.Username,
		user.FirstName,
		user.CreatedAt.Format("02.01.2006"),
		h.GetRoleDisplay(user.Role),
		subscriptionStatus,
	)

	keyboard := h.createSessionReplyKeyboard(user.ID, user.Timezone)

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

// createBuyKeyboard создает клавиатуру для покупки подписки
func (h *startHandlerImpl) createBuyKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "💎 Купить подписку", "callback_data": constants.PaymentConstants.CommandBuy},
			},
			{
				{"text": constants.ButtonTexts.Help, "callback_data": constants.CallbackHelp},
			},
		},
	}
}

// createSessionReplyKeyboard создает reply keyboard с кнопкой сессии
func (h *startHandlerImpl) createSessionReplyKeyboard(userID int, timezone string) interface{} {
	buttonText := constants.SessionButtonTexts.Start
	if h.tradingSessionService != nil {
		if session, ok := h.tradingSessionService.GetActive(userID); ok {
			buttonText = fmt.Sprintf("%s (до %s)",
				constants.SessionButtonTexts.Stop,
				formatInUserTZ(session.ExpiresAt, timezone),
			)
		}
	}

	return telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.ReplyKeyboardButton{
			{{Text: buttonText}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// formatInUserTZ форматирует время в часовом поясе пользователя
func formatInUserTZ(t time.Time, timezone string) string {
	if timezone == "" {
		timezone = "Europe/Moscow"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return t.Format("15:04")
	}
	return t.In(loc).Format("15:04")
}

// parseUserID парсит user_id из строки
func (h *startHandlerImpl) parseUserID(userIDStr string) (int, error) {
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
		"basic":      "📱 Доступ на 1 месяц",
		"pro":        "🚀 Доступ на 3 месяца",
		"enterprise": "🏢 Доступ на 12 месяцев",
		"test":       "🧪 Тестовый доступ",
	}
	if name, exists := plans[planID]; exists {
		return name
	}
	return "Неизвестный план"
}
