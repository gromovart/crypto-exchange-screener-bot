// internal/core/services/payment/stars_processor.go
package payment

import (
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

// processPayment реализация обработки платежа
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

	result, err := s.activateSubscription(
		invoiceData.UserID,
		invoiceData.SubscriptionPlanID,
		request.TelegramPaymentID,
	)
	if err != nil {
		return nil, err
	}

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
	)

	if err := s.publishPaymentSuccessEvent(
		invoiceData.UserID,
		request.TelegramPaymentID,
		request.StarsAmount,
	); err != nil {
		s.logger.Error("Не удалось опубликовать событие", "error", err)
	}

	return &StarsPaymentResult{
		Success:                 true,
		PaymentID:               request.TelegramPaymentID,
		SubscriptionActiveUntil: result.ActiveUntil,
		InvoiceID:               invoiceData.InvoiceID,
	}, nil
}
