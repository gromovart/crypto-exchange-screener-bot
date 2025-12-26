// internal/storage/types.go
package storage

import (
	"time"
)

// PriceData представляет точку данных цены
type PriceData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume24h float64   `json:"volume_24h"`
	Timestamp time.Time `json:"timestamp"`
}

// PriceSnapshot текущий снапшот цены
type PriceSnapshot struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume24h float64   `json:"volume_24h"`
	Timestamp time.Time `json:"timestamp"`
}

// StorageStats статистика хранилища
type StorageStats struct {
	TotalSymbols        int           `json:"total_symbols"`
	TotalDataPoints     int64         `json:"total_data_points"`
	MemoryUsageBytes    int64         `json:"memory_usage_bytes"`
	OldestTimestamp     time.Time     `json:"oldest_timestamp"`
	NewestTimestamp     time.Time     `json:"newest_timestamp"`
	UpdateRatePerSecond float64       `json:"update_rate_per_second"`
	StorageType         string        `json:"storage_type"`
	MaxHistoryPerSymbol int           `json:"max_history_per_symbol"`
	RetentionPeriod     time.Duration `json:"retention_period"`
}

// PriceHistory запрос истории цен
type PriceHistoryRequest struct {
	Symbol    string    `json:"symbol"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// PriceChange изменение цены
type PriceChange struct {
	Symbol        string    `json:"symbol"`
	CurrentPrice  float64   `json:"current_price"`
	PreviousPrice float64   `json:"previous_price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Interval      string    `json:"interval"`
	Timestamp     time.Time `json:"timestamp"`
}

// Ошибки хранилища
var (
	ErrSymbolNotFound  = StorageError{"symbol not found"}
	ErrStorageFull     = StorageError{"storage is full"}
	ErrInvalidLimit    = StorageError{"invalid limit"}
	ErrAlreadyExists   = StorageError{"symbol already exists"}
	ErrSubscriberError = StorageError{"subscriber error"}
)

// StorageError ошибка хранилища
type StorageError struct {
	Message string
}

func (e StorageError) Error() string {
	return e.Message
}
