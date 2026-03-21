// internal/delivery/max/bot/middleware/subscription.go
package middleware

import (
	"context"
	"fmt"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// SubscriptionMiddleware — middleware для проверки подписки в MAX боте
type SubscriptionMiddleware struct {
	subscriptionService *subscription.Service
}

// NewSubscriptionMiddleware создаёт новый middleware подписки
func NewSubscriptionMiddleware(subscriptionService *subscription.Service) *SubscriptionMiddleware {
	return &SubscriptionMiddleware{
		subscriptionService: subscriptionService,
	}
}

// RequireSubscription создаёт обёртку для хэндлера с проверкой подписки
func (m *SubscriptionMiddleware) RequireSubscription(handler handlers.Handler) handlers.Handler {
	return &subscriptionWrapper{
		handler:             handler,
		subscriptionService: m.subscriptionService,
	}
}

// subscriptionWrapper — обёртка хэндлера с проверкой подписки
type subscriptionWrapper struct {
	handler             handlers.Handler
	subscriptionService *subscription.Service
}

// Execute проверяет подписку и вызывает оригинальный хэндлер
func (w *subscriptionWrapper) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	mid := &SubscriptionMiddleware{subscriptionService: w.subscriptionService}
	msg, ok := mid.checkSubscription(params.User.ID)
	if !ok {
		rows := [][]map[string]string{
			{kb.B(kb.Btn.Buy, kb.CbBuy)},
			{kb.B("🏠 Главное меню", kb.CbMenuMain)},
		}
		return handlers.HandlerResult{
			Message:     msg,
			Keyboard:    kb.Keyboard(rows),
			EditMessage: params.MessageID != "",
		}, nil
	}

	return w.handler.Execute(params)
}

// GetName возвращает имя обёрнутого хэндлера
func (w *subscriptionWrapper) GetName() string {
	return "subscription_wrapper_" + w.handler.GetName()
}

// GetCommand возвращает команду обёрнутого хэндлера
func (w *subscriptionWrapper) GetCommand() string {
	return w.handler.GetCommand()
}

// GetType возвращает тип обёрнутого хэндлера
func (w *subscriptionWrapper) GetType() handlers.HandlerType {
	return w.handler.GetType()
}

// checkSubscription проверяет подписку и возвращает (сообщение, ok).
// ok=true — подписка активна, ok=false — нет доступа.
func (m *SubscriptionMiddleware) checkSubscription(userID int) (string, bool) {
	ctx := context.Background()

	activeSub, err := m.subscriptionService.GetActiveSubscription(ctx, userID)
	if err != nil {
		logger.Error("❌ MAX SubscriptionMiddleware: ошибка проверки подписки user %d: %v", userID, err)
		return "⚠️ Внутренняя ошибка при проверке подписки. Попробуйте позже.", false
	}

	if activeSub != nil {
		logger.Debug("✅ MAX: активная подписка у user %d до %v", userID, activeSub.CurrentPeriodEnd)
		return "", true
	}

	// Получаем последнюю подписку для информации
	latestSub, err := m.subscriptionService.GetLatestSubscription(ctx, userID)
	if err != nil {
		logger.Error("❌ MAX SubscriptionMiddleware: ошибка получения подписки user %d: %v", userID, err)
	}

	if latestSub != nil && latestSub.Status == "expired" {
		if latestSub.PlanCode == "free" {
			return m.freeExpiredMsg(latestSub), false
		}
		return m.paidExpiredMsg(latestSub), false
	}

	if latestSub != nil {
		return m.statusMsg(latestSub), false
	}

	return m.noSubscriptionMsg(), false
}

func (m *SubscriptionMiddleware) freeExpiredMsg(sub *models.UserSubscription) string {
	expiredDate := "неизвестно"
	if sub.CurrentPeriodEnd != nil {
		expiredDate = sub.CurrentPeriodEnd.Format("02.01.2006 15:04")
	}
	return fmt.Sprintf(
		"⏰ Пробный период закончился\n\n"+
			"Ваш бесплатный доступ истёк %s.\n\n"+
			"💎 Выберите тарифный план:\n"+
			"• 🧪 Тестовый — 10 ₽\n"+
			"• 📱 1 месяц — 1 490 ₽\n"+
			"• 🚀 3 месяца — 2 490 ₽\n"+
			"• 🏢 12 месяцев — 5 990 ₽\n\n"+
			"Нажмите кнопку ниже для выбора тарифа.",
		expiredDate,
	)
}

func (m *SubscriptionMiddleware) paidExpiredMsg(sub *models.UserSubscription) string {
	expiredDate := "неизвестно"
	if sub.CurrentPeriodEnd != nil {
		expiredDate = sub.CurrentPeriodEnd.Format("02.01.2006 15:04")
	}

	planName := sub.PlanName
	if planName == "" {
		switch sub.PlanCode {
		case "basic":
			planName = "1 месяц"
		case "pro":
			planName = "3 месяца"
		case "enterprise":
			planName = "12 месяцев"
		default:
			planName = sub.PlanCode
		}
	}

	return fmt.Sprintf(
		"⏰ Срок подписки истёк\n\n"+
			"Ваш тариф «%s» закончился %s.\n\n"+
			"✨ Продлите подписку и получите:\n"+
			"• 📈 Неограниченные сигналы\n"+
			"• ⚡ Мгновенные уведомления\n"+
			"• 🎯 Точные настройки порогов\n\n"+
			"Нажмите кнопку ниже для выбора тарифа.",
		planName,
		expiredDate,
	)
}

func (m *SubscriptionMiddleware) statusMsg(sub *models.UserSubscription) string {
	var statusText string
	switch sub.Status {
	case "canceled":
		statusText = "отменена"
	case "past_due":
		statusText = "просрочена"
	default:
		statusText = "неактивна"
	}

	planName := sub.PlanName
	if planName == "" {
		switch sub.PlanCode {
		case "free":
			planName = "Free"
		case "basic":
			planName = "1 месяц"
		case "pro":
			planName = "3 месяца"
		case "enterprise":
			planName = "12 месяцев"
		default:
			planName = sub.PlanCode
		}
	}

	return fmt.Sprintf(
		"⚠️ Подписка %s\n\n"+
			"Ваша подписка на тариф «%s» %s.\n\n"+
			"Для доступа ко всем функциям необходимо оформить новую подписку.\n\n"+
			"Нажмите кнопку ниже для выбора тарифа.",
		statusText,
		planName,
		statusText,
	)
}

func (m *SubscriptionMiddleware) noSubscriptionMsg() string {
	return "👋 Добро пожаловать!\n\n" +
		"Для доступа к боту необходима подписка.\n\n" +
		"💎 Тарифные планы:\n" +
		"• 🧪 Тестовый — 10 ₽\n" +
		"• 📱 1 месяц — 1 490 ₽\n" +
		"• 🚀 3 месяца — 2 490 ₽\n" +
		"• 🏢 12 месяцев — 5 990 ₽\n\n" +
		"Нажмите кнопку ниже для выбора тарифа."
}
