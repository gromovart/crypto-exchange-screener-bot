// internal/core/domain/payment/stars_processor.go
package payment

import (
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"time"
)

// createInvoice реализация создания инвойса
func (s *StarsService) createInvoice(request CreateInvoiceRequest) (*StarsInvoice, error) {
	if err := s.validateInvoiceRequest(request); err != nil {
		return nil, err
	}

	// Генерируем данные инвойса
	starsAmount := s.calculateStarsAmount(request.SubscriptionPlan.GetPriceCents())
	payload := s.generateInvoicePayload(request.UserID, request.SubscriptionPlan.GetID())

	// Создаем объект инвойса
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

	// Если есть starsClient, создаем реальный инвойс через Telegram API
	if s.starsClient != nil {
		// Подготавливаем данные для Telegram инвойса
		title := fmt.Sprintf("Подписка: %s", request.SubscriptionPlan.GetName())
		description := fmt.Sprintf("Оплата подписки через Telegram Stars (%d Stars)", starsAmount)

		// Создаем инвойс через Telegram API
		invoiceLink, err := s.starsClient.CreateSubscriptionInvoice(title, description, payload, starsAmount)
		if err != nil {
			s.logger.Error("Ошибка создания инвойса через Telegram API",
				"error", err,
				"userId", request.UserID,
				"plan", request.SubscriptionPlan.GetName(),
			)
			return nil, fmt.Errorf("ошибка создания Telegram инвойса: %w", err)
		}

		// Сохраняем ссылку на инвойс
		invoice.InvoiceURL = invoiceLink

		s.logger.Info("Создан Telegram Stars инвойс",
			"invoiceId", invoice.ID,
			"userId", request.UserID,
			"starsAmount", starsAmount,
			"plan", request.SubscriptionPlan.GetName(),
			"invoiceLink", invoiceLink,
		)
	} else {
		// Заглушка для разработки (без реального клиента)
		invoice.InvoiceURL = fmt.Sprintf("https://t.me/%s?start=%s",
			s.botUsername,
			payload,
		)

		s.logger.Warn("Создан локальный инвойс (Telegram клиент не доступен)",
			"invoiceId", invoice.ID,
			"userId", request.UserID,
			"starsAmount", starsAmount,
			"invoiceUrl", invoice.InvoiceURL,
		)
	}

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

	// Публикуем событие через интерфейс
	eventData := CreatePaymentEventData(
		request.TelegramPaymentID,
		invoiceData.UserID,
		invoiceData.SubscriptionPlanID,
		request.StarsAmount,
		"stars",
		invoiceData.InvoiceID,
	)

	if err := s.eventPublisher.PublishPaymentEvent(types.EventPaymentComplete, eventData.ToMap()); err != nil {
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

// ToMap конвертирует PaymentEventData в map
func (d PaymentEventData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"payment_id":   d.PaymentID,
		"user_id":      d.UserID,
		"plan_id":      d.PlanID,
		"stars_amount": d.StarsAmount,
		"payment_type": d.PaymentType,
		"timestamp":    d.Timestamp,
		"invoice_id":   d.InvoiceID,
		"metadata":     d.Metadata,
	}
}
