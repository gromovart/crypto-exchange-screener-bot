// internal/infrastructure/persistence/redis_storage/history_manager/service.go
package history_manager

import (
	"context"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"crypto-exchange-screener-bot/pkg/logger"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// NewHistoryManager создает нового менеджера истории
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		prefix: "price:",
		ctx:    context.Background(),
	}
}

// Initialize инициализирует менеджер истории
func (hm *HistoryManager) Initialize(client *redis.Client, config *storage.StorageConfig) {
	hm.client = client
	hm.config = config
}

// AddToHistory добавляет цену в историю
func (hm *HistoryManager) AddToHistory(pipe redis.Pipeliner, symbol string, snapshot storage.PriceSnapshotInterface) error {
	if hm.client == nil {
		return fmt.Errorf("клиент Redis не инициализирован")
	}

	historyItem := struct {
		Symbol       string    `json:"symbol"`
		Price        float64   `json:"price"`
		Volume24h    float64   `json:"volume_24h"`
		VolumeUSD    float64   `json:"volume_usd"`
		Timestamp    time.Time `json:"timestamp"`
		OpenInterest float64   `json:"open_interest"`
		FundingRate  float64   `json:"funding_rate"`
		Change24h    float64   `json:"change_24h"`
		High24h      float64   `json:"high_24h"`
		Low24h       float64   `json:"low_24h"`
	}{
		Symbol:       symbol,
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

	data, err := json.Marshal(historyItem)
	if err != nil {
		return err
	}

	// Используем ZSET для истории с timestamp как score
	historyKey := hm.prefix + "history:" + symbol
	pipe.ZAdd(hm.ctx, historyKey, &redis.Z{
		Score:  float64(snapshot.GetTimestamp().Unix()),
		Member: data,
	})

	// Ограничиваем размер истории
	pipe.ZRemRangeByRank(hm.ctx, historyKey, 0, -int64(hm.config.MaxHistoryPerSymbol+100))

	return nil
}

// GetHistory возвращает историю цен
func (hm *HistoryManager) GetHistory(symbol string, limit int) ([]storage.PriceDataInterface, error) {
	if hm.client == nil {
		return nil, fmt.Errorf("клиент Redis не инициализирован")
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > hm.config.MaxHistoryPerSymbol {
		limit = hm.config.MaxHistoryPerSymbol
	}

	historyKey := hm.prefix + "history:" + symbol

	// Получаем последние N записей (от новых к старым)
	results, err := hm.client.ZRevRangeByScore(hm.ctx, historyKey, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  int64(limit),
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории из Redis: %w", err)
	}

	var history []storage.PriceDataInterface
	for _, result := range results {
		var data struct {
			Symbol       string    `json:"symbol"`
			Price        float64   `json:"price"`
			Volume24h    float64   `json:"volume_24h"`
			VolumeUSD    float64   `json:"volume_usd"`
			Timestamp    time.Time `json:"timestamp"`
			OpenInterest float64   `json:"open_interest"`
			FundingRate  float64   `json:"funding_rate"`
			Change24h    float64   `json:"change_24h"`
			High24h      float64   `json:"high_24h"`
			Low24h       float64   `json:"low_24h"`
		}

		if err := json.Unmarshal([]byte(result), &data); err == nil {
			// Создаем PriceData структуру
			priceData := &storage.PriceData{
				Symbol:       data.Symbol,
				Price:        data.Price,
				Volume24h:    data.Volume24h,
				VolumeUSD:    data.VolumeUSD,
				Timestamp:    data.Timestamp,
				OpenInterest: data.OpenInterest,
				FundingRate:  data.FundingRate,
				Change24h:    data.Change24h,
				High24h:      data.High24h,
				Low24h:       data.Low24h,
			}
			history = append(history, priceData)
		}
	}

	// Сортируем по времени (старые -> новые)
	sort.Slice(history, func(i, j int) bool {
		return history[i].GetTimestamp().Before(history[j].GetTimestamp())
	})

	return history, nil
}

// GetHistoryRange возвращает историю за период
func (hm *HistoryManager) GetHistoryRange(symbol string, start, end time.Time) ([]storage.PriceDataInterface, error) {
	if hm.client == nil {
		return nil, fmt.Errorf("клиент Redis не инициализирован")
	}

	historyKey := hm.prefix + "history:" + symbol

	// Используем ZRANGEBYSCORE для получения данных в диапазоне
	results, err := hm.client.ZRangeByScore(hm.ctx, historyKey, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", start.Unix()),
		Max: fmt.Sprintf("%d", end.Unix()),
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории из Redis: %w", err)
	}

	var history []storage.PriceDataInterface
	for _, result := range results {
		var data struct {
			Symbol       string    `json:"symbol"`
			Price        float64   `json:"price"`
			Volume24h    float64   `json:"volume_24h"`
			VolumeUSD    float64   `json:"volume_usd"`
			Timestamp    time.Time `json:"timestamp"`
			OpenInterest float64   `json:"open_interest"`
			FundingRate  float64   `json:"funding_rate"`
			Change24h    float64   `json:"change_24h"`
			High24h      float64   `json:"high_24h"`
			Low24h       float64   `json:"low_24h"`
		}

		if err := json.Unmarshal([]byte(result), &data); err == nil {
			// Создаем PriceData структуру
			priceData := &storage.PriceData{
				Symbol:       data.Symbol,
				Price:        data.Price,
				Volume24h:    data.Volume24h,
				VolumeUSD:    data.VolumeUSD,
				Timestamp:    data.Timestamp,
				OpenInterest: data.OpenInterest,
				FundingRate:  data.FundingRate,
				Change24h:    data.Change24h,
				High24h:      data.High24h,
				Low24h:       data.Low24h,
			}
			history = append(history, priceData)
		}
	}

	return history, nil
}

// CleanupOldHistory очищает старые данные
func (hm *HistoryManager) CleanupOldHistory(maxAge time.Duration) (int, error) {
	if hm.client == nil {
		return 0, fmt.Errorf("клиент Redis не инициализирован")
	}

	cutoffTime := time.Now().Add(-maxAge)
	cutoffUnix := cutoffTime.Unix()

	// Получаем все ключи истории
	pattern := hm.prefix + "history:*"
	var cursor uint64
	keys := make([]string, 0)

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = hm.client.Scan(hm.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, err
		}

		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}

	totalRemoved := 0

	// Очищаем каждый ключ
	for _, key := range keys {
		removed, err := hm.client.ZRemRangeByScore(hm.ctx, key, "-inf", fmt.Sprintf("%d", cutoffUnix)).Result()
		if err != nil {
			logger.Warn("⚠️ Ошибка очистки истории для ключа %s: %v", key, err)
			continue
		}

		totalRemoved += int(removed)

		// Если история пустая, удаляем ключ
		count, err := hm.client.ZCard(hm.ctx, key).Result()
		if err == nil && count == 0 {
			hm.client.Del(hm.ctx, key)
		}
	}

	if totalRemoved > 0 {
		logger.Debug("🧹 HistoryManager: удалено %d старых записей (старше %v)", totalRemoved, maxAge)
	}

	return totalRemoved, nil
}

// TruncateHistory ограничивает историю
func (hm *HistoryManager) TruncateHistory(symbol string, maxPoints int) error {
	if hm.client == nil {
		return fmt.Errorf("клиент Redis не инициализирован")
	}

	historyKey := hm.prefix + "history:" + symbol

	// Получаем текущее количество
	count, err := hm.client.ZCard(hm.ctx, historyKey).Result()
	if err != nil {
		return err
	}

	if count <= int64(maxPoints) {
		return nil
	}

	// Удаляем лишние старые записи
	removeCount := count - int64(maxPoints)
	_, err = hm.client.ZRemRangeByRank(hm.ctx, historyKey, 0, removeCount-1).Result()
	return err
}

// GetSymbolsWithHistory возвращает символы с историей
func (hm *HistoryManager) GetSymbolsWithHistory() ([]string, error) {
	if hm.client == nil {
		return nil, fmt.Errorf("клиент Redis не инициализирован")
	}

	pattern := hm.prefix + "history:*"
	var cursor uint64
	symbols := make(map[string]bool)

	for {
		var keys []string
		var err error
		keys, cursor, err = hm.client.Scan(hm.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			// Извлекаем symbol из ключа: price:history:BTCUSDT
			parts := strings.Split(key, ":")
			if len(parts) >= 3 {
				symbol := parts[2]
				symbols[symbol] = true
			}
		}

		if cursor == 0 {
			break
		}
	}

	result := make([]string, 0, len(symbols))
	for symbol := range symbols {
		result = append(result, symbol)
	}

	sort.Strings(result)
	return result, nil
}
