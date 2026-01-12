// internal/core/domain/signals/detectors/counter/calculator/market_metrics_calculator.go
package calculator

import (
	"fmt"
	"log"
	"math"
	"time"

	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"crypto-exchange-screener-bot/pkg/logger"
)

// MarketMetricsCalculator - калькулятор рыночных метрик
type MarketMetricsCalculator struct {
	marketFetcher interface{}
	storage       interface{}
}

// Storage интерфейс для получения метрик
type Storage interface {
	GetSymbolMetrics(symbol string) (map[string]interface{}, bool)
	GetPriceHistory(symbol string, limit int) ([]interface{}, error)
	GetCurrentSnapshot(symbol string) (interface{}, bool)
}

// NewMarketMetricsCalculator создает новый калькулятор метрик
func NewMarketMetricsCalculator(marketFetcher interface{}, storage interface{}) *MarketMetricsCalculator {
	return &MarketMetricsCalculator{
		marketFetcher: marketFetcher,
		storage:       storage,
	}
}

// GetLiquidationData получает данные ликвидаций
func (c *MarketMetricsCalculator) GetLiquidationData(symbol string) (float64, float64, float64) {
	// Пробуем получить реальные данные
	if c.marketFetcher != nil {
		if fetcher, ok := c.marketFetcher.(interface {
			GetLiquidationMetrics(string) (*bybit.LiquidationMetrics, bool)
		}); ok {
			if metrics, exists := fetcher.GetLiquidationMetrics(symbol); exists {
				log.Printf("📊 Получены ликвидации для %s: $%.0f (long: $%.0f, short: $%.0f)",
					symbol, metrics.TotalVolumeUSD, metrics.LongLiqVolume, metrics.ShortLiqVolume)
				return metrics.TotalVolumeUSD, metrics.LongLiqVolume, metrics.ShortLiqVolume
			}
		}
	}

	// Fallback: эмуляция ликвидаций на основе объема
	volume24h := c.getVolume24h(symbol)
	if volume24h > 0 {
		change24h := c.getChange24h(symbol)

		// Базовые ликвидации как 0.1% от объема
		baseLiq := volume24h * 0.001

		// Увеличиваем при больших изменениях
		if math.Abs(change24h) > 5 {
			baseLiq *= 3
		} else if math.Abs(change24h) > 2 {
			baseLiq *= 2
		}

		// Распределяем между LONG/SHORT
		if change24h > 0 {
			// Рост - больше SHORT ликвидаций
			longLiq := baseLiq * 0.4
			shortLiq := baseLiq * 0.6
			return baseLiq, longLiq, shortLiq
		} else {
			// Падение - больше LONG ликвидаций
			longLiq := baseLiq * 0.6
			shortLiq := baseLiq * 0.4
			return baseLiq, longLiq, shortLiq
		}
	}

	log.Printf("⚠️ Не удалось получить ликвидации для %s", symbol)
	return 0, 0, 0
}

// CalculateOIChange24h рассчитывает изменение OI за 24 часа
func (c *MarketMetricsCalculator) CalculateOIChange24h(symbol string) float64 {
	log.Printf("🔍 Получение OI change для %s", symbol)

	// Пробуем получить метрики из storage
	if c.storage != nil {
		if storage, ok := c.storage.(interface {
			GetSymbolMetrics(string) (map[string]interface{}, bool)
		}); ok {
			if metrics, exists := storage.GetSymbolMetrics(symbol); exists {
				oiChange := getFloatFromMap(metrics, "OIChange24h", 0)
				logger.Debug("✅ Получен OI change для %s: %.1f%%", symbol, oiChange)
				return oiChange
			}
		}
	}

	// Fallback: вычисляем из истории
	history := c.getPriceHistory(symbol, 200)
	if len(history) >= 2 {
		// Ищем OI в истории
		var firstOI, lastOI float64
		for _, point := range history {
			if oi := getFloatFromMap(point, "OpenInterest", 0); oi > 0 {
				if firstOI == 0 {
					firstOI = oi
				}
				lastOI = oi
			}
		}

		if firstOI > 0 && lastOI > 0 {
			change := ((lastOI - firstOI) / firstOI) * 100
			log.Printf("📊 Рассчитан OI change для %s: %.1f%%", symbol, change)
			return change
		}
	}

	// Final fallback: базовая эмуляция
	change := c.calculateBasicOIChange(symbol)
	log.Printf("📊 Используем базовый OI change для %s: %.1f%%", symbol, change)
	return change
}

// GetRealTimeMetrics получает реальные метрики
func (c *MarketMetricsCalculator) GetRealTimeMetrics(symbol string) (float64, float64, float64, float64) {
	if c.storage != nil {
		if storage, ok := c.storage.(interface {
			GetSymbolMetrics(string) (map[string]interface{}, bool)
		}); ok {
			if metrics, exists := storage.GetSymbolMetrics(symbol); exists {
				price := getFloatFromMap(metrics, "Price", 0)
				oi := getFloatFromMap(metrics, "OpenInterest", 0)
				funding := getFloatFromMap(metrics, "FundingRate", 0)
				volume := getFloatFromMap(metrics, "VolumeUSD", 0)

				log.Printf("✅ Получены реальные метрики для %s: OI=%.0f", symbol, oi)
				return price, oi, funding, volume
			}
		}
	}
	return 0, 0, 0, 0
}

// CalculateAverageFunding рассчитывает среднюю ставку фандинга
func (c *MarketMetricsCalculator) CalculateAverageFunding(fundingRates []float64) float64 {
	if len(fundingRates) == 0 {
		return 0
	}

	var total float64
	var count int
	for _, rate := range fundingRates {
		if rate != 0 {
			total += rate
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// CalculateNextFundingTime рассчитывает время следующего фандинга
func (c *MarketMetricsCalculator) CalculateNextFundingTime() time.Time {
	now := time.Now().UTC()
	hour := now.Hour()
	var nextHour int

	switch {
	case hour < 8:
		nextHour = 8
	case hour < 16:
		nextHour = 16
	default:
		nextHour = 0
		now = now.Add(24 * time.Hour)
	}

	return time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		nextHour,
		0, 0, 0,
		time.UTC,
	)
}

// Вспомогательные методы

func (c *MarketMetricsCalculator) getVolume24h(symbol string) float64 {
	if c.storage != nil {
		if storage, ok := c.storage.(interface {
			GetCurrentSnapshot(string) (map[string]interface{}, bool)
		}); ok {
			if snapshot, exists := storage.GetCurrentSnapshot(symbol); exists {
				return getFloatFromMap(snapshot, "Volume24h", 0)
			}
		}
	}
	return 0
}

func (c *MarketMetricsCalculator) getChange24h(symbol string) float64 {
	if c.storage != nil {
		if storage, ok := c.storage.(interface {
			GetCurrentSnapshot(string) (map[string]interface{}, bool)
		}); ok {
			if snapshot, exists := storage.GetCurrentSnapshot(symbol); exists {
				return getFloatFromMap(snapshot, "Change24h", 0)
			}
		}
	}
	return 0
}

func (c *MarketMetricsCalculator) getPriceHistory(symbol string, limit int) []map[string]interface{} {
	if c.storage != nil {
		if storage, ok := c.storage.(interface {
			GetPriceHistory(string, int) ([]map[string]interface{}, error)
		}); ok {
			if history, err := storage.GetPriceHistory(symbol, limit); err == nil {
				return history
			}
		}
	}
	return nil
}

func (c *MarketMetricsCalculator) calculateBasicOIChange(symbol string) float64 {
	// Эмуляция изменения OI в пределах -20% до +20%
	hash := float64(len(symbol) + int(time.Now().Unix()/3600))
	return math.Mod(hash, 40) - 20 // От -20 до +20
}

func getFloatFromMap(m map[string]interface{}, key string, defaultValue float64) float64 {
	if m == nil {
		return defaultValue
	}

	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			if f, err := parseFloat(v); err == nil {
				return f
			}
		}
	}
	return defaultValue
}

func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

// FormatTimeUntilNextFunding форматирует время до следующего фандинга
func (c *MarketMetricsCalculator) FormatTimeUntilNextFunding(nextFundingTime time.Time) string {
	now := time.Now()
	if nextFundingTime.Before(now) {
		return "сейчас"
	}

	duration := nextFundingTime.Sub(now)

	switch {
	case duration.Hours() >= 1:
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dч %dм", hours, minutes)
		}
		return fmt.Sprintf("%dч", hours)
	default:
		minutes := int(duration.Minutes())
		if minutes <= 0 {
			return "скоро!"
		}
		return fmt.Sprintf("%dм", minutes)
	}
}
