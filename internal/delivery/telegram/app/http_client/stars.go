// internal/delivery/telegram/app/http_client/stars.go
package http_client

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
)

// StarsClient клиент для работы с Telegram Stars API
type StarsClient struct {
	*TelegramClient
	providerToken string // Токен платежного провайдера (Telegram Stars)
}

// NewStarsClient создает новый клиент для работы со Stars
func NewStarsClient(baseURL, providerToken string) *StarsClient {
	return &StarsClient{
		TelegramClient: NewTelegramClient(baseURL),
		providerToken:  providerToken,
	}
}

// CreateInvoiceLink создает инвойс для оплаты через Telegram Stars
func (c *StarsClient) CreateInvoiceLink(invoice telegram.Invoice) (string, error) {
	endpoint := "createInvoiceLink"

	if c.providerToken == "" {
		return "", fmt.Errorf("токен платежного провайдера не указан")
	}

	// Устанавливаем провайдера
	invoice.ProviderToken = c.providerToken

	// Если не указана валюта, используем Stars (XTR)
	if invoice.Currency == "" {
		invoice.Currency = "XTR"
	}

	params := map[string]interface{}{
		"title":                         invoice.Title,
		"description":                   invoice.Description,
		"payload":                       invoice.Payload,
		"provider_token":                invoice.ProviderToken,
		"currency":                      invoice.Currency,
		"prices":                        invoice.Prices,
		"max_tip_amount":                invoice.MaxTipAmount,
		"suggested_tip_amounts":         invoice.SuggestedTipAmounts,
		"start_parameter":               invoice.StartParameter,
		"photo_url":                     invoice.PhotoURL,
		"photo_size":                    invoice.PhotoSize,
		"photo_width":                   invoice.PhotoWidth,
		"photo_height":                  invoice.PhotoHeight,
		"need_name":                     invoice.NeedName,
		"need_phone_number":             invoice.NeedPhoneNumber,
		"need_email":                    invoice.NeedEmail,
		"need_shipping_address":         invoice.NeedShippingAddress,
		"send_phone_number_to_provider": invoice.SendPhoneNumberToProvider,
		"send_email_to_provider":        invoice.SendEmailToProvider,
		"is_flexible":                   invoice.IsFlexible,
	}

	var response telegram.CreateInvoiceResponse
	if err := c.makeRequest(endpoint, params, &response); err != nil {
		return "", fmt.Errorf("ошибка создания инвойса: %v", err)
	}

	if !response.OK {
		return "", fmt.Errorf("ошибка Telegram API: %s", response.Description)
	}

	if response.Result == nil {
		return "", fmt.Errorf("результат создания инвойса пуст")
	}

	logger.Info("Создан инвойс Stars: %s", response.Result.InvoiceLink)
	return response.Result.InvoiceLink, nil
}

// AnswerPreCheckoutQuery отвечает на pre-checkout запрос
func (c *StarsClient) AnswerPreCheckoutQuery(preCheckoutQueryID string, ok bool, errorMessage string) error {
	endpoint := "answerPreCheckoutQuery"

	params := map[string]interface{}{
		"pre_checkout_query_id": preCheckoutQueryID,
		"ok":                    ok,
	}

	if !ok && errorMessage != "" {
		params["error_message"] = errorMessage
	}

	var response struct {
		OK          bool   `json:"ok"`
		Description string `json:"description,omitempty"`
	}

	if err := c.makeRequest(endpoint, params, &response); err != nil {
		return fmt.Errorf("ошибка ответа на pre-checkout: %v", err)
	}

	if !response.OK {
		return fmt.Errorf("ошибка Telegram API: %s", response.Description)
	}

	logger.Debug("Ответ на pre-checkout отправлен: %s", preCheckoutQueryID)
	return nil
}

// CreateSubscriptionInvoice создает инвойс для подписки
func (c *StarsClient) CreateSubscriptionInvoice(title, description, payload string, starsAmount int) (string, error) {
	invoice := telegram.Invoice{
		Title:           title,
		Description:     description,
		Payload:         payload,
		Currency:        "XTR",
		StartParameter:  payload,
		NeedName:        false,
		NeedPhoneNumber: false,
		NeedEmail:       false,
		IsFlexible:      false,
	}

	// Добавляем цену
	invoice.Prices = []telegram.LabeledPrice{
		{
			Label:  "Подписка",
			Amount: starsAmount,
		},
	}

	return c.CreateInvoiceLink(invoice)
}
