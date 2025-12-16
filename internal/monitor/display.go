package monitor

import (
	"crypto-exchange-screener-bot/internal/types"
	"fmt"
	"sort"
	"strings"
	"time"
)

// DisplayManager управляет выводом информации
type DisplayManager struct {
	signalBuffer  []types.GrowthSignal
	compactMode   bool
	minChange     float64 // Минимальное изменение для отображения
	minConfidence float64 // Минимальная уверенность для отображения
	maxSignals    int     // Максимальное количество сигналов за вывод
}

// NewDisplayManager создает новый менеджер отображения
func NewDisplayManager(compact bool, minChange, minConfidence float64, maxSignals int) *DisplayManager {
	return &DisplayManager{
		signalBuffer:  make([]types.GrowthSignal, 0),
		compactMode:   compact,
		minChange:     minChange,
		minConfidence: minConfidence,
		maxSignals:    maxSignals,
	}
}

// AddSignal добавляет сигнал в буфер
func (dm *DisplayManager) AddSignal(signal types.GrowthSignal) {
	// Фильтруем по минимальному изменению
	changePercent := signal.GrowthPercent + signal.FallPercent
	if abs(changePercent) < dm.minChange {
		return
	}

	// Фильтруем по минимальной уверенности
	if signal.Confidence < dm.minConfidence {
		return
	}

	dm.signalBuffer = append(dm.signalBuffer, signal)
}

// Flush выводит все сигналы из буфера
func (dm *DisplayManager) Flush() {
	if len(dm.signalBuffer) == 0 {
		return
	}

	// Сортируем по абсолютному значению изменения (по убыванию)
	sort.Slice(dm.signalBuffer, func(i, j int) bool {
		changeI := abs(dm.signalBuffer[i].GrowthPercent + dm.signalBuffer[i].FallPercent)
		changeJ := abs(dm.signalBuffer[j].GrowthPercent + dm.signalBuffer[j].FallPercent)
		return changeI > changeJ
	})

	// Ограничиваем количество
	originalCount := len(dm.signalBuffer)
	displayCount := originalCount
	if dm.maxSignals > 0 && displayCount > dm.maxSignals {
		displayCount = dm.maxSignals
	}

	growthCount := 0
	fallCount := 0
	maxChange := 0.0
	var topSymbol string
	var topChange float64
	var topDirection string

	// Выводим заголовок только если есть что показывать
	if displayCount > 0 {
		fmt.Println(strings.Repeat("─", 80))
		if originalCount > displayCount {
			fmt.Printf("📊 СИГНАЛЫ (топ-%d из %d) %s\n",
				displayCount,
				originalCount,
				time.Now().Format("15:04:05"))
		} else {
			fmt.Printf("📊 СИГНАЛЫ (%d) %s\n",
				originalCount,
				time.Now().Format("15:04:05"))
		}
		fmt.Println(strings.Repeat("─", 80))

		for i := 0; i < displayCount && i < len(dm.signalBuffer); i++ {
			signal := dm.signalBuffer[i]

			if signal.Direction == "growth" {
				growthCount++
			} else {
				fallCount++
			}

			var icon string
			changePercent := signal.GrowthPercent + signal.FallPercent

			// Находим максимальное изменение
			if abs(changePercent) > abs(maxChange) {
				maxChange = changePercent
				topSymbol = signal.Symbol
				topChange = changePercent
				topDirection = signal.Direction
			}

			if signal.Direction == "growth" {
				icon = "🟢"
				fmt.Printf("%s %-12s ↑%6.2f%% %3dмин %.0f%%\n",
					icon, signal.Symbol, changePercent, signal.PeriodMinutes, signal.Confidence)
			} else {
				icon = "🔴"
				fmt.Printf("%s %-12s ↓%6.2f%% %3dмин %.0f%%\n",
					icon, signal.Symbol, -changePercent, signal.PeriodMinutes, signal.Confidence)
			}
		}

		fmt.Println(strings.Repeat("─", 80))

		// Статистика
		if topSymbol != "" {
			directionIcon := "🟢"
			if topDirection == "fall" {
				directionIcon = "🔴"
			}
			fmt.Printf("%s ТОП: %s %s%.2f%%\n",
				directionIcon, topSymbol,
				map[string]string{"growth": "↑", "fall": "↓"}[topDirection],
				abs(topChange))
		}

		percentGrowth := 0
		if displayCount > 0 {
			percentGrowth = (growthCount * 100) / displayCount
		}
		fmt.Printf("📈 Рост: %d (%d%%) | 📉 Падение: %d (%d%%)\n",
			growthCount, percentGrowth,
			fallCount, 100-percentGrowth)
		fmt.Println()
	}

	// Очищаем буфер
	dm.signalBuffer = make([]types.GrowthSignal, 0)
}

// Вспомогательная функция для модуля числа
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
