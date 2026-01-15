// internal/delivery/telegram/app/bot/formatters/provider.go
package formatters

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters/recommendation"
	"fmt"
	"strings"
	"time"
)

// FormatterProvider предоставляет доступ ко всем форматтерам
type FormatterProvider struct {
	HeaderFormatter      *HeaderFormatter
	SignalFormatter      *SignalFormatter
	MetricsFormatter     *MetricsFormatter
	TechnicalFormatter   *TechnicalFormatter
	ProgressFormatter    *ProgressFormatter
	FundingFormatter     *FundingFormatter
	LiquidationFormatter *LiquidationFormatter
	Recommendation       *recommendation.RecommendationFormatter
	NumberFormatter      *NumberFormatter
}

// NewFormatterProvider создает новый провайдер форматтеров
func NewFormatterProvider(exchange string) *FormatterProvider {
	return &FormatterProvider{
		HeaderFormatter:      NewHeaderFormatter(exchange),
		SignalFormatter:      NewSignalFormatter(),
		MetricsFormatter:     NewMetricsFormatter(),
		TechnicalFormatter:   NewTechnicalFormatter(),
		ProgressFormatter:    NewProgressFormatter(),
		FundingFormatter:     NewFundingFormatter(),
		LiquidationFormatter: NewLiquidationFormatter(),
		Recommendation:       recommendation.NewRecommendationFormatter(),
		NumberFormatter:      NewNumberFormatter(),
	}
}

// CounterData данные для форматирования counter сигнала
type CounterData struct {
	Symbol             string
	Direction          string
	ChangePercent      float64
	SignalCount        int
	MaxSignals         int
	Period             string
	CurrentPrice       float64
	Volume24h          float64
	OpenInterest       float64
	OIChange24h        float64
	FundingRate        float64
	NextFundingTime    time.Time
	LiquidationVolume  float64
	LongLiqVolume      float64
	ShortLiqVolume     float64
	VolumeDelta        float64
	VolumeDeltaPercent float64
	RSI                float64
	MACDSignal         float64
	DeltaSource        string
	Confidence         float64
	Timestamp          time.Time

	// НОВЫЕ ПОЛЯ для прогресса подтверждений
	Confirmations         int
	RequiredConfirmations int
	TotalSlots            int
	FilledSlots           int
	ProgressPercentage    float64
	NextAnalysis          time.Time
	NextSignal            time.Time
}

// FormatCounterSignal форматирует counter сигнал для отправки в Telegram
func (p *FormatterProvider) FormatCounterSignal(data CounterData) string {
	var builder strings.Builder

	// 1. ЗАГОЛОВОК
	// 🏷️ BYBIT • 1ч
	timeframe := p.HeaderFormatter.ExtractTimeframe(data.Period)
	intensityEmoji := p.HeaderFormatter.GetIntensityEmoji(data.ChangePercent)
	builder.WriteString(fmt.Sprintf("🏷️  %s • %s\n",
		p.HeaderFormatter.GetExchange(), timeframe))
	if intensityEmoji != "" {
		builder.WriteString(intensityEmoji + " ")
	}

	// 2. СИМВОЛ И ТИП КОНТРАКТА
	// 📛 DOLOUSDT
	// 📄 USDT-фьючерс
	contractType := p.HeaderFormatter.GetContractType(data.Symbol)
	builder.WriteString(fmt.Sprintf("📛 %s\n", data.Symbol))
	builder.WriteString(fmt.Sprintf("📄 %s\n", contractType))

	// 3. ВРЕМЯ
	// 🕐 22:07:06
	builder.WriteString(fmt.Sprintf("🕐 %s\n\n",
		data.Timestamp.Format("15:04:05")))

	// 4. СИГНАЛ И ЦЕНА
	// 🔴 ПАДЕНИЕ -60.00% 🚨
	// 💰 $0.07388
	builder.WriteString(p.SignalFormatter.FormatSignalBlock(
		data.Direction,
		data.ChangePercent,
		data.CurrentPrice,
	))

	// 5. РЫНОЧНЫЕ МЕТРИКИ
	// 📈 OI: $90.0M (🟢+7.0%)
	// 📊 Объем 24ч: $915M
	// 📈 Дельта: 🟠4.9K (🔴-33.4% ⚡) [API]
	builder.WriteString("📈 OI: ")
	builder.WriteString(p.MetricsFormatter.FormatOIWithChange(
		data.OpenInterest, data.OIChange24h))
	builder.WriteString("\n")

	builder.WriteString(fmt.Sprintf("📊 Объем 24ч: $%s\n",
		p.NumberFormatter.FormatDollarValue(data.Volume24h)))

	builder.WriteString("📈 Дельта: ")
	builder.WriteString(p.MetricsFormatter.FormatVolumeDelta(
		data.VolumeDelta, data.VolumeDeltaPercent, data.Direction))
	if data.DeltaSource != "" {
		builder.WriteString(GetSourceIndicator(data.DeltaSource))
	}
	builder.WriteString("\n\n")

	// 6. ТЕХНИЧЕСКИЙ АНАЛИЗ (если есть данные)
	// 📊 Тех. анализ:
	// RSI: 50.0 ⚪ (нейтральный)
	if data.RSI > 0 || data.MACDSignal != 0 {
		builder.WriteString("📊 Тех. анализ:\n")
		if data.RSI > 0 {
			builder.WriteString(p.TechnicalFormatter.FormatRSI(data.RSI))
			builder.WriteString("\n")
		}
		if data.MACDSignal != 0 {
			builder.WriteString(p.TechnicalFormatter.FormatMACD(data.MACDSignal))
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// 7. ПРОГРЕСС ПОДТВЕРЖДЕНИЙ (новый раздел)
	// 📡 Подтверждений: 3/6 🟢🟢🟢▫️▫️▫️ (50%)
	// 🕐 Следующий анализ: 10:10
	// ⏰ Следующий сигнал: 10:40 (через 20м)
	if data.RequiredConfirmations > 0 {
		builder.WriteString(p.ProgressFormatter.FormatConfirmationProgress(
			data.Confirmations,
			data.RequiredConfirmations,
			data.Period,
			data.NextAnalysis,
			data.NextSignal,
		))
		builder.WriteString("\n\n")
	} else {
		// Обратная совместимость со старым форматом
		builder.WriteString(p.ProgressFormatter.FormatProgressBlock(
			data.SignalCount,
			data.MaxSignals,
			data.Period,
		))
	}

	// 8. РЕКОМЕНДАЦИИ (если есть данные)
	// 🎯 РЕКОМЕНДАЦИЯ:
	// 📌 Направление: 🔴🔽 СИЛЬНЫЕ МЕДВЕЖЬИ СИГНАЛЫ
	//
	// 📊 Анализ сигналов:
	// 1.  ⚠️ RSI в зоне перепроданности (28.5) - осторожность с SHORT
	// 2.  📉 MACD: сильный медвежий тренд
	// 3.  📉 умеренная дельта продаж ($20762) - заметное давление продавцов
	// 4.  ✅ Объемы подтверждают ценовое движение
	//
	// 🟢 ОТКРЫТЬ ЛОНГ: умеренные бычьи сигналы
	//
	// 📊 УРОВНИ:
	// Стоп-лосс: $0.8560 (2.0%)
	// Тейк-профит: $0.8912 (4.0%)
	// Риск/Прибыль: 1:2.0
	//
	// 📈 РАЗМЕР ПОЗИЦИИ:
	// Рекомендуемый размер: 1-2% капитала
	//
	// 🎯 ЗАКЛЮЧЕНИЕ: умеренное движение с умеренной дельтой объемов
	recommendationText := p.Recommendation.GetEnhancedTradingRecommendation(
		data.Direction,
		data.RSI,
		data.MACDSignal,
		data.VolumeDelta,
		data.VolumeDeltaPercent,
		data.LongLiqVolume,
		data.ShortLiqVolume,
		data.CurrentPrice,  // НОВЫЙ ПАРАМЕТР
		data.ChangePercent, // НОВЫЙ ПАРАМЕТР
	)
	if recommendationText != "" {
		builder.WriteString(recommendationText)
		builder.WriteString("\n\n")
	}

	// 9. ФАНДИНГ (если есть данные)
	// 🎯 Фандинг: 🔴 -3.3459%
	// ⏰ Через: 59м
	if data.FundingRate != 0 && !data.NextFundingTime.IsZero() {
		builder.WriteString(p.FundingFormatter.FormatFundingBlock(
			data.FundingRate,
			data.NextFundingTime,
		))
		builder.WriteString("\n\n")
	}

	// 10. ЛИКВИДАЦИИ (если есть данные)
	if data.LiquidationVolume > 0 {
		builder.WriteString(p.LiquidationFormatter.FormatLiquidationBlock(
			data.Period,
			data.LiquidationVolume,
			data.LongLiqVolume,
			data.ShortLiqVolume,
			data.Volume24h,
		))
	}

	return strings.TrimSpace(builder.String())
}

// FormatCompactCounterSignal форматирует компактный counter сигнал
func (p *FormatterProvider) FormatCompactCounterSignal(data CounterData) string {
	icon, directionText, _ := p.SignalFormatter.GetDirectionInfo(data.Direction)
	return fmt.Sprintf("%s %s %s: %.2f%% (сигналов: %d/%d, дельта: $%s)",
		icon,
		directionText,
		data.Symbol,
		data.ChangePercent,
		data.SignalCount,
		data.MaxSignals,
		p.NumberFormatter.FormatDollarValue(data.VolumeDelta),
	)
}
