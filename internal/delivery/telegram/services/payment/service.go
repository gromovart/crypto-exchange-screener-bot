// internal/delivery/telegram/services/payment/service.go
package payment

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/payment"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	"crypto-exchange-screener-bot/pkg/logger"
)

// serviceImpl реализация Service
type serviceImpl struct {
	paymentService      *payment.PaymentService
	subscriptionService *subscription.Service
	userService         *users.Service
}

// NewService создает новый сервис обработки платежей
func NewService(deps Dependencies) Service {
	return &serviceImpl{
		paymentService:      deps.PaymentService,
		subscriptionService: deps.SubscriptionService,
		userService:         deps.UserService,
	}
}

// Exec выполняет операции с платежами
func (s *serviceImpl) Exec(params PaymentParams) (PaymentResult, error) {
	switch params.Action {
	case "pre_checkout":
		return s.handlePreCheckoutQuery(params)
	case "successful_payment":
		return s.handleSuccessfulPayment(params)
	case "activate_subscription":
		return s.activateSubscription(params)
	case "create_invoice":
		return s.createInvoice(params)
	default:
		return PaymentResult{}, fmt.Errorf("неподдерживаемое действие: %s", params.Action)
	}
}

// handlePreCheckoutQuery обрабатывает pre_checkout_query от Telegram
func (s *serviceImpl) handlePreCheckoutQuery(params PaymentParams) (PaymentResult, error) {
	logger.Info("Обработка pre_checkout_query для пользователя %d", params.UserID)

	// Извлекаем данные из params.Data
	queryID, _ := params.Data["query_id"].(string)
	invoicePayload, _ := params.Data["invoice_payload"].(string)
	totalAmount, _ := params.Data["total_amount"].(int)
	currency, _ := params.Data["currency"].(string)

	// Проверяем валюту (должна быть XTR для Stars)
	if currency != "XTR" {
		return PaymentResult{
			Success: false,
			Message: "Неверная валюта. Используйте Telegram Stars (XTR).",
		}, nil
	}

	// Проверяем дубликаты платежей
	isDuplicate, err := s.checkDuplicatePayment(queryID)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("ошибка проверки дубликата: %w", err)
	}

	if isDuplicate {
		return PaymentResult{
			Success: false,
			Message: "Этот платеж уже был обработан.",
		}, nil
	}

	// Валидируем платеж
	isValid, err := s.validatePayment(invoicePayload, totalAmount, params.UserID)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("ошибка валидации: %w", err)
	}

	if !isValid {
		return PaymentResult{
			Success: false,
			Message: "Неверные данные платежа.",
		}, nil
	}

	logger.Info("Pre-checkout запрос подтвержден: %s", queryID)

	return PaymentResult{
		Success:   true,
		Message:   "Платеж подтвержден.",
		PaymentID: queryID,
		Metadata: map[string]interface{}{
			"query_id":        queryID,
			"amount":          totalAmount,
			"currency":        currency,
			"validated_at":    time.Now(),
			"user_id":         params.UserID,
			"invoice_payload": invoicePayload,
		},
	}, nil
}

// handleSuccessfulPayment обрабатывает successful_payment от Telegram
func (s *serviceImpl) handleSuccessfulPayment(params PaymentParams) (PaymentResult, error) {
	logger.Info("Обработка successful_payment для пользователя %d", params.UserID)

	// Извлекаем данные
	telegramPaymentID, _ := params.Data["telegram_payment_charge_id"].(string)
	invoicePayload, _ := params.Data["invoice_payload"].(string)
	totalAmount, _ := params.Data["total_amount"].(int)

	// Парсим payload для получения данных подписки
	planID, userIDStr, err := s.parseInvoicePayload(invoicePayload)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("ошибка парсинга payload: %w", err)
	}

	// Проверяем что user_id совпадает
	userID, _ := strconv.Atoi(userIDStr)
	if userID != params.UserID {
		return PaymentResult{}, fmt.Errorf("несоответствие user_id в платеже")
	}

	// ⭐ 1. Обрабатываем платеж через сервис ядра (он сохранит в БД)
	ctx := context.Background()
	paymentRequest := payment.ProcessPaymentRequest{
		Payload:           invoicePayload,
		TelegramPaymentID: telegramPaymentID,
		StarsAmount:       totalAmount,
	}

	result, err := s.paymentService.ProcessPayment(ctx, paymentRequest)
	if err != nil {
		logger.Error("❌ Ошибка обработки платежа в PaymentService: %v", err)
		return PaymentResult{}, fmt.Errorf("ошибка обработки платежа: %w", err)
	}

	// ⭐ 2. Получаем ID платежа из результата
	var paymentID *int64
	if result.InvoiceID != "" {
		if id, err := strconv.ParseInt(result.InvoiceID, 10, 64); err == nil {
			paymentID = &id
			logger.Info("✅ Получен ID платежа из БД: %d", id)
		}
	}

	// ⭐ 3. Активируем подписку
	activationParams := PaymentParams{
		Action: "activate_subscription",
		UserID: params.UserID,
		Data: map[string]interface{}{
			"plan_id":    planID,
			"payment_id": paymentID,
		},
	}

	activationResult, err := s.activateSubscription(activationParams)
	if err != nil {
		logger.Error("❌ Ошибка активации подписки: %v", err)

		// Проверяем на уже существующую подписку
		if strings.Contains(err.Error(), "у пользователя уже есть активная подписка") {
			// Возвращаем успех, но с предупреждением
			return PaymentResult{
				Success:        true,
				Message:        "Платеж успешно обработан. У вас уже есть активная подписка.",
				PaymentID:      telegramPaymentID,
				StarsAmount:    totalAmount,
				SubscriptionID: "",
				Metadata: map[string]interface{}{
					"payment_id":    telegramPaymentID,
					"db_payment_id": paymentID,
					"plan_id":       planID,
					"stars_amount":  totalAmount,
					"processed_at":  time.Now(),
					"warning":       "existing_subscription",
				},
			}, nil
		}

		return PaymentResult{}, fmt.Errorf("ошибка активации подписки: %w", err)
	}

	// ⭐ 4. Обновляем платеж с subscription_id
	if paymentID != nil && activationResult.SubscriptionID != "" {
		subID, err := strconv.ParseInt(activationResult.SubscriptionID, 10, 64)
		if err != nil {
			// Если не удалось распарсить, пытаемся извлечь число из строки
			logger.Error("❌ Ошибка парсинга subscription_id '%s': %v",
				activationResult.SubscriptionID, err)

			// Пробуем извлечь число из строки (например, если пришло "sub_123")
			re := regexp.MustCompile(`\d+`)
			numbers := re.FindAllString(activationResult.SubscriptionID, -1)
			if len(numbers) > 0 {
				if id, err := strconv.ParseInt(numbers[0], 10, 64); err == nil {
					subID = id
					logger.Info("✅ Извлечен subscription_id из строки: %d", subID)
				} else {
					logger.Error("❌ Не удалось извлечь число из '%s'", activationResult.SubscriptionID)
					return PaymentResult{}, fmt.Errorf("неверный формат subscription_id: %s", activationResult.SubscriptionID)
				}
			} else {
				return PaymentResult{}, fmt.Errorf("неверный формат subscription_id: %s", activationResult.SubscriptionID)
			}
		}

		// Обновляем платеж
		updateCtx := context.Background()
		if err := s.paymentService.UpdatePaymentWithSubscription(updateCtx, *paymentID, subID); err != nil {
			logger.Error("⚠️ Не удалось обновить платеж с subscription_id: %v", err)
		} else {
			logger.Info("✅ Платеж %d обновлен: subscription_id=%d", *paymentID, subID)
		}
	}

	logger.Info("✅ Платеж успешно обработан: %s, подписка: %s", telegramPaymentID, planID)

	return PaymentResult{
		Success:        true,
		Message:        "✅ *Платеж успешно обработан!*\n\nПодписка активирована.",
		PaymentID:      telegramPaymentID,
		SubscriptionID: activationResult.SubscriptionID,
		StarsAmount:    totalAmount,
		ActivatedUntil: activationResult.ActivatedUntil,
		Metadata: map[string]interface{}{
			"payment_id":      telegramPaymentID,
			"db_payment_id":   paymentID,
			"plan_id":         planID,
			"stars_amount":    totalAmount,
			"processed_at":    time.Now(),
			"subscription_id": activationResult.SubscriptionID,
			"activated_until": activationResult.ActivatedUntil,
		},
	}, nil
}

// activateSubscription активирует подписку пользователя
func (s *serviceImpl) activateSubscription(params PaymentParams) (PaymentResult, error) {
	planID, _ := params.Data["plan_id"].(string)
	paymentIDObj, _ := params.Data["payment_id"].(interface{})

	logger.Info("Активация подписки %s для пользователя %d", planID, params.UserID)

	// Преобразуем paymentID в *int64
	var paymentIDPtr *int64
	if paymentIDObj != nil {
		switch v := paymentIDObj.(type) {
		case int64:
			paymentIDPtr = &v
		case int:
			id := int64(v)
			paymentIDPtr = &id
		case *int64:
			paymentIDPtr = v
		default:
			logger.Warn("⚠️ Неизвестный тип payment_id: %T", v)
		}
	}

	// Создаем подписку через сервис ядра
	ctx := context.Background()
	subscription, err := s.subscriptionService.CreateSubscription(ctx, params.UserID, planID, paymentIDPtr, false)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("ошибка создания подписки: %w", err)
	}

	// Определяем дату истечения
	var activatedUntil time.Time
	if subscription.CurrentPeriodEnd != nil {
		activatedUntil = *subscription.CurrentPeriodEnd
	} else {
		activatedUntil = time.Now().Add(30 * 24 * time.Hour)
	}

	return PaymentResult{
		Success:        true,
		SubscriptionID: strconv.Itoa(subscription.ID),
		ActivatedUntil: activatedUntil,
		Metadata: map[string]interface{}{
			"plan_id":         planID,
			"payment_id":      paymentIDPtr,
			"activated_at":    time.Now(),
			"subscription_id": subscription.ID,
		},
	}, nil
}

// createInvoice создает инвойс для оплаты
func (s *serviceImpl) createInvoice(params PaymentParams) (PaymentResult, error) {
	planID, _ := params.Data["plan_id"].(string)

	logger.Info("Создание инвойс для плана %s, пользователь %d", planID, params.UserID)

	// Получаем план подписки
	plan, err := s.subscriptionService.GetPlan(planID)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("ошибка получения плана: %w", err)
	}

	// Создаем адаптер плана
	planAdapter := &subscriptionPlanAdapter{plan: plan}

	// Создаем инвойс через сервис ядра
	userIDStr := strconv.Itoa(params.UserID)
	invoiceRequest := payment.CreateInvoiceRequest{
		UserID:           userIDStr,
		SubscriptionPlan: planAdapter,
	}

	ctx := context.Background()
	invoice, err := s.paymentService.CreateInvoice(ctx, invoiceRequest)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("ошибка создания инвойса: %w", err)
	}

	return PaymentResult{
		Success:     true,
		InvoiceURL:  invoice.InvoiceURL,
		StarsAmount: invoice.StarsAmount,
		Message:     "Инвойс успешно создан.",
		PaymentID:   invoice.ID,
		Metadata: map[string]interface{}{
			"plan_id":      planID,
			"invoice_id":   invoice.ID,
			"stars_amount": invoice.StarsAmount,
			"created_at":   time.Now(),
			"expires_at":   invoice.ExpiresAt,
		},
	}, nil
}

// Вспомогательные методы
func (s *serviceImpl) checkDuplicatePayment(paymentID string) (bool, error) {
	return false, nil
}

func (s *serviceImpl) validatePayment(payload string, amount int, userID int) (bool, error) {
	if payload == "" || amount <= 0 {
		return false, nil
	}
	return true, nil
}

func (s *serviceImpl) parseInvoicePayload(payload string) (planID, userID string, err error) {
	parts := strings.Split(payload, "_")
	if len(parts) < 4 || parts[0] != "sub" {
		return "", "", fmt.Errorf("неверный формат payload")
	}
	return parts[1], parts[2], nil
}

// subscriptionPlanAdapter адаптер для models.Plan
type subscriptionPlanAdapter struct {
	plan *models.Plan
}

func (a *subscriptionPlanAdapter) GetID() string {
	return a.plan.Code
}

func (a *subscriptionPlanAdapter) GetName() string {
	return a.plan.Name
}

func (a *subscriptionPlanAdapter) GetPriceCents() int {
	if a.plan.StarsPriceMonthly > 0 {
		return a.plan.StarsPriceMonthly
	}
	switch a.plan.Code {
	case "basic":
		return 299
	case "pro":
		return 999
	case "enterprise":
		return 2499
	default:
		return 100
	}
}
