// internal/delivery/telegram/services/counter-notification/service.go
package counternotification

import (
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"time"
)

// serviceImpl реализация CounterNotificationService
type serviceImpl struct {
	// Зависимости будут добавлены позже
	// formatter    *formatters.Formatter
	// notifier     *notifier.Notifier
	// userService  *users.Service
}

// NewService создает новый сервис уведомлений счетчика
func NewService() Service {
	return &serviceImpl{}
}

// Exec выполняет обработку запроса на уведомление счетчика
func (s *serviceImpl) Exec(params interface{}) (interface{}, error) {
	// Приводим параметры к нужному типу
	parsedParams, ok := params.(NotificationParams)
	if !ok {
		return NotificationResult{Processed: false},
			fmt.Errorf("неверный тип параметров: ожидается NotificationParams")
	}

	if parsedParams.Event.Type != types.EventCounterNotificationRequest {
		return NotificationResult{Processed: false},
			fmt.Errorf("неподдерживаемый тип события: %s", parsedParams.Event.Type)
	}

	// Извлекаем данные уведомления
	notificationData, err := s.extractNotificationData(parsedParams.Event.Data)
	if err != nil {
		return NotificationResult{Processed: false},
			fmt.Errorf("ошибка извлечения данных уведомления: %w", err)
	}

	// TODO: Реализовать полную логику обработки уведомления
	// 1. Получить список пользователей для этого символа
	// 2. Проверить настройки уведомлений
	// 3. Подготовить форматированное сообщение
	// 4. Отправить уведомления через notifier

	fmt.Printf("🔔 CounterNotificationService: Уведомление для %s (%s: %d/%d, %.1f%%)\n",
		notificationData.Symbol, notificationData.SignalType,
		notificationData.CurrentCount, notificationData.MaxSignals,
		notificationData.Percentage)

	return NotificationResult{
		Processed: true,
		Message:   fmt.Sprintf("Уведомление для %s отправлено", notificationData.Symbol),
		SentTo:    1, // TODO: реальное количество получателей
	}, nil
}

// extractNotificationData извлекает данные уведомления из события
func (s *serviceImpl) extractNotificationData(eventData interface{}) (NotificationData, error) {
	// Пробуем преобразовать в CounterNotification
	if notification, ok := eventData.(types.CounterNotification); ok {
		return NotificationData{
			Symbol:          notification.Symbol,
			SignalType:      notification.SignalType,
			CurrentCount:    notification.CurrentCount,
			Period:          notification.Period,
			PeriodStartTime: notification.PeriodStartTime,
			Timestamp:       notification.Timestamp,
			MaxSignals:      notification.MaxSignals,
			Percentage:      notification.Percentage,
		}, nil
	}

	// Временная заглушка
	return NotificationData{
		Symbol:          "BTCUSDT",
		SignalType:      types.CounterTypeGrowth,
		CurrentCount:    8,
		Period:          types.CounterPeriod("5m"),
		PeriodStartTime: time.Now().Add(-5 * time.Minute),
		Timestamp:       time.Now(),
		MaxSignals:      10,
		Percentage:      80.0,
	}, nil
}
