// internal/delivery/telegram/app/bot/formatters/sr_zones.go
package formatters

import (
	"fmt"
	"strings"
)

// SRZoneData — данные зоны S/R для форматирования
type SRZoneData struct {
	Price      float64
	Strength   float64
	DistPct    float64
	HasWall    bool
	WallSizeUSD float64
}

// SRZonesFormatter форматирует блок зон поддержки/сопротивления
type SRZonesFormatter struct {
	nf *NumberFormatter
}

// NewSRZonesFormatter создаёт форматтер
func NewSRZonesFormatter() *SRZonesFormatter {
	return &SRZonesFormatter{nf: NewNumberFormatter()}
}

// FormatSRZonesBlock форматирует блок S/R зон для Telegram-сообщения.
// period — строка вида "1h", "15m" и т.д.
// support и resistance могут быть nil.
func (f *SRZonesFormatter) FormatSRZonesBlock(period string, support, resistance *SRZoneData) string {
	if support == nil && resistance == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📐 Зоны S/R (%s):\n", period))

	if support != nil {
		sb.WriteString(f.formatLine("🟢", "Поддержка", support, true))
		sb.WriteString("\n")
	}
	if resistance != nil {
		sb.WriteString(f.formatLine("🔴", "Сопротивление", resistance, false))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatLine форматирует одну строку зоны.
func (f *SRZonesFormatter) formatLine(emoji, label string, z *SRZoneData, isSupport bool) string {
	dirArrow := "↑"
	if isSupport {
		dirArrow = "↓"
	}

	line := fmt.Sprintf("%s %s: $%s %s%.1f%% | сила: %.0f%%",
		emoji,
		label,
		f.nf.FormatPrice(z.Price),
		dirArrow,
		z.DistPct,
		z.Strength,
	)

	if z.HasWall && z.WallSizeUSD > 0 {
		line += fmt.Sprintf(" 🧱 $%s", f.nf.FormatDollarValue(z.WallSizeUSD))
	}

	return line
}
