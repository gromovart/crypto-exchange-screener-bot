// internal/core/domain/analysis/sr_zones/calculator.go
package sr_zones

import (
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
	"math"
	"sort"
	"time"
)

const (
	// lookback — сколько свечей смотрим до/после для определения pivot
	defaultLookback = 3
	// tolerance — допустимое отклонение цен в кластере (0.5%)
	defaultTolerance = 0.005
	// maxZones — максимум зон на один период
	maxZones = 10

	// minWallUSD — минимальный абсолютный размер стены ордеров в USD.
	// Ордер считается "стеной" только если его USD-объём превышает этот порог
	// И статистически значимо превышает средний размер ордеров в стакане (mean+2σ).
	// Предотвращает ложные стены на монетах с маленькими ордерами.
	minWallUSD = 50_000
)

// Calculator вычисляет S/R зоны по истории свечей.
type Calculator struct {
	lookback  int
	tolerance float64
}

// NewCalculator создаёт калькулятор с параметрами по умолчанию.
func NewCalculator() *Calculator {
	return &Calculator{
		lookback:  defaultLookback,
		tolerance: defaultTolerance,
	}
}

// FindZones находит зоны поддержки и сопротивления.
// candles должны быть отсортированы от старых к новым.
func (c *Calculator) FindZones(symbol, period string, candles []storage.CandleInterface) []Zone {
	if len(candles) < c.lookback*2+1 {
		return nil
	}

	pivotHighs := c.findPivotHighs(candles)
	pivotLows := c.findPivotLows(candles)

	resistances := c.clusterLevels(symbol, period, pivotHighs, ZoneTypeResistance)
	supports := c.clusterLevels(symbol, period, pivotLows, ZoneTypeSupport)

	all := append(resistances, supports...)

	// Сортируем по силе убыванием и берём топ maxZones
	sort.Slice(all, func(i, j int) bool {
		return all[i].Strength > all[j].Strength
	})

	if len(all) > maxZones {
		all = all[:maxZones]
	}
	return all
}

// pivotPoint — точка разворота со временем касания и объёмом
type pivotPoint struct {
	price     float64
	volume    float64
	touchTime time.Time
}

// findPivotHighs ищет локальные максимумы.
func (c *Calculator) findPivotHighs(candles []storage.CandleInterface) []pivotPoint {
	var pivots []pivotPoint
	n := len(candles)
	lb := c.lookback

	for i := lb; i < n-lb; i++ {
		high := candles[i].GetHigh()
		isPivot := true
		for j := 1; j <= lb; j++ {
			if candles[i-j].GetHigh() >= high || candles[i+j].GetHigh() >= high {
				isPivot = false
				break
			}
		}
		if isPivot {
			pivots = append(pivots, pivotPoint{
				price:     high,
				volume:    candles[i].GetVolumeUSD(),
				touchTime: candles[i].GetStartTime(),
			})
		}
	}
	return pivots
}

// findPivotLows ищет локальные минимумы.
func (c *Calculator) findPivotLows(candles []storage.CandleInterface) []pivotPoint {
	var pivots []pivotPoint
	n := len(candles)
	lb := c.lookback

	for i := lb; i < n-lb; i++ {
		low := candles[i].GetLow()
		isPivot := true
		for j := 1; j <= lb; j++ {
			if candles[i-j].GetLow() <= low || candles[i+j].GetLow() <= low {
				isPivot = false
				break
			}
		}
		if isPivot {
			pivots = append(pivots, pivotPoint{
				price:     low,
				volume:    candles[i].GetVolumeUSD(),
				touchTime: candles[i].GetStartTime(),
			})
		}
	}
	return pivots
}

// clusterLevels группирует pivot-точки в зоны.
func (c *Calculator) clusterLevels(symbol, period string, pivots []pivotPoint, zoneType ZoneType) []Zone {
	if len(pivots) == 0 {
		return nil
	}

	// Сортируем pivot-точки по цене
	sort.Slice(pivots, func(i, j int) bool {
		return pivots[i].price < pivots[j].price
	})

	var zones []Zone
	used := make([]bool, len(pivots))

	for i := range pivots {
		if used[i] {
			continue
		}

		// Начинаем новый кластер
		cluster := []pivotPoint{pivots[i]}
		used[i] = true
		refPrice := pivots[i].price

		for j := i + 1; j < len(pivots); j++ {
			if used[j] {
				continue
			}
			// Если цена в пределах tolerance от центра кластера
			if math.Abs(pivots[j].price-refPrice)/refPrice <= c.tolerance {
				cluster = append(cluster, pivots[j])
				used[j] = true
				// Пересчитываем опорную цену кластера как среднее
				sum := 0.0
				for _, p := range cluster {
					sum += p.price
				}
				refPrice = sum / float64(len(cluster))
			}
		}

		// Нужно минимум 2 касания для валидной зоны
		if len(cluster) < 2 {
			continue
		}

		zone := c.buildZone(symbol, period, zoneType, cluster)
		zones = append(zones, zone)
	}
	return zones
}

// buildZone строит Zone из кластера pivot-точек.
func (c *Calculator) buildZone(symbol, period string, zoneType ZoneType, cluster []pivotPoint) Zone {
	var sumPrice, sumVolume float64
	priceMin := math.MaxFloat64
	priceMax := -math.MaxFloat64
	var lastTouch time.Time

	for _, p := range cluster {
		sumPrice += p.price
		sumVolume += p.volume
		if p.price < priceMin {
			priceMin = p.price
		}
		if p.price > priceMax {
			priceMax = p.price
		}
		if p.touchTime.After(lastTouch) {
			lastTouch = p.touchTime
		}
	}

	center := sumPrice / float64(len(cluster))
	touchCount := len(cluster)

	// Скоринг: базовый вклад каждого касания — 15 баллов, макс 100
	strength := math.Min(100, float64(touchCount)*15)
	// +10 баллов за высокий объём (если суммарный объём > 0)
	if sumVolume > 0 {
		strength = math.Min(100, strength+10)
	}

	return Zone{
		Symbol:      symbol,
		Period:      period,
		Type:        zoneType,
		PriceCenter: center,
		PriceHigh:   priceMax,
		PriceLow:    priceMin,
		Strength:    strength,
		TouchCount:  touchCount,
		Volume:      sumVolume,
		LastTouch:   lastTouch,
		CreatedAt:   time.Now(),
	}
}

// EnrichWithOrderBook обогащает зоны данными стакана ордеров.
func EnrichWithOrderBook(zones []Zone, book *OrderBook) []Zone {
	if book == nil || (len(book.Bids) == 0 && len(book.Asks) == 0) {
		return zones
	}

	// Считаем mean+2σ для bids и asks отдельно
	bidMean, bidStd := meanStd(book.Bids)
	askMean, askStd := meanStd(book.Asks)

	wallThresholdBid := bidMean + 2*bidStd
	wallThresholdAsk := askMean + 2*askStd

	for i := range zones {
		z := &zones[i]
		var wallUSD float64

		// Ищем крупные ордера в диапазоне зоны
		if z.Type == ZoneTypeSupport {
			for _, level := range book.Bids {
				if level.Price >= z.PriceLow && level.Price <= z.PriceHigh {
					levelUSD := level.Size * level.Price
					if level.Size >= wallThresholdBid && levelUSD >= minWallUSD {
						wallUSD += levelUSD
					}
				}
			}
		} else {
			for _, level := range book.Asks {
				if level.Price >= z.PriceLow && level.Price <= z.PriceHigh {
					levelUSD := level.Size * level.Price
					if level.Size >= wallThresholdAsk && levelUSD >= minWallUSD {
						wallUSD += levelUSD
					}
				}
			}
		}

		if wallUSD > 0 {
			z.HasOrderWall = true
			z.OrderWallSizeUSD = wallUSD
		}
	}
	return zones
}

// meanStd вычисляет среднее и стандартное отклонение размеров ордеров.
func meanStd(levels []OrderLevel) (mean, std float64) {
	if len(levels) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, l := range levels {
		sum += l.Size
	}
	mean = sum / float64(len(levels))

	variance := 0.0
	for _, l := range levels {
		d := l.Size - mean
		variance += d * d
	}
	variance /= float64(len(levels))
	std = math.Sqrt(variance)
	return
}
