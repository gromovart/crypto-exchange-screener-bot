// internal/telegram/types.go
package telegram

// ReplyKeyboardButton - кнопка reply клавиатуры
type ReplyKeyboardButton struct {
	Text string `json:"text"`
}

// ReplyKeyboardMarkup - разметка reply клавиатуры
type ReplyKeyboardMarkup struct {
	Keyboard        [][]ReplyKeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool                    `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard bool                    `json:"one_time_keyboard,omitempty"`
	Selective       bool                    `json:"selective,omitempty"`
	RemoveKeyboard  bool                    `json:"remove_keyboard,omitempty"`
}
