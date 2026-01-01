// internal/core/domain/signals/filters/rate_limit_filter.go
package filters

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"sync"
	"time"
)

// RateLimitFilter - фильтр, ограничивающий частоту сигналов по символу
type RateLimitFilter struct {
	minDelay   time.Duration
	lastSignal map[string]time.Time
	mu         sync.RWMutex
	stats      FilterStats
}

// NewRateLimitFilter создает новый RateLimitFilter
func NewRateLimitFilter(minDelay time.Duration) *RateLimitFilter {
	return &RateLimitFilter{
		minDelay:   minDelay,
		lastSignal: make(map[string]time.Time),
		stats: FilterStats{
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

func (f *RateLimitFilter) GetStats() FilterStats {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.stats
}
