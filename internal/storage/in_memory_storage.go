// in_memory_storage.go
package storage

import (
	"container/list"
	"crypto_exchange_screener_bot/internal/types/common"
	"crypto_exchange_screener_bot/internal/types/fetcher"
	"crypto_exchange_screener_bot/internal/types/storage"
	"regexp"
	"sort"
	"sync"
	"time"
)

// InMemoryPriceStorage реализация in-memory хранилища
type InMemoryPriceStorage struct {
	mu sync.RWMutex

	// Текущие цены
	current map[string]*storage.PriceSnapshot

	// История цен (двусторонний список для каждой пары)
	history map[string]*list.List

	// Статистика
	stats storage.StorageStats

	// Подписки
	subscriptions *SubscriptionManager

	// Конфигурация
	config *storage.StorageConfig

	// Вспомогательные структуры
	symbolsByVolume []storage.SymbolVolume
	lastCleanup     time.Time
}

// StorePrice сохраняет цену
func (s *InMemoryPriceStorage) StorePrice(symbol string, price, volume24h float64, timestamp time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем лимит символов
	if len(s.current) >= s.config.MaxSymbols && !s.SymbolExists(symbol) {
		return storage.ErrStorageFull
	}

	// Обновляем текущую цену
	snapshot := &storage.PriceSnapshot{
		Symbol:    common.Symbol(symbol),
		Price:     price,
		Volume24h: volume24h,
		Timestamp: timestamp,
	}
	s.current[symbol] = snapshot

	// Добавляем в историю
	if _, exists := s.history[symbol]; !exists {
		s.history[symbol] = list.New()
	}

	historyList := s.history[symbol]
	historyList.PushBack(common.PriceData{
		Symbol:    common.Symbol(symbol),
		Price:     price,
		Volume24h: volume24h,
		Timestamp: timestamp,
	})

	// Ограничиваем глубину истории
	if historyList.Len() > s.config.MaxHistoryPerSymbol {
		if front := historyList.Front(); front != nil {
			historyList.Remove(front)
		}
	}

	// Обновляем статистику
	s.updateStats()

	// Обновляем сортировку по объему
	s.updateSymbolVolume(symbol, volume24h)

	// Уведомляем подписчиков (без блокировки, чтобы избежать deadlock)
	go s.subscriptions.NotifyAll(symbol, price, volume24h, timestamp)

	return nil
}

// GetCurrentPrice возвращает текущую цену
func (s *InMemoryPriceStorage) GetCurrentPrice(symbol string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if snapshot, exists := s.current[symbol]; exists {
		return snapshot.Price, true
	}
	return 0, false
}

// GetCurrentSnapshot возвращает текущий снапшот
func (s *InMemoryPriceStorage) GetCurrentSnapshot(symbol string) (*storage.PriceSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, exists := s.current[symbol]
	return snapshot, exists
}

// GetAllCurrentPrices возвращает все текущие цены
func (s *InMemoryPriceStorage) GetAllCurrentPrices() map[string]storage.PriceSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]storage.PriceSnapshot, len(s.current))
	for symbol, snapshot := range s.current {
		result[symbol] = *snapshot
	}
	return result
}

// GetSymbols возвращает все символы
func (s *InMemoryPriceStorage) GetSymbols() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	symbols := make([]string, 0, len(s.current))
	for symbol := range s.current {
		symbols = append(symbols, symbol)
	}

	// Сортируем для детерминированности
	sort.Strings(symbols)
	return symbols
}

// SymbolExists проверяет существование символа
func (s *InMemoryPriceStorage) SymbolExists(symbol string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.current[symbol]
	return exists
}

// GetPriceHistory возвращает историю цен
func (s *InMemoryPriceStorage) GetPriceHistory(symbol string, limit int) ([]common.PriceData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	historyList, exists := s.history[symbol]
	if !exists {
		return nil, storage.ErrSymbolNotFound
	}

	// Если лимит не указан или больше размера, берем все
	if limit <= 0 || limit > historyList.Len() {
		limit = historyList.Len()
	}

	result := make([]common.PriceData, 0, limit)

	// Идем с конца (последние данные)
	element := historyList.Back()
	for i := 0; i < limit && element != nil; i++ {
		if priceData, ok := element.Value.(common.PriceData); ok {
			result = append(result, priceData)
		}
		element = element.Prev()
	}

	// Разворачиваем, чтобы получить правильный порядок (старые -> новые)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// GetPriceHistoryRange возвращает историю за период
func (s *InMemoryPriceStorage) GetPriceHistoryRange(symbol string, start, end time.Time) ([]common.PriceData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	historyList, exists := s.history[symbol]
	if !exists {
		return nil, storage.ErrSymbolNotFound
	}

	var result []common.PriceData

	// Проходим по всей истории
	for element := historyList.Front(); element != nil; element = element.Next() {
		if priceData, ok := element.Value.(common.PriceData); ok {
			// Проверяем попадает ли в диапазон
			if !priceData.Timestamp.Before(start) && !priceData.Timestamp.After(end) {
				result = append(result, priceData)
			}
		}
	}

	return result, nil
}

// GetLatestPrice возвращает последнюю цену
func (s *InMemoryPriceStorage) GetLatestPrice(symbol string) (*common.PriceData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	historyList, exists := s.history[symbol]
	if !exists || historyList.Len() == 0 {
		return nil, false
	}

	// Берем последний элемент
	lastElement := historyList.Back()
	if lastElement == nil {
		return nil, false
	}

	if priceData, ok := lastElement.Value.(common.PriceData); ok {
		return &priceData, true
	}

	return nil, false
}

// CalculatePriceChange рассчитывает изменение цены
func (s *InMemoryPriceStorage) CalculatePriceChange(symbol string, interval time.Duration) (*fetcher.PriceChange, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	currentSnapshot, exists := s.current[symbol]
	if !exists {
		return nil, storage.ErrSymbolNotFound
	}

	// Ищем цену за указанный интервал назад
	targetTime := time.Now().Add(-interval)

	historyList, exists := s.history[symbol]
	if !exists {
		return nil, storage.ErrSymbolNotFound
	}

	var previousPrice *common.PriceData

	// Ищем ближайшую цену к targetTime
	for element := historyList.Front(); element != nil; element = element.Next() {
		if priceData, ok := element.Value.(common.PriceData); ok {
			if priceData.Timestamp.After(targetTime) {
				previousPrice = &priceData
				break
			}
		}
	}

	if previousPrice == nil {
		// Если не нашли, берем самую старую
		if front := historyList.Front(); front != nil {
			if priceData, ok := front.Value.(common.PriceData); ok {
				previousPrice = &priceData
			}
		}
	}

	if previousPrice == nil {
		return nil, storage.ErrSymbolNotFound
	}

	// Рассчитываем изменение
	change := currentSnapshot.Price - previousPrice.Price
	changePercent := (change / previousPrice.Price) * 100

	return &fetcher.PriceChange{
		Symbol:        common.Symbol(symbol),
		CurrentPrice:  currentSnapshot.Price,
		PreviousPrice: previousPrice.Price,
		Change:        change,
		ChangePercent: changePercent,
		Interval:      interval.String(),
		Timestamp:     time.Now(),
	}, nil
}

// GetAveragePrice возвращает среднюю цену за период
func (s *InMemoryPriceStorage) GetAveragePrice(symbol string, period time.Duration) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	historyList, exists := s.history[symbol]
	if !exists {
		return 0, storage.ErrSymbolNotFound
	}

	cutoffTime := time.Now().Add(-period)
	var sum float64
	count := 0

	// Проходим с конца (новые данные сначала)
	for element := historyList.Back(); element != nil; element = element.Prev() {
		if priceData, ok := element.Value.(common.PriceData); ok {
			if priceData.Timestamp.Before(cutoffTime) {
				break
			}
			sum += priceData.Price
			count++
		}
	}

	if count == 0 {
		return 0, storage.ErrSymbolNotFound
	}

	return sum / float64(count), nil
}

// GetMinMaxPrice возвращает min и max за период
func (s *InMemoryPriceStorage) GetMinMaxPrice(symbol string, period time.Duration) (min, max float64, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	historyList, exists := s.history[symbol]
	if !exists {
		return 0, 0, storage.ErrSymbolNotFound
	}

	cutoffTime := time.Now().Add(-period)
	min = 1e9 // Большое число
	max = 0
	count := 0

	for element := historyList.Back(); element != nil; element = element.Prev() {
		if priceData, ok := element.Value.(common.PriceData); ok {
			if priceData.Timestamp.Before(cutoffTime) {
				break
			}
			if priceData.Price < min {
				min = priceData.Price
			}
			if priceData.Price > max {
				max = priceData.Price
			}
			count++
		}
	}

	if count == 0 {
		return 0, 0, storage.ErrSymbolNotFound
	}

	return min, max, nil
}

// Subscribe подписывает на обновления
func (s *InMemoryPriceStorage) Subscribe(symbol string, subscriber Subscriber) error {
	s.subscriptions.Subscribe(symbol, subscriber)
	return nil
}

// Unsubscribe отписывает от обновлений
func (s *InMemoryPriceStorage) Unsubscribe(symbol string, subscriber Subscriber) error {
	s.subscriptions.Unsubscribe(symbol, subscriber)
	return nil
}

// GetSubscriberCount возвращает количество подписчиков
func (s *InMemoryPriceStorage) GetSubscriberCount(symbol string) int {
	return s.subscriptions.GetSubscriberCount(symbol)
}

// CleanOldData очищает старые данные
func (s *InMemoryPriceStorage) CleanOldData(maxAge time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffTime := time.Now().Add(-maxAge)
	removedCount := 0

	for symbol, historyList := range s.history {
		// Удаляем старые элементы с начала списка
		for {
			front := historyList.Front()
			if front == nil {
				break
			}

			if priceData, ok := front.Value.(common.PriceData); ok {
				if priceData.Timestamp.Before(cutoffTime) {
					historyList.Remove(front)
					removedCount++
				} else {
					break // Дошли до новых данных
				}
			} else {
				historyList.Remove(front)
			}
		}

		// Если история пустая, удаляем символ
		if historyList.Len() == 0 {
			delete(s.history, symbol)
			delete(s.current, symbol)
		}
	}

	s.updateStats()
	return removedCount, nil
}

// TruncateHistory ограничивает историю
func (s *InMemoryPriceStorage) TruncateHistory(symbol string, maxPoints int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	historyList, exists := s.history[symbol]
	if !exists {
		return storage.ErrSymbolNotFound
	}

	// Удаляем лишние элементы с начала
	for historyList.Len() > maxPoints {
		if front := historyList.Front(); front != nil {
			historyList.Remove(front)
		}
	}

	return nil
}

// RemoveSymbol удаляет символ
func (s *InMemoryPriceStorage) RemoveSymbol(symbol string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.current, symbol)
	delete(s.history, symbol)

	s.updateStats()

	// Уведомляем подписчиков (асинхронно)
	go func() {
		s.subscriptions.NotifySymbolRemoved(symbol)
	}()

	return nil
}

// Clear очищает все данные
func (s *InMemoryPriceStorage) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.current = make(map[string]*storage.PriceSnapshot)
	s.history = make(map[string]*list.List)
	s.symbolsByVolume = nil

	s.updateStats()

	return nil
}

// GetStats возвращает статистику
func (s *InMemoryPriceStorage) GetStats() storage.StorageStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stats
}

// GetSymbolStats возвращает статистику по символу
func (s *InMemoryPriceStorage) GetSymbolStats(symbol string) (storage.SymbolStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, exists := s.current[symbol]
	if !exists {
		return storage.SymbolStats{}, storage.ErrSymbolNotFound
	}

	historyList, exists := s.history[symbol]
	if !exists || historyList.Len() == 0 {
		return storage.SymbolStats{}, storage.ErrSymbolNotFound
	}

	// Находим первую и последнюю цены
	var firstData, lastData common.PriceData

	if front := historyList.Front(); front != nil {
		if data, ok := front.Value.(common.PriceData); ok {
			firstData = data
		}
	}

	if back := historyList.Back(); back != nil {
		if data, ok := back.Value.(common.PriceData); ok {
			lastData = data
		}
	}

	// Рассчитываем средний объем
	var totalVolume float64
	volumeCount := 0

	for element := historyList.Front(); element != nil; element = element.Next() {
		if priceData, ok := element.Value.(common.PriceData); ok {
			totalVolume += priceData.Volume24h
			volumeCount++
		}
	}

	avgVolume := 0.0
	if volumeCount > 0 {
		avgVolume = totalVolume / float64(volumeCount)
	}

	// Рассчитываем изменение за 24 часа
	priceChange24h := 0.0
	if lastData.Price > 0 && firstData.Price > 0 {
		priceChange24h = ((lastData.Price - firstData.Price) / firstData.Price) * 100
	}

	return storage.SymbolStats{
		Symbol:         common.Symbol(symbol),
		DataPoints:     historyList.Len(),
		FirstTimestamp: firstData.Timestamp,
		LastTimestamp:  lastData.Timestamp,
		CurrentPrice:   snapshot.Price,
		AvgVolume24h:   avgVolume,
		PriceChange24h: priceChange24h,
	}, nil
}

// FindSymbolsByPattern ищет символы по шаблону
func (s *InMemoryPriceStorage) FindSymbolsByPattern(pattern string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []string

	// Простой поиск по подстроке
	for symbol := range s.current {
		if pattern == "*" || pattern == "" {
			result = append(result, symbol)
		} else if matched, _ := regexp.MatchString(pattern, symbol); matched {
			result = append(result, symbol)
		} else if contains(symbol, pattern) {
			result = append(result, symbol)
		}
	}

	sort.Strings(result)
	return result, nil
}

// GetTopSymbolsByVolume возвращает топ символов по объему
func (s *InMemoryPriceStorage) GetTopSymbolsByVolume(limit int) ([]storage.SymbolVolume, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.symbolsByVolume) {
		limit = len(s.symbolsByVolume)
	}

	if limit > len(s.symbolsByVolume) {
		limit = len(s.symbolsByVolume)
	}

	result := make([]storage.SymbolVolume, limit)
	copy(result, s.symbolsByVolume[:limit])

	return result, nil
}

// Вспомогательные методы

func (s *InMemoryPriceStorage) updateStats() {
	s.stats = storage.StorageStats{
		TotalSymbols:        len(s.current),
		TotalDataPoints:     s.calculateTotalDataPoints(),
		MemoryUsageBytes:    s.estimateMemoryUsage(),
		OldestTimestamp:     s.findOldestTimestamp(),
		NewestTimestamp:     s.findNewestTimestamp(),
		UpdateRatePerSecond: 0, // Можно рассчитать позже
		StorageType:         "in_memory",
		MaxHistoryPerSymbol: s.config.MaxHistoryPerSymbol,
		RetentionPeriod:     s.config.RetentionPeriod,
	}
}

func (s *InMemoryPriceStorage) calculateTotalDataPoints() int64 {
	var total int64
	for _, historyList := range s.history {
		total += int64(historyList.Len())
	}
	return total
}

func (s *InMemoryPriceStorage) estimateMemoryUsage() int64 {
	// Оценка использования памяти
	// Каждый PriceData ~ 40 байт, каждый PriceSnapshot ~ 40 байт
	dataPoints := s.calculateTotalDataPoints()
	symbols := int64(len(s.current))

	return dataPoints*40 + symbols*40
}

func (s *InMemoryPriceStorage) findOldestTimestamp() time.Time {
	var oldest time.Time
	first := true

	for _, historyList := range s.history {
		if front := historyList.Front(); front != nil {
			if priceData, ok := front.Value.(common.PriceData); ok {
				if first || priceData.Timestamp.Before(oldest) {
					oldest = priceData.Timestamp
					first = false
				}
			}
		}
	}

	if first {
		return time.Time{}
	}
	return oldest
}

func (s *InMemoryPriceStorage) findNewestTimestamp() time.Time {
	var newest time.Time

	for _, snapshot := range s.current {
		if snapshot.Timestamp.After(newest) {
			newest = snapshot.Timestamp
		}
	}

	return newest
}

func (s *InMemoryPriceStorage) updateSymbolVolume(symbol string, volume float64) {
	// Находим символ в списке
	found := false
	for i, sv := range s.symbolsByVolume {
		if sv.Symbol == symbol {
			s.symbolsByVolume[i].Volume = volume
			found = true
			break
		}
	}

	// Если не нашли, добавляем
	if !found {
		s.symbolsByVolume = append(s.symbolsByVolume, storage.SymbolVolume{
			Symbol: symbol,
			Volume: volume,
		})
	}

	// Сортируем по убыванию объема
	sort.Slice(s.symbolsByVolume, func(i, j int) bool {
		return s.symbolsByVolume[i].Volume > s.symbolsByVolume[j].Volume
	})
}

func (s *InMemoryPriceStorage) startCleanupRoutine() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Since(s.lastCleanup) >= s.config.CleanupInterval {
				s.CleanOldData(s.config.RetentionPeriod)
				s.lastCleanup = time.Now()
			}
		}
	}
}

// Вспомогательная функция для поиска подстроки
func contains(s, substr string) bool {
	if substr == "" {
		return true
	}

	// Простой поиск без учета регистра
	substr = toUpper(substr)
	sUpper := toUpper(s)

	// Если есть wildcard *
	if idx := index(substr, "*"); idx != -1 {
		if idx == 0 {
			// * в начале
			return hasSuffix(sUpper, substr[1:])
		} else if idx == len(substr)-1 {
			// * в конце
			return hasPrefix(sUpper, substr[:len(substr)-1])
		} else {
			// * в середине
			parts := split(substr, "*")
			return hasPrefix(sUpper, parts[0]) && hasSuffix(sUpper, parts[1])
		}
	}

	return index(sUpper, substr) != -1
}

// Простые строковые функции для избежания импорта strings
func toUpper(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'a' <= c && c <= 'z' {
			c -= 'a' - 'A'
		}
		result = append(result, c)
	}
	return string(result)
}

func index(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	result = append(result, s[start:])
	return result
}
