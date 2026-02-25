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

	// Параметры определения стены ордеров.

	// bucketWidthPct — ширина ценового бакета при агрегации стакана (0.1%).
	// Ордера в пределах 0.1% ценового диапазона объединяются в один бакет.
	// Это позволяет учитывать скопление (плотность) ордеров на уровне,
	// а не только размер отдельных ордеров.
	bucketWidthPct = 0.001

	// wallSearchRadiusPct — радиус поиска стены вокруг центра зоны (±0.5%).
	// Стена ищется в диапазоне [center*(1-radius), center*(1+radius)].
	wallSearchRadiusPct = 0.005

	// minWallVolumePct — минимальный порог стены как процент от 24h объёма (0.05%).
	// Для ARCUSDT $241M → порог $120K; для BTC $30B → порог $15M.
	minWallVolumePct = 0.0005

	// minWallUSD — абсолютный минимальный порог стены в USD.
	// Применяется если 24h объём недоступен или даёт меньшее значение.
	minWallUSD = 50_000

	// wallMultiplier — бакет стены должен быть минимум в 3× больше среднего бакета.
	// Это гарантирует, что стена ВСЕГДА выделяется на фоне обычного стакана —
	// даже при равномерном распределении ордеров (малом σ).
	// Пример: средний бакет $50K → минимальная стена $150K.
	wallMultiplier = 3.0
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

// priceBucket — агрегированный ценовой бакет стакана ордеров.
type priceBucket struct {
	priceKey  float64 // нижняя граница бакета
	volumeUSD float64 // суммарный USD-объём всех ордеров в бакете
	count     int     // количество ордеров в бакете
}

// buildBuckets агрегирует ордера в ценовые бакеты шириной bucketWidthPct.
// Ордера в пределах одного бакета суммируются — это даёт "плотность" на уровне.
func buildBuckets(levels []OrderLevel, widthPct float64) []priceBucket {
	if len(levels) == 0 {
		return nil
	}

	buckets := make(map[float64]*priceBucket)
	for _, l := range levels {
		if l.Price <= 0 {
			continue
		}
		// Ключ бакета: округляем цену вниз до ближайшего кратного bucketWidthPct
		key := math.Floor(l.Price/l.Price*l.Price/(l.Price*widthPct)) * (l.Price * widthPct)
		// Более простая формула: key = floor(price / (price * widthPct)) * (price * widthPct)
		// Эквивалентно: round price to nearest bucketWidthPct fraction
		bucketSize := l.Price * widthPct
		key = math.Floor(l.Price/bucketSize) * bucketSize

		b, ok := buckets[key]
		if !ok {
			b = &priceBucket{priceKey: key}
			buckets[key] = b
		}
		b.volumeUSD += l.Size * l.Price
		b.count++
	}

	result := make([]priceBucket, 0, len(buckets))
	for _, b := range buckets {
		result = append(result, *b)
	}
	return result
}

// bucketMeanStd вычисляет среднее и стандартное отклонение USD-объёма бакетов.
func bucketMeanStd(buckets []priceBucket) (mean, std float64) {
	if len(buckets) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, b := range buckets {
		sum += b.volumeUSD
	}
	mean = sum / float64(len(buckets))

	variance := 0.0
	for _, b := range buckets {
		d := b.volumeUSD - mean
		variance += d * d
	}
	variance /= float64(len(buckets))
	std = math.Sqrt(variance)
	return
}

// computeWallThreshold вычисляет итоговый порог стены как максимум из трёх критериев:
//
//  1. Относительный: bucketMean × wallMultiplier (3×)
//     Бакет всегда должен быть в 3× выше среднего — независимо от σ.
//     Если средний бакет $50K, минимальная стена = $150K, а не $50K.
//
//  2. Статистический: bucketMean + 2σ
//     Стена статистически выделяется на фоне остального стакана.
//
//  3. Динамический: max(vol24h × 0.05%, minWallUSD)
//     Стена осмысленна в контексте торгового объёма монеты.
//
// Итоговый порог = max(1, 2, 3). Бакет обязан превышать все три условия
// через единое значение — нет дублирующих AND-проверок.
func computeWallThreshold(bucketMean, bucketStd, volume24hUSD float64) float64 {
	// 1. Относительный: в wallMultiplier раз выше среднего
	relative := bucketMean * wallMultiplier

	// 2. Статистический: mean + 2σ
	statistical := bucketMean + 2*bucketStd

	// 3. Динамический: процент от 24h объёма или абсолютный минимум
	dynamic := volume24hUSD * minWallVolumePct
	if dynamic < minWallUSD {
		dynamic = minWallUSD
	}

	return math.Max(relative, math.Max(statistical, dynamic))
}

// EnrichWithOrderBook обогащает зоны данными стакана ордеров.
//
// Алгоритм:
//  1. Агрегируем ордера в ценовые бакеты по 0.1% — учитываем плотность (много
//     мелких ордеров на одном уровне суммируются).
//  2. Единый порог = max(mean×3, mean+2σ, vol24h×0.05%) — бакет стены обязан
//     быть в 3× выше среднего независимо от σ, статистически выделяться
//     и быть значимым относительно объёма торгов.
//  3. Ищем стены в радиусе ±0.5% от центра зоны.
func EnrichWithOrderBook(zones []Zone, book *OrderBook, volume24hUSD float64) []Zone {
	if book == nil || (len(book.Bids) == 0 && len(book.Asks) == 0) {
		return zones
	}

	// Строим бакеты
	bidBuckets := buildBuckets(book.Bids, bucketWidthPct)
	askBuckets := buildBuckets(book.Asks, bucketWidthPct)

	// Единый порог для bids и asks: max(mean×3, mean+2σ, dynFloor)
	bidMean, bidStd := bucketMeanStd(bidBuckets)
	askMean, askStd := bucketMeanStd(askBuckets)
	bidThreshold := computeWallThreshold(bidMean, bidStd, volume24hUSD)
	askThreshold := computeWallThreshold(askMean, askStd, volume24hUSD)

	for i := range zones {
		z := &zones[i]

		// Диапазон поиска стены: ±wallSearchRadiusPct вокруг центра зоны
		searchLow := z.PriceCenter * (1 - wallSearchRadiusPct)
		searchHigh := z.PriceCenter * (1 + wallSearchRadiusPct)

		var wallUSD float64
		var threshold float64
		var buckets []priceBucket

		if z.Type == ZoneTypeSupport {
			buckets = bidBuckets
			threshold = bidThreshold
		} else {
			buckets = askBuckets
			threshold = askThreshold
		}

		for _, b := range buckets {
			if b.priceKey < searchLow || b.priceKey > searchHigh {
				continue
			}
			if b.volumeUSD >= threshold {
				wallUSD += b.volumeUSD
			}
		}

		if wallUSD > 0 {
			z.HasOrderWall = true
			z.OrderWallSizeUSD = wallUSD
		}
	}
	return zones
}
