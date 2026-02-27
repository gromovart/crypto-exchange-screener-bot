// internal/core/domain/signals/detectors/counter/calculator/volume_delta_fallback.go
package calculator

import (
	"math"
	"time"

	"crypto-exchange-screener-bot/internal/types"
	"crypto-exchange-screener-bot/pkg/logger"
)

// getFromStorage получает дельту из хранилища
func (c *VolumeDeltaCalculator) getFromStorage(symbol, direction string) *types.VolumeDeltaData {
	// Эта функция требует доступа к storage интерфейсу
	// Возвращаем nil - реализация зависит от конкретного storage
	// В реальной реализации здесь нужно получить данные из storage

	// Временная заглушка - можно раскомментировать когда будет storage доступ
	/*
		if c.storage != nil {
			// Пример: если storage имеет метод GetSnapshot
			if storage, ok := c.storage.(interface {
				GetCurrentSnapshot(string) (interface{}, bool)
			}); ok {
				if snapshot, exists := storage.GetCurrentSnapshot(symbol); exists {
					// Используем snapshot для расчета дельты
				}
			}
		}
	*/

	return nil
}

// calculateBasicDelta базовая эмуляция дельты
func (c *VolumeDeltaCalculator) calculateBasicDelta(symbol, direction string) *types.VolumeDeltaData {
	// Эмуляция дельты на основе направления
	var deltaPercent float64

	if direction == "growth" {
		deltaPercent = 2.0 // +2.0% для роста
	} else if direction == "fall" {
		deltaPercent = -2.0 // -2.0% для падения
	} else {
		deltaPercent = 1.0 // +1.0% для нейтрального
	}

	// Базовый объем
	volume24h := c.estimateVolume(symbol)
	delta := volume24h * deltaPercent / 100

	// Ограничиваем максимальную/минимальную дельту
	maxDelta := volume24h * 0.05
	if math.Abs(delta) > maxDelta {
		delta = maxDelta * math.Copysign(1, deltaPercent)
		deltaPercent = (delta / volume24h) * 100
	}

	logger.Info("📊 Базовая дельта для %s: $%.0f (%.1f%%) от объема $%.0f",
		symbol, delta, deltaPercent, volume24h)

	return &types.VolumeDeltaData{
		Delta:        delta,
		DeltaPercent: deltaPercent,
		Source:       types.VolumeDeltaSourceEmulated,
		Timestamp:    time.Now(),
		IsRealData:   false,
	}
}

// estimateVolume оценивает объем для символа
func (c *VolumeDeltaCalculator) estimateVolume(symbol string) float64 {
	// Базовые оценки объемов для разных типов символов
	if len(symbol) >= 4 && symbol[len(symbol)-4:] == "USDT" {
		return 5000000 // $5M для USDT пар
	}
	if len(symbol) >= 3 && symbol[len(symbol)-3:] == "USD" {
		return 3000000 // $3M для USD пар
	}
	return 2000000 // $2M по умолчанию
}

// GetCacheInfo возвращает информацию о кэше
func (c *VolumeDeltaCalculator) GetCacheInfo() map[string]interface{} {
	c.volumeDeltaCacheMu.RLock()
	defer c.volumeDeltaCacheMu.RUnlock()

	info := make(map[string]interface{})
	info["cache_size"] = len(c.volumeDeltaCache)
	info["ttl"] = c.volumeDeltaTTL.String()

	symbolsInfo := make(map[string]interface{})
	for symbol, cache := range c.volumeDeltaCache {
		age := time.Since(cache.updateTime).Round(time.Second)
		symbolsInfo[symbol] = map[string]interface{}{
			"delta":         cache.deltaData.Delta,
			"delta_percent": cache.deltaData.DeltaPercent,
			"source":        cache.deltaData.Source,
			"age":           age.String(),
			"expires_in":    time.Until(cache.expiration).Round(time.Second).String(),
		}
	}
	info["cached_symbols"] = symbolsInfo

	return info
}
