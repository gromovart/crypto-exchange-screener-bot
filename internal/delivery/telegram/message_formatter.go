// internal/delivery/telegram/message_formatter.go
package telegram

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/formatters"
	"crypto-exchange-screener-bot/internal/delivery/telegram/formatters/recommendation"
	"crypto-exchange-screener-bot/internal/infrastructure/api/exchanges/bybit"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

// MarketMessageFormatter форматирует сообщения с рыночными данными
type MarketMessageFormatter struct {
	exchange                string
	headerFormatter         *formatters.HeaderFormatter
	signalFormatter         *formatters.SignalFormatter
	metricsFormatter        *formatters.MetricsFormatter
	technicalFormatter      *formatters.TechnicalFormatter
	liquidationFormatter    *formatters.LiquidationFormatter
	progressFormatter       *formatters.ProgressFormatter
	fundingFormatter        *formatters.FundingFormatter
	numberFormatter         *formatters.NumberFormatter
	recommendationFormatter *recommendation.RecommendationFormatter
}

// NewMarketMessageFormatter создает новый форматтер
func NewMarketMessageFormatter(exchange string) *MarketMessageFormatter {
	return &MarketMessageFormatter{
		exchange:                strings.ToUpper(exchange),
		headerFormatter:         formatters.NewHeaderFormatter(exchange),
		signalFormatter:         formatters.NewSignalFormatter(),
		metricsFormatter:        formatters.NewMetricsFormatter(),
		technicalFormatter:      formatters.NewTechnicalFormatter(),
		liquidationFormatter:    formatters.NewLiquidationFormatter(),
		progressFormatter:       formatters.NewProgressFormatter(),
		fundingFormatter:        formatters.NewFundingFormatter(),
		numberFormatter:         formatters.NewNumberFormatter(),
		recommendationFormatter: recommendation.NewRecommendationFormatter(),
	}
}

// FormatMessage создает сообщение в чистом формате без рамки
func (f *MarketMessageFormatter) FormatMessage(params *MessageParams) string {
	var builder strings.Builder

	// Заголовок
	builder.WriteString(f.formatHeader(params.Symbol, params.Period))

	// Сигнал и цена
	builder.WriteString(f.formatSignal(params.Direction, params.Change, params.CurrentPrice))

	// Метрики
	builder.WriteString(f.formatMetrics(params))

	// Технический анализ
	if params.RSI > 0 || params.MACDSignal != 0 {
		builder.WriteString("📊 Тех. анализ:\n")
		if params.RSI > 0 {
			builder.WriteString(f.technicalFormatter.FormatRSI(params.RSI) + "\n")
		}
		if params.MACDSignal != 0 {
			builder.WriteString(f.technicalFormatter.FormatMACD(params.MACDSignal) + "\n")
		}
		builder.WriteString("\n")
	}

	// Ликвидации
	if liqBlock := f.liquidationFormatter.FormatLiquidationBlock(
		params.Period, params.LiquidationVolume, params.LongLiqVolume,
		params.ShortLiqVolume, params.Volume24h,
	); liqBlock != "" {
		builder.WriteString(liqBlock)
	}

	// Прогресс
	builder.WriteString(f.progressFormatter.FormatProgressBlock(
		params.SignalCount, params.MaxSignals, params.Period,
	))

	// Рекомендации
	if rec := f.recommendationFormatter.GetEnhancedTradingRecommendation(
		params.Direction, params.RSI, params.MACDSignal,
		params.VolumeDelta, params.VolumeDeltaPercent,
		params.LongLiqVolume, params.ShortLiqVolume,
	); rec != "" {
		builder.WriteString(fmt.Sprintf("🎯 РЕКОМЕНДАЦИЯ:\n%s\n\n", rec))
	}

	// Фандинг
	builder.WriteString(f.fundingFormatter.FormatFundingBlock(
		params.FundingRate, params.NextFundingTime,
	))

	return builder.String()
}

// MessageParams параметры сообщения
type MessageParams struct {
	Symbol             string
	Direction          string
	Change             float64
	SignalCount        int
	MaxSignals         int
	CurrentPrice       float64
	Volume24h          float64
	OpenInterest       float64
	OIChange24h        float64
	FundingRate        float64
	AverageFunding     float64
	NextFundingTime    time.Time
	Period             string
	LiquidationVolume  float64
	LongLiqVolume      float64
	ShortLiqVolume     float64
	VolumeDelta        float64
	VolumeDeltaPercent float64
	RSI                float64
	MACDSignal         float64
	DeltaSource        string
}

// formatHeader форматирует заголовок
func (f *MarketMessageFormatter) formatHeader(symbol, period string) string {
	timeframe := f.headerFormatter.ExtractTimeframe(period)
	contractType := f.headerFormatter.GetContractType(symbol)

	return fmt.Sprintf("🏷️  %s • %s\n📛 %s\n📄 %s\n🕐 %s\n\n",
		f.exchange, timeframe, symbol, contractType,
		time.Now().Format("15:04:05"),
	)
}

// formatSignal форматирует сигнал
func (f *MarketMessageFormatter) formatSignal(direction string, change, price float64) string {
	icon, text, prefix := f.signalFormatter.GetDirectionInfo(direction)
	intensity := f.headerFormatter.GetIntensityEmoji(math.Abs(change))

	return fmt.Sprintf("%s %s %s%.2f%% %s\n💰 $%s\n\n",
		icon, text, prefix, math.Abs(change), intensity,
		f.numberFormatter.FormatPrice(price),
	)
}

// formatMetrics форматирует метрики
func (f *MarketMessageFormatter) formatMetrics(params *MessageParams) string {
	var builder strings.Builder

	// OI
	builder.WriteString(fmt.Sprintf("📈 OI: %s\n",
		f.metricsFormatter.FormatOIWithChange(params.OpenInterest, params.OIChange24h),
	))

	// Объем
	builder.WriteString(fmt.Sprintf("📊 Объем 24ч: $%s\n",
		f.numberFormatter.FormatDollarValue(params.Volume24h),
	))

	// Дельта
	if params.VolumeDelta != 0 || params.VolumeDeltaPercent != 0 {
		deltaStr := f.metricsFormatter.FormatVolumeDelta(
			params.VolumeDelta, params.VolumeDeltaPercent, params.Direction,
		)
		if params.DeltaSource != "" {
			deltaStr += formatters.GetSourceIndicator(params.DeltaSource)
		}
		builder.WriteString(fmt.Sprintf("📈 Дельта: %s\n\n", deltaStr))
	} else {
		builder.WriteString("\n")
	}

	return builder.String()
}

// FormatMessageWithFullDelta создает сообщение с полной дельтой
func (f *MarketMessageFormatter) FormatMessageWithFullDelta(
	params *MessageParams,
	volumeDelta *bybit.VolumeDelta,
) string {
	// Адаптируем параметры для полной дельты
	fullParams := *params

	if volumeDelta != nil {
		fullParams.VolumeDelta = volumeDelta.Delta
		fullParams.VolumeDeltaPercent = volumeDelta.DeltaPercent
		fullParams.DeltaSource = "api"

		log.Printf("📊 Реальные данные дельты для %s", params.Symbol)
	}

	// Используем базовый форматтер
	return f.FormatMessage(&fullParams)
}
