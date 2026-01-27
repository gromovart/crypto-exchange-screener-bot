// internal/infrastructure/persistence/redis_storage/candle_tracker/candle_tracker.go
package candletracker

import (
	"fmt"

	"crypto-exchange-screener-bot/pkg/logger"
)

// Initialize инициализирует трекер
func (ct *CandleTracker) Initialize() error {
	if ct.redisService == nil {
		return fmt.Errorf("сервис Redis не инициализирован")
	}

	ct.client = ct.redisService.GetClient()
	if ct.client == nil {
		return fmt.Errorf("клиент Redis недоступен")
	}

	logger.Info("✅ CandleTracker инициализирован (TTL: %v)", ct.ttl)
	return nil
}

// MarkCandleProcessedAtomically атомарно помечает свечу как обработанную
// Возвращает true если свеча была успешно помечена как обработанная (не была обработана ранее)
// Возвращает false если свеча уже была обработана
func (ct *CandleTracker) MarkCandleProcessedAtomically(symbol, period string, startTime int64) (bool, error) {
	if ct.client == nil {
		return false, fmt.Errorf("клиент Redis не инициализирован")
	}

	key := ct.generateKey(symbol, period, startTime)

	// Используем SETNX для атомарной проверки и установки
	// SETNX возвращает 1 если ключа не было, 0 если ключ уже существовал
	result, err := ct.client.SetNX(ct.ctx, key, "1", ct.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("ошибка SETNX для ключа %s: %w", key, err)
	}

	return result, nil
}

// IsCandleProcessed проверяет была ли свеча обработана
func (ct *CandleTracker) IsCandleProcessed(symbol, period string, startTime int64) (bool, error) {
	if ct.client == nil {
		return false, fmt.Errorf("клиент Redis не инициализирован")
	}

	key := ct.generateKey(symbol, period, startTime)
	exists, err := ct.client.Exists(ct.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("ошибка проверки ключа %s: %w", key, err)
	}

	return exists == 1, nil
}

// MarkCandleProcessedUnsafe помечает свечу как обработанную (без атомарной проверки)
func (ct *CandleTracker) MarkCandleProcessedUnsafe(symbol, period string, startTime int64) error {
	if ct.client == nil {
		return fmt.Errorf("клиент Redis не инициализирован")
	}

	key := ct.generateKey(symbol, period, startTime)
	err := ct.client.Set(ct.ctx, key, "1", ct.ttl).Err()
	if err != nil {
		return fmt.Errorf("ошибка установки ключа %s: %w", key, err)
	}

	return nil
}

// CleanupOldEntries очищает старые записи (не нужно, так как TTL автоматический)
func (ct *CandleTracker) CleanupOldEntries() (int64, error) {
	// Redis автоматически удаляет ключи с истекшим TTL
	// Этот метод может быть полезен для принудительной очистки
	if ct.client == nil {
		return 0, fmt.Errorf("клиент Redis не инициализирован")
	}

	// Получаем все ключи с префиксом
	pattern := ct.prefix + "*"
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = ct.client.Scan(ct.ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return 0, fmt.Errorf("ошибка SCAN ключей: %w", err)
		}

		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return 0, nil
	}

	// Используем pipeline для массового удаления
	pipe := ct.client.Pipeline()
	for _, key := range keys {
		pipe.Expire(ct.ctx, key, ct.ttl)
	}

	_, err := pipe.Exec(ct.ctx)
	if err != nil {
		return 0, fmt.Errorf("ошибка установки TTL для ключей: %w", err)
	}

	logger.Debug("🧹 CandleTracker: обновлен TTL для %d ключей", len(keys))
	return int64(len(keys)), nil
}

// GetStats возвращает статистику трекера
func (ct *CandleTracker) GetStats() (map[string]interface{}, error) {
	if ct.client == nil {
		return nil, fmt.Errorf("клиент Redis не инициализирован")
	}

	pattern := ct.prefix + "*"
	var cursor uint64
	totalKeys := 0

	for {
		var keys []string
		var err error
		keys, cursor, err = ct.client.Scan(ct.ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("ошибка SCAN ключей для статистики: %w", err)
		}

		totalKeys += len(keys)
		if cursor == 0 {
			break
		}
	}

	return map[string]interface{}{
		"total_tracked_candles": totalKeys,
		"ttl":                   ct.ttl.String(),
		"prefix":                ct.prefix,
	}, nil
}

// generateKey генерирует ключ для свечи
func (ct *CandleTracker) generateKey(symbol, period string, startTime int64) string {
	// Формат: processed_candle:{symbol}:{period}:{startTimeUnix}
	return fmt.Sprintf("%s%s:%s:%d", ct.prefix, symbol, period, startTime)
}

// TestConnection тестирует подключение к Redis
func (ct *CandleTracker) TestConnection() error {
	if ct.client == nil {
		return fmt.Errorf("клиент Redis не инициализирован")
	}

	_, err := ct.client.Ping(ct.ctx).Result()
	if err != nil {
		return fmt.Errorf("ошибка подключения к Redis: %w", err)
	}

	return nil
}
