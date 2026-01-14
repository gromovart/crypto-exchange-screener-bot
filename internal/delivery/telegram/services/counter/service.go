// internal/delivery/telegram/services/counter/service.go
package counter

import (
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
)

// serviceImpl реализация CounterService
type serviceImpl struct {
	// Зависимости будут добавлены позже
	// formatter    *formatters.Formatter
	// notifier     *notifier.Notifier
	// userService  *users.Service
}

// NewService создает новый сервис счетчика
func NewService() Service {
	return &serviceImpl{}
}

// Exec выполняет обработку события счетчика
func (s *serviceImpl) Exec(params interface{}) (interface{}, error) {
	// Приводим параметры к нужному типу
	parsedParams, ok := params.(CounterParams)
	if !ok {
		return CounterResult{Processed: false},
			fmt.Errorf("неверный тип параметров: ожидается CounterParams")
	}

	if parsedParams.Event.Type != types.EventCounterSignalDetected {
		return CounterResult{Processed: false},
			fmt.Errorf("неподдерживаемый тип события: %s", parsedParams.Event.Type)
	}

	// Извлекаем данные счетчика
	counterData, err := s.extractCounterData(parsedParams.Event.Data)
	if err != nil {
		return CounterResult{Processed: false},
			fmt.Errorf("ошибка извлечения данных счетчика: %w", err)
	}

	// TODO: Реализовать полную логику обработки счетчика
	// 1. Проверить пороговые значения
	// 2. Подготовить уведомление
	// 3. Отправить через notifier

	fmt.Printf("🔢 CounterService: Обработка счетчика для %s (рост: %d, падение: %d, период: %s)\n",
		counterData.Symbol, counterData.GrowthCount, counterData.FallCount, counterData.Period)

	return CounterResult{
		Processed: true,
		Message:   fmt.Sprintf("Счетчик %s обработан", counterData.Symbol),
	}, nil
}

// CounterData данные счетчика для обработки
type counterData struct {
	Symbol          string              `json:"symbol"`
	GrowthCount     int                 `json:"growth_count"`
	FallCount       int                 `json:"fall_count"`
	Period          types.CounterPeriod `json:"period"`
	PeriodStartTime string              `json:"period_start_time"`
	LastGrowthTime  string              `json:"last_growth_time,omitempty"`
	LastFallTime    string              `json:"last_fall_time,omitempty"`
}

// extractCounterData извлекает данные счетчика из события
func (s *serviceImpl) extractCounterData(eventData interface{}) (counterData, error) {
	// Пробуем разные типы данных счетчика
	switch data := eventData.(type) {
	case types.SignalCounter:
		return counterData{
			Symbol:          data.Symbol,
			GrowthCount:     data.GrowthCount,
			FallCount:       data.FallCount,
			Period:          data.Period,
			PeriodStartTime: data.PeriodStartTime.Format("2006-01-02 15:04:05"),
			LastGrowthTime:  data.LastGrowthTime.Format("2006-01-02 15:04:05"),
			LastFallTime:    data.LastFallTime.Format("2006-01-02 15:04:05"),
		}, nil

	case types.CounterNotification:
		return counterData{
			Symbol:          data.Symbol,
			GrowthCount:     data.CurrentCount,
			FallCount:       0, // TODO: определить для fall
			Period:          data.Period,
			PeriodStartTime: data.PeriodStartTime.Format("2006-01-02 15:04:05"),
		}, nil

	default:
		// Временная заглушка
		return counterData{
			Symbol:          "BTCUSDT",
			GrowthCount:     5,
			FallCount:       2,
			Period:          types.CounterPeriod("5m"),
			PeriodStartTime: "2024-01-01 12:00:00",
		}, nil
	}
}
