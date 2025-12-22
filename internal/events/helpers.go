// internal/events/helpers.go
package events

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/common"
	"crypto_exchange_screener_bot/internal/types/events"
	"time"

	"github.com/google/uuid"
)

// PublishPriceUpdate публикует событие обновления цены
func PublishPriceUpdate(bus *EventBus, source string, priceData common.PriceData, metadata map[string]interface{}) error {
	event := events.Event{
		ID:        uuid.New().String(),
		Type:      events.EventPriceUpdated,
		Source:    source,
		Timestamp: time.Now(),
		Payload:   priceData, // Структура в Payload
		Data:      metadata,  // Метаданные в Data
	}

	return bus.Publish(event)
}

// PublishSignalDetected публикует событие обнаружения сигнала
func PublishSignalDetected(bus *EventBus, source string, signal analysis.Signal, metadata map[string]interface{}) error {
	event := events.Event{
		ID:        uuid.New().String(),
		Type:      events.EventSignalDetected,
		Source:    source,
		Timestamp: time.Now(),
		Payload:   signal,   // Структура Signal в Payload
		Data:      metadata, // Метаданные в Data
	}

	return bus.Publish(event)
}

// PublishTrendSignalDetected публикует событие обнаружения тренд-сигнала
func PublishTrendSignalDetected(bus *EventBus, source string, trendSignal analysis.TrendSignal, metadata map[string]interface{}) error {
	event := events.Event{
		ID:        uuid.New().String(),
		Type:      events.EventSignalDetected,
		Source:    source,
		Timestamp: time.Now(),
		Payload:   trendSignal, // Структура TrendSignal в Payload
		Data:      metadata,    // Метаданные в Data
	}

	return bus.Publish(event)
}

// PublishError публикует событие ошибки
func PublishError(bus *EventBus, source, component string, err error, context string, metadata map[string]interface{}) error {
	errorEvent := events.ErrorEvent{
		Error:     err,
		Component: component,
		Context:   context,
	}

	event := events.Event{
		ID:        uuid.New().String(),
		Type:      events.EventError,
		Source:    source,
		Timestamp: time.Now(),
		Payload:   errorEvent, // Структура ErrorEvent в Payload
		Data:      metadata,   // Дополнительные данные в Data
	}

	return bus.Publish(event)
}

// PublishAnalysisStarted публикует событие начала анализа
func PublishAnalysisStarted(bus *EventBus, source string, symbol string, metadata map[string]interface{}) error {
	event := events.Event{
		ID:        uuid.New().String(),
		Type:      events.EventAnalysisStarted,
		Source:    source,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{ // Простые данные тоже можно в Payload
			"symbol": symbol,
			"time":   time.Now().Format(time.RFC3339),
		},
		Data: metadata,
	}

	return bus.Publish(event)
}
