// internal/delivery/telegram/app/bot/handlers/commands/buy/handler.go
package buy

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
)

// buyCommandHandler реализация обработчика команды /buy
type buyCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /buy
func NewHandler() handlers.Handler {
	return &buyCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "buy_command_handler",
			Command: constants.PaymentConstants.CommandBuy,
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /buy
func (h *buyCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Проверяем авторизацию пользователя
	if params.User == nil || params.User.ID == 0 {
		return h.createUnauthorizedMessage()
	}

	// Получаем доступные планы
	plans := h.getAvailablePlans()

	// Проверяем текущую подписку пользователя
	currentSubscription := h.getUserSubscription(params.User.ID)

	// Создаем сообщение
	message := h.createPlansMessage(params.User, plans, currentSubscription)
	keyboard := h.createPlansKeyboard(plans, currentSubscription)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id":          params.User.ID,
			"plans_count":      len(plans),
			"has_subscription": currentSubscription != nil,
		},
	}, nil
}

// createUnauthorizedMessage создает сообщение для неавторизованных пользователей
func (h *buyCommandHandler) createUnauthorizedMessage() (handlers.HandlerResult, error) {
	message := "🔒 *Авторизация требуется*\n\n" +
		"Для покупки подписки необходимо авторизоваться.\n\n" +
		"Используйте кнопку ниже для входа."

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.AuthButtonTexts.Login, "callback_data": constants.CallbackAuthLogin},
			},
			{
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
	}, nil
}

// getAvailablePlans возвращает доступные планы (заглушка)
func (h *buyCommandHandler) getAvailablePlans() []*SubscriptionPlan {
	return []*SubscriptionPlan{
		{
			ID:          "basic",
			Name:        "📱 Basic",
			Description: "• До 10 символов\n• 50 сигналов/день\n• Базовые уведомления",
			PriceCents:  299, // $2.99
			Features:    []string{"10_symbols", "50_signals", "basic_notifications"},
		},
		{
			ID:          "pro",
			Name:        "🚀 Pro",
			Description: "• До 50 символов\n• 200 сигналов/день\n• Расширенные уведомления\n• Приоритетная поддержка",
			PriceCents:  999, // $9.99
			Features:    []string{"50_symbols", "200_signals", "advanced_notifications", "priority_support"},
		},
		{
			ID:          "enterprise",
			Name:        "🏢 Enterprise",
			Description: "• Неограниченные символы\n• 1000+ сигналов/день\n• Кастомные настройки\n• API доступ",
			PriceCents:  2499, // $24.99
			Features:    []string{"unlimited_symbols", "1000_signals", "custom_settings", "api_access"},
		},
	}
}

// getUserSubscription возвращает текущую подписку пользователя (заглушка)
func (h *buyCommandHandler) getUserSubscription(userID int) *UserSubscription {
	// TODO: Получить из базы данных
	return nil
}

// createPlansMessage создает сообщение со списком планов
func (h *buyCommandHandler) createPlansMessage(
	user *models.User,
	plans []*SubscriptionPlan,
	currentSubscription *UserSubscription,
) string {
	message := "💎 *Выберите тарифный план*\n\n"

	// Если есть текущая подписка
	if currentSubscription != nil {
		message += fmt.Sprintf("Ваш текущий план: *%s*\n", currentSubscription.PlanName)
		message += fmt.Sprintf("Действует до: %s\n\n", currentSubscription.ExpiresAt)
	}

	for _, plan := range plans {
		// Расчет стоимости в Stars
		starsAmount := h.calculateStars(plan.PriceCents)
		usdPrice := float64(plan.PriceCents) / 100

		message += fmt.Sprintf("📋 *%s*\n", plan.Name)
		message += fmt.Sprintf("💰 *%d Stars* ($%.2f)\n", starsAmount, usdPrice)
		message += fmt.Sprintf("%s\n\n", plan.Description)
	}

	message += "ℹ️ *Информация о платежах:*\n"
	message += "• 1 Star ≈ $0.01\n"
	message += "• Комиссия Telegram: 5%\n"
	message += "• Подписка продляется автоматически\n"
	message += "• Можно отменить в любой момент\n\n"
	message += "Выберите план для продолжения:"

	return message
}

// createPlansKeyboard создает клавиатуру с планами
func (h *buyCommandHandler) createPlansKeyboard(
	plans []*SubscriptionPlan,
	currentSubscription *UserSubscription,
) interface{} {
	var keyboard [][]map[string]string

	// Кнопки для каждого плана
	for _, plan := range plans {
		buttonText := fmt.Sprintf("📋 %s - %d Stars", plan.Name, h.calculateStars(plan.PriceCents))
		callbackData := fmt.Sprintf("%s%s", constants.PaymentConstants.CallbackPaymentPlan, plan.ID)

		keyboard = append(keyboard, []map[string]string{
			{"text": buttonText, "callback_data": callbackData},
		})
	}

	// Дополнительные кнопки
	keyboard = append(keyboard, []map[string]string{
		{"text": constants.PaymentButtonTexts.History, "callback_data": constants.PaymentConstants.CallbackPaymentHistory},
	})
	keyboard = append(keyboard, []map[string]string{
		{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
	})

	return map[string]interface{}{
		"inline_keyboard": keyboard,
	}
}

// calculateStars рассчитывает количество Stars с учетом комиссии
func (h *buyCommandHandler) calculateStars(usdCents int) int {
	baseStars := usdCents / 100
	if baseStars < 1 {
		baseStars = 1
	}
	commission := baseStars / 20 // 5%
	if commission < 1 {
		commission = 1
	}
	return baseStars + commission
}

// Вспомогательные типы (временные)
type SubscriptionPlan struct {
	ID          string
	Name        string
	Description string
	PriceCents  int
	Features    []string
}

type UserSubscription struct {
	PlanName  string
	ExpiresAt string
}
