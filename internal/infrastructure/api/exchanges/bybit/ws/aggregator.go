// internal/infrastructure/api/exchanges/bybit/ws/aggregator.go
package ws

import (
	bybit "crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"sync"
	"time"
)

// liqEvent — одна ликвидация, записанная в скользящее окно
type liqEvent struct {
	sizeUSD   float64
	isLong    bool // true = лонг ликвидирован (Side=="Sell")
	timestamp time.Time
}

// SlidingWindowAggregator агрегирует ликвидации в скользящем окне
type SlidingWindowAggregator struct {
	mu        sync.Mutex
	windows   map[string][]liqEvent
	windowDur time.Duration
}

// NewSlidingWindowAggregator создает агрегатор с заданной шириной окна
func NewSlidingWindowAggregator(windowDur time.Duration) *SlidingWindowAggregator {
	return &SlidingWindowAggregator{
		windows:   make(map[string][]liqEvent),
		windowDur: windowDur,
	}
}

// Add добавляет событие ликвидации в окно для символа
func (a *SlidingWindowAggregator) Add(symbol string, e liqEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.windows[symbol] = append(a.windows[symbol], e)
}

// GetMetrics возвращает агрегированные метрики для символа за последнее окно.
// Возвращает nil если событий не было.
func (a *SlidingWindowAggregator) GetMetrics(symbol string) *bybit.LiquidationMetrics {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-a.windowDur)

	events := a.windows[symbol]
	if len(events) == 0 {
		return nil
	}

	// Отсекаем старые события
	firstValid := len(events)
	for i, ev := range events {
		if ev.timestamp.After(cutoff) {
			firstValid = i
			break
		}
	}
	events = events[firstValid:]
	a.windows[symbol] = events

	if len(events) == 0 {
		return nil
	}

	// Агрегируем
	var totalUSD, longUSD, shortUSD float64
	var longCount, shortCount int
	for _, ev := range events {
		totalUSD += ev.sizeUSD
		if ev.isLong {
			longUSD += ev.sizeUSD
			longCount++
		} else {
			shortUSD += ev.sizeUSD
			shortCount++
		}
	}

	return &bybit.LiquidationMetrics{
		Symbol:         symbol,
		TotalVolumeUSD: totalUSD,
		LongLiqVolume:  longUSD,
		ShortLiqVolume: shortUSD,
		LongLiqCount:   longCount,
		ShortLiqCount:  shortCount,
		UpdateTime:     now,
	}
}

// Symbols возвращает список символов, для которых есть данные
func (a *SlidingWindowAggregator) Symbols() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := make([]string, 0, len(a.windows))
	for sym := range a.windows {
		result = append(result, sym)
	}
	return result
}
