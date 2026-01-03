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

// PriceData данные о цене
type PriceData struct {
	Symbol       string                 `json:"symbol"`
	Price        float64                `json:"price"`
	Volume24h    float64                `json:"volume_24h"`
	VolumeUSD    float64                `json:"volume_usd"` // ДОБАВЬТЕ ЭТУ СТРОКУ
	Timestamp    time.Time              `json:"timestamp"`
	Exchange     string                 `json:"exchange,omitempty"`
	Category     string                 `json:"category,omitempty"` // spot, futures
	High24h      float64                `json:"high24h,omitempty"`
	Low24h       float64                `json:"low24h,omitempty"`
	Change24h    float64                `json:"change24h,omitempty"`
	FundingRate  float64                `json:"fundingRate,omitempty"`  // только для futures
	OpenInterest float64                `json:"openInterest,omitempty"` // открытый интерес для фьючерсов
	Basis        float64                `json:"basis,omitempty"`        // базис (для фьючерсов)
	Liquidation  float64                `json:"liquidation,omitempty"`  // объем ликвидаций
	Metadata     map[string]interface{} `json:"metadata,omitempty"`     // дополнительные метаданные
}

// NotificationService интерфейс сервиса уведомлений
type NotificationService interface {
	Send(signal TrendSignal) error
	SendBatch(signals []TrendSignal) error
	SetEnabled(enabled bool)
	IsEnabled() bool
	GetStats() map[string]interface{}
}
