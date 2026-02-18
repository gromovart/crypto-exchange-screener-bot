// internal/delivery/telegram/app/bot/handlers/commands/buy/handler.go
// Добавляем проверку на дублирование

package buy

import (
	"fmt"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Кэш для предотвращения дублирования (простейшая реализация)
var (
	lastBuyCommand     = make(map[int]time.Time)
	lastBuyCommandLock sync.RWMutex
	duplicateThreshold = 2 * time.Second // Защита от дублирования в течение 2 секунд
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
	// Проверка на дублирование команды
	if h.isDuplicateCommand(params.User.ID) {
		logger.Debug("Пропускаем дублирующую команду /buy от пользователя %d", params.User.ID)
		return handlers.HandlerResult{
			Message: "⏳ *Команда уже обрабатывается...*\n\nПожалуйста, подождите несколько секунд.",
		}, nil
	}

	// Помечаем команду как обработанную
	h.markCommandProcessed(params.User.ID)

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
			"timestamp":        time.Now(),
		},
	}, nil
}

// isDuplicateCommand проверяет дублирование команды
func (h *buyCommandHandler) isDuplicateCommand(userID int) bool {
	lastBuyCommandLock.RLock()
	lastTime, exists := lastBuyCommand[userID]
	lastBuyCommandLock.RUnlock()

	if !exists {
		return false
	}

	return time.Since(lastTime) < duplicateThreshold
}

// markCommandProcessed помечает команду как обработанную
func (h *buyCommandHandler) markCommandProcessed(userID int) {
	lastBuyCommandLock.Lock()
	lastBuyCommand[userID] = time.Now()
	lastBuyCommandLock.Unlock()
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

// getAvailablePlans возвращает доступные планы
func (h *buyCommandHandler) getAvailablePlans() []*SubscriptionPlan {
	return []*SubscriptionPlan{
		{
			ID:          "basic",
			Name:        "📱 Доступ на 1 месяц",
			Description: "• Получение сигналов в течении 1 месяца\n• После завершения периода требуется ручное продление\n• Все виды уведомлений",
			PriceCents:  1500, // ⭐ $15.00
			Features:    []string{"10_symbols", "50_signals", "basic_notifications"},
		},
		{
			ID:          "pro",
			Name:        "🚀 Доступ на 3 месяца",
			Description: "• Получение сигналов в течении 3 месяцев\n• После завершения периода требуется ручное продление\n• Все виды уведомлений",
			PriceCents:  3000, // ⭐ $30.00
			Features:    []string{"50_symbols", "200_signals", "advanced_notifications", "priority_support"},
		},
		{
			ID:          "enterprise",
			Name:        "🏢 Доступ на 12 месяцев",
			Description: "• Неограниченные символы\n• После завершения периода требуется ручное продление\n• Кастомные настройки\n• Все виды уведомлений",
			PriceCents:  7500, // ⭐ $75.00
			Features:    []string{"unlimited_symbols", "1000_signals", "custom_settings", "api_access"},
		},
	}
}

// getUserSubscription возвращает текущую подписку пользователя
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
	message += "• 1 Star ≈ $0.03\n"
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
	// USD центы → Stars (1 Star = $0.03 = 3 цента)
	// 1500 центов ($15) / 3 = 500 Stars
	// 3000 центов ($30) / 3 = 1000 Stars
	// 7500 центов ($75) / 3 = 2500 Stars
	return usdCents / 3
}

// Вспомогательные типы
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
