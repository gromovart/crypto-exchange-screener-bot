// internal/delivery/telegram/app/bot/formatters/provider.go
package formatters

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/formatters/recommendation"
	"crypto-exchange-screener-bot/pkg/logger"
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
	SRZonesFormatter     *SRZonesFormatter
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
		SRZonesFormatter:     NewSRZonesFormatter(),
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
	RSIStatus          string
	MACDSignal         float64
	MACDStatus         string
	MACDDescription    string
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

	// Зоны поддержки/сопротивления
	SRSupport    *SRZoneData
	SRResistance *SRZoneData
}

// FormatCounterSignal форматирует counter сигнал для отправки в Telegram
func (p *FormatterProvider) FormatCounterSignal(data CounterData) string {
	// В начале метода FormatCounterSignal добавить:
	logger.Warn("📝 Форматирование сигнала %s: подтверждений %d/%d, слотов %d/%d",
		data.Symbol, data.Confirmations, data.RequiredConfirmations,
		data.FilledSlots, data.TotalSlots)

	var builder strings.Builder

	// 1. ЗАГОЛОВОК
	// 🔴 ПАДЕНИЕ -60.00% 🚨
	builder.WriteString(p.SignalFormatter.FormatSignalHeader(
		data.Direction,
		data.ChangePercent,
		data.CurrentPrice,
	))

	// 2. СИМВОЛ
	// 📛 DOLOUSDT
	builder.WriteString(fmt.Sprintf("📛 %s\n\n", data.Symbol))

	// 3. БИРЖА
	// 🏷️ BYBIT • 1ч
	timeframe := p.HeaderFormatter.ExtractTimeframe(data.Period)
	intensityEmoji := p.HeaderFormatter.GetIntensityEmoji(data.ChangePercent)
	builder.WriteString(fmt.Sprintf("🏷️  %s • %s\n",
		p.HeaderFormatter.GetExchange(), timeframe))
	if intensityEmoji != "" {
		builder.WriteString(intensityEmoji + " ")
	}

	// 4. ВРЕМЯ
	// 🕐 22:07:06
	builder.WriteString(fmt.Sprintf("🕐 %s\n\n",
		data.Timestamp.Format("15:04:05")))

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

		// ⭐ ИСПОЛЬЗУЕМ РЕАЛЬНЫЕ ДАННЫЕ С СТАТУСАМИ
		if data.RSI > 0 {
			if data.RSIStatus != "" {
				// Используем реальный статус из CounterAnalyzer
				builder.WriteString(p.TechnicalFormatter.FormatRSIWithStatus(data.RSI, data.RSIStatus))
			} else {
				// Fallback: статический расчет (для обратной совместимости)
				builder.WriteString(p.TechnicalFormatter.FormatRSI(data.RSI))
			}
			builder.WriteString("\n")
		}

		if data.MACDSignal != 0 {
			if data.MACDDescription != "" {
				// Используем реальное описание из CounterAnalyzer
				builder.WriteString(p.TechnicalFormatter.FormatMACDWithDescription(data.MACDDescription))
			} else if data.MACDStatus != "" {
				// Используем статус из CounterAnalyzer
				builder.WriteString(fmt.Sprintf("MACD: %s", data.MACDStatus))
			} else {
				// Fallback: статический расчет (для обратной совместимости)
				builder.WriteString(p.TechnicalFormatter.FormatMACD(data.MACDSignal))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	// 7. ЗОНЫ S/R (если есть данные)
	if srBlock := p.SRZonesFormatter.FormatSRZonesBlock(
		data.Period, data.SRSupport, data.SRResistance,
	); srBlock != "" {
		builder.WriteString(srBlock)
		builder.WriteString("\n")
	}

	// ⭐ ИЗМЕНЕНО: Только торговая рекомендация с уровнями (без дублирования анализа)
	tradingRecommendation := p.Recommendation.GetTradingRecommendationOnly(
		data.Direction,
		data.RSI,
		data.MACDSignal,
		data.VolumeDelta,
		data.VolumeDeltaPercent,
		data.LongLiqVolume,
		data.ShortLiqVolume,
		data.CurrentPrice,
		data.ChangePercent,
	)

	if tradingRecommendation != "" {
		builder.WriteString(tradingRecommendation)
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
	// 💥 Ликвидации за 5м: $12.5M
	// LONG: $7.8M, SHORT: $4.7M
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
