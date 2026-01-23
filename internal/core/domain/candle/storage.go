// internal/core/domain/candle/storage.go
package candle

import (
	"sort"
	"sync"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"
)

// CandleStorage - хранилище свечей
type CandleStorage struct {
	mu sync.RWMutex

	// Активные свечи (текущие, незакрытые)
	activeCandles map[string]map[string]*Candle // symbol -> period -> candle

	// История свечей (закрытые)
	candleHistory map[string]map[string][]*Candle // symbol -> period -> candles

	// Конфигурация
	config CandleConfig

	// Статистика
	stats CandleStats
}

// NewCandleStorage создает новое хранилище свечей
func NewCandleStorage(config CandleConfig) *CandleStorage {
	if len(config.SupportedPeriods) == 0 {
		config.SupportedPeriods = []string{"5m", "15m", "30m", "1h", "4h", "1d"}
	}
	if config.MaxHistory <= 0 {
		config.MaxHistory = 1000
	}

	return &CandleStorage{
		activeCandles: make(map[string]map[string]*Candle),
		candleHistory: make(map[string]map[string][]*Candle),
		config:        config,
		stats: CandleStats{
			PeriodsCount: make(map[string]int),
		},
	}
}

// SaveActiveCandle сохраняет активную свечу
func (cs *CandleStorage) SaveActiveCandle(candle *Candle) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	symbol := candle.Symbol
	period := candle.Period

	// Инициализируем карту если нужно
	if _, exists := cs.activeCandles[symbol]; !exists {
		cs.activeCandles[symbol] = make(map[string]*Candle)
	}

	// Сохраняем свечу
	cs.activeCandles[symbol][period] = candle

	// Обновляем статистику
	cs.updateStats()
}

// GetActiveCandle получает активную свечу
func (cs *CandleStorage) GetActiveCandle(symbol, period string) (*Candle, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if symbolCandles, exists := cs.activeCandles[symbol]; exists {
		if candle, exists := symbolCandles[period]; exists {
			return candle, true
		}
	}

	return nil, false
}

// CloseAndArchiveCandle закрывает свечу и архивирует
func (cs *CandleStorage) CloseAndArchiveCandle(candle *Candle) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	symbol := candle.Symbol
	period := candle.Period

	// Закрываем свечу
	candle.IsClosed = true
	candle.EndTime = time.Now()

	// Удаляем из активных
	if symbolCandles, exists := cs.activeCandles[symbol]; exists {
		delete(symbolCandles, period)
		if len(symbolCandles) == 0 {
			delete(cs.activeCandles, symbol)
		}
	}

	// Добавляем в историю
	cs.addToHistory(candle)

	// Обновляем статистику
	cs.updateStats()

	logger.Debug("📊 Архивирована свеча %s %s: %.6f → %.6f (%.2f%%)",
		symbol, period, candle.Open, candle.Close,
		((candle.Close-candle.Open)/candle.Open)*100)
}

// addToHistory добавляет свечу в историю
func (cs *CandleStorage) addToHistory(candle *Candle) {
	symbol := candle.Symbol
	period := candle.Period

	// Инициализируем историю если нужно
	if _, exists := cs.candleHistory[symbol]; !exists {
		cs.candleHistory[symbol] = make(map[string][]*Candle)
	}
	if _, exists := cs.candleHistory[symbol][period]; !exists {
		cs.candleHistory[symbol][period] = make([]*Candle, 0)
	}

	// Добавляем свечу
	history := cs.candleHistory[symbol][period]
	history = append(history, candle)

	// Сортируем по времени (старые -> новые)
	sort.Slice(history, func(i, j int) bool {
		return history[i].StartTime.Before(history[j].StartTime)
	})

	// Ограничиваем глубину истории
	if len(history) > cs.config.MaxHistory {
		history = history[len(history)-cs.config.MaxHistory:]
	}

	cs.candleHistory[symbol][period] = history
}

// GetHistory возвращает историю свечей
func (cs *CandleStorage) GetHistory(symbol, period string, limit int) ([]*Candle, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if symbolHistory, exists := cs.candleHistory[symbol]; exists {
		if periodHistory, exists := symbolHistory[period]; exists {
			if limit <= 0 || limit > len(periodHistory) {
				limit = len(periodHistory)
			}

			// Возвращаем последние limit свечей
			start := len(periodHistory) - limit
			if start < 0 {
				start = 0
			}

			result := make([]*Candle, limit)
			copy(result, periodHistory[start:])
			return result, nil
		}
	}

	return nil, nil // Возвращаем nil вместо ошибки
}

// GetLatestCandle возвращает последнюю свечу
func (cs *CandleStorage) GetLatestCandle(symbol, period string) (*Candle, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Сначала проверяем активные свечи
	if candle, exists := cs.GetActiveCandle(symbol, period); exists {
		return candle, true
	}

	// Затем историю
	if symbolHistory, exists := cs.candleHistory[symbol]; exists {
		if periodHistory, exists := symbolHistory[period]; exists && len(periodHistory) > 0 {
			return periodHistory[len(periodHistory)-1], true
		}
	}

	return nil, false
}

// CleanupOldCandles очищает старые свечи
func (cs *CandleStorage) CleanupOldCandles(maxAge time.Duration) int {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cutoffTime := time.Now().Add(-maxAge)
	removedCount := 0

	// Очищаем историю
	for symbol, periodHistory := range cs.candleHistory {
		for period, candles := range periodHistory {
			var filtered []*Candle
			for _, candle := range candles {
				if candle.EndTime.After(cutoffTime) {
					filtered = append(filtered, candle)
				} else {
					removedCount++
				}
			}
			cs.candleHistory[symbol][period] = filtered
		}

		// Удаляем пустые записи
		if len(cs.candleHistory[symbol]) == 0 {
			delete(cs.candleHistory, symbol)
		}
	}

	// Очищаем активные свечи (те, что слишком долго активны)
	for symbol, periodCandles := range cs.activeCandles {
		for period, candle := range periodCandles {
			if time.Since(candle.StartTime) > maxAge*2 {
				delete(periodCandles, period)
				removedCount++
				logger.Warn("⚠️ Удалена старая активная свеча %s %s (возраст: %v)",
					symbol, period, time.Since(candle.StartTime))
			}
		}

		if len(cs.activeCandles[symbol]) == 0 {
			delete(cs.activeCandles, symbol)
		}
	}

	if removedCount > 0 {
		logger.Debug("🧹 Очищено %d старых свечей (старше %v)", removedCount, maxAge)
	}

	cs.updateStats()
	return removedCount
}

// GetSymbols возвращает все символы с данными
func (cs *CandleStorage) GetSymbols() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var symbols []string
	for symbol := range cs.candleHistory {
		symbols = append(symbols, symbol)
	}
	for symbol := range cs.activeCandles {
		found := false
		for _, s := range symbols {
			if s == symbol {
				found = true
				break
			}
		}
		if !found {
			symbols = append(symbols, symbol)
		}
	}

	sort.Strings(symbols)
	return symbols
}

// GetPeriodsForSymbol возвращает периоды для символа
func (cs *CandleStorage) GetPeriodsForSymbol(symbol string) []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var periods []string

	// Периоды из истории
	if symbolHistory, exists := cs.candleHistory[symbol]; exists {
		for period := range symbolHistory {
			periods = append(periods, period)
		}
	}

	// Периоды из активных свечей
	if activeCandles, exists := cs.activeCandles[symbol]; exists {
		for period := range activeCandles {
			found := false
			for _, p := range periods {
				if p == period {
					found = true
					break
				}
			}
			if !found {
				periods = append(periods, period)
			}
		}
	}

	sort.Strings(periods)
	return periods
}

// GetStats возвращает статистику
func (cs *CandleStorage) GetStats() CandleStats {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.stats
}

// updateStats обновляет статистику
func (cs *CandleStorage) updateStats() {
	stats := CandleStats{
		PeriodsCount: make(map[string]int),
		OldestCandle: time.Now(),
		NewestCandle: time.Time{},
	}

	// Активные свечи
	activeCount := 0
	for _, periods := range cs.activeCandles {
		activeCount += len(periods)
		for period := range periods {
			stats.PeriodsCount[period]++
		}
	}
	stats.ActiveCandles = activeCount

	// Исторические свечи
	historyCount := 0
	for _, periodHistory := range cs.candleHistory {
		for period, candles := range periodHistory {
			historyCount += len(candles)
			stats.PeriodsCount[period] += len(candles)

			// Находим самую старую и новую свечу
			if len(candles) > 0 {
				if candles[0].StartTime.Before(stats.OldestCandle) {
					stats.OldestCandle = candles[0].StartTime
				}
				if candles[len(candles)-1].EndTime.After(stats.NewestCandle) {
					stats.NewestCandle = candles[len(candles)-1].EndTime
				}
			}
		}
	}
	stats.TotalCandles = activeCount + historyCount

	// Символы
	symbols := make(map[string]bool)
	for symbol := range cs.candleHistory {
		symbols[symbol] = true
	}
	for symbol := range cs.activeCandles {
		symbols[symbol] = true
	}
	stats.SymbolsCount = len(symbols)

	cs.stats = stats
}
