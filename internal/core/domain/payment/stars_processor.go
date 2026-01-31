// internal/core/domain/payment/stars_processor.go
package payment

import (
	types "crypto-exchange-screener-bot/internal/types"
	"fmt"
	"time"
)

// createInvoice реализация создания инвойса
func (s *StarsService) createInvoice(request CreateInvoiceRequest) (*StarsInvoice, error) {
	if err := s.validateInvoiceRequest(request); err != nil {
		return nil, err
	}

	starsAmount := s.calculateStarsAmount(request.SubscriptionPlan.GetPriceCents())
	payload := s.generateInvoicePayload(request.UserID, request.SubscriptionPlan.GetID())

	invoice := &StarsInvoice{
		ID:                 s.generateInvoiceID(),
		UserID:             request.UserID,
		SubscriptionPlanID: request.SubscriptionPlan.GetID(),
		StarsAmount:        starsAmount,
		FiatAmount:         request.SubscriptionPlan.GetPriceCents(),
		Currency:           "USD",
		Payload:            payload,
		Status:             PaymentStatusPending,
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(24 * time.Hour),
	}

	s.logger.Info("Создан инвойс Stars",
		"invoiceId", invoice.ID,
		"userId", request.UserID,
		"starsAmount", starsAmount,
		"plan", request.SubscriptionPlan.GetName(),
	)

	return invoice, nil
}

// processPayment реализация обработки платежа (упрощенная версия без вызова subscription)
func (s *StarsService) processPayment(request ProcessPaymentRequest) (*StarsPaymentResult, error) {
	if err := s.validatePaymentRequest(request); err != nil {
		return nil, err
	}

	invoiceData, err := s.parseInvoicePayload(request.Payload)
	if err != nil {
		return nil, err
	}

	isValid, err := s.validateTelegramPayment(
		request.TelegramPaymentID,
		request.StarsAmount,
		invoiceData,
	)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return nil, fmt.Errorf("валидация платежа не пройдена")
	}

	// Записываем транзакцию
	if err := s.recordPaymentTransaction(
		request.TelegramPaymentID,
		invoiceData.UserID,
		request.StarsAmount,
		invoiceData.SubscriptionPlanID,
	); err != nil {
		s.logger.Error("Не удалось записать транзакцию", "error", err)
	}

	s.logger.Info("Платеж Stars обработан",
		"paymentId", request.TelegramPaymentID,
		"userId", invoiceData.UserID,
		"starsAmount", request.StarsAmount,
		"planId", invoiceData.SubscriptionPlanID,
	)

	// Публикуем событие
	eventData := map[string]interface{}{
		"payment_id":   request.TelegramPaymentID,
		"user_id":      invoiceData.UserID,
		"plan_id":      invoiceData.SubscriptionPlanID,
		"stars_amount": request.StarsAmount,
		"payment_type": "stars",
		"timestamp":    time.Now(),
		"invoice_id":   invoiceData.InvoiceID,
	}

	event := types.Event{
		Type:      types.EventPaymentComplete,
		Source:    "stars_processor",
		Data:      eventData,
		Timestamp: time.Now(),
	}

	if err := s.eventBus.Publish(event); err != nil {
		s.logger.Error("Не удалось опубликовать событие платежа", "error", err)
	}

	return &StarsPaymentResult{
		Success:   true,
		PaymentID: request.TelegramPaymentID,
		UserID:    invoiceData.UserID,
		PlanID:    invoiceData.SubscriptionPlanID,
		InvoiceID: invoiceData.InvoiceID,
		Timestamp: time.Now(),
	}, nil
}
