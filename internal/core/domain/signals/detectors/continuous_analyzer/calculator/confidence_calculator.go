// internal/core/domain/signals/detectors/continuous_analyzer/calculator/confidence_calculator.go
package calculator

import (
	"crypto-exchange-screener-bot/internal/core/domain/signals/detectors/common"
	"math"
)

// ConfidenceCalculator калькулятор уверенности для анализа непрерывности
type ConfidenceCalculator struct {
	config common.AnalyzerConfig
}

// NewConfidenceCalculator создает новый калькулятор уверенности
func NewConfidenceCalculator(config common.AnalyzerConfig) *ConfidenceCalculator {
	return &ConfidenceCalculator{
		config: config,
	}
}

// Calculate вычисляет уверенность сигнала
func (c *ConfidenceCalculator) Calculate(points int, change float64) float64 {
	absoluteChange := math.Abs(change)

	// Базовая уверенность на основе количества точек (максимум 60%)
	baseConfidence := math.Min(float64(points)*20.0, 60.0)

	// Дополнительная уверенность на основе величины изменения (максимум 40%)
	changeConfidence := math.Min(absoluteChange*2.0, 40.0)

	totalConfidence := baseConfidence + changeConfidence

	// Ограничиваем 100%
	return math.Min(totalConfidence, 100.0)
}

// CalculateSequenceQuality рассчитывает качество последовательности
func (c *ConfidenceCalculator) CalculateSequenceQuality(
	length int,
	avgChange float64,
	avgGap float64,
	consistency float64,
) float64 {
	// Веса для различных метрик
	const (
		lengthWeight      = 0.3
		changeWeight      = 0.4
		gapWeight         = 0.2
		consistencyWeight = 0.1
	)

	// Нормализуем метрики
	lengthScore := math.Min(float64(length)/10.0, 1.0)    // максимум 10 точек = 1.0
	changeScore := math.Min(math.Abs(avgChange)/5.0, 1.0) // 5% изменение = 1.0
	gapScore := 1.0 - math.Min(avgGap/0.5, 1.0)           // меньший gap лучше
	consistencyScore := consistency / 100.0

	// Рассчитываем итоговый score
	totalScore := lengthScore*lengthWeight +
		changeScore*changeWeight +
		gapScore*gapWeight +
		consistencyScore*consistencyWeight

	return totalScore * 100.0 // Конвертируем в проценты
}

// CalculateContinuityScore рассчитывает балл непрерывности
func (c *ConfidenceCalculator) CalculateContinuityScore(
	directionChanges int,
	totalPoints int,
	avgGap float64,
) float64 {
	if totalPoints < 2 {
		return 0.0
	}

	// Рассчитываем консистентность направления
	directionConsistency := 1.0 - float64(directionChanges)/float64(totalPoints-1)

	// Рассчитываем стабильность gap
	gapStability := 1.0 - math.Min(avgGap*10.0, 1.0) // нормализуем gap

	// Комбинируем метрики
	continuityScore := (directionConsistency * 0.6) + (gapStability * 0.4)

	return continuityScore * 100.0
}

// UpdateConfig обновляет конфигурацию
func (c *ConfidenceCalculator) UpdateConfig(config common.AnalyzerConfig) {
	c.config = config
}

// GetName возвращает имя калькулятора
func (c *ConfidenceCalculator) GetName() string {
	return "confidence_calculator"
}
