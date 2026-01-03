package storage

import (
	"time"
)

// PriceData представляет точку данных цены (обновленная версия)
type PriceData struct {
	Symbol       string                 `json:"symbol"`
	Price        float64                `json:"price"`
	Volume24h    float64                `json:"volume_24h"` // Объем в базовой валюте
	VolumeUSD    float64                `json:"volume_usd"` // Объем в USDT
	Timestamp    time.Time              `json:"timestamp"`
	OpenInterest float64                `json:"open_interest"`         // Открытый интерес ← ДОБАВЛЕНО!
	FundingRate  float64                `json:"funding_rate"`          // Ставка фандинга ← ДОБАВЛЕНО!
	Change24h    float64                `json:"change_24h"`            // Изменение за 24ч ← ДОБАВЛЕНО!
	High24h      float64                `json:"high_24h"`              // Максимум за 24ч ← ДОБАВЛЕНО!
	Low24h       float64                `json:"low_24h"`               // Минимум за 24ч ← ДОБАВЛЕНО!
	Category     string                 `json:"category,omitempty"`    // Категория (spot/futures)
	Basis        float64                `json:"basis,omitempty"`       // Базис (для фьючерсов)
	Liquidation  float64                `json:"liquidation,omitempty"` // Объем ликвидаций
	Metadata     map[string]interface{} `json:"metadata,omitempty"`    // Дополнительные метаданные
}

// PriceSnapshot текущий снапшот цены (обновленная версия)
// PriceSnapshot текущий снапшот цены
type PriceSnapshot struct {
	Symbol       string    `json:"symbol"`
	Price        float64   `json:"price"`
	Volume24h    float64   `json:"volume_24h"`
	VolumeUSD    float64   `json:"volume_usd"`
	Timestamp    time.Time `json:"timestamp"`
	OpenInterest float64   `json:"open_interest"`
	FundingRate  float64   `json:"funding_rate"`
	Change24h    float64   `json:"change_24h"`
	High24h      float64   `json:"high_24h"`
	Low24h       float64   `json:"low_24h"`
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
	SymbolsWithOI       int           `json:"symbols_with_oi"`      // Добавлено
	SymbolsWithFunding  int           `json:"symbols_with_funding"` // Добавлено
}

// PriceHistoryRequest запрос истории цен
type PriceHistoryRequest struct {
	Symbol    string    `json:"symbol"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// PriceChange изменение цены (обновленная версия)
type PriceChange struct {
	Symbol        string    `json:"symbol"`
	CurrentPrice  float64   `json:"current_price"`
	PreviousPrice float64   `json:"previous_price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Interval      string    `json:"interval"`
	Timestamp     time.Time `json:"timestamp"`
	VolumeUSD     float64   `json:"volume_usd,omitempty"` // ← ДОБАВЛЕНО!
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
