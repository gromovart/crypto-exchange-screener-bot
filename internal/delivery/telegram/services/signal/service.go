// internal/delivery/telegram/services/signal/service.go
// internal/delivery/telegram/services/signal/service.go
package signal

import (
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"time"
)

// serviceImpl реализация SignalService
type serviceImpl struct {
	// Зависимости будут добавлены позже
	// formatter    *formatters.Formatter
	// notifier     *notifier.Notifier
	// userService  *users.Service
}

// NewService создает новый сервис сигналов
func NewService() Service {
	return &serviceImpl{}
}

// SignalParams параметры для Exec
type signalParams struct {
	Event types.Event `json:"event"`
}

// SignalResult результат Exec
type signalResult struct {
	Processed bool   `json:"processed"`
	Message   string `json:"message,omitempty"`
}

// Exec выполняет обработку сигнала
func (s *serviceImpl) Exec(params interface{}) (interface{}, error) {
	// Приводим параметры к нужному типу
	parsedParams, ok := params.(signalParams)
	if !ok {
		return signalResult{Processed: false},
			fmt.Errorf("неверный тип параметров: ожидается signalParams")
	}

	if parsedParams.Event.Type != types.EventSignalDetected {
		return signalResult{Processed: false},
			fmt.Errorf("неподдерживаемый тип события: %s", parsedParams.Event.Type)
	}

	// Извлекаем данные сигнала
	signalData, err := s.extractSignalData(parsedParams.Event.Data)
	if err != nil {
		return signalResult{Processed: false},
			fmt.Errorf("ошибка извлечения данных сигнала: %w", err)
	}

	// TODO: Реализовать полную логику обработки сигнала
	// 1. Получить список пользователей для этого символа/типа сигнала
	// 2. Проверить настройки каждого пользователя
	// 3. Отформатировать сообщение через formatter
	// 4. Отправить уведомления через notifier

	fmt.Printf("📡 SignalService: Обработка сигнала %s для %s (%.2f%%)\n",
		signalData.SignalType, signalData.Symbol, signalData.ChangePercent)

	return signalResult{
		Processed: true,
		Message:   fmt.Sprintf("Сигнал %s обработан", signalData.Symbol),
	}, nil
}

// SignalData данные сигнала для обработки
type signalData struct {
	Symbol        string                 `json:"symbol"`
	SignalType    string                 `json:"signal_type"`
	Direction     string                 `json:"direction"`
	ChangePercent float64                `json:"change_percent"`
	Confidence    float64                `json:"confidence"`
	Timestamp     time.Time              `json:"timestamp"`
	Price         float64                `json:"price,omitempty"`
	Volume24h     float64                `json:"volume_24h,omitempty"`
	PeriodMinutes int                    `json:"period_minutes,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// extractSignalData извлекает данные сигнала из события
func (s *serviceImpl) extractSignalData(eventData interface{}) (signalData, error) {
	// TODO: Реализовать преобразование для разных структур сигналов
	// Проверяем тип данных и преобразуем в зависимости от типа события

	// Временная заглушка - возвращаем тестовые данные
	return signalData{
		Symbol:        "BTCUSDT",
		SignalType:    "growth",
		Direction:     "growth",
		ChangePercent: 2.5,
		Confidence:    0.8,
		Timestamp:     time.Now(),
		Price:         50000.0,
		Volume24h:     1000000.0,
		PeriodMinutes: 5,
		Metadata:      map[string]interface{}{"source": "test"},
	}, nil
}
