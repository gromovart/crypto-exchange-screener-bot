// internal/delivery/telegram/app/bot/handlers/events/payment/precheckout/handler.go
package pre_checkout

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	"crypto-exchange-screener-bot/internal/delivery/telegram/services/payment"
	"crypto-exchange-screener-bot/pkg/logger"
)

// preCheckoutHandler реализация обработчика pre_checkout_query
type preCheckoutHandler struct {
	*base.BaseHandler
	paymentService payment.Service
}

// Execute выполняет обработку pre_checkout_query
func (h *preCheckoutHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Извлекаем данные pre_checkout_query из params.Data
	// Формат: pre_checkout_query:{query_id}:{payload}:{amount}:{currency}:{user_id}
	queryData := h.parsePreCheckoutData(params.Data)
	if queryData.QueryID == "" {
		return handlers.HandlerResult{}, fmt.Errorf("неверный формат pre_checkout данных")
	}

	// Проверяем user_id
	if params.User == nil || params.User.ID == 0 {
		return handlers.HandlerResult{}, fmt.Errorf("пользователь не авторизован")
	}

	// Обработка через payment service
	paymentParams := payment.PaymentParams{
		Action: "pre_checkout",
		UserID: params.User.ID,
		ChatID: params.ChatID,
		Data: map[string]interface{}{
			"query_id":        queryData.QueryID,
			"invoice_payload": queryData.Payload,
			"total_amount":    queryData.TotalAmount,
			"currency":        queryData.Currency,
		},
	}

	result, err := h.paymentService.Exec(paymentParams)
	if err != nil {
		return handlers.HandlerResult{}, err
	}

	// Формируем ответ для Telegram API
	return handlers.HandlerResult{
		Message: result.Message,
		Metadata: map[string]interface{}{
			"payment_id":                 result.PaymentID,
			"success":                    result.Success,
			"telegram_response_required": true,
			"telegram_method":            "answerPreCheckoutQuery",
			"telegram_params": map[string]interface{}{
				"pre_checkout_query_id": queryData.QueryID,
				"ok":                    result.Success,
				"error_message":         result.Message,
			},
		},
	}, nil
}

// preCheckoutData структура для данных pre_checkout_query
type preCheckoutData struct {
	QueryID     string
	Payload     string
	TotalAmount int
	Currency    string
	UserID      int64
}

// parsePreCheckoutData парсит данные pre_checkout_query из строки
func (h *preCheckoutHandler) parsePreCheckoutData(data string) preCheckoutData {
	// Формат: pre_checkout_query:{query_id}:{payload}:{amount}:{currency}:{user_id}
	logger.Warn("📦 Парсинг pre_checkout данных: '%s'", data)

	parts := strings.Split(data, ":")
	logger.Warn("📊 Разделено на %d частей: %v", len(parts), parts)

	if len(parts) < 6 || parts[0] != "pre_checkout_query" {
		logger.Error("❌ Неверный формат: ожидается 6 частей, получено %d, первый элемент: '%s'",
			len(parts), parts[0])
		return preCheckoutData{}
	}

	amount, _ := strconv.Atoi(parts[3])
	userID, _ := strconv.ParseInt(parts[5], 10, 64)

	logger.Warn("✅ Успешно распарсено: queryID=%s, payload=%s, amount=%d, currency=%s, userID=%d",
		parts[1], parts[2], amount, parts[4], userID)

	return preCheckoutData{
		QueryID:     parts[1],
		Payload:     parts[2],
		TotalAmount: amount,
		Currency:    parts[4],
		UserID:      userID,
	}
}
