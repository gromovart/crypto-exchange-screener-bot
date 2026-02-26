// internal/infrastructure/persistence/redis_storage/sr_storage/storage.go
package sr_storage

import (
	"context"
	"crypto-exchange-screener-bot/internal/core/domain/analysis/sr_zones"
	redis_service "crypto-exchange-screener-bot/internal/infrastructure/cache/redis"
	"crypto-exchange-screener-bot/pkg/logger"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	zonesKeyPrefix = "sr:zones:"
)

// SRZoneStorage — Redis-хранилище зон поддержки/сопротивления.
// Ключ: sr:zones:{symbol}:{period}
// Структура: ZSET, score = PriceCenter, value = JSON зоны.
type SRZoneStorage struct {
	client *redis.Client
	ctx    context.Context
}

// NewSRZoneStorage создаёт новое хранилище.
func NewSRZoneStorage(redisService *redis_service.RedisService) (*SRZoneStorage, error) {
	if redisService == nil {
		return nil, fmt.Errorf("redisService не инициализирован")
	}
	client := redisService.GetClient()
	if client == nil {
		return nil, fmt.Errorf("redis клиент недоступен")
	}
	return &SRZoneStorage{
		client: client,
		ctx:    context.Background(),
	}, nil
}

func (s *SRZoneStorage) zsetKey(symbol, period string) string {
	return zonesKeyPrefix + symbol + ":" + period
}

// SaveZones сохраняет список зон в Redis (ZSET по цене) и выставляет TTL.
func (s *SRZoneStorage) SaveZones(symbol, period string, zones []sr_zones.Zone) error {
	key := s.zsetKey(symbol, period)

	pipe := s.client.Pipeline()
	// Удаляем старые зоны
	pipe.Del(s.ctx, key)

	for _, z := range zones {
		data, err := json.Marshal(z)
		if err != nil {
			logger.Warn("⚠️ sr_storage: ошибка сериализации зоны %s/%s: %v", symbol, period, err)
			continue
		}
		pipe.ZAdd(s.ctx, key, &redis.Z{
			Score:  z.PriceCenter,
			Member: string(data),
		})
	}

	// TTL = 3× длительность периода
	ttl := periodTTL(period)
	pipe.Expire(s.ctx, key, ttl)

	_, err := pipe.Exec(s.ctx)
	if err != nil {
		return fmt.Errorf("sr_storage: ошибка сохранения зон %s/%s: %w", symbol, period, err)
	}

	logger.Debug("💾 sr_storage: сохранено %d зон для %s/%s (TTL: %v)", len(zones), symbol, period, ttl)
	return nil
}

// GetZones возвращает все зоны для пары symbol/period.
func (s *SRZoneStorage) GetZones(symbol, period string) ([]sr_zones.Zone, error) {
	key := s.zsetKey(symbol, period)

	results, err := s.client.ZRangeByScore(s.ctx, key, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("sr_storage: ошибка чтения зон %s/%s: %w", symbol, period, err)
	}

	zones := make([]sr_zones.Zone, 0, len(results))
	for _, raw := range results {
		var z sr_zones.Zone
		if err := json.Unmarshal([]byte(raw), &z); err != nil {
			logger.Warn("⚠️ sr_storage: ошибка десериализации зоны: %v", err)
			continue
		}
		zones = append(zones, z)
	}
	return zones, nil
}

// GetNearestZones находит ближайшую поддержку и сопротивление к currentPrice.
func (s *SRZoneStorage) GetNearestZones(symbol, period string, currentPrice float64) (sr_zones.NearestZones, error) {
	zones, err := s.GetZones(symbol, period)
	if err != nil {
		return sr_zones.NearestZones{}, err
	}

	var (
		nearestSupport    *sr_zones.Zone
		nearestResistance *sr_zones.Zone
		bestSupportDist   = math.MaxFloat64
		bestResistDist    = math.MaxFloat64
	)

	for i := range zones {
		z := &zones[i]
		diff := z.PriceCenter - currentPrice
		absDiff := math.Abs(diff)

		if diff < 0 {
			// Зона ниже текущей цены → поддержка
			if absDiff < bestSupportDist {
				bestSupportDist = absDiff
				nearestSupport = z
			}
		} else {
			// Зона выше текущей цены → сопротивление
			if absDiff < bestResistDist {
				bestResistDist = absDiff
				nearestResistance = z
			}
		}
	}

	result := sr_zones.NearestZones{
		Support:    nearestSupport,
		Resistance: nearestResistance,
	}

	if nearestSupport != nil && currentPrice > 0 {
		result.DistToSupportPct = (currentPrice - nearestSupport.PriceCenter) / currentPrice * 100
	}
	if nearestResistance != nil && currentPrice > 0 {
		result.DistToResistPct = (nearestResistance.PriceCenter - currentPrice) / currentPrice * 100
	}

	return result, nil
}

// GetNearestZonesWithFallback ищет ближайшие зоны сначала для primaryPeriod,
// затем последовательно для fallbackPeriods, если зоны не найдены.
// Возвращает зоны, строку с фактически использованным периодом и ошибку.
func (s *SRZoneStorage) GetNearestZonesWithFallback(
	symbol, primaryPeriod string,
	currentPrice float64,
	fallbackPeriods []string,
) (sr_zones.NearestZones, string, error) {
	// Собираем полный список периодов для перебора: первичный + fallback (без дубликатов)
	periodsToTry := make([]string, 0, 1+len(fallbackPeriods))
	periodsToTry = append(periodsToTry, primaryPeriod)
	for _, p := range fallbackPeriods {
		if p != primaryPeriod {
			periodsToTry = append(periodsToTry, p)
		}
	}

	for _, period := range periodsToTry {
		nearest, err := s.GetNearestZones(symbol, period, currentPrice)
		if err != nil {
			logger.Debug("⚠️ sr_storage: нет зон %s/%s: %v", symbol, period, err)
			continue
		}
		// Считаем успехом наличие хотя бы одной зоны (поддержка ИЛИ сопротивление)
		if nearest.Support != nil || nearest.Resistance != nil {
			if period != primaryPeriod {
				logger.Debug("📐 sr_storage: fallback зоны %s/%s → использован %s",
					symbol, primaryPeriod, period)
			}
			return nearest, period, nil
		}
	}

	return sr_zones.NearestZones{}, "", fmt.Errorf("нет зон ни для одного периода %s %v",
		symbol, periodsToTry)
}

// periodTTL возвращает TTL для зон в зависимости от периода.
func periodTTL(period string) time.Duration {
	switch period {
	case "1m":
		return 3 * time.Minute
	case "5m":
		return 15 * time.Minute
	case "15m":
		return 45 * time.Minute
	case "30m":
		return 90 * time.Minute
	case "1h":
		return 3 * time.Hour
	case "4h":
		return 12 * time.Hour
	case "1d":
		return 3 * 24 * time.Hour
	default:
		return 1 * time.Hour
	}
}
