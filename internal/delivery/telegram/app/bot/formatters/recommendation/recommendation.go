// internal/delivery/telegram/app/bot/formatters/recommendation/recommendation.go
package recommendation

import (
	"fmt"
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
	currentPrice float64, // ДОБАВЛЕНО: текущая цена для уровней
	changePercent float64, // ДОБАВЛЕНО: изменение цены для анализа
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

	// Получаем торговую рекомендацию с уровнями
	tradingRecommendation := f.scorer.GetEntryRecommendation(
		analysis.Recommendations,
		rsi,
		changePercent,
		volumeDelta,
		currentPrice,
	)

	// Форматируем базовый результат
	formattedResult := f.formatter.FormatResult(primarySignal, analysis.Recommendations, analysis.Strength)

	// Объединяем с торговой рекомендацией
	var result strings.Builder
	result.WriteString(formattedResult)
	result.WriteString("\n\n")
	result.WriteString(tradingRecommendation)

	return strings.TrimSpace(result.String())
}

// GetEnhancedTradingRecommendationWithFullDelta улучшенные рекомендации с полными данными дельты
func (f *RecommendationFormatter) GetEnhancedTradingRecommendationWithFullDelta(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta *VolumeDeltaData,
	isRealData bool,
	longLiqVolume, shortLiqVolume float64,
	currentPrice float64,
	changePercent float64,
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
		currentPrice,
		changePercent,
	)

	// Если нет базовых рекомендаций
	if baseResult == "" {
		return ""
	}

	// Добавляем информацию о качестве данных
	var result strings.Builder

	// Добавляем заголовок качества данных
	if isRealData {
		result.WriteString("📊 Анализ на основе реальных данных:\n\n")
	} else {
		result.WriteString("📊 Анализ на основе эмулированных данных:\n\n")
	}

	// Добавляем базовые рекомендации
	result.WriteString(baseResult)

	return strings.TrimSpace(result.String())
}

// GetTradingRecommendationOnly возвращает только торговую рекомендацию без анализа
func (f *RecommendationFormatter) GetTradingRecommendationOnly(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta float64,
	volumeDeltaPercent float64,
	longLiqVolume, shortLiqVolume float64,
	currentPrice float64,
	changePercent float64,
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

	// Получаем торговую рекомендацию с уровнями
	tradingRecommendation := f.scorer.GetEntryRecommendation(
		analysis.Recommendations,
		rsi,
		changePercent,
		volumeDelta,
		currentPrice,
	)

	return tradingRecommendation
}

// GetCompactRecommendation возвращает компактную рекомендацию для сигналов
func (f *RecommendationFormatter) GetCompactRecommendation(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta float64,
	currentPrice float64,
	changePercent float64,
) string {
	// Упрощенный анализ
	analysis := f.analyzer.AnalyzeData(
		direction, rsi, macdSignal,
		volumeDelta, 0, // volumeDeltaPercent = 0 для упрощения
		0, 0, // liquidations = 0
	)

	if len(analysis.Recommendations) == 0 {
		return "⚪ Нет четких сигналов"
	}

	scores := f.scorer.CalculateSignalScores(analysis.Recommendations)
	action := f.scorer.GetTradingAction(scores, analysis.Recommendations, rsi, changePercent, volumeDelta)

	// Компактный формат
	var result strings.Builder
	result.WriteString("🎯 " + action)

	// Добавляем уровни если есть торговая рекомендация
	if strings.Contains(action, "ЛОНГ") || strings.Contains(action, "ШОРТ") {
		stopLossPercent := 2.0
		if strings.Contains(action, "ЛОНГ") {
			stopPrice := currentPrice * (1 - stopLossPercent/100)
			result.WriteString(fmt.Sprintf("\nSL: $%.4f (2%%)", stopPrice))
		} else if strings.Contains(action, "ШОРТ") {
			stopPrice := currentPrice * (1 + stopLossPercent/100)
			result.WriteString(fmt.Sprintf("\nSL: $%.4f (2%%)", stopPrice))
		}
	}

	return result.String()
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

// TradingRecommendation торговые рекомендации с уровнями
type TradingRecommendation struct {
	Action            string  // "LONG", "SHORT", "WAIT", "AVOID_LONG", "AVOID_SHORT"
	StopLoss          float64 // Уровень стоп-лосса
	TakeProfit        float64 // Уровень тейк-профита
	StopLossPercent   float64 // Процент стоп-лосса
	TakeProfitPercent float64 // Процент тейк-профита
	PositionSize      string  // Рекомендуемый размер позиции
	Confidence        string  // Уверенность: "HIGH", "MEDIUM", "LOW"
	Reason            string  // Причина рекомендации
}

// GetStructuredTradingRecommendation возвращает структурированные торговые рекомендации
func (f *RecommendationFormatter) GetStructuredTradingRecommendation(
	direction string,
	rsi float64,
	macdSignal float64,
	volumeDelta float64,
	volumeDeltaPercent float64,
	longLiqVolume, shortLiqVolume float64,
	currentPrice float64,
	changePercent float64,
) TradingRecommendation {
	// Анализируем данные
	analysis := f.analyzer.AnalyzeData(
		direction, rsi, macdSignal,
		volumeDelta, volumeDeltaPercent,
		longLiqVolume, shortLiqVolume,
	)

	scores := f.scorer.CalculateSignalScores(analysis.Recommendations)
	actionText := f.scorer.GetTradingAction(scores, analysis.Recommendations, rsi, changePercent, volumeDelta)

	// Создаем структурированную рекомендацию
	rec := TradingRecommendation{
		StopLossPercent:   2.0,
		TakeProfitPercent: 4.0,
	}

	// Определяем действие
	switch {
	case strings.Contains(actionText, "ОТКРЫТЬ ЛОНГ"):
		rec.Action = "LONG"
		rec.StopLoss = currentPrice * (1 - rec.StopLossPercent/100)
		rec.TakeProfit = currentPrice * (1 + rec.TakeProfitPercent/100)

	case strings.Contains(actionText, "ОТКРЫТЬ ШОРТ"):
		rec.Action = "SHORT"
		rec.StopLoss = currentPrice * (1 + rec.StopLossPercent/100)
		rec.TakeProfit = currentPrice * (1 - rec.TakeProfitPercent/100)

	case strings.Contains(actionText, "НЕ ОТКРЫВАТЬ LONG"):
		rec.Action = "AVOID_LONG"

	case strings.Contains(actionText, "НЕ ОТКРЫВАТЬ SHORT"):
		rec.Action = "AVOID_SHORT"

	default:
		rec.Action = "WAIT"
	}

	// Определяем уверенность
	totalConfidence := scores.BullishScore + scores.BearishScore
	if strings.Contains(actionText, "сильные") || totalConfidence >= 6 {
		rec.Confidence = "HIGH"
		rec.PositionSize = "2-3% капитала"
	} else if strings.Contains(actionText, "умеренные") || totalConfidence >= 4 {
		rec.Confidence = "MEDIUM"
		rec.PositionSize = "1-2% капитала"
	} else {
		rec.Confidence = "LOW"
		rec.PositionSize = "0.5-1% капитала"
	}

	// Упрощаем причину
	if strings.Contains(actionText, "RSI") {
		rec.Reason = "Сигнал RSI"
	} else if strings.Contains(actionText, "MACD") {
		rec.Reason = "Сигнал MACD"
	} else if strings.Contains(actionText, "дельта") {
		rec.Reason = "Сигнал объемов"
	} else {
		rec.Reason = "Смешанные сигналы"
	}

	return rec
}
