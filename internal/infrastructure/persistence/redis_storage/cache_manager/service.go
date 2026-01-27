// internal/infrastructure/persistence/redis_storage/cache_manager/service.go
package cache_manager

import (
	"context"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/pkg/logger"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// NewCacheManager создает нового менеджера кэша
func NewCacheManager() *CacheManager {
	return &CacheManager{
		prefix:       "price:",
		ctx:          context.Background(),
		localCache:   make(map[string]storage.PriceSnapshotInterface),
		localCacheMu: sync.RWMutex{},
	}
}

// Initialize инициализирует менеджер кэша
func (cm *CacheManager) Initialize(client *redis.Client) {
	cm.client = client
}

// SaveSnapshot сохраняет снапшот в Redis
func (cm *CacheManager) SaveSnapshot(pipe redis.Pipeliner, symbol string, snapshot storage.PriceSnapshotInterface) {
	if cm.client == nil {
		return
	}

	// Сохраняем в локальный кэш
	cm.localCacheMu.Lock()
	cm.localCache[symbol] = snapshot
	cm.localCacheMu.Unlock()

	// Преобразуем в конкретный тип для сериализации
	priceSnapshot := &storage.PriceSnapshot{
		Symbol:       snapshot.GetSymbol(),
		Price:        snapshot.GetPrice(),
		Volume24h:    snapshot.GetVolume24h(),
		VolumeUSD:    snapshot.GetVolumeUSD(),
		Timestamp:    snapshot.GetTimestamp(),
		OpenInterest: snapshot.GetOpenInterest(),
		FundingRate:  snapshot.GetFundingRate(),
		Change24h:    snapshot.GetChange24h(),
		High24h:      snapshot.GetHigh24h(),
		Low24h:       snapshot.GetLow24h(),
	}

	// Сохраняем в Redis
	currentKey := cm.prefix + "current:" + symbol
	snapshotJSON, err := json.Marshal(priceSnapshot)
	if err != nil {
		logger.Warn("⚠️ Ошибка маршалинга снапшота для %s: %v", symbol, err)
		return
	}

	pipe.Set(cm.ctx, currentKey, snapshotJSON, 24*time.Hour)
}

// GetSnapshot получает снапшот из кэша
func (cm *CacheManager) GetSnapshot(symbol string) (storage.PriceSnapshotInterface, bool) {
	// Сначала проверяем локальный кэш
	cm.localCacheMu.RLock()
	if snapshot, exists := cm.localCache[symbol]; exists {
		cm.localCacheMu.RUnlock()
		return snapshot, true
	}
	cm.localCacheMu.RUnlock()

	// Если нет в локальном кэше, загружаем из Redis
	return cm.loadFromRedis(symbol)
}

// loadFromRedis загружает снапшот из Redis
func (cm *CacheManager) loadFromRedis(symbol string) (storage.PriceSnapshotInterface, bool) {
	if cm.client == nil {
		return nil, false
	}

	key := cm.prefix + "current:" + symbol
	data, err := cm.client.Get(cm.ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		logger.Warn("⚠️ Ошибка загрузки снапшота из Redis для %s: %v", symbol, err)
		return nil, false
	}

	var snapshot storage.PriceSnapshot
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		logger.Warn("⚠️ Ошибка парсинга снапшота для %s: %v", symbol, err)
		return nil, false
	}

	// Сохраняем в локальный кэш
	cm.localCacheMu.Lock()
	cm.localCache[symbol] = &snapshot
	cm.localCacheMu.Unlock()

	return &snapshot, true
}

// GetAllSnapshots возвращает все снапшоты
func (cm *CacheManager) GetAllSnapshots() map[string]storage.PriceSnapshotInterface {
	cm.localCacheMu.RLock()
	defer cm.localCacheMu.RUnlock()

	result := make(map[string]storage.PriceSnapshotInterface)
	for symbol, snapshot := range cm.localCache {
		result[symbol] = snapshot
	}

	return result
}

// GetSymbols возвращает все символы из кэша
func (cm *CacheManager) GetSymbols() []string {
	cm.localCacheMu.RLock()
	defer cm.localCacheMu.RUnlock()

	symbols := make([]string, 0, len(cm.localCache))
	for symbol := range cm.localCache {
		symbols = append(symbols, symbol)
	}

	sort.Strings(symbols)
	return symbols
}

// ClearCache очищает локальный кэш
func (cm *CacheManager) ClearCache() {
	cm.localCacheMu.Lock()
	cm.localCache = make(map[string]storage.PriceSnapshotInterface)
	cm.localCacheMu.Unlock()
}

// UpdateLocalCache обновляет локальный кэш
func (cm *CacheManager) UpdateLocalCache(symbol string, snapshot storage.PriceSnapshotInterface) {
	cm.localCacheMu.Lock()
	cm.localCache[symbol] = snapshot
	cm.localCacheMu.Unlock()
}

// RemoveFromCache удаляет символ из кэша
func (cm *CacheManager) RemoveFromCache(symbol string) {
	cm.localCacheMu.Lock()
	delete(cm.localCache, symbol)
	cm.localCacheMu.Unlock()
}
