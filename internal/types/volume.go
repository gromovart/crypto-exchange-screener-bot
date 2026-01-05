// internal/types/volume.go
package types

import "time"

// ==================== ТИПЫ ДЛЯ ДЕЛЬТЫ ОБЪЕМОВ ====================

// VolumeDeltaSource источник данных дельты
type VolumeDeltaSource string

const (
	VolumeDeltaSourceAPI      VolumeDeltaSource = "api"      // API Bybit
	VolumeDeltaSourceStorage  VolumeDeltaSource = "storage"  // Хранилище
	VolumeDeltaSourceEmulated VolumeDeltaSource = "emulated" // Эмуляция
	VolumeDeltaSourceCache    VolumeDeltaSource = "cache"    // Кэш
)

// VolumeDeltaData данные дельты с источником
type VolumeDeltaData struct {
	Delta        float64
	DeltaPercent float64
	Source       VolumeDeltaSource
	Timestamp    time.Time
	BuyVolume    float64 // Покупки
	SellVolume   float64 // Продажи
	TotalTrades  int     // Всего сделок
	IsRealData   bool    // Реальные данные (true) или эмулированные (false)
}

// VolumeDeltaDataForFormatter данные дельты для форматтера (без циклического импорта)
type VolumeDeltaDataForFormatter struct {
	Delta        float64
	DeltaPercent float64
	Source       string
	Timestamp    time.Time
	BuyVolume    float64
	SellVolume   float64
	TotalTrades  int
	IsRealData   bool
}

// ToFormatterData конвертирует VolumeDeltaData в VolumeDeltaDataForFormatter
func (v *VolumeDeltaData) ToFormatterData() *VolumeDeltaDataForFormatter {
	if v == nil {
		return &VolumeDeltaDataForFormatter{}
	}

	return &VolumeDeltaDataForFormatter{
		Delta:        v.Delta,
		DeltaPercent: v.DeltaPercent,
		Source:       string(v.Source),
		Timestamp:    v.Timestamp,
		BuyVolume:    v.BuyVolume,
		SellVolume:   v.SellVolume,
		TotalTrades:  v.TotalTrades,
		IsRealData:   v.IsRealData,
	}
}

func (s VolumeDeltaSource) String() string {
	return string(s)
}
