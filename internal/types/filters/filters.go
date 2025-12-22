package filters

import "crypto_exchange_screener_bot/internal/types/analysis"

// Filter - интерфейс фильтра
type Filter interface {
	Name() string
	Apply(signal analysis.Signal) bool
	GetStats() FilterStats
}

// FilterStats - статистика фильтра
type FilterStats struct {
	TotalProcessed int64 `json:"total_processed"`
	PassedThrough  int64 `json:"passed_through"`
	FilteredOut    int64 `json:"filtered_out"`
}
