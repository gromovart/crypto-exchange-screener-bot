// internal/infrastructure/persistence/redis_storage/candle_storage/candle_storage.go
package candle_storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"crypto-exchange-screener-bot/internal/core/domain/candle"
	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/pkg/logger"

	"github.com/go-redis/redis/v8"
)

// NewRedisCandleStorage создает новое Redis хранилище свечей
func NewRedisCandleStorage(redisService *redis_service.RedisService, config candle.CandleConfig) (*RedisCandleStorage, error) {
	if redisService == nil {
		return nil, fmt.Errorf("сервис Redis не инициализирован")
	}

	client := redisService.GetClient()
	if client == nil {
		return nil, fmt.Errorf("клиент Redis недоступен")
	}

	if len(config.SupportedPeriods) == 0 {
		config.SupportedPeriods = []string{"5m", "15m", "30m", "1h", "4h", "1d"}
	}
	if config.MaxHistory <= 0 {
		config.MaxHistory = 1000
	}

	return &RedisCandleStorage{
		client:       client,
		ctx:          context.Background(),
		prefix:       "candle:",
		candlePrefix: "candle:data:",
		config:       config,
	}, nil
}

// SaveActiveCandle сохраняет активную свечу
func (rcs *RedisCandleStorage) SaveActiveCandle(candle *candle.Candle) error {
	key := rcs.getActiveCandleKey(candle.Symbol, candle.Period)
	return rcs.saveCandleToRedis(key, candle, 1*time.Hour) // TTL 1 час для активных свечей
}

// GetActiveCandle получает активную свечу
func (rcs *RedisCandleStorage) GetActiveCandle(symbol, period string) (*candle.Candle, bool) {
	key := rcs.getActiveCandleKey(symbol, period)
	return rcs.loadCandleFromRedis(key)
}

// CloseAndArchiveCandle закрывает свечу и архивирует
func (rcs *RedisCandleStorage) CloseAndArchiveCandle(candle *candle.Candle) error {
	// Закрываем свечу
	candle.IsClosed = true
	candle.EndTime = time.Now()

	// Удаляем из активных
	activeKey := rcs.getActiveCandleKey(candle.Symbol, candle.Period)
	if err := rcs.client.Del(rcs.ctx, activeKey).Err(); err != nil {
		logger.Warn("⚠️ Ошибка удаления активной свечи %s %s: %v",
			candle.Symbol, candle.Period, err)
	}

	// Добавляем в историю
	return rcs.addToHistory(candle)
}

// GetHistory возвращает историю свечей
func (rcs *RedisCandleStorage) GetHistory(symbol, period string, limit int) ([]*candle.Candle, error) {
	historyKey := rcs.getHistoryKey(symbol, period)

	// Получаем последние N записей
	results, err := rcs.client.ZRevRangeByScore(rcs.ctx, historyKey, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  int64(limit),
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории из Redis: %w", err)
	}

	var candles []*candle.Candle
	for _, result := range results {
		candle, err := rcs.unmarshalCandle(result)
		if err == nil {
			candles = append(candles, candle)
		}
	}

	// Сортируем по времени (старые -> новые)
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].StartTime.Before(candles[j].StartTime)
	})

	return candles, nil
}

// GetLatestCandle возвращает последнюю свечу
func (rcs *RedisCandleStorage) GetLatestCandle(symbol, period string) (*candle.Candle, bool) {
	// Сначала проверяем активные свечи
	if candle, exists := rcs.GetActiveCandle(symbol, period); exists {
		return candle, true
	}

	// Затем историю
	history, err := rcs.GetHistory(symbol, period, 1)
	if err != nil || len(history) == 0 {
		return nil, false
	}

	return history[len(history)-1], true
}

// GetCandle получает свечу (активную или последнюю из истории)
func (rcs *RedisCandleStorage) GetCandle(symbol, period string) (*candle.Candle, error) {
	// Сначала активную
	if candle, exists := rcs.GetActiveCandle(symbol, period); exists {
		return candle, nil
	}

	// Затем последнюю из истории
	history, err := rcs.GetHistory(symbol, period, 1)
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("свеча не найдена для %s %s", symbol, period)
	}

	return history[0], nil
}

// CleanupOldCandles очищает старые свечи
func (rcs *RedisCandleStorage) CleanupOldCandles(maxAge time.Duration) int {
	cutoffTime := time.Now().Add(-maxAge)
	cutoffUnix := cutoffTime.Unix()

	// Получаем все ключи истории свечей
	pattern := rcs.prefix + "history:*"
	var cursor uint64
	keys := make([]string, 0)

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = rcs.client.Scan(rcs.ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Warn("⚠️ Ошибка SCAN истории свечей: %v", err)
			break
		}

		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}

	totalRemoved := 0

	// Очищаем каждый ключ
	for _, key := range keys {
		removed, err := rcs.client.ZRemRangeByScore(rcs.ctx, key, "-inf", fmt.Sprintf("%d", cutoffUnix)).Result()
		if err != nil {
			logger.Warn("⚠️ Ошибка очистки истории свечей для ключа %s: %v", key, err)
			continue
		}

		totalRemoved += int(removed)

		// Если история пустая, удаляем ключ
		count, err := rcs.client.ZCard(rcs.ctx, key).Result()
		if err == nil && count == 0 {
			rcs.client.Del(rcs.ctx, key)
		}
	}

	if totalRemoved > 0 {
		logger.Debug("🧹 RedisCandleStorage: удалено %d старых свечей (старше %v)", totalRemoved, maxAge)
	}

	return totalRemoved
}

// GetSymbols возвращает все символы с данными
func (rcs *RedisCandleStorage) GetSymbols() []string {
	// Получаем все ключи активных свечей
	pattern := rcs.prefix + "active:*"
	var cursor uint64
	symbolsMap := make(map[string]bool)

	for {
		var keys []string
		var err error
		keys, cursor, err = rcs.client.Scan(rcs.ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Warn("⚠️ Ошибка SCAN активных свечей: %v", err)
			break
		}

		for _, key := range keys {
			// Извлекаем symbol из ключа: candle:active:BTCUSDT:5m
			parts := strings.Split(key, ":")
			if len(parts) >= 3 {
				symbol := parts[2]
				symbolsMap[symbol] = true
			}
		}

		if cursor == 0 {
			break
		}
	}

	// Также проверяем историю
	pattern = rcs.prefix + "history:*"
	cursor = 0

	for {
		var keys []string
		var err error
		keys, cursor, err = rcs.client.Scan(rcs.ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Warn("⚠️ Ошибка SCAN истории свечей: %v", err)
			break
		}

		for _, key := range keys {
			// Извлекаем symbol из ключа: candle:history:BTCUSDT:5m
			parts := strings.Split(key, ":")
			if len(parts) >= 3 {
				symbol := parts[2]
				symbolsMap[symbol] = true
			}
		}

		if cursor == 0 {
			break
		}
	}

	symbols := make([]string, 0, len(symbolsMap))
	for symbol := range symbolsMap {
		symbols = append(symbols, symbol)
	}

	sort.Strings(symbols)
	return symbols
}

// GetPeriodsForSymbol возвращает периоды для символа
func (rcs *RedisCandleStorage) GetPeriodsForSymbol(symbol string) []string {
	periodsMap := make(map[string]bool)

	// Проверяем активные свечи
	pattern := rcs.prefix + "active:" + symbol + ":*"
	var cursor uint64

	for {
		var keys []string
		var err error
		keys, cursor, err = rcs.client.Scan(rcs.ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Warn("⚠️ Ошибка SCIN периодов для символа %s: %v", symbol, err)
			break
		}

		for _, key := range keys {
			// Извлекаем period из ключа: candle:active:BTCUSDT:5m
			parts := strings.Split(key, ":")
			if len(parts) >= 4 {
				period := parts[3]
				periodsMap[period] = true
			}
		}

		if cursor == 0 {
			break
		}
	}

	// Проверяем историю
	pattern = rcs.prefix + "history:" + symbol + ":*"
	cursor = 0

	for {
		var keys []string
		var err error
		keys, cursor, err = rcs.client.Scan(rcs.ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Warn("⚠️ Ошибка SCAN истории периодов для символа %s: %v", symbol, err)
			break
		}

		for _, key := range keys {
			parts := strings.Split(key, ":")
			if len(parts) >= 4 {
				period := parts[3]
				periodsMap[period] = true
			}
		}

		if cursor == 0 {
			break
		}
	}

	periods := make([]string, 0, len(periodsMap))
	for period := range periodsMap {
		periods = append(periods, period)
	}

	sort.Strings(periods)
	return periods
}

// GetStats возвращает статистику
func (rcs *RedisCandleStorage) GetStats() candle.CandleStats {
	symbols := rcs.GetSymbols()
	stats := candle.CandleStats{
		PeriodsCount: make(map[string]int),
		OldestCandle: time.Now(),
		NewestCandle: time.Time{},
	}

	// Подсчитываем активные свечи
	activeCount := 0
	for _, symbol := range symbols {
		periods := rcs.GetPeriodsForSymbol(symbol)
		for _, period := range periods {
			if _, exists := rcs.GetActiveCandle(symbol, period); exists {
				activeCount++
				stats.PeriodsCount[period]++
			}
		}
	}
	stats.ActiveCandles = activeCount

	// Подсчитываем исторические свечи
	historyCount := 0
	for _, symbol := range symbols {
		periods := rcs.GetPeriodsForSymbol(symbol)
		for _, period := range periods {
			history, err := rcs.GetHistory(symbol, period, rcs.config.MaxHistory)
			if err == nil {
				historyCount += len(history)
				stats.PeriodsCount[period] += len(history)

				// Находим самую старую и новую свечу
				if len(history) > 0 {
					if history[0].StartTime.Before(stats.OldestCandle) {
						stats.OldestCandle = history[0].StartTime
					}
					if history[len(history)-1].EndTime.After(stats.NewestCandle) {
						stats.NewestCandle = history[len(history)-1].EndTime
					}
				}
			}
		}
	}
	stats.TotalCandles = activeCount + historyCount
	stats.SymbolsCount = len(symbols)

	return stats
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ====================

// saveCandleToRedis сохраняет свечу в Redis
func (rcs *RedisCandleStorage) saveCandleToRedis(key string, candle *candle.Candle, ttl time.Duration) error {
	data, err := json.Marshal(candle)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга свечи: %w", err)
	}

	return rcs.client.Set(rcs.ctx, key, data, ttl).Err()
}

// loadCandleFromRedis загружает свечу из Redis
func (rcs *RedisCandleStorage) loadCandleFromRedis(key string) (*candle.Candle, bool) {
	data, err := rcs.client.Get(rcs.ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		logger.Warn("⚠️ Ошибка загрузки свечи из Redis: %v", err)
		return nil, false
	}

	candle, err := rcs.unmarshalCandle(data)
	if err != nil {
		return nil, false
	}

	return candle, true
}

// unmarshalCandle преобразует JSON в свечу
func (rcs *RedisCandleStorage) unmarshalCandle(data string) (*candle.Candle, error) {
	var candle candle.Candle
	if err := json.Unmarshal([]byte(data), &candle); err != nil {
		return nil, fmt.Errorf("ошибка парсинга свечи: %w", err)
	}
	return &candle, nil
}

// addToHistory добавляет свечу в историю
func (rcs *RedisCandleStorage) addToHistory(candle *candle.Candle) error {
	data, err := json.Marshal(candle)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга свечи для истории: %w", err)
	}

	historyKey := rcs.getHistoryKey(candle.Symbol, candle.Period)
	_, err = rcs.client.ZAdd(rcs.ctx, historyKey, &redis.Z{
		Score:  float64(candle.StartTime.Unix()),
		Member: data,
	}).Result()

	if err != nil {
		return fmt.Errorf("ошибка добавления в историю: %w", err)
	}

	// Ограничиваем размер истории
	rcs.client.ZRemRangeByRank(rcs.ctx, historyKey, 0, -int64(rcs.config.MaxHistory+100))

	logger.Debug("📊 Архивирована свеча %s %s: %.6f → %.6f (%.2f%%)",
		candle.Symbol, candle.Period, candle.Open, candle.Close,
		((candle.Close-candle.Open)/candle.Open)*100)

	return nil
}

// getActiveCandleKey возвращает ключ для активной свечи
func (rcs *RedisCandleStorage) getActiveCandleKey(symbol, period string) string {
	return fmt.Sprintf("%sactive:%s:%s", rcs.prefix, symbol, period)
}

// getHistoryKey возвращает ключ для истории свечей
func (rcs *RedisCandleStorage) getHistoryKey(symbol, period string) string {
	return fmt.Sprintf("%shistory:%s:%s", rcs.prefix, symbol, period)
}
