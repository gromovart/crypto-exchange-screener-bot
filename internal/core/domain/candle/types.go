// internal/core/domain/candle/types.go

package candle

import (
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"time"
)

// PricePoint - точка цены для построения свечи
type PricePoint struct {
	Symbol    string
	Price     float64
	Volume    float64
	VolumeUSD float64
	Timestamp time.Time
}

// CandleConfig - конфигурация построителя
type CandleConfig struct {
	SupportedPeriods []string      // Поддерживаемые периоды
	MaxHistory       int           // Максимальная история свечей
	CleanupInterval  time.Duration // Интервал очистки
	AutoBuild        bool          // Автоматическое построение
}

// BuildResult - результат построения свечи
type BuildResult struct {
	Candle   *storage.Candle // Изменено на storage.Candle
	Error    error
	IsNew    bool
	Duration time.Duration
}

// HistoryRequest - запрос на получение истории
type HistoryRequest struct {
	Symbol   string
	Period   string
	Limit    int
	FromTime time.Time
	ToTime   time.Time
}

// CandleStats - статистика свечей
type CandleStats struct {
	TotalCandles  int
	ActiveCandles int
	SymbolsCount  int
	PeriodsCount  map[string]int
	OldestCandle  time.Time
	NewestCandle  time.Time
	QueueSize     int
	BuildErrors   int
	BuildSuccess  int
}
