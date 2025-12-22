// internal/analysis/filters/confidence_filter.go
package filters

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/filters"
	"sync"
)

// ConfidenceFilter - фильтр по уверенности сигнала
type ConfidenceFilter struct {
	MinConfidence float64
	stats         filters.FilterStats
	mu            sync.RWMutex
}

// NewConfidenceFilter создает новый ConfidenceFilter
func NewConfidenceFilter(minConfidence float64) *ConfidenceFilter {
	return &ConfidenceFilter{
		MinConfidence: minConfidence,
		stats: filters.FilterStats{
			TotalProcessed: 0,
			PassedThrough:  0,
			FilteredOut:    0,
		},
	}
}

func (f *ConfidenceFilter) Name() string {
	return "confidence_filter"
}

func (f *ConfidenceFilter) Apply(signal analysis.Signal) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.stats.TotalProcessed++

	if signal.Confidence < f.MinConfidence {
		f.stats.FilteredOut++
		return false
	}

	f.stats.PassedThrough++
	return true
}

func (f *ConfidenceFilter) GetStats() filters.FilterStats {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.stats
}
