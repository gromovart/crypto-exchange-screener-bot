// internal/delivery/telegram/controllers/payment/controller.go
package payment

import (
	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// paymentControllerImpl реализация PaymentController
type paymentControllerImpl struct{}

// NewController создает новый контроллер платежей
func NewController() Controller {
	return &paymentControllerImpl{}
}

// HandleEvent обрабатывает событие от EventBus
func (c *paymentControllerImpl) HandleEvent(event types.Event) error {
	logger.Warn("💰 [PAYMENT CONTROLLER] Получено событие: %s", event.Type)

	// Проверяем тип события
	switch event.Type {
	case types.EventPaymentComplete:
		return c.handlePaymentComplete(event)
	case types.EventPaymentCreated:
		return c.handlePaymentCreated(event)
	case types.EventPaymentFailed:
		return c.handlePaymentFailed(event)
	case types.EventPaymentRefunded:
		return c.handlePaymentRefunded(event)
	default:
		logger.Warn("⚠️ [PAYMENT CONTROLLER] Неизвестный тип события: %s", event.Type)
		return nil
	}
}

// handlePaymentComplete обрабатывает успешный платеж
func (c *paymentControllerImpl) handlePaymentComplete(event types.Event) error {
	logger.Warn("💰💰💰 [PAYMENT CONTROLLER] УСПЕШНЫЙ ПЛАТЕЖ!")

	// Пытаемся извлечь данные
	if event.Data == nil {
		logger.Warn("⚠️ [PAYMENT CONTROLLER] Данные события отсутствуют")
		return nil
	}

	// Преобразуем данные в map
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		logger.Warn("⚠️ [PAYMENT CONTROLLER] Неверный формат данных: %T", event.Data)
		return nil
	}

	// Логируем все поля
	logger.Warn("📋 [PAYMENT CONTROLLER] Детали платежа:")
	for key, value := range data {
		logger.Warn("   • %s: %v", key, value)
	}

	// Извлекаем основные поля
	if paymentID, ok := data["payment_id"].(string); ok {
		logger.Warn("   ✅ PaymentID: %s", paymentID)
	}
	if userID, ok := data["user_id"].(string); ok {
		logger.Warn("   👤 UserID: %s", userID)
	}
	if planID, ok := data["plan_id"].(string); ok {
		logger.Warn("   📋 PlanID: %s", planID)
	}
	if starsAmount, ok := data["stars_amount"].(int); ok {
		logger.Warn("   ⭐ Stars: %d", starsAmount)
	}
	if timestamp, ok := data["timestamp"]; ok {
		logger.Warn("   🕐 Время: %v", timestamp)
	}

	logger.Warn("✅ [PAYMENT CONTROLLER] Обработка завершена")
	return nil
}

// handlePaymentCreated обрабатывает создание платежа
func (c *paymentControllerImpl) handlePaymentCreated(event types.Event) error {
	logger.Warn("📝 [PAYMENT CONTROLLER] Платеж создан")
	logger.Warn("   • Данные: %+v", event.Data)
	return nil
}

// handlePaymentFailed обрабатывает неудачный платеж
func (c *paymentControllerImpl) handlePaymentFailed(event types.Event) error {
	logger.Warn("❌ [PAYMENT CONTROLLER] Платеж не удался")
	logger.Warn("   • Данные: %+v", event.Data)
	return nil
}

// handlePaymentRefunded обрабатывает возврат платежа
func (c *paymentControllerImpl) handlePaymentRefunded(event types.Event) error {
	logger.Warn("↩️ [PAYMENT CONTROLLER] Платеж возвращен")
	logger.Warn("   • Данные: %+v", event.Data)
	return nil
}

// GetName возвращает имя контроллера
func (c *paymentControllerImpl) GetName() string {
	return "payment_controller"
}

// GetSubscribedEvents возвращает типы событий для подписки
func (c *paymentControllerImpl) GetSubscribedEvents() []types.EventType {
	return []types.EventType{
		types.EventPaymentComplete,
		types.EventPaymentCreated,
		types.EventPaymentFailed,
		types.EventPaymentRefunded,
	}
}
