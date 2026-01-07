// internal/delivery/telegram/formatters/recommendation/recommendation.go
package recommendation

import (
	"strings"
	"time"
)

// RecommendationFormatter отвечает за форматирование рекомендаций
type RecommendationFormatter struct {
	analyzer  *Analyzer
	scorer    *Scorer
	formatter *Formatter
}

// NewRecommendationFormatter создает новый форматтер рекомендаций
func NewRecommendationFormatter() *RecommendationFormatter {
	return &RecommendationFormatter{
		analyzer:  NewAnalyzer(),
		scorer:    NewScorer(),
		formatter: NewFormatter(),
	}
}

// GetEnhancedTradingRecommendation возвращает улучшенные рекомендации по торговле
func (f *RecommendationFormatter) GetEnhancedTradingRecommendation(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta float64,
	volumeDeltaPercent float64,
	longLiqVolume float64,
	shortLiqVolume float64,
) string {
	// Анализируем данные
	analysis := f.analyzer.AnalyzeData(
		direction, rsi, macdSignal,
		volumeDelta, volumeDeltaPercent,
		longLiqVolume, shortLiqVolume,
	)

	// Если рекомендаций нет
	if len(analysis.Recommendations) == 0 {
		return ""
	}

	// Подсчитываем баллы
	scores := f.scorer.CalculateSignalScores(analysis.Recommendations)

	// Определяем основной сигнал
	primarySignal := f.scorer.DeterminePrimarySignal(scores, analysis.Recommendations)

	// Форматируем результат
	return f.formatter.FormatResult(primarySignal, analysis.Recommendations, analysis.Strength)
}

// GetEnhancedTradingRecommendationWithFullDelta улучшенные рекомендации с полными данными дельты
func (f *RecommendationFormatter) GetEnhancedTradingRecommendationWithFullDelta(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta *VolumeDeltaData,
	isRealData bool,
	longLiqVolume, shortLiqVolume float64,
) string {
	// Конвертируем данные дельты
	var delta, deltaPercent float64
	if volumeDelta != nil {
		delta = volumeDelta.Delta
		deltaPercent = volumeDelta.DeltaPercent
	}

	// Получаем базовые рекомендации
	baseResult := f.GetEnhancedTradingRecommendation(
		direction, rsi, macdSignal,
		delta, deltaPercent,
		longLiqVolume, shortLiqVolume,
	)

	// Если нет базовых рекомендаций
	if baseResult == "" {
		return ""
	}

	// Добавляем информацию о качестве данных
	var result strings.Builder
	if isRealData {
		result.WriteString("📊 Анализ на основе реальных данных:\n")
	} else {
		result.WriteString("📊 Анализ на основе эмулированных данных:\n")
	}

	// Добавляем базовые рекомендации (без первой строки - заголовка)
	lines := strings.Split(baseResult, "\n")
	if len(lines) > 1 {
		for _, line := range lines[1:] {
			if strings.TrimSpace(line) != "" {
				result.WriteString(line + "\n")
			}
		}
	}

	return strings.TrimSpace(result.String())
}

// VolumeDeltaData данные дельты для рекомендаций
type VolumeDeltaData struct {
	Delta        float64
	DeltaPercent float64
	BuyVolume    float64
	SellVolume   float64
	TotalTrades  int
	Timestamp    time.Time
	IsRealData   bool
}
