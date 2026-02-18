// internal/delivery/telegram/app/bot/middlewares/subscription.go
package middlewares

import (
	"context"
	"fmt"

	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// SubscriptionMiddleware - middleware для проверки подписки
type SubscriptionMiddleware struct {
	subscriptionService *subscription.Service
}

// NewSubscriptionMiddleware создает новый middleware подписки
func NewSubscriptionMiddleware(subscriptionService *subscription.Service) *SubscriptionMiddleware {
	return &SubscriptionMiddleware{
		subscriptionService: subscriptionService,
	}
}

// ProcessSubscription проверяет подписку пользователя
func (m *SubscriptionMiddleware) ProcessSubscription(userID int) error {
	ctx := context.Background()

	// Получаем активную подписку
	activeSub, err := m.subscriptionService.GetActiveSubscription(ctx, userID)
	if err != nil {
		logger.Error("❌ Ошибка проверки подписки для user %d: %v", userID, err)
		return fmt.Errorf("внутренняя ошибка проверки подписки")
	}

	// Если есть активная подписка - всё ок
	if activeSub != nil {
		logger.Debug("✅ У пользователя %d есть активная подписка до %v",
			userID, activeSub.CurrentPeriodEnd)
		return nil
	}

	// Получаем последнюю подписку для информации о дате окончания
	latestSub, err := m.subscriptionService.GetLatestSubscription(ctx, userID)
	if err != nil {
		logger.Error("❌ Ошибка получения истории подписок для user %d: %v", userID, err)
	}

	// Если есть истекшая подписка
	if latestSub != nil && latestSub.Status == "expired" {
		if latestSub.PlanCode == "free" {
			return m.formatFreeExpiredMessage(latestSub)
		}
		return m.formatPaidExpiredMessage(latestSub)
	}

	// Если есть подписка с другим статусом (canceled, past_due и т.д.)
	if latestSub != nil {
		return m.formatStatusMessage(latestSub)
	}

	// Нет никакой подписки
	return m.formatNoSubscriptionMessage()
}

// formatFreeExpiredMessage форматирует сообщение для истекшей free подписки
func (m *SubscriptionMiddleware) formatFreeExpiredMessage(sub *models.UserSubscription) error {
	expiredDate := "неизвестно"
	if sub.CurrentPeriodEnd != nil {
		expiredDate = sub.CurrentPeriodEnd.Format("02.01.2006 15:04")
	}

	message := fmt.Sprintf(
		"⏰ *Пробный период закончился*\n\n"+
			"Ваш бесплатный доступ истек %s.\n\n"+
			"💎 *Выберите тарифный план:*\n"+
			"• 📱 *Доступ на 1 месяц* — базовые сигналы\n"+
			"• 🚀 *Доступ на 3 месяца* — расширенные возможности\n"+
			"• 🏢 *Доступ на 12 месяцев* — максимум функций\n\n"+
			"Оплата удобными Telegram Stars ⭐\n\n"+
			"➡️ Нажмите /buy для просмотра тарифов",
		expiredDate,
	)

	return fmt.Errorf(message)
}

// formatPaidExpiredMessage форматирует сообщение для истекшей платной подписки
func (m *SubscriptionMiddleware) formatPaidExpiredMessage(sub *models.UserSubscription) error {
	expiredDate := "неизвестно"
	if sub.CurrentPeriodEnd != nil {
		expiredDate = sub.CurrentPeriodEnd.Format("02.01.2006 15:04")
	}

	// Определяем правильное название тарифа из базы данных
	planDisplayName := sub.PlanName
	if planDisplayName == "" {
		switch sub.PlanCode {
		case "basic":
			planDisplayName = "📱 Доступ на 1 месяц"
		case "pro":
			planDisplayName = "🚀 Доступ на 3 месяца"
		case "enterprise":
			planDisplayName = "🏢 Доступ на 12 месяцев"
		default:
			planDisplayName = sub.PlanCode
		}
	}

	message := fmt.Sprintf(
		"⏰ *Срок подписки истек*\n\n"+
			"Ваш тариф *%s* закончился %s.\n\n"+
			"✨ *Продлите подписку сейчас* и получите:\n"+
			"• 📈 Неограниченные сигналы\n"+
			"• ⚡ Мгновенные уведомления\n"+
			"• 🎯 Точные настройки порогов\n"+
			"• 🔔 Приоритетная поддержка\n\n"+
			"💳 Оплата через Telegram Stars\n\n"+
			"➡️ Нажмите /buy для продления",
		planDisplayName,
		expiredDate,
	)

	return fmt.Errorf(message)
}

// formatStatusMessage форматирует сообщение для других статусов
func (m *SubscriptionMiddleware) formatStatusMessage(sub *models.UserSubscription) error {
	var statusText string
	switch sub.Status {
	case "canceled":
		statusText = "отменена"
	case "past_due":
		statusText = "просрочена"
	default:
		statusText = "неактивна"
	}

	// Определяем правильное название тарифа
	planDisplayName := sub.PlanName
	if planDisplayName == "" {
		switch sub.PlanCode {
		case "free":
			planDisplayName = "🆓 Free"
		case "basic":
			planDisplayName = "📱 Доступ на 1 месяц"
		case "pro":
			planDisplayName = "🚀 Доступ на 3 месяца"
		case "enterprise":
			planDisplayName = "🏢 Доступ на 12 месяцев"
		default:
			planDisplayName = sub.PlanCode
		}
	}

	message := fmt.Sprintf(
		"⚠️ *Подписка %s*\n\n"+
			"Ваша подписка на тариф *%s* %s.\n\n"+
			"💎 Для доступа ко всем функциям бота необходимо оформить новую подписку.\n\n"+
			"➡️ Нажмите /buy для выбора тарифа",
		statusText,
		planDisplayName,
		statusText,
	)

	return fmt.Errorf(message)
}

// formatNoSubscriptionMessage форматирует сообщение для пользователей без подписки
func (m *SubscriptionMiddleware) formatNoSubscriptionMessage() error {
	message := fmt.Sprintf(
		"👋 *Добро пожаловать!*\n\n" +
			"Для доступа к боту необходима подписка.\n\n" +
			"💎 *Тарифные планы:*\n" +
			"• 📱 *Доступ на 1 месяц* — базовые сигналы роста/падения\n" +
			"• 🚀 *Доступ на 3 месяца* — расширенные настройки и фильтры\n" +
			"• 🏢 *Доступ на 12 месяцев* — все функции + приоритетная поддержка\n\n" +
			"✨ *Все тарифы включают:*\n" +
			"✅ Мгновенные сигналы роста/падения\n" +
			"✅ Настройка порогов срабатывания\n" +
			"✅ Фильтрация по объему\n" +
			"✅ Telegram уведомления\n\n" +
			"⭐ Оплата удобными Telegram Stars\n\n" +
			"➡️ Нажмите /buy для просмотра деталей",
	)

	return fmt.Errorf(message)
}

// RequireSubscription создает обертку для хэндлера с проверкой подписки
func (m *SubscriptionMiddleware) RequireSubscription(handler handlers.Handler) handlers.Handler {
	return &subscriptionWrapper{
		handler:             handler,
		subscriptionService: m.subscriptionService,
	}
}

// subscriptionWrapper обертка для хэндлера с проверкой подписки
type subscriptionWrapper struct {
	handler             handlers.Handler
	subscriptionService *subscription.Service
}

// Execute выполняет проверку подписки и вызывает оригинальный хэндлер
func (w *subscriptionWrapper) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if params.User == nil {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Создаем временный middleware для проверки
	middleware := &SubscriptionMiddleware{
		subscriptionService: w.subscriptionService,
	}

	// Проверяем подписку
	if err := middleware.ProcessSubscription(params.User.ID); err != nil {
		// Возвращаем красивое сообщение без технической ошибки
		return handlers.HandlerResult{
			Message: err.Error(),
			Keyboard: map[string]interface{}{
				"inline_keyboard": [][]map[string]string{
					{
						{"text": "💎 Купить подписку", "callback_data": "buy"},
						{"text": "🏠 Главное меню", "callback_data": "menu_main"},
					},
				},
			},
		}, nil
	}

	return w.handler.Execute(params)
}

// GetName возвращает имя обернутого хэндлера
func (w *subscriptionWrapper) GetName() string {
	return "subscription_wrapper_" + w.handler.GetName()
}

// GetCommand возвращает команду обернутого хэндлера
func (w *subscriptionWrapper) GetCommand() string {
	return w.handler.GetCommand()
}

// GetType возвращает тип обернутого хэндлера
func (w *subscriptionWrapper) GetType() handlers.HandlerType {
	return w.handler.GetType()
}
