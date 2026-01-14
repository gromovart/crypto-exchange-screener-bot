// internal/core/domain/signals/types.go
package analysis

import (
	"encoding/json"
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
	Strategy       string                 `json:"strategy"`
	Tags           []string               `json:"tags"`
	Indicators     map[string]float64     `json:"indicators"`
	IsContinuous   bool                   `json:"is_continuous"`
	ContinuousFrom int                    `json:"continuous_from,omitempty"`
	ContinuousTo   int                    `json:"continuous_to,omitempty"`
	Patterns       []string               `json:"patterns"`
	Custom         map[string]interface{} `json:"custom,omitempty"` // НОВОЕ поле
}

// ToMap преобразует Signal в map[string]interface{}
func (s *Signal) ToMap() map[string]interface{} {
	data := map[string]interface{}{
		"id":             s.ID,
		"symbol":         s.Symbol,
		"type":           s.Type,
		"direction":      s.Direction,
		"change_percent": s.ChangePercent,
		"period":         s.Period,
		"confidence":     s.Confidence,
		"data_points":    s.DataPoints,
		"start_price":    s.StartPrice,
		"end_price":      s.EndPrice,
		"volume":         s.Volume,
		"timestamp":      s.Timestamp.Format(time.RFC3339),
	}

	// Добавляем метаданные
	if s.Metadata.Strategy != "" {
		data["strategy"] = s.Metadata.Strategy
	}
	if len(s.Metadata.Tags) > 0 {
		data["tags"] = s.Metadata.Tags
	}
	if len(s.Metadata.Indicators) > 0 {
		data["indicators"] = s.Metadata.Indicators
	}
	if len(s.Metadata.Patterns) > 0 {
		data["patterns"] = s.Metadata.Patterns
	}
	if s.Metadata.IsContinuous {
		data["is_continuous"] = s.Metadata.IsContinuous
		data["continuous_from"] = s.Metadata.ContinuousFrom
		data["continuous_to"] = s.Metadata.ContinuousTo
	}
	if len(s.Metadata.Custom) > 0 {
		// Добавляем все кастомные поля
		for key, value := range s.Metadata.Custom {
			data[key] = value
		}
	}

	return data
}

// FromMap создает Signal из map[string]interface{}
func (s *Signal) FromMap(data map[string]interface{}) error {
	// Конвертируем timestamp строку обратно в time.Time
	if timestampStr, ok := data["timestamp"].(string); ok {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			data["timestamp"] = timestamp
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, s)
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
