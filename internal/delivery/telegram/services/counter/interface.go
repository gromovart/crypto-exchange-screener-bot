// internal/delivery/telegram/services/counter/interface.go
package counter

import "time"

// Service интерфейс сервиса счетчика
type Service interface {
	// Exec выполняет обработку параметров счетчика
	Exec(params CounterParams) (CounterResult, error)
}

// CounterParams параметры для Exec
type CounterParams struct {
	// Базовые поля
	Symbol        string
	Direction     string
	ChangePercent float64
	Period        string
	Timestamp     time.Time

	// Подтверждения
	Confirmations int

	// Данные из indicators
	CurrentPrice       float64
	Volume24h          float64
	OpenInterest       float64
	FundingRate        float64
	RSI                float64
	MACDSignal         float64
	VolumeDelta        float64
	VolumeDeltaPercent float64

	// НОВЫЕ ПОЛЯ: Данные прогресса из сигнала
	ProgressFilledGroups int     `json:"progress_filled_groups,omitempty"`
	ProgressTotalGroups  int     `json:"progress_total_groups,omitempty"`
	ProgressPercentage   float64 `json:"progress_percentage,omitempty"`

	// Зоны S/R
	SRSupportPrice       float64
	SRSupportStrength    float64
	SRSupportDistPct     float64
	SRSupportHasWall     bool
	SRSupportWallUSD     float64
	SRResistancePrice    float64
	SRResistanceStrength float64
	SRResistanceDistPct  float64
	SRResistanceHasWall  bool
	SRResistanceWallUSD  float64
}

// CounterResult результат Exec
type CounterResult struct {
	Processed bool   `json:"processed"`
	Message   string `json:"message,omitempty"`
	SentTo    int    `json:"sent_to,omitempty"`
}

// RawCounterData сырые данные счетчика
type RawCounterData struct {
	Symbol             string    `json:"symbol"`
	Direction          string    `json:"direction"`
	ChangePercent      float64   `json:"change"`
	SignalCount        int       `json:"signal_count"`
	MaxSignals         int       `json:"max_signals"`
	Period             string    `json:"period"` // "5m", "15m", "30m", "1h", "4h", "1d"
	CurrentPrice       float64   `json:"current_price"`
	Volume24h          float64   `json:"volume_24h"`
	OpenInterest       float64   `json:"open_interest"`
	OIChange24h        float64   `json:"oi_change_24h"`
	FundingRate        float64   `json:"funding_rate"`
	NextFundingTime    time.Time `json:"next_funding_time"`
	LiquidationVolume  float64   `json:"liquidation_volume"`
	LongLiqVolume      float64   `json:"long_liq_volume"`
	ShortLiqVolume     float64   `json:"short_liq_volume"`
	VolumeDelta        float64   `json:"volume_delta"`
	VolumeDeltaPercent float64   `json:"volume_delta_percent"`
	RSI                float64   `json:"rsi"`
	MACDSignal         float64   `json:"macd_signal"`
	DeltaSource        string    `json:"delta_source"`
	Confidence         float64   `json:"confidence"`
	Timestamp          time.Time `json:"timestamp"`

	Confirmations         int `json:"confirmations"`          // текущие подтверждения
	RequiredConfirmations int `json:"required_confirmations"` // нужно подтверждений
	TotalSlots            int `json:"total_slots"`            // всего слотов (групп)
	FilledSlots           int `json:"filled_slots"`           // заполненные слоты

	NextAnalysis       time.Time `json:"next_analysis"`       // следующий анализ
	NextSignal         time.Time `json:"next_signal"`         // следующий сигнал
	ProgressPercentage float64   `json:"progress_percentage"` // процент прогресса (вычисляемое)

	// Зоны S/R
	SRSupportPrice       float64
	SRSupportStrength    float64
	SRSupportDistPct     float64
	SRSupportHasWall     bool
	SRSupportWallUSD     float64
	SRResistancePrice    float64
	SRResistanceStrength float64
	SRResistanceDistPct  float64
	SRResistanceHasWall  bool
	SRResistanceWallUSD  float64
}
