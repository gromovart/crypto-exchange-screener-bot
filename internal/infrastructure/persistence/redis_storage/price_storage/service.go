// internal/infrastructure/persistence/redis_storage/price_storage/service.go
package price_storage

import (
	"context"
	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	redis_storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/cache_manager"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/history_manager"
	"crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage/subscription_manager"
	"fmt"
	"sort"
	"strings"
	"time"

	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/go-redis/redis/v8"
)

// PriceStorage реализация PriceStorage с использованием Redis
type PriceStorage struct {
	redisService *redis_service.RedisService
	client       *redis.Client
	prefix       string
	ctx          context.Context

	// Подсистемы (используем интерфейсы)
	cacheManager    redis_storage.CacheManagerInterface
	subscriptionMgr redis_storage.SubscriptionManagerInterface
	historyManager  redis_storage.HistoryManagerInterface

	// Конфигурация
	config *redis_storage.StorageConfig
}

// NewPriceStorage создает новое Redis хранилище цен
func NewPriceStorage(
	redisService *redis_service.RedisService,
	config *redis_storage.StorageConfig,
	cacheManager redis_storage.CacheManagerInterface,
	subscriptionMgr redis_storage.SubscriptionManagerInterface,
	historyManager redis_storage.HistoryManagerInterface,
) *PriceStorage {
	if config == nil {
		config = &redis_storage.StorageConfig{
			MaxHistoryPerSymbol: 10000,
			MaxSymbols:          1000,
			CleanupInterval:     5 * time.Minute,
			RetentionPeriod:     48 * time.Hour,
		}
	}

	return &PriceStorage{
		redisService:    redisService,
		prefix:          "price:",
		ctx:             context.Background(),
		cacheManager:    cacheManager,
		subscriptionMgr: subscriptionMgr,
		historyManager:  historyManager,
		config:          config,
	}
}

// NewPriceStorageSimple создает новое Redis хранилище цен (упрощенная версия)
func NewPriceStorageSimple(redisService *redis_service.RedisService, config *redis_storage.StorageConfig) *PriceStorage {
	if config == nil {
		config = &redis_storage.StorageConfig{
			MaxHistoryPerSymbol: 10000,
			MaxSymbols:          1000,
			CleanupInterval:     5 * time.Minute,
			RetentionPeriod:     48 * time.Hour,
		}
	}

	// Создаем подсистемы
	cacheManager := cache_manager.NewCacheManager()
	subscriptionMgr := subscription_manager.NewSubscriptionManager()
	historyManager := history_manager.NewHistoryManager()

	return NewPriceStorage(redisService, config, cacheManager, subscriptionMgr, historyManager)
}

// Initialize инициализирует Redis хранилище
func (rps *PriceStorage) Initialize() error {
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

	logger.Info("✅ PriceStorage инициализирован")
	return nil
}

// StorePrice сохраняет цену со всеми данными
func (rps *PriceStorage) StorePrice(
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
	snapshot := &redis_storage.PriceSnapshot{
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
	err := rps.historyManager.AddToHistory(pipe, symbol, snapshot)
	if err != nil {
		return err
	}

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
	_, err = pipe.Exec(rps.ctx)
	if err != nil {
		return fmt.Errorf("ошибка сохранения в Redis: %w", err)
	}

	// Уведомляем подписчиков
	go rps.subscriptionMgr.NotifyAll(symbol, price, volume24h, volumeUSD, timestamp)

	return nil
}

// StorePriceData сохраняет готовый объект PriceData
func (rps *PriceStorage) StorePriceData(priceData redis_storage.PriceDataInterface) error {
	return rps.StorePrice(
		priceData.GetSymbol(),
		priceData.GetPrice(),
		priceData.GetVolume24h(),
		priceData.GetVolumeUSD(),
		priceData.GetTimestamp(),
		priceData.GetOpenInterest(),
		priceData.GetFundingRate(),
		priceData.GetChange24h(),
		priceData.GetHigh24h(),
		priceData.GetLow24h(),
	)
}

// GetCurrentPrice возвращает текущую цену
func (rps *PriceStorage) GetCurrentPrice(symbol string) (float64, bool) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return 0, false
	}
	return snapshot.GetPrice(), true
}

// GetCurrentSnapshot возвращает текущий снапшот
func (rps *PriceStorage) GetCurrentSnapshot(symbol string) (redis_storage.PriceSnapshotInterface, bool) {
	return rps.cacheManager.GetSnapshot(symbol)
}

// GetAllCurrentPrices возвращает все текущие цены
func (rps *PriceStorage) GetAllCurrentPrices() map[string]redis_storage.PriceSnapshotInterface {
	return rps.cacheManager.GetAllSnapshots()
}

// GetSymbols возвращает все символы
func (rps *PriceStorage) GetSymbols() []string {
	return rps.cacheManager.GetSymbols()
}

// SymbolExists проверяет существование символа
func (rps *PriceStorage) SymbolExists(symbol string) bool {
	_, exists := rps.GetCurrentSnapshot(symbol)
	return exists
}

// GetPriceHistory возвращает историю цен
func (rps *PriceStorage) GetPriceHistory(symbol string, limit int) ([]redis_storage.PriceDataInterface, error) {
	return rps.historyManager.GetHistory(symbol, limit)
}

// GetPriceHistoryRange возвращает историю за период
func (rps *PriceStorage) GetPriceHistoryRange(symbol string, start, end time.Time) ([]redis_storage.PriceDataInterface, error) {
	return rps.historyManager.GetHistoryRange(symbol, start, end)
}

// GetLatestPrice возвращает последнюю цену
func (rps *PriceStorage) GetLatestPrice(symbol string) (redis_storage.PriceDataInterface, bool) {
	history, err := rps.GetPriceHistory(symbol, 1)
	if err != nil || len(history) == 0 {
		return nil, false
	}
	return history[len(history)-1], true
}

// CalculatePriceChange рассчитывает изменение цены
func (rps *PriceStorage) CalculatePriceChange(symbol string, interval time.Duration) (redis_storage.PriceChangeInterface, error) {
	currentSnapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return nil, redis_storage.ErrSymbolNotFound
	}

	// Ищем цену за указанный интервал назад
	targetTime := time.Now().Add(-interval)
	history, err := rps.GetPriceHistoryRange(symbol, targetTime.Add(-1*time.Minute), targetTime.Add(1*time.Minute))
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, redis_storage.ErrSymbolNotFound
	}

	// Находим ближайшую цену к targetTime
	var previousPrice redis_storage.PriceDataInterface
	var minDiff time.Duration = 24 * time.Hour

	for i := range history {
		diff := history[i].GetTimestamp().Sub(targetTime)
		if diff.Abs() < minDiff.Abs() {
			minDiff = diff
			previousPrice = history[i]
		}
	}

	if previousPrice == nil {
		return nil, redis_storage.ErrSymbolNotFound
	}

	// Рассчитываем изменение
	change := currentSnapshot.GetPrice() - previousPrice.GetPrice()
	changePercent := (change / previousPrice.GetPrice()) * 100

	return &PriceChange{
		Symbol:        symbol,
		CurrentPrice:  currentSnapshot.GetPrice(),
		PreviousPrice: previousPrice.GetPrice(),
		Change:        change,
		ChangePercent: changePercent,
		Interval:      interval.String(),
		Timestamp:     time.Now(),
		VolumeUSD:     currentSnapshot.GetVolumeUSD(),
	}, nil
}

// GetAveragePrice возвращает среднюю цену за период
func (rps *PriceStorage) GetAveragePrice(symbol string, period time.Duration) (float64, error) {
	cutoffTime := time.Now().Add(-period)
	history, err := rps.GetPriceHistoryRange(symbol, cutoffTime, time.Now())
	if err != nil {
		return 0, err
	}

	if len(history) == 0 {
		return 0, redis_storage.ErrSymbolNotFound
	}

	var sum float64
	for _, data := range history {
		sum += data.GetPrice()
	}

	return sum / float64(len(history)), nil
}

// GetMinMaxPrice возвращает min и max за период
func (rps *PriceStorage) GetMinMaxPrice(symbol string, period time.Duration) (min, max float64, err error) {
	cutoffTime := time.Now().Add(-period)
	history, err := rps.GetPriceHistoryRange(symbol, cutoffTime, time.Now())
	if err != nil {
		return 0, 0, err
	}

	if len(history) == 0 {
		return 0, 0, redis_storage.ErrSymbolNotFound
	}

	min = history[0].GetPrice()
	max = history[0].GetPrice()

	for _, data := range history {
		price := data.GetPrice()
		if price < min {
			min = price
		}
		if price > max {
			max = price
		}
	}

	return min, max, nil
}

// GetOpenInterest возвращает открытый интерес
func (rps *PriceStorage) GetOpenInterest(symbol string) (float64, bool) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return 0, false
	}
	return snapshot.GetOpenInterest(), true
}

// GetFundingRate возвращает ставку фандинга
func (rps *PriceStorage) GetFundingRate(symbol string) (float64, bool) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return 0, false
	}
	return snapshot.GetFundingRate(), true
}

// GetSymbolMetrics возвращает все метрики символа
func (rps *PriceStorage) GetSymbolMetrics(symbol string) (redis_storage.SymbolMetricsInterface, bool) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return nil, false
	}

	// Рассчитываем изменения
	oiChange24h, fundingChange := rps.calculateChanges(symbol)

	logger.Debug("💾 RedisStorage.GetSymbolMetrics: %s - OI=%.0f, Funding=%.6f",
		symbol, snapshot.GetOpenInterest(), snapshot.GetFundingRate())

	return &redis_storage.SymbolMetrics{
		Symbol:        snapshot.GetSymbol(),
		Price:         snapshot.GetPrice(),
		Volume24h:     snapshot.GetVolume24h(),
		VolumeUSD:     snapshot.GetVolumeUSD(),
		OpenInterest:  snapshot.GetOpenInterest(),
		FundingRate:   snapshot.GetFundingRate(),
		Change24h:     snapshot.GetChange24h(),
		High24h:       snapshot.GetHigh24h(),
		Low24h:        snapshot.GetLow24h(),
		OIChange24h:   oiChange24h,
		FundingChange: fundingChange,
		Timestamp:     snapshot.GetTimestamp(),
	}, true
}

// calculateChanges рассчитывает изменения OI и фандинга
func (rps *PriceStorage) calculateChanges(symbol string) (float64, float64) {
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
	currentOI := currentSnapshot.GetOpenInterest()
	oldestOI := oldest.GetOpenInterest()
	if currentOI > 0 && oldestOI > 0 {
		oiChange24h = ((currentOI - oldestOI) / oldestOI) * 100
	}

	fundingChange := 0.0
	currentFunding := currentSnapshot.GetFundingRate()
	oldestFunding := oldest.GetFundingRate()
	if currentFunding != 0 && oldestFunding != 0 {
		fundingChange = ((currentFunding - oldestFunding) / oldestFunding) * 100
	}

	return oiChange24h, fundingChange
}

// Subscribe подписывает на обновления
func (rps *PriceStorage) Subscribe(symbol string, subscriber redis_storage.SubscriberInterface) error {
	return rps.subscriptionMgr.Subscribe(symbol, subscriber)
}

// Unsubscribe отписывает от обновлений
func (rps *PriceStorage) Unsubscribe(symbol string, subscriber redis_storage.SubscriberInterface) error {
	return rps.subscriptionMgr.Unsubscribe(symbol, subscriber)
}

// GetSubscriberCount возвращает количество подписчиков
func (rps *PriceStorage) GetSubscriberCount(symbol string) int {
	return rps.subscriptionMgr.GetSubscriberCount(symbol)
}

// CleanOldData очищает старые данные
func (rps *PriceStorage) CleanOldData(maxAge time.Duration) (int, error) {
	if rps.client == nil {
		return 0, redis_storage.ErrRedisNotReady
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
func (rps *PriceStorage) TruncateHistory(symbol string, maxPoints int) error {
	return rps.historyManager.TruncateHistory(symbol, maxPoints)
}

// RemoveSymbol удаляет символ
func (rps *PriceStorage) RemoveSymbol(symbol string) error {
	if rps.client == nil {
		return redis_storage.ErrRedisNotReady
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
func (rps *PriceStorage) Clear() error {
	if rps.client == nil {
		return redis_storage.ErrRedisNotReady
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
func (rps *PriceStorage) GetStats() redis_storage.StorageStatsInterface {
	if rps.client == nil {
		return &redis_storage.StorageStats{
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
			if snapshot.GetOpenInterest() > 0 {
				symbolsWithOI++
			}
			if snapshot.GetFundingRate() != 0 {
				symbolsWithFunding++
			}
		}
	}

	return &redis_storage.StorageStats{
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
func (rps *PriceStorage) GetSymbolStats(symbol string) (redis_storage.SymbolStatsInterface, error) {
	snapshot, exists := rps.GetCurrentSnapshot(symbol)
	if !exists {
		return nil, redis_storage.ErrSymbolNotFound
	}

	history, err := rps.GetPriceHistory(symbol, 10000)
	if err != nil || len(history) == 0 {
		return nil, redis_storage.ErrSymbolNotFound
	}

	// Рассчитываем средний объем
	var totalVolume24h, totalVolumeUSD float64
	for _, data := range history {
		totalVolume24h += data.GetVolume24h()
		totalVolumeUSD += data.GetVolumeUSD()
	}

	avgVolume24h := totalVolume24h / float64(len(history))
	avgVolumeUSD := totalVolumeUSD / float64(len(history))

	// Рассчитываем изменение за 24 часа
	firstPrice := history[0].GetPrice()
	lastPrice := history[len(history)-1].GetPrice()
	priceChange24h := 0.0
	if firstPrice > 0 {
		priceChange24h = ((lastPrice - firstPrice) / firstPrice) * 100
	}

	// Рассчитываем изменения OI и фандинга
	oiChange24h, fundingChange := rps.calculateChanges(symbol)

	return &redis_storage.SymbolStats{
		Symbol:         symbol,
		DataPoints:     len(history),
		FirstTimestamp: history[0].GetTimestamp(),
		LastTimestamp:  history[len(history)-1].GetTimestamp(),
		CurrentPrice:   snapshot.GetPrice(),
		AvgVolume24h:   avgVolume24h,
		AvgVolumeUSD:   avgVolumeUSD,
		PriceChange24h: priceChange24h,
		OpenInterest:   snapshot.GetOpenInterest(),
		OIChange24h:    oiChange24h,
		FundingRate:    snapshot.GetFundingRate(),
		FundingChange:  fundingChange,
		High24h:        snapshot.GetHigh24h(),
		Low24h:         snapshot.GetLow24h(),
	}, nil
}

// GetTopSymbolsByVolumeUSD возвращает топ символов по объему в USDT
func (rps *PriceStorage) GetTopSymbolsByVolumeUSD(limit int) ([]redis_storage.SymbolVolumeInterface, error) {
	if rps.client == nil {
		return nil, redis_storage.ErrRedisNotReady
	}

	sortedSetKey := "prices:sorted_by_volume"

	// Получаем топ символов с их объемами
	results, err := rps.client.ZRevRangeWithScores(rps.ctx, sortedSetKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	var symbols []redis_storage.SymbolVolumeInterface
	for _, result := range results {
		symbol := result.Member.(string)
		volumeUSD := result.Score

		// Получаем объем в базовой валюте
		var volume24h float64
		if snapshot, exists := rps.GetCurrentSnapshot(symbol); exists {
			volume24h = snapshot.GetVolume24h()
		}

		symbols = append(symbols, &redis_storage.SymbolVolume{
			Symbol:    symbol,
			Volume:    volume24h,
			VolumeUSD: volumeUSD,
		})
	}

	return symbols, nil
}

// GetTopSymbolsByVolume возвращает топ символов по объему
func (rps *PriceStorage) GetTopSymbolsByVolume(limit int) ([]redis_storage.SymbolVolumeInterface, error) {
	symbols := rps.GetSymbols()
	var symbolVolumes []redis_storage.SymbolVolumeInterface

	for _, symbol := range symbols {
		if snapshot, exists := rps.GetCurrentSnapshot(symbol); exists {
			symbolVolumes = append(symbolVolumes, &redis_storage.SymbolVolume{
				Symbol:    symbol,
				Volume:    snapshot.GetVolume24h(),
				VolumeUSD: snapshot.GetVolumeUSD(),
			})
		}
	}

	// Сортируем по убыванию объема
	sort.Slice(symbolVolumes, func(i, j int) bool {
		return symbolVolumes[i].GetVolume() > symbolVolumes[j].GetVolume()
	})

	if limit <= 0 || limit > len(symbolVolumes) {
		limit = len(symbolVolumes)
	}

	return symbolVolumes[:limit], nil
}

// FindSymbolsByPattern ищет символы по шаблону
func (rps *PriceStorage) FindSymbolsByPattern(pattern string) ([]string, error) {
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
func (rps *PriceStorage) StorePriceLegacy(symbol string, price, volume24h float64, timestamp time.Time) error {
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
