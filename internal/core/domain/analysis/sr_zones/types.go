// internal/core/domain/analysis/sr_zones/types.go
package sr_zones

import "time"

// ZoneType — тип зоны
type ZoneType string

const (
	ZoneTypeSupport    ZoneType = "support"
	ZoneTypeResistance ZoneType = "resistance"
)

// Zone — зона поддержки или сопротивления
type Zone struct {
	Symbol           string    `json:"symbol"`
	Period           string    `json:"period"`
	Type             ZoneType  `json:"type"`
	PriceCenter      float64   `json:"price_center"`
	PriceHigh        float64   `json:"price_high"`
	PriceLow         float64   `json:"price_low"`
	Strength         float64   `json:"strength"`            // 0-100, с учётом пробоев
	TouchCount       int       `json:"touch_count"`         // сколько раз уровень устоял
	BreakthroughCount int      `json:"breakthrough_count"`  // сколько раз уровень был пробит
	Volume           float64   `json:"volume"`              // суммарный объём при касаниях
	HasOrderWall     bool      `json:"has_order_wall"`      // есть ли крупная стена в стакане
	OrderWallSizeUSD float64   `json:"order_wall_size_usd"` // объём стены в USDT
	LastTouch        time.Time `json:"last_touch"`
	CreatedAt        time.Time `json:"created_at"`
}

// NearestZones — ближайшие зоны к текущей цене
type NearestZones struct {
	Support        *Zone
	Resistance     *Zone
	DistToSupportPct float64 // % расстояние до поддержки (положительное)
	DistToResistPct  float64 // % расстояние до сопротивления (положительное)
}

// OrderLevel — уровень в стакане ордеров
type OrderLevel struct {
	Price float64
	Size  float64
}

// OrderBook — стакан ордеров
type OrderBook struct {
	Symbol string
	Bids   []OrderLevel // покупатели (ниже цены)
	Asks   []OrderLevel // продавцы (выше цены)
}
