// internal/infrastructure/persistence/redis_storage/price_storage/implements.go
package price_storage

import (
	"time"
)





// Реализация методов интерфейса PriceChange
func (pc *PriceChange) GetSymbol() string         { return pc.Symbol }
func (pc *PriceChange) GetCurrentPrice() float64  { return pc.CurrentPrice }
func (pc *PriceChange) GetPreviousPrice() float64 { return pc.PreviousPrice }
func (pc *PriceChange) GetChange() float64        { return pc.Change }
func (pc *PriceChange) GetChangePercent() float64 { return pc.ChangePercent }
func (pc *PriceChange) GetInterval() string       { return pc.Interval }
func (pc *PriceChange) GetTimestamp() time.Time   { return pc.Timestamp }
func (pc *PriceChange) GetVolumeUSD() float64     { return pc.VolumeUSD }
