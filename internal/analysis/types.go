// internal/analysis/analysis_types.go
package analysis

import (
	"time"
)

// Signal - структура сигнала анализа
type Signal struct {
	ID            string    `json:"id"`
	Symbol        string    `json:"symbol"`
	Type          string    `json:"type"`           // "growth", "fall", "breakout", "volume_spike"
	Direction     string    `json:"direction"`      // "up", "down"
	ChangePercent float64   `json:"change_percent"` // процент изменения
	Period        int       `json:"period"`         // период в минутах
	Confidence    float64   `json:"confidence"`     // уверенность 0-100
	DataPoints    int       `json:"data_points"`    // количество точек данных
	StartPrice    float64   `json:"start_price"`
	EndPrice      float64   `json:"end_price"`
	Volume        float64   `json:"volume"`
	Timestamp     time.Time `json:"timestamp"`
	Metadata      Metadata  `json:"metadata"`
}

// Metadata - метаданные сигнала
type Metadata struct {
	Strategy     string             `json:"strategy"`
	Tags         []string           `json:"tags"`
	Indicators   map[string]float64 `json:"indicators"`
	IsContinuous bool               `json:"is_continuous"`
	Patterns     []string           `json:"patterns"`
}

// AnalysisRequest - запрос на анализ
type AnalysisRequest struct {
	Symbol         string        `json:"symbol"`
	Period         time.Duration `json:"period"`
	Strategies     []string      `json:"strategies"`
	IncludeHistory bool          `json:"include_history"`
}

// AnalysisResult - результат анализа
type AnalysisResult struct {
	Symbol    string        `json:"symbol"`
	Signals   []Signal      `json:"signals"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}
