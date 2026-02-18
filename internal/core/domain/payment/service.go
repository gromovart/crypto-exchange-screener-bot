// internal/core/domain/payment/service.go
package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/models"
	invoice_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/invoice"
	payment_repo "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres/repository/payment"
	"crypto-exchange-screener-bot/pkg/logger"
)

// PaymentService сервис для работы с платежами в ядре
type PaymentService struct {
	starsService *StarsService
	paymentRepo  payment_repo.PaymentRepository
	invoiceRepo  invoice_repo.InvoiceRepository // ⭐ Добавляем репозиторий инвойсов
	logger       *logger.Logger
}

// NewPaymentService создает новый сервис платежей
func NewPaymentService(
	starsService *StarsService,
	paymentRepo payment_repo.PaymentRepository,
	invoiceRepo invoice_repo.InvoiceRepository, // ⭐ Добавляем параметр
	logger *logger.Logger,
) *PaymentService {
	return &PaymentService{
		starsService: starsService,
		paymentRepo:  paymentRepo,
		invoiceRepo:  invoiceRepo,
		logger:       logger,
	}
}

// CreateInvoice создает инвойс через Stars и сохраняет в БД
func (s *PaymentService) CreateInvoice(ctx context.Context, request CreateInvoiceRequest) (*StarsInvoice, error) {
	// Создаем инвойс через StarsService
	invoice, err := s.starsService.CreateInvoice(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания инвойса в Stars: %w", err)
	}

	// Парсим payload для получения invoiceData
	invoiceData, err := s.starsService.parseInvoicePayload(invoice.Payload)
	if err != nil {
		s.logger.Warn("⚠️ Не удалось распарсить payload: %v", err)
		invoiceData = &InvoiceData{
			SubscriptionPlanID: request.SubscriptionPlan.GetID(),
			UserID:             request.UserID,
			InvoiceID:          invoice.ID,
		}
	}

	now := time.Now()

	// Создаем metadata из invoiceData
	metadataMap := map[string]interface{}{
		"invoice_data": map[string]interface{}{
			"plan_id":          invoiceData.SubscriptionPlanID,
			"user_id":          invoiceData.UserID,
			"invoice_id":       invoiceData.InvoiceID,
			"original_payload": invoice.Payload,
		},
		"created_at": now,
	}

	metadataJSON, err := json.Marshal(metadataMap)
	if err != nil {
		s.logger.Warn("⚠️ Ошибка сериализации metadata: %v", err)
		metadataJSON = []byte("{}")
	}

	// Создаем инвойс в БД
	dbInvoice := &models.Invoice{
		UserID:      parseInt64(request.UserID),
		PlanID:      request.SubscriptionPlan.GetID(),
		ExternalID:  invoice.ID,
		Title:       fmt.Sprintf("Подписка %s", request.SubscriptionPlan.GetName()),
		Description: fmt.Sprintf("Оплата подписки %s через Telegram Stars", request.SubscriptionPlan.GetName()),
		AmountUSD:   float64(invoice.StarsAmount) / 100,
		StarsAmount: invoice.StarsAmount,
		FiatAmount:  invoice.FiatAmount,
		Currency:    "USD",
		Status:      models.InvoiceStatusPending,
		Provider:    models.InvoiceProviderTelegram,
		InvoiceURL:  invoice.InvoiceURL,
		Payload:     invoice.Payload,
		Metadata:    metadataJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   invoice.ExpiresAt,
		PaidAt:      nil,
	}

	if err := s.invoiceRepo.Create(ctx, dbInvoice); err != nil {
		s.logger.Error("❌ Не удалось создать инвойс в БД: %v", err)
		return nil, fmt.Errorf("ошибка создания инвойса в БД: %w", err)
	}

	s.logger.Info("✅ Инвойс создан в БД: ID=%d, ExternalID=%s, Metadata=%s",
		dbInvoice.ID, invoice.ID, string(metadataJSON))

	invoice.ID = fmt.Sprintf("%d", dbInvoice.ID)
	return invoice, nil
}

// ProcessPayment обрабатывает успешный платеж
func (s *PaymentService) ProcessPayment(ctx context.Context, request ProcessPaymentRequest) (*StarsPaymentResult, error) {
	// Проверяем существует ли уже платеж
	existing, _ := s.paymentRepo.GetByExternalID(ctx, request.TelegramPaymentID)
	if existing != nil {
		s.logger.Warn("⚠️ Платеж уже существует: %s", request.TelegramPaymentID)
		return &StarsPaymentResult{
			Success:   false,
			PaymentID: request.TelegramPaymentID,
			InvoiceID: fmt.Sprintf("%d", existing.ID),
		}, fmt.Errorf("платеж уже существует")
	}

	// Обрабатываем через StarsService
	result, err := s.starsService.ProcessPayment(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка обработки платежа в Stars: %w", err)
	}

	// Парсим payload для получения invoiceData
	invoiceData, err := s.starsService.parseInvoicePayload(request.Payload)
	if err != nil {
		s.logger.Warn("⚠️ Не удалось распарсить payload: %v", err)
		invoiceData = &InvoiceData{
			SubscriptionPlanID: result.PlanID,
			UserID:             result.UserID,
			InvoiceID:          "unknown",
		}
	}

	userID := parseInt64(result.UserID)
	now := time.Now()
	expiresAt := now.Add(30 * 24 * time.Hour)

	// ⭐ Правильная конвертация Stars → USD (курс 36.23 Stars = 1 USD)
	usdAmount := float64(request.StarsAmount) / 36.23
	centsAmount := int(usdAmount * 100) // 500/36.23*100 = 1380 центов = $13.80

	// ⭐ Создаем metadata из invoiceData
	metadataMap := map[string]interface{}{
		"invoice_data": map[string]interface{}{
			"plan_id":          invoiceData.SubscriptionPlanID,
			"user_id":          invoiceData.UserID,
			"invoice_id":       invoiceData.InvoiceID,
			"original_payload": request.Payload,
		},
		"stars_result": map[string]interface{}{
			"payment_id": result.PaymentID,
			"user_id":    result.UserID,
			"plan_id":    result.PlanID,
			"success":    result.Success,
		},
		"conversion": map[string]interface{}{
			"rate":         36.23,
			"stars_amount": request.StarsAmount,
			"usd_amount":   usdAmount,
			"cents":        centsAmount,
		},
		"processed_at": now,
	}

	metadataJSON, err := json.Marshal(metadataMap)
	if err != nil {
		s.logger.Warn("⚠️ Ошибка сериализации metadata: %v", err)
		metadataJSON = []byte("{}")
	}

	// ⭐ 1. Сначала создаем инвойс в БД
	invoice := &models.Invoice{
		UserID:      userID,
		PlanID:      result.PlanID,
		ExternalID:  request.TelegramPaymentID,
		Title:       fmt.Sprintf("Подписка %s", result.PlanID),
		Description: fmt.Sprintf("Оплата подписки %s через Telegram Stars", result.PlanID),
		AmountUSD:   usdAmount,           // ⭐ $13.80 для 500 Stars
		StarsAmount: request.StarsAmount, // ⭐ 500 Stars
		FiatAmount:  centsAmount,         // ⭐ 1380 центов
		Currency:    "USD",
		Status:      models.InvoiceStatusPaid,
		Provider:    models.InvoiceProviderTelegram,
		InvoiceURL:  "",
		Payload:     request.Payload,
		Metadata:    metadataJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   expiresAt,
		PaidAt:      &now,
	}

	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		s.logger.Error("❌ Не удалось создать инвойс: %v", err)
		return nil, fmt.Errorf("ошибка создания инвойса: %w", err)
	}
	s.logger.Info("✅ Инвойс создан в БД: ID=%d, ExternalID=%s, Stars=%d, USD=$%.2f",
		invoice.ID, request.TelegramPaymentID, request.StarsAmount, usdAmount)

	// ⭐ 2. Теперь создаем платеж с invoice_id
	payment := &models.Payment{
		UserID:         userID,
		SubscriptionID: nil,
		InvoiceID:      &invoice.ID, // Заполняем ID созданного инвойса
		ExternalID:     request.TelegramPaymentID,
		Amount:         usdAmount, // ⭐ $13.80
		Currency:       models.CurrencyUSD,
		StarsAmount:    request.StarsAmount, // ⭐ 500
		FiatAmount:     centsAmount,         // ⭐ 1380 центов
		PaymentType:    models.PaymentTypeStars,
		Status:         models.PaymentStatusCompleted,
		Provider:       "telegram_stars",
		Description:    fmt.Sprintf("Подписка %s", result.PlanID),
		Payload:        request.Payload,
		Metadata:       metadataJSON,
		CreatedAt:      now,
		UpdatedAt:      now,
		PaidAt:         &now,
		ExpiresAt:      &expiresAt,
	}

	s.logger.Info("💾 Конвертация: %d Stars = $%.2f (%d центов) по курсу 36.23",
		request.StarsAmount, usdAmount, centsAmount)

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		s.logger.Error("❌ Не удалось сохранить платеж в БД: %v", err)
		return nil, fmt.Errorf("ошибка сохранения платежа: %w", err)
	}

	result.InvoiceID = fmt.Sprintf("%d", payment.ID)
	s.logger.Info("✅ Платеж сохранен в БД: ID=%d, ExternalID=%s, InvoiceID=%d, Amount=$%.2f",
		payment.ID, request.TelegramPaymentID, invoice.ID, usdAmount)

	return result, nil
}

// UpdatePaymentWithSubscription обновляет платеж с ID подписки
func (s *PaymentService) UpdatePaymentWithSubscription(ctx context.Context, paymentID int64, subscriptionID int64) error {
	// 1. Получаем платеж из БД
	payment, err := s.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("ошибка получения платежа: %w", err)
	}
	if payment == nil {
		return fmt.Errorf("платеж не найден: %d", paymentID)
	}

	// 2. Обновляем subscription_id
	payment.SubscriptionID = &subscriptionID
	payment.UpdatedAt = time.Now()

	// 3. Сохраняем в БД
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("ошибка обновления платежа: %w", err)
	}

	s.logger.Info("✅ Платеж %d обновлен: subscription_id=%d", paymentID, subscriptionID)
	return nil
}

// GetPayment возвращает платеж по ID
func (s *PaymentService) GetPayment(ctx context.Context, id int64) (*models.Payment, error) {
	return s.paymentRepo.GetByID(ctx, id)
}

// GetPaymentByExternalID возвращает платеж по внешнему ID
func (s *PaymentService) GetPaymentByExternalID(ctx context.Context, externalID string) (*models.Payment, error) {
	return s.paymentRepo.GetByExternalID(ctx, externalID)
}

// GetUserPayments возвращает платежи пользователя
func (s *PaymentService) GetUserPayments(ctx context.Context, userID int64, filter models.PaymentFilter) ([]*models.Payment, error) {
	return s.paymentRepo.GetByUserID(ctx, userID, filter)
}

// Вспомогательная функция
func parseInt64(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}
