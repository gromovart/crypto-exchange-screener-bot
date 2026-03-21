// internal/infrastructure/http/tbank/types.go
package tbank

// InitRequest запрос на инициализацию платежа
type InitRequest struct {
	TerminalKey     string `json:"TerminalKey"`
	Amount          int64  `json:"Amount"` // в копейках
	OrderId         string `json:"OrderId"`
	Description     string `json:"Description,omitempty"`
	NotificationURL string `json:"NotificationURL,omitempty"`
	SuccessURL      string `json:"SuccessURL,omitempty"`
	FailURL         string `json:"FailURL,omitempty"`
	PayType         string `json:"PayType,omitempty"` // "O" - одностадийный
	Language        string `json:"Language,omitempty"`
	RedirectDueDate string `json:"RedirectDueDate,omitempty"`
	Token           string `json:"Token"`
}

// InitResponse ответ на инициализацию платежа
type InitResponse struct {
	TerminalKey string `json:"TerminalKey"`
	Amount      int64  `json:"Amount"`
	OrderId     string `json:"OrderId"`
	Success     bool   `json:"Success"`
	Status      string `json:"Status"`
	PaymentId   string `json:"PaymentId"`
	ErrorCode   string `json:"ErrorCode"`
	Message     string `json:"Message"`
	Details     string `json:"Details"`
	PaymentURL  string `json:"PaymentURL"`
}

// Notification входящее уведомление от Т-Банк о статусе платежа
type Notification struct {
	TerminalKey string `json:"TerminalKey"`
	OrderId     string `json:"OrderId"`
	Success     bool   `json:"Success"`
	Status      string `json:"Status"` // CONFIRMED, REJECTED, AUTHORIZED и др.
	PaymentId   string `json:"PaymentId"`
	ErrorCode   string `json:"ErrorCode"`
	Amount      int64  `json:"Amount"`
	CardId      string `json:"CardId,omitempty"`
	Pan         string `json:"Pan,omitempty"`
	ExpDate     string `json:"ExpDate,omitempty"`
	Token       string `json:"Token"`
}
