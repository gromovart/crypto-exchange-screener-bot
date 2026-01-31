// internal/core/domain/payment/stars_utils.go
package payment

import (
	"fmt"
	"strings"
	"time"

	types "crypto-exchange-screener-bot/internal/types"

	"github.com/google/uuid"
)

// validateInvoiceRequest валидирует запрос на создание инвойса
func (s *StarsService) validateInvoiceRequest(request CreateInvoiceRequest) error {
	if request.UserID == "" {
		return fmt.Errorf("идентификатор пользователя обязателен")
	}
	if request.SubscriptionPlan == nil {
		return fmt.Errorf("план подписки обязателен")
	}
	if request.SubscriptionPlan.GetPriceCents() <= 0 {
		return fmt.Errorf("неверная цена подписки")
	}
	starsAmount := s.getStarsAmount(request.SubscriptionPlan.GetPriceCents())
	if starsAmount > 10000 { // Telegram лимит
		return fmt.Errorf("сумма превышает максимальный лимит Stars")
	}
	return nil
}

// validatePaymentRequest валидирует запрос на обработку платежа
func (s *StarsService) validatePaymentRequest(request ProcessPaymentRequest) error {
	if request.Payload == "" {
		return fmt.Errorf("payload платежа обязателен")
	}
	if request.TelegramPaymentID == "" {
		return fmt.Errorf("идентификатор платежа Telegram обязателен")
	}
	if request.StarsAmount < 1 {
		return fmt.Errorf("неверная сумма Stars")
	}
	return nil
}

// getStarsAmount конвертирует USD центы в Stars
func (s *StarsService) getStarsAmount(usdCents int) int {
	stars := usdCents / 100
	if stars < 1 {
		return 1
	}
	return stars
}

// getUsdAmount конвертирует Stars в USD центы
func (s *StarsService) getUsdAmount(stars int) int {
	return stars * 100
}

// calculateStarsAmount рассчитывает сумму в Stars с учетом комиссии
func (s *StarsService) calculateStarsAmount(usdCents int) int {
	baseStars := s.getStarsAmount(usdCents)
	commission := int(float64(baseStars) * StarsCommissionRate)
	if commission < 1 {
		commission = 1
	}
	return baseStars + commission
}

// generateInvoiceID генерирует уникальный ID инвойса
func (s *StarsService) generateInvoiceID() string {
	return "inv_" + uuid.New().String()
}

// generateInvoicePayload генерирует payload для инвойса
func (s *StarsService) generateInvoicePayload(userID, planID string) string {
	nonce := uuid.New().String()[:8]
	return fmt.Sprintf("sub_%s_%s_%s", planID, userID, nonce)
}

// parseInvoicePayload парсит payload инвойса
func (s *StarsService) parseInvoicePayload(payload string) (*InvoiceData, error) {
	parts := strings.Split(payload, "_")
	if len(parts) < 4 || parts[0] != "sub" {
		return nil, fmt.Errorf("неверный формат payload инвойса")
	}
	return &InvoiceData{
		SubscriptionPlanID: parts[1],
		UserID:             parts[2],
		InvoiceID:          "inv_" + strings.Join(parts[3:], "_"),
	}, nil
}

// validateTelegramPayment валидирует платеж через Telegram API
func (s *StarsService) validateTelegramPayment(
	paymentID string,
	starsAmount int,
	invoiceData *InvoiceData,
) (bool, error) {
	// TODO: интеграция с Telegram API для верификации платежа
	// временная заглушка для разработки
	return true, nil
}

// recordPaymentTransaction записывает транзакцию платежа
func (s *StarsService) recordPaymentTransaction(
	paymentID, userID string,
	starsAmount int,
	planID string,
) error {
	s.logger.Debug("Транзакция платежа записана",
		"paymentId", paymentID,
		"userId", userID,
		"starsAmount", starsAmount,
		"planId", planID,
	)
	return nil
}

// publishPaymentSuccessEvent публикует событие об успешном платеже
func (s *StarsService) publishPaymentSuccessEvent(
	paymentID, userID, planID, invoiceID string,
	starsAmount int,
) error {
	event := types.Event{
		Type: "payment.completed",
		Data: map[string]interface{}{
			"payment_id":   paymentID,
			"user_id":      userID,
			"plan_id":      planID,
			"stars_amount": starsAmount,
			"payment_type": "stars",
			"timestamp":    time.Now(),
			"invoice_id":   invoiceID,
		},
	}
	return s.eventBus.Publish(event)
}

// validateWebhook проверяет валидность webhook от Telegram
func (s *StarsService) validateWebhook(data map[string]interface{}) (bool, error) {
	isValidSignature, err := s.validateTelegramSignature(data)
	if err != nil {
		return false, fmt.Errorf("ошибка валидации подписи: %w", err)
	}

	hasRequiredFields := data["telegram_payment_id"] != nil &&
		data["stars_amount"] != nil &&
		data["payload"] != nil

	paymentID, _ := data["telegram_payment_id"].(string)
	isDuplicate, err := s.checkDuplicatePayment(paymentID)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки дубликата: %w", err)
	}

	return isValidSignature && hasRequiredFields && !isDuplicate, nil
}

// validateTelegramSignature валидирует HMAC подпись от Telegram
func (s *StarsService) validateTelegramSignature(data map[string]interface{}) (bool, error) {
	// TODO: реализация валидации HMAC подписи
	// временная заглушка для разработки
	return true, nil
}

// checkDuplicatePayment проверяет дубликаты платежей
func (s *StarsService) checkDuplicatePayment(paymentID string) (bool, error) {
	// TODO: проверка дубликатов платежей в БД
	// временная заглушка
	return false, nil
}
