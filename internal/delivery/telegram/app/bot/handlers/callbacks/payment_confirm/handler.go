// internal/delivery/telegram/app/bot/handlers/callbacks/payment_confirm/handler.go
package payment_confirm

import (
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	telegram_http "crypto-exchange-screener-bot/internal/delivery/telegram/app/http_client"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
)

// paymentConfirmHandler обработчик подтверждения платежа
type paymentConfirmHandler struct {
	*base.BaseHandler
	config      *config.Config
	starsClient *telegram_http.StarsClient
}

// NewHandler создает новый обработчик подтверждения платежа
func NewHandler(deps Dependencies) handlers.Handler {
	// Получаем токен бота из конфига
	botToken := deps.Config.Telegram.BotToken
	if botToken == "" {
		logger.Error("❌ TELEGRAM_BOT_TOKEN не настроен в конфигурации")
		return nil
	}

	// Создаем базовый URL с токеном бота
	baseURL := fmt.Sprintf("https://api.telegram.org/bot%s/", botToken)

	// ВАЖНО: Создаем StarsClient с пустым providerToken,
	// но сохраняем его в структуре Dependencies как требуется
	starsClient := telegram_http.NewStarsClient(baseURL, "")

	return &paymentConfirmHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "payment_confirm_handler",
			Command: constants.PaymentConstants.CallbackPaymentConfirm,
			Type:    handlers.TypeCallback,
		},
		config:      deps.Config,
		starsClient: starsClient, // Используем созданный клиент
	}
}

// Execute выполняет обработку подтверждения платежа
func (h *paymentConfirmHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Извлекаем ID плана из callback_data (формат: payment_confirm:basic)
	planID := h.extractPlanID(params.Data)
	if planID == "" {
		return handlers.HandlerResult{}, fmt.Errorf("неверный формат callback: %s", params.Data)
	}

	// Получаем информацию о плане
	plan := h.getPlanByID(planID)
	if plan == nil {
		return handlers.HandlerResult{}, fmt.Errorf("план не найден: %s", planID)
	}

	// Создаем инвойс через Telegram Stars API
	invoiceLink, err := h.createTelegramInvoice(params.User.ID, plan)
	if err != nil {
		logger.Error("❌ Ошибка создания инвойса: %v", err)
		return handlers.HandlerResult{
			Message: "❌ *Ошибка создания платежной формы*\n\nПожалуйста, попробуйте позже или обратитесь в поддержку.",
			Keyboard: map[string]interface{}{
				"inline_keyboard": [][]map[string]string{
					{
						{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
					},
				},
			},
		}, nil
	}

	// Сообщение с инструкцией по оплате
	message := h.createPaymentMessage(plan, invoiceLink)
	keyboard := h.createPaymentKeyboard(planID, invoiceLink)

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"plan_id":      planID,
			"user_id":      params.User.ID,
			"invoice_link": invoiceLink,
			"stars_amount": h.calculateStars(plan.PriceCents),
		},
	}, nil
}

// createTelegramInvoice создает инвойс через Telegram API
func (h *paymentConfirmHandler) createTelegramInvoice(userID int, plan *SubscriptionPlan) (string, error) {
	// Создаем уникальный payload для инвойса
	// Формат: sub_{plan_id}_{user_id}_{timestamp}
	timestamp := time.Now().Unix()
	payload := fmt.Sprintf("sub_%s_%d_%d", plan.ID, userID, timestamp)

	title := fmt.Sprintf("Подписка: %s", plan.Name)
	description := fmt.Sprintf("Доступ к функциям тарифа %s", plan.Name)
	starsAmount := h.calculateStars(plan.PriceCents)

	logger.Info("💰 Создание инвойса Stars: план=%s, пользователь=%d, сумма=%d Stars, payload=%s",
		plan.ID, userID, starsAmount, payload)

	// Используем StarsClient для создания инвойса
	// provider_token будет пустым внутри клиента (мы передали "" при создании)
	invoiceLink, err := h.starsClient.CreateSubscriptionInvoice(title, description, payload, starsAmount)
	if err != nil {
		logger.Error("❌ Ошибка создания инвойса: %v", err)
		return "", err
	}

	logger.Info("✅ Инвойс создан успешно: %s", invoiceLink)
	return invoiceLink, nil
}

// extractPlanID извлекает ID плана из callback_data
func (h *paymentConfirmHandler) extractPlanID(callbackData string) string {
	prefix := constants.PaymentConstants.CallbackPaymentConfirm
	if len(callbackData) <= len(prefix) {
		return ""
	}
	return callbackData[len(prefix):]
}

// getPlanByID возвращает план по ID
func (h *paymentConfirmHandler) getPlanByID(planID string) *SubscriptionPlan {
	plans := map[string]*SubscriptionPlan{
		"basic": {
			ID:         "basic",
			Name:       "📱 Доступ на 1 месяц",
			PriceCents: 1500, // ⭐ $15.00 = 1500 центов
		},
		"pro": {
			ID:         "pro",
			Name:       "🚀 Доступ на 3 месяца",
			PriceCents: 3000, // ⭐ $30.00 = 3000 центов
		},
		"enterprise": {
			ID:         "enterprise",
			Name:       "🏢 Доступ на 12 месяцев",
			PriceCents: 7500, // ⭐ $75.00 = 7500 центов
		},
	}
	return plans[planID]
}

// createPaymentMessage создает сообщение с инструкцией по оплате
func (h *paymentConfirmHandler) createPaymentMessage(plan *SubscriptionPlan, invoiceLink string) string {
	starsAmount := h.calculateStars(plan.PriceCents)
	usdPrice := float64(plan.PriceCents) / 100

	message := "💳 *Оплата через Telegram Stars*\n\n"
	message += fmt.Sprintf("План: *%s*\n", plan.Name)
	message += fmt.Sprintf("Сумма: *%d Stars* ($%.2f)\n\n", starsAmount, usdPrice)

	message += "📋 *Как оплатить:*\n"
	message += "1. Убедитесь, что у вас есть Stars в @wallet\n"
	message += "2. Нажмите кнопку '💳 Оплатить сейчас'\n"
	message += "3. Подтвердите платеж в открывшемся окне\n"
	message += "4. После успешной оплаты вы получите уведомление\n\n"

	message += "🔄 *После оплаты:*\n"
	message += "• Подписка активируется автоматически\n"
	message += "• Вы получите подтверждение в этот чат\n"
	message += "• Все функции плана будут доступны сразу\n\n"

	message += "❓ *Проблемы с оплатой?*\n"
	message += "Напишите в поддержку через /help"

	return message
}

// createPaymentKeyboard создает клавиатуру для оплаты
func (h *paymentConfirmHandler) createPaymentKeyboard(planID, invoiceLink string) interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "💳 Оплатить сейчас", "url": invoiceLink},
			},
			{
				{"text": "🔄 Проверить статус", "callback_data": fmt.Sprintf("%s%s",
					constants.PaymentConstants.CallbackPaymentCheck, planID)},
			},
			{
				{"text": constants.PaymentButtonTexts.BackToPlans, "callback_data": constants.PaymentConstants.CommandBuy},
				{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}

// calculateStars рассчитывает количество Stars с учетом комиссии Telegram
// Согласно документации, комиссия уже включена в цену Stars для пользователя
func (h *paymentConfirmHandler) calculateStars(usdCents int) int {
	return usdCents / 3 // 1500/3 = 500, 3000/3 = 1000, 7500/3 = 2500
}

// SubscriptionPlan вспомогательный тип для планов подписки
type SubscriptionPlan struct {
	ID         string
	Name       string
	PriceCents int
}
