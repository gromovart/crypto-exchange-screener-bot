// internal/infrastructure/persistence/redis_storage/candle_storage/implements.go
package candle_storage

import (
	"time"
)

// Реализация интерфейса interfaces.Candle для CandleData
func (c *CandleData) GetSymbol() string       { return c.Symbol }
func (c *CandleData) GetPeriod() string       { return c.Period }
func (c *CandleData) GetOpen() float64        { return c.Open }
func (c *CandleData) GetHigh() float64        { return c.High }
func (c *CandleData) GetLow() float64         { return c.Low }
func (c *CandleData) GetClose() float64       { return c.Close }
func (c *CandleData) GetVolume() float64      { return c.Volume }
func (c *CandleData) GetVolumeUSD() float64   { return c.VolumeUSD }
func (c *CandleData) GetTrades() int          { return c.Trades }
func (c *CandleData) GetStartTime() time.Time { return c.StartTime }
func (c *CandleData) GetEndTime() time.Time   { return c.EndTime }
func (c *CandleData) IsClosed() bool          { return c.IsClosedFlag }
func (c *CandleData) IsReal() bool            { return c.IsRealFlag }

// Реализация интерфейса interfaces.CandleConfig для CandleConfigData
func (c *CandleConfigData) GetSupportedPeriods() []string     { return c.SupportedPeriods }
func (c *CandleConfigData) GetMaxHistory() int                { return c.MaxHistory }
func (c *CandleConfigData) GetCleanupInterval() time.Duration { return c.CleanupInterval }

// Реализация интерфейса interfaces.CandleStats для CandleStatsData
func (s *CandleStatsData) GetTotalCandles() int            { return s.TotalCandles }
func (s *CandleStatsData) GetActiveCandles() int           { return s.ActiveCandles }
func (s *CandleStatsData) GetSymbolsCount() int            { return s.SymbolsCount }
func (s *CandleStatsData) GetOldestCandle() time.Time      { return s.OldestCandle }
func (s *CandleStatsData) GetNewestCandle() time.Time      { return s.NewestCandle }
func (s *CandleStatsData) GetPeriodsCount() map[string]int { return s.PeriodsCount }
