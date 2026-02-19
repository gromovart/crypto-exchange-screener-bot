// internal/core/domain/payment/stars_processor.go
package payment

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/internal/types"
)

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

	// ⭐ КОНВЕРТИРУЕМ USERID В INT
	userID, err := strconv.Atoi(invoiceData.UserID)
	if err != nil {
		return nil, fmt.Errorf("неверный формат user_id: %s", invoiceData.UserID)
	}

	ctx := context.Background()
	var subscription *models.UserSubscription

	// ⭐ ПРОВЕРЯЕМ ТЕКУЩУЮ ПОДПИСКУ
	activeSub, err := s.subscriptionService.GetActiveSubscription(ctx, userID)
	if err != nil {
		s.logger.Error("❌ Ошибка проверки подписки", "error", err, "userId", userID)
		return nil, fmt.Errorf("ошибка проверки подписки: %w", err)
	}

	// ⭐ ЛОГИКА СОЗДАНИЯ/ОБНОВЛЕНИЯ ПОДПИСКИ
	if activeSub != nil {
		// ⭐ ТЕСТОВЫЙ ПЛАН - создаем отдельно, не заменяя существующую
		if invoiceData.SubscriptionPlanID == "test" {
			s.logger.Info("🧪 Тестовый платеж для user %d с существующей подпиской %s",
				userID, activeSub.PlanCode)

			// Создаем тестовую подписку (она будет отдельной записью)
			subscription, err = s.subscriptionService.CreateSubscription(
				ctx,
				userID,
				"test",
				nil,
				false,
			)
			if err != nil {
				s.logger.Error("❌ Ошибка создания тестовой подписки",
					"error", err,
					"userId", userID,
				)
				return nil, fmt.Errorf("ошибка создания тестовой подписки: %w", err)
			}
			s.logger.Info("✅ Создана тестовая подписка для user %d на 5 минут", userID)
		} else if activeSub.PlanCode == models.PlanFree {
			// Есть free - делаем апгрейд
			s.logger.Info("🔄 Апгрейд FREE -> %s для user %d",
				invoiceData.SubscriptionPlanID, userID)

			subscription, err = s.subscriptionService.UpgradeSubscription(
				ctx,
				userID,
				invoiceData.SubscriptionPlanID,
				nil,
			)
			if err != nil {
				s.logger.Error("❌ Ошибка апгрейда подписки",
					"error", err,
					"userId", userID,
					"plan", invoiceData.SubscriptionPlanID,
				)
				return nil, fmt.Errorf("ошибка апгрейда подписки: %w", err)
			}
			s.logger.Info("✅ Подписка обновлена с FREE до %s для user %d",
				invoiceData.SubscriptionPlanID, userID)
		} else {
			// Есть платная подписка (не тестовая) - ошибка
			s.logger.Error("❌ У пользователя уже есть активная платная подписка",
				"userId", userID,
				"existingPlan", activeSub.PlanCode,
				"newPlan", invoiceData.SubscriptionPlanID,
			)
			return nil, fmt.Errorf("у пользователя уже есть активная подписка %s",
				activeSub.PlanCode)
		}
	} else {
		// Нет подписки - создаем новую
		s.logger.Info("➕ Создание новой подписки %s для user %d",
			invoiceData.SubscriptionPlanID, userID)

		subscription, err = s.subscriptionService.CreateSubscription(
			ctx,
			userID,
			invoiceData.SubscriptionPlanID,
			nil,
			false,
		)
		if err != nil {
			s.logger.Error("❌ Ошибка создания подписки",
				"error", err,
				"userId", userID,
				"plan", invoiceData.SubscriptionPlanID,
			)
			return nil, fmt.Errorf("ошибка создания подписки: %w", err)
		}
		s.logger.Info("✅ Создана новая подписка %s для user %d",
			invoiceData.SubscriptionPlanID, userID)
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

	s.logger.Info("💰 Платеж Stars обработан",
		"paymentId", request.TelegramPaymentID,
		"userId", invoiceData.UserID,
		"starsAmount", request.StarsAmount,
		"planId", invoiceData.SubscriptionPlanID,
		"subscriptionId", subscription.ID,
	)

	// Публикуем событие через интерфейс
	eventData := CreatePaymentEventData(
		request.TelegramPaymentID,
		invoiceData.UserID,
		invoiceData.SubscriptionPlanID,
		request.StarsAmount,
		"stars",
		invoiceData.InvoiceID,
		subscription.ID,
	)

	event := types.Event{
		Type:      types.EventPaymentComplete,
		Source:    "payment_service",
		Data:      eventData.ToMap(),
		Timestamp: time.Now(),
		Metadata: types.Metadata{
			Tags: []string{"payment"},
		},
	}

	if err := s.eventPublisher.Publish(event); err != nil {
		s.logger.Error("Не удалось опубликовать событие платежа", "error", err)
	}

	// ⭐ ВОЗВРАЩАЕМ РЕЗУЛЬТАТ С ID ПОДПИСКИ
	return &StarsPaymentResult{
		Success:        true,
		PaymentID:      request.TelegramPaymentID,
		UserID:         invoiceData.UserID,
		PlanID:         invoiceData.SubscriptionPlanID,
		InvoiceID:      invoiceData.InvoiceID,
		SubscriptionID: subscription.ID,
		Timestamp:      time.Now(),
	}, nil
}

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
			s.logger.Error("❌ Ошибка создания инвойса через Telegram API",
				"error", err,
				"userId", request.UserID,
				"plan", request.SubscriptionPlan.GetName(),
			)
			return nil, fmt.Errorf("ошибка создания Telegram инвойса: %w", err)
		}

		// Сохраняем ссылку на инвойс
		invoice.InvoiceURL = invoiceLink

		s.logger.Info("✅ Создан Telegram Stars инвойс",
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

		s.logger.Warn("⚠️ Создан локальный инвойс (Telegram клиент не доступен)",
			"invoiceId", invoice.ID,
			"userId", request.UserID,
			"starsAmount", starsAmount,
			"invoiceUrl", invoice.InvoiceURL,
		)
	}

	return invoice, nil
}
