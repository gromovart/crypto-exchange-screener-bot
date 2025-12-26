// internal/analysis/filters/volume_filter.go
package filters

import (
	analysis "crypto-exchange-screener-bot/internal/core/domain/signals"
	"sync"
)

// VolumeFilter - фильтр по объему
type VolumeFilter struct {
	MinVolume float64
	stats     FilterStats
	mu        sync.RWMutex
}

// NewVolumeFilter создает новый VolumeFilter
func NewVolumeFilter(minVolume float64) *VolumeFilter {
	return &VolumeFilter{
		MinVolume: minVolume,
		stats: FilterStats{
			TotalProcessed: 0,
			PassedThrough:  0,
			FilteredOut:    0,
		},
	}
}

func (f *VolumeFilter) Name() string {
	return "volume_filter"
}

func (f *VolumeFilter) Apply(signal analysis.Signal) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.stats.TotalProcessed++

	// Проверяем объем сигнала (если он есть)
	if signal.Volume < f.MinVolume {
		f.stats.FilteredOut++
		return false
	}

	f.stats.PassedThrough++
	return true
}

func (f *VolumeFilter) GetStats() FilterStats {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.stats
}
