// internal/infrastructure/persistence/redis_storage/implements.go
package redis_storage

import "time"

// Реализация методов интерфейса StorageConfig для структуры StorageConfig
func (sc *StorageConfig) GetMaxHistoryPerSymbol() int       { return sc.MaxHistoryPerSymbol }
func (sc *StorageConfig) GetMaxSymbols() int                { return sc.MaxSymbols }
func (sc *StorageConfig) GetCleanupInterval() time.Duration { return sc.CleanupInterval }
func (sc *StorageConfig) GetRetentionPeriod() time.Duration { return sc.RetentionPeriod }
func (sc *StorageConfig) GetEnableCompression() bool        { return sc.EnableCompression }
func (sc *StorageConfig) GetEnablePersistence() bool        { return sc.EnablePersistence }
func (sc *StorageConfig) GetPersistencePath() string        { return sc.PersistencePath }

// Реализация методов интерфейса SymbolMetrics для структуры SymbolMetrics
func (sm *SymbolMetrics) GetSymbol() string         { return sm.Symbol }
func (sm *SymbolMetrics) GetPrice() float64         { return sm.Price }
func (sm *SymbolMetrics) GetVolume24h() float64     { return sm.Volume24h }
func (sm *SymbolMetrics) GetVolumeUSD() float64     { return sm.VolumeUSD }
func (sm *SymbolMetrics) GetTimestamp() time.Time   { return sm.Timestamp }
func (sm *SymbolMetrics) GetOpenInterest() float64  { return sm.OpenInterest }
func (sm *SymbolMetrics) GetFundingRate() float64   { return sm.FundingRate }
func (sm *SymbolMetrics) GetChange24h() float64     { return sm.Change24h }
func (sm *SymbolMetrics) GetHigh24h() float64       { return sm.High24h }
func (sm *SymbolMetrics) GetLow24h() float64        { return sm.Low24h }
func (sm *SymbolMetrics) GetOIChange24h() float64   { return sm.OIChange24h }
func (sm *SymbolMetrics) GetFundingChange() float64 { return sm.FundingChange }

// Реализация методов интерфейса SymbolStats для структуры SymbolStats
func (ss *SymbolStats) GetSymbol() string            { return ss.Symbol }
func (ss *SymbolStats) GetDataPoints() int           { return ss.DataPoints }
func (ss *SymbolStats) GetFirstTimestamp() time.Time { return ss.FirstTimestamp }
func (ss *SymbolStats) GetLastTimestamp() time.Time  { return ss.LastTimestamp }
func (ss *SymbolStats) GetCurrentPrice() float64     { return ss.CurrentPrice }
func (ss *SymbolStats) GetAvgVolume24h() float64     { return ss.AvgVolume24h }
func (ss *SymbolStats) GetAvgVolumeUSD() float64     { return ss.AvgVolumeUSD }
func (ss *SymbolStats) GetPriceChange24h() float64   { return ss.PriceChange24h }
func (ss *SymbolStats) GetOpenInterest() float64     { return ss.OpenInterest }
func (ss *SymbolStats) GetOIChange24h() float64      { return ss.OIChange24h }
func (ss *SymbolStats) GetFundingRate() float64      { return ss.FundingRate }
func (ss *SymbolStats) GetFundingChange() float64    { return ss.FundingChange }
func (ss *SymbolStats) GetHigh24h() float64          { return ss.High24h }
func (ss *SymbolStats) GetLow24h() float64           { return ss.Low24h }

// Реализация методов интерфейса StorageStats для структуры StorageStats
func (ss *StorageStats) GetTotalSymbols() int              { return ss.TotalSymbols }
func (ss *StorageStats) GetTotalDataPoints() int64         { return ss.TotalDataPoints }
func (ss *StorageStats) GetMemoryUsageBytes() int64        { return ss.MemoryUsageBytes }
func (ss *StorageStats) GetOldestTimestamp() time.Time     { return ss.OldestTimestamp }
func (ss *StorageStats) GetNewestTimestamp() time.Time     { return ss.NewestTimestamp }
func (ss *StorageStats) GetUpdateRatePerSecond() float64   { return ss.UpdateRatePerSecond }
func (ss *StorageStats) GetStorageType() string            { return ss.StorageType }
func (ss *StorageStats) GetMaxHistoryPerSymbol() int       { return ss.MaxHistoryPerSymbol }
func (ss *StorageStats) GetRetentionPeriod() time.Duration { return ss.RetentionPeriod }
func (ss *StorageStats) GetSymbolsWithOI() int             { return ss.SymbolsWithOI }
func (ss *StorageStats) GetSymbolsWithFunding() int        { return ss.SymbolsWithFunding }

// Реализация методов интерфейса SymbolVolume для структуры SymbolVolume
func (sv *SymbolVolume) GetSymbol() string     { return sv.Symbol }
func (sv *SymbolVolume) GetVolume() float64    { return sv.Volume }
func (sv *SymbolVolume) GetVolumeUSD() float64 { return sv.VolumeUSD }

// Реализация методов интерфейса PriceData для структуры PriceData
func (pd *PriceData) GetSymbol() string        { return pd.Symbol }
func (pd *PriceData) GetPrice() float64        { return pd.Price }
func (pd *PriceData) GetVolume24h() float64    { return pd.Volume24h }
func (pd *PriceData) GetVolumeUSD() float64    { return pd.VolumeUSD }
func (pd *PriceData) GetTimestamp() time.Time  { return pd.Timestamp }
func (pd *PriceData) GetOpenInterest() float64 { return pd.OpenInterest }
func (pd *PriceData) GetFundingRate() float64  { return pd.FundingRate }
func (pd *PriceData) GetChange24h() float64    { return pd.Change24h }
func (pd *PriceData) GetHigh24h() float64      { return pd.High24h }
func (pd *PriceData) GetLow24h() float64       { return pd.Low24h }

// Реализация методов интерфейса PriceSnapshot для структуры PriceSnapshot
func (ps *PriceSnapshot) GetSymbol() string        { return ps.Symbol }
func (ps *PriceSnapshot) GetPrice() float64        { return ps.Price }
func (ps *PriceSnapshot) GetVolume24h() float64    { return ps.Volume24h }
func (ps *PriceSnapshot) GetVolumeUSD() float64    { return ps.VolumeUSD }
func (ps *PriceSnapshot) GetTimestamp() time.Time  { return ps.Timestamp }
func (ps *PriceSnapshot) GetOpenInterest() float64 { return ps.OpenInterest }
func (ps *PriceSnapshot) GetFundingRate() float64  { return ps.FundingRate }
func (ps *PriceSnapshot) GetChange24h() float64    { return ps.Change24h }
func (ps *PriceSnapshot) GetHigh24h() float64      { return ps.High24h }
func (ps *PriceSnapshot) GetLow24h() float64       { return ps.Low24h }
