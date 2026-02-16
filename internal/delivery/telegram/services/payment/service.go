// internal/delivery/telegram/services/payment/service.go
package payment

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/payment"
	"crypto-exchange-screener-bot/internal/core/domain/subscription"
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	payment_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/payment"
	"crypto-exchange-screener-bot/pkg/logger"
)

// serviceImpl реализация Service
type serviceImpl struct {
	paymentService      *payment.StarsService
	subscriptionService *subscription.Service
	userService         *users.Service
	paymentRepo         payment_repo.PaymentRepository // Сохраняем репозиторий
}

// NewService создает новый сервис обработки платежей
func NewService(deps Dependencies) Service {
	return &serviceImpl{
		paymentService:      deps.PaymentService,
		subscriptionService: deps.SubscriptionService,
		userService:         deps.UserService,
		paymentRepo:         deps.PaymentRepository,
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

	// ⭐ 1. СНАЧАЛА СОЗДАЕМ ЗАПИСЬ О ПЛАТЕЖЕ В БД
	var paymentID *int64
	if s.paymentRepo != nil {
		logger.Warn("💰 [DEBUG] Сохраняем платеж в БД: telegramPaymentID=%s, userID=%d, starsAmount=%d",
			telegramPaymentID, userID, totalAmount)

		now := time.Now()
		payment := &models.Payment{
			UserID:      int64(userID),
			ExternalID:  telegramPaymentID,
			StarsAmount: totalAmount,
			FiatAmount:  totalAmount * 100,
			Currency:    models.CurrencyUSD,
			Status:      models.PaymentStatusCompleted,
			PaymentType: models.PaymentTypeStars,
			CreatedAt:   now,
			PaidAt:      &now,
			Description: fmt.Sprintf("Подписка %s", planID),
			Payload:     invoicePayload,
		}

		if err := s.paymentRepo.Create(context.Background(), payment); err != nil {
			logger.Error("❌ [DEBUG] Не удалось сохранить платеж в БД: %v", err)
			return PaymentResult{}, fmt.Errorf("ошибка сохранения платежа: %w", err)
		}

		paymentID = &payment.ID
		logger.Warn("✅ [DEBUG] Платеж сохранен в БД: ID=%d, ExternalID=%s", payment.ID, telegramPaymentID)
	} else {
		logger.Warn("⚠️ [DEBUG] PaymentRepository не доступен, платеж не сохранен в БД")
	}

	// ⭐ 2. ОБРАБАТЫВАЕМ ПЛАТЕЖ ЧЕРЕЗ СЕРВИС ЯДРА
	paymentRequest := payment.ProcessPaymentRequest{
		Payload:           invoicePayload,
		TelegramPaymentID: telegramPaymentID,
		StarsAmount:       totalAmount,
	}

	_, err = s.paymentService.ProcessPayment(paymentRequest)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("ошибка обработки платежа: %w", err)
	}

	// ⭐ 3. АКТИВИРУЕМ ПОДПИСКУ С ID ПЛАТЕЖА ИЗ БД
	activationParams := PaymentParams{
		Action: "activate_subscription",
		UserID: params.UserID,
		Data: map[string]interface{}{
			"plan_id":    planID,
			"payment_id": paymentID, // Передаем ID из БД, а не внешний ID
		},
	}

	activationResult, err := s.activateSubscription(activationParams)
	if err != nil {
		// Если подписка не создалась, но платеж сохранен - нужно вернуть ошибку
		return PaymentResult{}, fmt.Errorf("ошибка активации подписки: %w", err)
	}

	// Отправляем подтверждение пользователю
	err = s.sendConfirmation(params.UserID, params.ChatID, planID, totalAmount)
	if err != nil {
		logger.Warn("Не удалось отправить подтверждение: %v", err)
	}

	logger.Info("Платеж успешно обработан: %s, подписка: %s", telegramPaymentID, planID)

	return PaymentResult{
		Success:        true,
		Message:        "Платеж успешно обработан. Подписка активирована.",
		PaymentID:      telegramPaymentID,
		SubscriptionID: activationResult.SubscriptionID,
		StarsAmount:    totalAmount,
		ActivatedUntil: activationResult.ActivatedUntil,
		Metadata: map[string]interface{}{
			"payment_id":      telegramPaymentID,
			"db_payment_id":   paymentID, // Добавляем ID из БД
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
		// Дефолт: 30 дней если не указано
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

	// Создаем адаптер плана для payment service
	planAdapter := &subscriptionPlanAdapter{plan: plan}

	// Создаем инвойс через сервис ядра
	userIDStr := strconv.Itoa(params.UserID)
	invoiceRequest := payment.CreateInvoiceRequest{
		UserID:           userIDStr,
		SubscriptionPlan: planAdapter,
	}

	invoice, err := s.paymentService.CreateInvoice(invoiceRequest)
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

// checkDuplicatePayment проверяет дубликаты платежей
func (s *serviceImpl) checkDuplicatePayment(paymentID string) (bool, error) {
	// TODO: реализовать проверку дубликатов в БД
	// временная заглушка
	return false, nil
}

// validatePayment валидирует платеж
func (s *serviceImpl) validatePayment(payload string, amount int, userID int) (bool, error) {
	// TODO: расширенная валидация через сервис ядра
	// временная базовая проверка
	if payload == "" || amount <= 0 {
		return false, nil
	}
	return true, nil
}

// parseInvoicePayload парсит payload инвойса
func (s *serviceImpl) parseInvoicePayload(payload string) (planID, userID string, err error) {
	// Формат: sub_{plan_id}_{user_id}_{nonce}
	parts := strings.Split(payload, "_")
	if len(parts) < 4 || parts[0] != "sub" {
		return "", "", fmt.Errorf("неверный формат payload")
	}
	return parts[1], parts[2], nil
}

// sendConfirmation отправляет подтверждение пользователю
func (s *serviceImpl) sendConfirmation(userID int, chatID int64, planID string, starsAmount int) error {
	// TODO: реализовать отправку через message_sender
	logger.Info("Подтверждение отправлено: пользователь %d, план %s, сумма %d Stars",
		userID, planID, starsAmount)
	return nil
}

// subscriptionPlanAdapter адаптер для models.Plan к payment.SubscriptionPlan
type subscriptionPlanAdapter struct {
	plan *models.Plan
}

func (a *subscriptionPlanAdapter) GetID() string {
	// План ID может быть string (code) для подписок
	return a.plan.Code
}

func (a *subscriptionPlanAdapter) GetName() string {
	return a.plan.Name
}

func (a *subscriptionPlanAdapter) GetPriceCents() int {
	// Используем StarsPriceMonthly для Telegram Stars
	// Конвертируем Stars в USD центы (1 Star = $0.01 = 1 цент)
	if a.plan.StarsPriceMonthly > 0 {
		return a.plan.StarsPriceMonthly
	}
	// Дефолтное значение если не указано
	switch a.plan.Code {
	case "basic":
		return 299 // $2.99
	case "pro":
		return 999 // $9.99
	case "enterprise":
		return 2499 // $24.99
	default:
		return 100 // $1.00
	}
}
