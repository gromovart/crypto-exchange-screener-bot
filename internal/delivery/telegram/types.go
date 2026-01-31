// internal/delivery/telegram/types.go
package telegram

import (
	"sync"
	"time"
)

// RateLimiter - ограничитель частоты запросов
type RateLimiter struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
	minDelay time.Duration
}

// TelegramResponse - ответ от Telegram API
type TelegramResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
}

// InlineKeyboardButton - кнопка inline клавиатуры
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
	URL          string `json:"url,omitempty"`
}

// InlineKeyboardMarkup - разметка inline клавиатуры
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// TelegramMessage - сообщение с клавиатурой
type TelegramMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
	// ParseMode   string      `json:"parse_mode,omitempty"`
	ReplyMarkup interface{} `json:"reply_markup,omitempty"`
}

// ReplyKeyboardButton - кнопка reply клавиатуры
type ReplyKeyboardButton struct {
	Text string `json:"text"`
}

// ReplyKeyboardMarkup - разметка reply клавиатуры
type ReplyKeyboardMarkup struct {
	Keyboard        [][]ReplyKeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool                    `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard bool                    `json:"one_time_keyboard,omitempty"`
	RemoveKeyboard  bool                    `json:"remove_keyboard,omitempty"`
	Selective       bool                    `json:"selective,omitempty"`
	IsPersistent    bool                    `json:"is_persistent,omitempty"` // Новая опция
}

// NewRateLimiter создает новый ограничитель частоты
func NewRateLimiter(minDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		lastSent: make(map[string]time.Time),
		minDelay: minDelay,
	}
}

// CanSend проверяет, можно ли отправить сообщение
func (rl *RateLimiter) CanSend(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if last, exists := rl.lastSent[key]; exists {
		if now.Sub(last) < rl.minDelay {
			return false
		}
	}
	rl.lastSent[key] = now
	return true
}

// BotCommand представляет команду в меню бота
type BotCommand struct {
	Command     string `json:"command"`     // Команда (1-32 символа)
	Description string `json:"description"` // Описание (1-256 символов)
}

// SetMyCommandsParams параметры для установки команд
type SetMyCommandsParams struct {
	Commands []BotCommand     `json:"commands"`                // Список команд
	Scope    *BotCommandScope `json:"scope,omitempty"`         // Область видимости
	Language string           `json:"language_code,omitempty"` // Язык
}

// BotCommandScope - область видимости команд
type BotCommandScope struct {
	Type string `json:"type"` // Тип области: "default", "all_private_chats", "all_group_chats", "all_chat_administrators"
}

// SetMyCommandsResponse - ответ на установку команд
type SetMyCommandsResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
}

// LabeledPrice представляет цену с меткой для Telegram Stars
type LabeledPrice struct {
	Label  string `json:"label"`  // Метка цены (например, "Подписка")
	Amount int    `json:"amount"` // Сумма в минимальных единицах валюты
}

// Invoice представляет инвойс для Telegram Stars
type Invoice struct {
	Title                     string         `json:"title"`                                   // Название товара (1-32 символа)
	Description               string         `json:"description"`                             // Описание товара (1-255 символов)
	Payload                   string         `json:"payload"`                                 // Уникальный payload (1-128 байт)
	ProviderToken             string         `json:"provider_token"`                          // Токен платежного провайдера
	Currency                  string         `json:"currency"`                                // Валюта (XTR для Stars)
	Prices                    []LabeledPrice `json:"prices"`                                  // Цены
	MaxTipAmount              int            `json:"max_tip_amount,omitempty"`                // Максимальная сумма чаевых
	SuggestedTipAmounts       []int          `json:"suggested_tip_amounts,omitempty"`         // Предлагаемые чаевые
	StartParameter            string         `json:"start_parameter,omitempty"`               // Параметр для /start
	PhotoURL                  string         `json:"photo_url,omitempty"`                     // URL фото товара
	PhotoSize                 int            `json:"photo_size,omitempty"`                    // Размер фото
	PhotoWidth                int            `json:"photo_width,omitempty"`                   // Ширина фото
	PhotoHeight               int            `json:"photo_height,omitempty"`                  // Высота фото
	NeedName                  bool           `json:"need_name,omitempty"`                     // Требовать имя
	NeedPhoneNumber           bool           `json:"need_phone_number,omitempty"`             // Требовать телефон
	NeedEmail                 bool           `json:"need_email,omitempty"`                    // Требовать email
	NeedShippingAddress       bool           `json:"need_shipping_address,omitempty"`         // Требовать адрес
	SendPhoneNumberToProvider bool           `json:"send_phone_number_to_provider,omitempty"` // Отправлять телефон провайдеру
	SendEmailToProvider       bool           `json:"send_email_to_provider,omitempty"`        // Отправлять email провайдеру
	IsFlexible                bool           `json:"is_flexible,omitempty"`                   // Гибкая цена
}

// CreateInvoiceResponse ответ на создание инвойса
type CreateInvoiceResponse struct {
	OK          bool           `json:"ok"`
	Result      *InvoiceResult `json:"result,omitempty"`
	Description string         `json:"description,omitempty"`
	ErrorCode   int            `json:"error_code,omitempty"`
}

// InvoiceResult результат создания инвойса
type InvoiceResult struct {
	InvoiceLink string `json:"invoice_link"` // Ссылка на инвойс
}

// PreCheckoutQuery предварительный запрос на проверку
type PreCheckoutQuery struct {
	ID               string     `json:"id"`
	From             User       `json:"from"`
	Currency         string     `json:"currency"`
	TotalAmount      int        `json:"total_amount"`
	InvoicePayload   string     `json:"invoice_payload"`
	ShippingOptionID string     `json:"shipping_option_id,omitempty"`
	OrderInfo        *OrderInfo `json:"order_info,omitempty"`
}

// OrderInfo информация о заказе
type OrderInfo struct {
	Name            string           `json:"name,omitempty"`
	PhoneNumber     string           `json:"phone_number,omitempty"`
	Email           string           `json:"email,omitempty"`
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty"`
}

// ShippingAddress адрес доставки
type ShippingAddress struct {
	CountryCode string `json:"country_code"`
	State       string `json:"state,omitempty"`
	City        string `json:"city"`
	StreetLine1 string `json:"street_line1"`
	StreetLine2 string `json:"street_line2,omitempty"`
	PostCode    string `json:"post_code"`
}

// SuccessfulPayment успешный платеж
type SuccessfulPayment struct {
	Currency                string     `json:"currency"`
	TotalAmount             int        `json:"total_amount"`
	InvoicePayload          string     `json:"invoice_payload"`
	ShippingOptionID        string     `json:"shipping_option_id,omitempty"`
	OrderInfo               *OrderInfo `json:"order_info,omitempty"`
	TelegramPaymentChargeID string     `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID string     `json:"provider_payment_charge_id"`
}

// AnswerPreCheckoutQueryParams параметры для ответа на pre-checkout
type AnswerPreCheckoutQueryParams struct {
	PreCheckoutQueryID string `json:"pre_checkout_query_id"`
	OK                 bool   `json:"ok"`
	ErrorMessage       string `json:"error_message,omitempty"`
}
type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}
