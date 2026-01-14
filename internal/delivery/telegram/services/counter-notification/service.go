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

// NotificationParams параметры для Exec
type notificationParams struct {
	Event types.Event `json:"event"`
}

// NotificationResult результат Exec
type notificationResult struct {
	Processed bool   `json:"processed"`
	Message   string `json:"message,omitempty"`
	SentTo    int    `json:"sent_to,omitempty"`
}

// Exec выполняет обработку запроса на уведомление счетчика
func (s *serviceImpl) Exec(params interface{}) (interface{}, error) {
	// Приводим параметры к нужному типу
	parsedParams, ok := params.(notificationParams)
	if !ok {
		return notificationResult{Processed: false},
			fmt.Errorf("неверный тип параметров: ожидается notificationParams")
	}

	if parsedParams.Event.Type != types.EventCounterNotificationRequest {
		return notificationResult{Processed: false},
			fmt.Errorf("неподдерживаемый тип события: %s", parsedParams.Event.Type)
	}

	// Извлекаем данные уведомления
	notificationData, err := s.extractNotificationData(parsedParams.Event.Data)
	if err != nil {
		return notificationResult{Processed: false},
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

	return notificationResult{
		Processed: true,
		Message:   fmt.Sprintf("Уведомление для %s отправлено", notificationData.Symbol),
		SentTo:    1, // TODO: реальное количество получателей
	}, nil
}

// NotificationData данные уведомления счетчика
type notificationData struct {
	Symbol          string                  `json:"symbol"`
	SignalType      types.CounterSignalType `json:"signal_type"`
	CurrentCount    int                     `json:"current_count"`
	Period          types.CounterPeriod     `json:"period"`
	PeriodStartTime time.Time               `json:"period_start_time"`
	Timestamp       time.Time               `json:"timestamp"`
	MaxSignals      int                     `json:"max_signals"`
	Percentage      float64                 `json:"percentage"`
}

// extractNotificationData извлекает данные уведомления из события
func (s *serviceImpl) extractNotificationData(eventData interface{}) (notificationData, error) {
	// Пробуем преобразовать в CounterNotification
	if notification, ok := eventData.(types.CounterNotification); ok {
		return notificationData{
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
	return notificationData{
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
