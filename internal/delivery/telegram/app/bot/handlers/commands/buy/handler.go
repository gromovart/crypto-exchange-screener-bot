// internal/delivery/telegram/app/bot/handlers/commands/buy/handler.go
package buy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	currency_client "crypto-exchange-screener-bot/internal/infrastructure/http/currency"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// Кэш для предотвращения дублирования (простейшая реализация)
var (
	lastBuyCommand     = make(map[int]time.Time)
	lastBuyCommandLock sync.RWMutex
	duplicateThreshold = 2 * time.Second
)

// Dependencies зависимости хэндлера
type Dependencies struct {
	IsDev          bool
	CurrencyClient *currency_client.Client
}

// buyCommandHandler реализация обработчика команды /buy
type buyCommandHandler struct {
	*base.BaseHandler
	isDev          bool
	currencyClient *currency_client.Client
}

// NewHandler создает новый обработчик команды /buy
func NewHandler(deps ...Dependencies) handlers.Handler {
	isDev := false
	var cc *currency_client.Client
	if len(deps) > 0 {
		isDev = deps[0].IsDev
		cc = deps[0].CurrencyClient
	}
	return &buyCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "buy_command_handler",
			Command: constants.PaymentConstants.CommandBuy,
			Type:    handlers.TypeCommand,
		},
		isDev:          isDev,
		currencyClient: cc,
	}
}

// Execute выполняет обработку команды /buy
func (h *buyCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if h.isDuplicateCommand(params.User.ID) {
		logger.Debug("Пропускаем дублирующую команду /buy от пользователя %d", params.User.ID)
		return handlers.HandlerResult{
			Message: "⏳ *Команда уже обрабатывается...*\n\nПожалуйста, подождите несколько секунд.",
		}, nil
	}
	h.markCommandProcessed(params.User.ID)

	if params.User == nil || params.User.ID == 0 {
		return h.createUnauthorizedMessage()
	}

	usdRubRate := h.getRate()
	plans := h.getAvailablePlans()
	currentSubscription := h.getUserSubscription(params.User.ID)

	message := h.createPlansMessage(params.User, plans, currentSubscription, usdRubRate)
	keyboard := h.createPlansKeyboard(plans, usdRubRate)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id":          params.User.ID,
			"plans_count":      len(plans),
			"has_subscription": currentSubscription != nil,
			"usd_rub_rate":     usdRubRate,
			"timestamp":        time.Now(),
		},
	}, nil
}

func (h *buyCommandHandler) getRate() float64 {
	if h.currencyClient != nil {
		return h.currencyClient.GetUSDRUB(context.Background())
	}
	return currency_client.FallbackRate
}

func (h *buyCommandHandler) isDuplicateCommand(userID int) bool {
	lastBuyCommandLock.RLock()
	lastTime, exists := lastBuyCommand[userID]
	lastBuyCommandLock.RUnlock()
	if !exists {
		return false
	}
	return time.Since(lastTime) < duplicateThreshold
}

func (h *buyCommandHandler) markCommandProcessed(userID int) {
	lastBuyCommandLock.Lock()
	lastBuyCommand[userID] = time.Now()
	lastBuyCommandLock.Unlock()
}

func (h *buyCommandHandler) createUnauthorizedMessage() (handlers.HandlerResult, error) {
	message := "🔒 *Авторизация требуется*\n\n" +
		"Для покупки подписки необходимо авторизоваться.\n\n" +
		"Используйте кнопку ниже для входа."

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": constants.AuthButtonTexts.Login, "callback_data": constants.CallbackAuthLogin}},
			{{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain}},
		},
	}
	return handlers.HandlerResult{Message: message, Keyboard: keyboard}, nil
}

func (h *buyCommandHandler) getAvailablePlans() []*SubscriptionPlan {
	var plans []*SubscriptionPlan
	if h.isDev {
		plans = append(plans, &SubscriptionPlan{
			ID:          "test",
			Name:        "🧪 Тестовый доступ",
			Description: "• Проверка платежей\n• Действует 5 минут",
			PriceRub:    10,
		})
	}
	plans = append(plans,
		&SubscriptionPlan{
			ID:          "basic",
			Name:        "📱 1 месяц",
			Description: "• Все сигналы\n• Все виды уведомлений",
			PriceRub:    1490,
		},
		&SubscriptionPlan{
			ID:          "pro",
			Name:        "🚀 3 месяца",
			Description: "• Все сигналы\n• Все виды уведомлений\n• Приоритетная поддержка",
			PriceRub:    2490,
		},
		&SubscriptionPlan{
			ID:          "enterprise",
			Name:        "🏢 12 месяцев",
			Description: "• Все сигналы\n• Кастомные настройки\n• Поддержка 24/7",
			PriceRub:    5990,
		},
	)
	return plans
}

func (h *buyCommandHandler) getUserSubscription(userID int) *UserSubscription {
	return nil
}

func (h *buyCommandHandler) createPlansMessage(
	user *models.User,
	plans []*SubscriptionPlan,
	currentSubscription *UserSubscription,
	usdRubRate float64,
) string {
	message := "💎 *Выберите тарифный план*\n\n"

	if currentSubscription != nil {
		message += fmt.Sprintf("Ваш текущий план: *%s*\n", currentSubscription.PlanName)
		message += fmt.Sprintf("Действует до: %s\n\n", currentSubscription.ExpiresAt)
	}

	for _, plan := range plans {
		stars := calculateStars(plan.PriceRub, usdRubRate)
		message += fmt.Sprintf("*%s* — %d ₽\n", plan.Name, plan.PriceRub)
		message += fmt.Sprintf("⭐ %d Stars\n", stars)
		message += fmt.Sprintf("%s\n\n", plan.Description)
	}

	message += fmt.Sprintf("💱 Курс: %.2f ₽/$\n", usdRubRate)
	message += "\nВыберите план для продолжения:"
	return message
}

func (h *buyCommandHandler) createPlansKeyboard(plans []*SubscriptionPlan, usdRubRate float64) interface{} {
	var keyboard [][]map[string]string

	for _, plan := range plans {
		buttonText := fmt.Sprintf("%s — %d ₽", plan.Name, plan.PriceRub)
		callbackData := fmt.Sprintf("%s%s", constants.PaymentConstants.CallbackPaymentPlan, plan.ID)
		keyboard = append(keyboard, []map[string]string{
			{"text": buttonText, "callback_data": callbackData},
		})
	}

	keyboard = append(keyboard,
		[]map[string]string{
			{"text": constants.PaymentButtonTexts.History, "callback_data": constants.PaymentConstants.CallbackPaymentHistory},
		},
		[]map[string]string{
			{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
		},
	)

	return map[string]interface{}{"inline_keyboard": keyboard}
}

// calculateStars рассчитывает кол-во Stars: ceil((₽ / курс) / $0.013)
func calculateStars(priceRub int, usdRubRate float64) int {
	if usdRubRate <= 0 {
		usdRubRate = currency_client.FallbackRate
	}
	usd := float64(priceRub) / usdRubRate
	return int(math.Ceil(usd / 0.013))
}

// Вспомогательные типы
type SubscriptionPlan struct {
	ID          string
	Name        string
	Description string
	PriceRub    int
}

type UserSubscription struct {
	PlanName  string
	ExpiresAt string
}
