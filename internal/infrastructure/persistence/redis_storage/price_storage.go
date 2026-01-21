// internal/infrastructure/persistence/redis_storage/price_storage.go(переименован)
package redis_storage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/go-redis/redis/v8"
)

// RedisPriceStorage реализация PriceStorage с использованием Redis
type RedisPriceStorage struct {
	redisService *redis_service.RedisService
	client       *redis.Client
	prefix       string
	ctx          context.Context

	// Подсистемы
	cacheManager    *CacheManager
	subscriptionMgr *SubscriptionManager
	historyManager  *HistoryManager

	// Конфигурация
	config *StorageConfig
}

// NewRedisPriceStorage создает новое Redis хранилище цен
func NewRedisPriceStorage(redisService *redis_service.RedisService, config *StorageConfig) *RedisPriceStorage {
	if config == nil {
		config = &StorageConfig{
			MaxHistoryPerSymbol: 10000,
			MaxSymbols:          1000,
			CleanupInterval:     5 * time.Minute,
			RetentionPeriod:     48 * time.Hour,
		}
	}

	return &RedisPriceStorage{
		redisService:    redisService,
		prefix:          "price:",
		ctx:             context.Background(),
		cacheManager:    NewCacheManager(),
		subscriptionMgr: NewSubscriptionManager(),
		historyManager:  NewHistoryManager(),
		config:          config,
	}
}

// Initialize инициализирует Redis хранилище
func (rps *RedisPriceStorage) Initialize() error {
	if rps.redisService == nil {
		return fmt.Errorf("сервис Redis не инициализирован")
	}

	rps.client = rps.redisService.GetClient()
	if rps.client == nil {
		return fmt.Errorf("клиент Redis недоступен")
	}

	// Инициализируем подсистемы
	rps.cacheManager.Initialize(rps.client)
	rps.historyManager.Initialize(rps.client, rps.config)

	logger.Info("✅ RedisPriceStorage инициализирован")
	return nil
}

// StorePrice сохраняет цену со всеми данными
func (rps *RedisPriceStorage) StorePrice(
	symbol string,
	price, volume24h, volumeUSD float64,
	timestamp time.Time,
	openInterest float64,
	fundingRate float64,
	change24h float64,
	high24h float64,
	low24h float64,
) error {
	if rps.client == nil {
		return fmt.Errorf("клиент Redis не инициализирован")
	}

	logger.Debug("💾 RedisStorage: сохранение %s: цена=%.6f, OI=%.0f, фандинг=%.6f",
		symbol, price, openInterest, fundingRate)

	// Создаем снапшот
	snapshot := &PriceSnapshot{
		Symbol:       symbol,
		Price:        price,
		Volume24h:    volume24h,
		VolumeUSD:    volumeUSD,
		Timestamp:    timestamp,
		OpenInterest: openInterest,
		FundingRate:  fundingRate,
		Change24h:    change24h,
		High24h:      high24h,
		Low24h:       low24h,
	}

	// Используем pipeline для атомарности
	pipe := rps.client.Pipeline()

	// 1. Сохраняем в кэш и текущие цены
	rps.cacheManager.SaveSnapshot(pipe, symbol, snapshot)

	// 2. Добавляем в историю
	rps.historyManager.AddToHistory(pipe, symbol, snapshot)

	// 3. Обновляем сортированный набор по объему
	if snapshot.VolumeUSD > 0 {
		volumeSortedKey := "prices:sorted_by_volume"
		pipe.ZAdd(rps.ctx, volumeSortedKey, &redis.Z{
			Score:  snapshot.VolumeUSD,
			Member: symbol,
		})
		// Ограничиваем размер
		pipe.ZRemRangeByRank(rps.ctx, volumeSortedKey, 0, -int64(rps.config.MaxSymbols+100))
	}

	// Выполняем все команды
	_, err := pipe.Exec(rps.ctx)
	if err != nil {
		return fmt.Errorf("ошибка сохранения в Redis: %w", err)
	}

	// Уведомляем подписчиков
	go rps.subscriptionMgr.NotifyAll(symbol, price, volume24h, volumeUSD, timestamp)

	return nil
}

// StorePriceData сохраняет готовый объект PriceData
func (rps *RedisPriceStorage) StorePriceData(priceData PriceData) error {
	return rps.StorePrice(
		priceData.Symbol,
		priceData.Price,
		priceData.Volume24h,
		priceData.VolumeUSD,
		priceData.Timestamp,
		priceData.OpenInterest,
		priceData.FundingRate,
		priceData.Change24h,
		priceData.High24h,
		priceData.Low24h,
	)
}

// GetCurrentPrice возвращает текущую цену
func (rps *RedisPriceStorage) GetCurrentPrice(symbol string) (float64, bool) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return 0, false
	}
	return snapshot.Price, true
}

// GetCurrentSnapshot возвращает текущий снапшот
func (rps *RedisPriceStorage) GetCurrentSnapshot(symbol string) (*PriceSnapshot, bool) {
	return rps.cacheManager.GetSnapshot(symbol)
}

// GetAllCurrentPrices возвращает все текущие цены
func (rps *RedisPriceStorage) GetAllCurrentPrices() map[string]PriceSnapshot {
	return rps.cacheManager.GetAllSnapshots()
}

// GetSymbols возвращает все символы
func (rps *RedisPriceStorage) GetSymbols() []string {
	return rps.cacheManager.GetSymbols()
}

// SymbolExists проверяет существование символа
func (rps *RedisPriceStorage) SymbolExists(symbol string) bool {
	_, exists := rps.GetCurrentSnapshot(symbol)
	return exists
}

// GetPriceHistory возвращает историю цен
func (rps *RedisPriceStorage) GetPriceHistory(symbol string, limit int) ([]PriceData, error) {
	return rps.historyManager.GetHistory(symbol, limit)
}

// GetPriceHistoryRange возвращает историю за период
func (rps *RedisPriceStorage) GetPriceHistoryRange(symbol string, start, end time.Time) ([]PriceData, error) {
	return rps.historyManager.GetHistoryRange(symbol, start, end)
}

// GetLatestPrice возвращает последнюю цену
func (rps *RedisPriceStorage) GetLatestPrice(symbol string) (*PriceData, bool) {
	history, err := rps.GetPriceHistory(symbol, 1)
	if err != nil || len(history) == 0 {
		return nil, false
	}
	return &history[len(history)-1], true
}

// CalculatePriceChange рассчитывает изменение цены
func (rps *RedisPriceStorage) CalculatePriceChange(symbol string, interval time.Duration) (*PriceChange, error) {
	currentSnapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return nil, ErrSymbolNotFound
	}

	// Ищем цену за указанный интервал назад
	targetTime := time.Now().Add(-interval)
	history, err := rps.GetPriceHistoryRange(symbol, targetTime.Add(-1*time.Minute), targetTime.Add(1*time.Minute))
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, ErrSymbolNotFound
	}

	// Находим ближайшую цену к targetTime
	var previousPrice *PriceData
	var minDiff time.Duration = 24 * time.Hour

	for i := range history {
		diff := history[i].Timestamp.Sub(targetTime)
		if diff.Abs() < minDiff.Abs() {
			minDiff = diff
			previousPrice = &history[i]
		}
	}

	if previousPrice == nil {
		return nil, ErrSymbolNotFound
	}

	// Рассчитываем изменение
	change := currentSnapshot.Price - previousPrice.Price
	changePercent := (change / previousPrice.Price) * 100

	return &PriceChange{
		Symbol:        symbol,
		CurrentPrice:  currentSnapshot.Price,
		PreviousPrice: previousPrice.Price,
		Change:        change,
		ChangePercent: changePercent,
		Interval:      interval.String(),
		Timestamp:     time.Now(),
		VolumeUSD:     currentSnapshot.VolumeUSD,
	}, nil
}

// GetAveragePrice возвращает среднюю цену за период
func (rps *RedisPriceStorage) GetAveragePrice(symbol string, period time.Duration) (float64, error) {
	cutoffTime := time.Now().Add(-period)
	history, err := rps.GetPriceHistoryRange(symbol, cutoffTime, time.Now())
	if err != nil {
		return 0, err
	}

	if len(history) == 0 {
		return 0, ErrSymbolNotFound
	}

	var sum float64
	for _, data := range history {
		sum += data.Price
	}

	return sum / float64(len(history)), nil
}

// GetMinMaxPrice возвращает min и max за период
func (rps *RedisPriceStorage) GetMinMaxPrice(symbol string, period time.Duration) (min, max float64, err error) {
	cutoffTime := time.Now().Add(-period)
	history, err := rps.GetPriceHistoryRange(symbol, cutoffTime, time.Now())
	if err != nil {
		return 0, 0, err
	}

	if len(history) == 0 {
		return 0, 0, ErrSymbolNotFound
	}

	min = history[0].Price
	max = history[0].Price

	for _, data := range history {
		if data.Price < min {
			min = data.Price
		}
		if data.Price > max {
			max = data.Price
		}
	}

	return min, max, nil
}

// GetOpenInterest возвращает открытый интерес
func (rps *RedisPriceStorage) GetOpenInterest(symbol string) (float64, bool) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return 0, false
	}
	return snapshot.OpenInterest, true
}

// GetFundingRate возвращает ставку фандинга
func (rps *RedisPriceStorage) GetFundingRate(symbol string) (float64, bool) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return 0, false
	}
	return snapshot.FundingRate, true
}

// GetSymbolMetrics возвращает все метрики символа
func (rps *RedisPriceStorage) GetSymbolMetrics(symbol string) (*SymbolMetrics, bool) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return nil, false
	}

	// Рассчитываем изменения
	oiChange24h, fundingChange := rps.calculateChanges(symbol)

	logger.Debug("💾 RedisStorage.GetSymbolMetrics: %s - OI=%.0f, Funding=%.6f",
		symbol, snapshot.OpenInterest, snapshot.FundingRate)

	return &SymbolMetrics{
		Symbol:        snapshot.Symbol,
		Price:         snapshot.Price,
		Volume24h:     snapshot.Volume24h,
		VolumeUSD:     snapshot.VolumeUSD,
		OpenInterest:  snapshot.OpenInterest,
		FundingRate:   snapshot.FundingRate,
		Change24h:     snapshot.Change24h,
		High24h:       snapshot.High24h,
		Low24h:        snapshot.Low24h,
		OIChange24h:   oiChange24h,
		FundingChange: fundingChange,
		Timestamp:     snapshot.Timestamp,
	}, true
}

// calculateChanges рассчитывает изменения OI и фандинга
func (rps *RedisPriceStorage) calculateChanges(symbol string) (float64, float64) {
	// Получаем историю за 24 часа
	history, err := rps.GetPriceHistoryRange(symbol, time.Now().Add(-24*time.Hour), time.Now())
	if err != nil || len(history) < 2 {
		return 0, 0
	}

	currentSnapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return 0, 0
	}

	// Находим самую старую запись
	oldest := history[0]

	// Рассчитываем изменения
	oiChange24h := 0.0
	if currentSnapshot.OpenInterest > 0 && oldest.OpenInterest > 0 {
		oiChange24h = ((currentSnapshot.OpenInterest - oldest.OpenInterest) / oldest.OpenInterest) * 100
	}

	fundingChange := 0.0
	if currentSnapshot.FundingRate != 0 && oldest.FundingRate != 0 {
		fundingChange = ((currentSnapshot.FundingRate - oldest.FundingRate) / oldest.FundingRate) * 100
	}

	return oiChange24h, fundingChange
}

// Subscribe подписывает на обновления
func (rps *RedisPriceStorage) Subscribe(symbol string, subscriber Subscriber) error {
	rps.subscriptionMgr.Subscribe(symbol, subscriber)
	return nil
}

// Unsubscribe отписывает от обновлений
func (rps *RedisPriceStorage) Unsubscribe(symbol string, subscriber Subscriber) error {
	rps.subscriptionMgr.Unsubscribe(symbol, subscriber)
	return nil
}

// GetSubscriberCount возвращает количество подписчиков
func (rps *RedisPriceStorage) GetSubscriberCount(symbol string) int {
	return rps.subscriptionMgr.GetSubscriberCount(symbol)
}

// CleanOldData очищает старые данные
func (rps *RedisPriceStorage) CleanOldData(maxAge time.Duration) (int, error) {
	if rps.client == nil {
		return 0, ErrRedisNotReady
	}

	// Очищаем историю
	removed, err := rps.historyManager.CleanupOldHistory(maxAge)
	if err != nil {
		return 0, err
	}

	// Очищаем устаревшие снапшоты из кэша
	rps.cacheManager.ClearCache()

	return removed, nil
}

// TruncateHistory ограничивает историю
func (rps *RedisPriceStorage) TruncateHistory(symbol string, maxPoints int) error {
	return rps.historyManager.TruncateHistory(symbol, maxPoints)
}

// RemoveSymbol удаляет символ
func (rps *RedisPriceStorage) RemoveSymbol(symbol string) error {
	if rps.client == nil {
		return ErrRedisNotReady
	}

	// Удаляем все ключи связанные с символом
	keys := []string{
		rps.prefix + "current:" + symbol,
		rps.prefix + "metrics:" + symbol,
		rps.prefix + "history:" + symbol,
	}

	_, err := rps.client.Del(rps.ctx, keys...).Result()
	if err != nil {
		return err
	}

	// Удаляем из сортированного набора по объему
	sortedSetKey := "prices:sorted_by_volume"
	rps.client.ZRem(rps.ctx, sortedSetKey, symbol)

	// Удаляем из кэша
	rps.cacheManager.RemoveFromCache(symbol)

	// Уведомляем подписчиков
	go rps.subscriptionMgr.NotifySymbolRemoved(symbol)

	return nil
}

// Clear очищает все данные
func (rps *RedisPriceStorage) Clear() error {
	if rps.client == nil {
		return ErrRedisNotReady
	}

	// Удаляем все ключи с префиксом
	pattern := rps.prefix + "*"
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = rps.client.Scan(rps.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}

	// Также удаляем сортированный набор
	keys = append(keys, "prices:sorted_by_volume")

	if len(keys) > 0 {
		_, err := rps.client.Del(rps.ctx, keys...).Result()
		if err != nil {
			return err
		}
	}

	// Очищаем кэш
	rps.cacheManager.ClearCache()

	return nil
}

// GetStats возвращает статистику
func (rps *RedisPriceStorage) GetStats() StorageStats {
	if rps.client == nil {
		return StorageStats{
			StorageType:  "redis",
			TotalSymbols: 0,
		}
	}

	// Получаем количество символов
	symbols := rps.GetSymbols()

	// Получаем статистику истории
	var estimatedDataPoints int64
	symbolsWithHistory, err := rps.historyManager.GetSymbolsWithHistory()
	if err == nil {
		for _, symbol := range symbolsWithHistory {
			historyKey := rps.prefix + "history:" + symbol
			if count, err := rps.client.ZCard(rps.ctx, historyKey).Result(); err == nil {
				estimatedDataPoints += count
			}
		}
	}

	// Получаем информацию о памяти
	var memoryUsage int64 = 0
	info, err := rps.client.Info(rps.ctx, "memory").Result()
	if err == nil {
		// Парсим использование памяти
		lines := strings.Split(info, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "used_memory:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &memoryUsage)
				}
			}
		}
	}

	// Подсчитываем символы с OI и фандингом
	symbolsWithOI := 0
	symbolsWithFunding := 0

	for _, symbol := range symbols {
		if snapshot, exists := rps.GetCurrentSnapshot(symbol); exists {
			if snapshot.OpenInterest > 0 {
				symbolsWithOI++
			}
			if snapshot.FundingRate != 0 {
				symbolsWithFunding++
			}
		}
	}

	return StorageStats{
		TotalSymbols:        len(symbols),
		TotalDataPoints:     estimatedDataPoints,
		MemoryUsageBytes:    memoryUsage,
		OldestTimestamp:     time.Time{}, // Сложно определить для Redis
		NewestTimestamp:     time.Now(),
		UpdateRatePerSecond: 0,
		StorageType:         "redis",
		MaxHistoryPerSymbol: rps.config.MaxHistoryPerSymbol,
		RetentionPeriod:     rps.config.RetentionPeriod,
		SymbolsWithOI:       symbolsWithOI,
		SymbolsWithFunding:  symbolsWithFunding,
	}
}

// GetSymbolStats возвращает статистику по символу
func (rps *RedisPriceStorage) GetSymbolStats(symbol string) (SymbolStats, error) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return SymbolStats{}, ErrSymbolNotFound
	}

	history, err := rps.GetPriceHistory(symbol, 10000)
	if err != nil || len(history) == 0 {
		return SymbolStats{}, ErrSymbolNotFound
	}

	// Рассчитываем средний объем
	var totalVolume24h, totalVolumeUSD float64
	for _, data := range history {
		totalVolume24h += data.Volume24h
		totalVolumeUSD += data.VolumeUSD
	}

	avgVolume24h := totalVolume24h / float64(len(history))
	avgVolumeUSD := totalVolumeUSD / float64(len(history))

	// Рассчитываем изменение за 24 часа
	firstPrice := history[0].Price
	lastPrice := history[len(history)-1].Price
	priceChange24h := 0.0
	if firstPrice > 0 {
		priceChange24h = ((lastPrice - firstPrice) / firstPrice) * 100
	}

	// Рассчитываем изменения OI и фандинга
	oiChange24h, fundingChange := rps.calculateChanges(symbol)

	return SymbolStats{
		Symbol:         symbol,
		DataPoints:     len(history),
		FirstTimestamp: history[0].Timestamp,
		LastTimestamp:  history[len(history)-1].Timestamp,
		CurrentPrice:   snapshot.Price,
		AvgVolume24h:   avgVolume24h,
		AvgVolumeUSD:   avgVolumeUSD,
		PriceChange24h: priceChange24h,
		OpenInterest:   snapshot.OpenInterest,
		OIChange24h:    oiChange24h,
		FundingRate:    snapshot.FundingRate,
		FundingChange:  fundingChange,
		High24h:        snapshot.High24h,
		Low24h:         snapshot.Low24h,
	}, nil
}

// GetTopSymbolsByVolumeUSD возвращает топ символов по объему в USDT
func (rps *RedisPriceStorage) GetTopSymbolsByVolumeUSD(limit int) ([]SymbolVolume, error) {
	if rps.client == nil {
		return nil, ErrRedisNotReady
	}

	sortedSetKey := "prices:sorted_by_volume"

	// Получаем топ символов с их объемами
	results, err := rps.client.ZRevRangeWithScores(rps.ctx, sortedSetKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	var symbols []SymbolVolume
	for _, result := range results {
		symbol := result.Member.(string)
		volumeUSD := result.Score

		// Получаем объем в базовой валюте
		var volume24h float64
		if snapshot, exists := rps.GetCurrentSnapshot(symbol); exists {
			volume24h = snapshot.Volume24h
		}

		symbols = append(symbols, SymbolVolume{
			Symbol:    symbol,
			Volume:    volume24h,
			VolumeUSD: volumeUSD,
		})
	}

	return symbols, nil
}

// GetTopSymbolsByVolume возвращает топ символов по объему
func (rps *RedisPriceStorage) GetTopSymbolsByVolume(limit int) ([]SymbolVolume, error) {
	symbols := rps.GetSymbols()
	var symbolVolumes []SymbolVolume

	for _, symbol := range symbols {
		if snapshot, exists := rps.GetCurrentSnapshot(symbol); exists {
			symbolVolumes = append(symbolVolumes, SymbolVolume{
				Symbol:    symbol,
				Volume:    snapshot.Volume24h,
				VolumeUSD: snapshot.VolumeUSD,
			})
		}
	}

	// Сортируем по убыванию объема
	sort.Slice(symbolVolumes, func(i, j int) bool {
		return symbolVolumes[i].Volume > symbolVolumes[j].Volume
	})

	if limit <= 0 || limit > len(symbolVolumes) {
		limit = len(symbolVolumes)
	}

	return symbolVolumes[:limit], nil
}

// FindSymbolsByPattern ищет символы по шаблону
func (rps *RedisPriceStorage) FindSymbolsByPattern(pattern string) ([]string, error) {
	symbols := rps.GetSymbols()
	var result []string

	patternUpper := strings.ToUpper(pattern)
	for _, symbol := range symbols {
		symbolUpper := strings.ToUpper(symbol)

		if pattern == "*" || pattern == "" {
			result = append(result, symbol)
		} else if strings.Contains(symbolUpper, patternUpper) {
			result = append(result, symbol)
		} else if strings.Contains(pattern, "*") {
			// Простая wildcard логика
			patternParts := strings.Split(patternUpper, "*")
			if len(patternParts) == 1 {
				if strings.HasPrefix(symbolUpper, patternParts[0]) {
					result = append(result, symbol)
				}
			} else if len(patternParts) == 2 {
				if strings.HasPrefix(symbolUpper, patternParts[0]) &&
					strings.HasSuffix(symbolUpper, patternParts[1]) {
					result = append(result, symbol)
				}
			}
		}
	}

	sort.Strings(result)
	return result, nil
}

// StorePriceLegacy поддерживает старый интерфейс
func (rps *RedisPriceStorage) StorePriceLegacy(symbol string, price, volume24h float64, timestamp time.Time) error {
	volumeUSD := price * volume24h
	return rps.StorePrice(
		symbol,
		price,
		volume24h,
		volumeUSD,
		timestamp,
		0, 0, 0, 0, 0,
	)
}
