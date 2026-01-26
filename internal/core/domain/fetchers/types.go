package fetchers

import (
	"time"
)

// PriceFetcher интерфейс
type PriceFetcher interface {
	Start(interval time.Duration) error
	Stop() error
	IsRunning() bool
	GetStats() map[string]interface{}
}

// PriceFetcherConfig - конфигурация PriceFetcher
type PriceFetcherConfig struct {
	UpdateInterval      time.Duration
	MaxConcurrent       int
	RequestTimeout      time.Duration
	SymbolFilter        string
	ExcludeSymbols      string
	MaxSymbolsToMonitor int
	MinVolumeFilter     float64
	InitialDataFetch    bool
	DataFetchLimit      int
	FuturesCategory     string
}

// TrendSignal сигнал тренда
type TrendSignal struct {
	Symbol        string    `json:"symbol"`
	Direction     string    `json:"direction"` // "growth" или "fall"
	ChangePercent float64   `json:"change_percent"`
	PeriodMinutes int       `json:"period_minutes"`
	Confidence    float64   `json:"confidence"`
	Timestamp     time.Time `json:"timestamp"`
	DataPoints    int       `json:"data_points"`
	VolumeUSD     float64   `json:"volume_usd,omitempty"` // ← ОПЦИОНАЛЬНО: можно добавить для анализа
}

// NotificationService интерфейс сервиса уведомлений
type NotificationService interface {
	Send(signal TrendSignal) error
	SendBatch(signals []TrendSignal) error
	SetEnabled(enabled bool)
	IsEnabled() bool
	GetStats() map[string]interface{}
}

// PriceChange изменение цены
type PriceChange struct {
	Symbol        string    `json:"symbol"`
	CurrentPrice  float64   `json:"current_price"`
	PreviousPrice float64   `json:"previous_price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Interval      string    `json:"interval"`
	Timestamp     time.Time `json:"timestamp"`
	VolumeUSD     float64   `json:"volume_usd,omitempty"` // ← ДОБАВЛЕНО для анализа
}

// TradeData представляет данные о сделке
type TradeData struct {
	Symbol string    `json:"symbol"`
	Side   string    `json:"side"` // "Buy" или "Sell"
	Price  float64   `json:"price"`
	Size   float64   `json:"size"`
	Time   time.Time `json:"time"`
}

// VolumeDelta представляет дельту объемов
type VolumeDelta struct {
	Symbol       string    `json:"symbol"`
	Period       string    `json:"period"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	BuyVolume    float64   `json:"buy_volume"`
	SellVolume   float64   `json:"sell_volume"`
	Delta        float64   `json:"delta"`         // buyVolume - sellVolume
	DeltaPercent float64   `json:"delta_percent"` // Процентное изменение
	TotalTrades  int       `json:"total_trades"`
	UpdateTime   time.Time `json:"update_time"`
}
