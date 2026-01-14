// internal/delivery/telegram/services/counter-notification/interface.go
package counternotification

import (
	"crypto-exchange-screener-bot/internal/types"
	"time"
)

type Service interface {
	Exec(params interface{}) (interface{}, error)
}

// NotificationParams параметры для Exec
type NotificationParams struct {
	Event types.Event `json:"event"`
}

// NotificationResult результат Exec
type NotificationResult struct {
	Processed bool   `json:"processed"`
	Message   string `json:"message,omitempty"`
	SentTo    int    `json:"sent_to,omitempty"`
}

// NotificationData данные уведомления счетчика
type NotificationData struct {
	Symbol          string                  `json:"symbol"`
	SignalType      types.CounterSignalType `json:"signal_type"`
	CurrentCount    int                     `json:"current_count"`
	Period          types.CounterPeriod     `json:"period"`
	PeriodStartTime time.Time               `json:"period_start_time"`
	Timestamp       time.Time               `json:"timestamp"`
	MaxSignals      int                     `json:"max_signals"`
	Percentage      float64                 `json:"percentage"`
}
