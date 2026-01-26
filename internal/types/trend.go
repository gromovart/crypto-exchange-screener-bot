// internal/types/trend.go
package types

import (
	"time"
)

// TrendSignal сигнал тренда
type TrendSignal struct {
	Symbol        string    `json:"symbol"`
	Direction     string    `json:"direction"` // "growth" или "fall"
	ChangePercent float64   `json:"change_percent"`
	PeriodMinutes int       `json:"period_minutes"`
	Confidence    float64   `json:"confidence"`
	Timestamp     time.Time `json:"timestamp"`
	DataPoints    int       `json:"data_points"`
}

// NotificationService интерфейс сервиса уведомлений
type NotificationService interface {
	Send(signal TrendSignal) error
	SendBatch(signals []TrendSignal) error
	SetEnabled(enabled bool)
	IsEnabled() bool
	GetStats() map[string]interface{}
}
