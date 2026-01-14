// internal/delivery/telegram/app/bot/formatters/types.go
package formatters

// FormatterTypes содержит общие типы для форматтеров
type FormatterTypes struct{}

// BarConfig конфигурация для создания баров
type BarConfig struct {
	Percentage float64
	Emoji      string
	MaxBars    int
}

// ProgressBarConfig конфигурация для прогресс-баров
type ProgressBarConfig struct {
	Percentage  float64
	Count       int
	MaxCount    int
	LowColor    string // 🟢
	MediumColor string // 🟡
	HighColor   string // 🔴
}
