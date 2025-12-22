// internal/analysis/filters/rate_limit_filter.go
package filters

import (
	"crypto_exchange_screener_bot/internal/types/analysis"
	"crypto_exchange_screener_bot/internal/types/filters"
	"sync"
	"time"
)

// RateLimitFilter - фильтр, ограничивающий частоту сигналов по символу
type RateLimitFilter struct {
	minDelay   time.Duration
	lastSignal map[string]time.Time
	mu         sync.RWMutex
	stats      filters.FilterStats
}

// NewRateLimitFilter создает новый RateLimitFilter
func NewRateLimitFilter(minDelay time.Duration) *RateLimitFilter {
	return &RateLimitFilter{
		minDelay:   minDelay,
		lastSignal: make(map[string]time.Time),
		stats: filters.FilterStats{
			TotalProcessed: 0,
			PassedThrough:  0,
			FilteredOut:    0,
		},
	}
}

func (f *RateLimitFilter) Name() string {
	return "rate_limit_filter"
}

func (f *RateLimitFilter) Apply(signal analysis.Signal) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	last, exists := f.lastSignal[signal.Symbol]

	// Увеличиваем счетчик обработанных сигналов
	f.stats.TotalProcessed++

	if exists && now.Sub(last) < f.minDelay {
		f.stats.FilteredOut++
		return false
	}

	f.lastSignal[signal.Symbol] = now
	f.stats.PassedThrough++
	return true
}

func (f *RateLimitFilter) GetStats() filters.FilterStats {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.stats
}
