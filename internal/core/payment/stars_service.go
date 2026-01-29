// internal/core/services/payment/stars_service.go
package payment

import (
	event_bus "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	types "crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StarsCommissionRate комиссия Telegram Stars (5%)
const StarsCommissionRate = 0.05

// StarsService сервис обработки платежей через Telegram Stars
type StarsService struct {
	logger              logger.Logger
	subscriptionService SubscriptionService
	userManager         UserManager
	eventBus            event_bus.EventBus
}

// SubscriptionService интерфейс сервиса подписок
type SubscriptionService interface {
	ActivateSubscription(userID, planID, paymentID string) (*ActivationResult, error)
}

// UserManager интерфейс менеджера пользователей
type UserManager interface {
	GetUser(userID string) (User, error)
}

// User интерфейс пользователя
type User interface {
	GetID() string
}

// SubscriptionPlan интерфейс плана подписки
type SubscriptionPlan interface {
	GetID() string
	GetName() string
	GetPriceCents() int
}

// ActivationResult результат активации подписки
type ActivationResult struct {
	ActiveUntil time.Time
}

// NewStarsService создает новый сервис оплаты Stars
func NewStarsService(
	subscriptionService SubscriptionService,
	userManager UserManager,
	eventBus event_bus.EventBus,
	logger logger.Logger,
) *StarsService {
	return &StarsService{
		logger:              logger,
		subscriptionService: subscriptionService,
		userManager:         userManager,
		eventBus:            eventBus,
	}
}

// CreateInvoiceRequest запрос на создание инвойса
type CreateInvoiceRequest struct {
	UserID           string
	SubscriptionPlan SubscriptionPlan
}

// ProcessPaymentRequest запрос на обработку платежа
type ProcessPaymentRequest struct {
	Payload           string
	TelegramPaymentID string
	StarsAmount       int
}

// StarsInvoice инвойс для оплаты Stars
type StarsInvoice struct {
	ID                 string
	UserID             string
	SubscriptionPlanID string
	StarsAmount        int
	FiatAmount         int
	Currency           string
	Payload            string
	Status             PaymentStatus
	CreatedAt          time.Time
	ExpiresAt          time.Time
}

// PaymentStatus статус платежа
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// StarsPaymentResult результат обработки платежа
type StarsPaymentResult struct {
	Success                 bool
	PaymentID               string
	SubscriptionActiveUntil time.Time
	InvoiceID               string
}

// InvoiceData данные из парсинга payload
type InvoiceData struct {
	UserID             string
	SubscriptionPlanID string
	InvoiceID          string
}

// CreateInvoice создает инвойс для оплаты через Stars
func (s *StarsService) CreateInvoice(request CreateInvoiceRequest) (*StarsInvoice, error) {
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

// ProcessPayment обрабатывает успешный платеж Stars
func (s *StarsService) ProcessPayment(request ProcessPaymentRequest) (*StarsPaymentResult, error) {
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

// ValidateWebhook проверяет валидность webhook от Telegram
func (s *StarsService) ValidateWebhook(data map[string]interface{}) (bool, error) {
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

// GetStarsAmount конвертирует USD центы в Stars
func (s *StarsService) GetStarsAmount(usdCents int) int {
	stars := usdCents / 100
	if stars < 1 {
		return 1
	}
	return stars
}

// GetUsdAmount конвертирует Stars в USD центы
func (s *StarsService) GetUsdAmount(stars int) int {
	return stars * 100
}

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
	starsAmount := s.GetStarsAmount(request.SubscriptionPlan.GetPriceCents())
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

// calculateStarsAmount рассчитывает сумму в Stars с учетом комиссии
func (s *StarsService) calculateStarsAmount(usdCents int) int {
	baseStars := s.GetStarsAmount(usdCents)
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

// activateSubscription активирует подписку пользователя
func (s *StarsService) activateSubscription(
	userID, planID, paymentID string,
) (*ActivationResult, error) {
	user, err := s.userManager.GetUser(userID)
	if err != nil {
		return nil, fmt.Errorf("пользователь не найден: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("пользователь не найден")
	}
	return s.subscriptionService.ActivateSubscription(userID, planID, paymentID)
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
	userID, paymentID string,
	starsAmount int,
) error {
	event := types.Event{
		Type: "payment.success",
		Data: map[string]interface{}{
			"userId":    userID,
			"paymentId": paymentID,
			"amount":    starsAmount,
			"timestamp": time.Now(),
		},
	}
	return s.eventBus.Publish(event)
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
