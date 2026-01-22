// internal/infrastructure/persistence/redis_storage/price_storage/structs.go
package price_storage

import "time"

// PriceChange изменение цены
type PriceChange struct {
	Symbol        string    `json:"symbol"`
	CurrentPrice  float64   `json:"current_price"`
	PreviousPrice float64   `json:"previous_price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Interval      string    `json:"interval"`
	Timestamp     time.Time `json:"timestamp"`
	VolumeUSD     float64   `json:"volume_usd,omitempty"`
}
