// internal/core/services/payment/stars_service.go
package payment

import (
	event_bus "crypto-exchange-screener-bot/internal/infrastructure/transport/event_bus"
	"crypto-exchange-screener-bot/pkg/logger"
)

// StarsService сервис обработки платежей через Telegram Stars
type StarsService struct {
	logger              logger.Logger
	subscriptionService SubscriptionService
	userManager         UserManager
	eventBus            event_bus.EventBus
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

// CreateInvoice создает инвойс для оплаты через Stars
func (s *StarsService) CreateInvoice(request CreateInvoiceRequest) (*StarsInvoice, error) {
	return s.createInvoice(request)
}

// ProcessPayment обрабатывает успешный платеж Stars
func (s *StarsService) ProcessPayment(request ProcessPaymentRequest) (*StarsPaymentResult, error) {
	return s.processPayment(request)
}

// ValidateWebhook проверяет валидность webhook от Telegram
func (s *StarsService) ValidateWebhook(data map[string]interface{}) (bool, error) {
	return s.validateWebhook(data)
}

// GetStarsAmount конвертирует USD центы в Stars
func (s *StarsService) GetStarsAmount(usdCents int) int {
	return s.getStarsAmount(usdCents)
}

// GetUsdAmount конвертирует Stars в USD центы
func (s *StarsService) GetUsdAmount(stars int) int {
	return s.getUsdAmount(stars)
}
