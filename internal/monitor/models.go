// models.go
package monitor

import (
	"time"
)

// Конфигурация интервалов
type Interval string

const (
	Interval1Min   Interval = "1"
	Interval5Min   Interval = "5"
	Interval10Min  Interval = "10"
	Interval15Min  Interval = "15"
	Interval30Min  Interval = "30"
	Interval1Hour  Interval = "60"
	Interval2Hour  Interval = "120"
	Interval4Hour  Interval = "240"
	Interval8Hour  Interval = "480"
	Interval12Hour Interval = "720"
	Interval24Hour Interval = "1440"
)

// Структуры данных
type PriceData struct {
	Symbol    string
	Price     float64
	Timestamp time.Time
	Volume24h float64
}

type PriceChange struct {
	Symbol        string  `json:"symbol"`
	CurrentPrice  float64 `json:"current_price"`
	PreviousPrice float64 `json:"previous_price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Interval      string  `json:"interval"`
	Volume24h     float64 `json:"volume_24h"`
	Timestamp     string  `json:"timestamp"`
}
