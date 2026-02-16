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
		providerToken:  "",
	}
}

// CreateInvoiceLink создает инвойс для оплаты через Telegram Stars
func (c *StarsClient) CreateInvoiceLink(invoice telegram.Invoice) (string, error) {
	endpoint := "createInvoiceLink"

	// ВАЖНО: Для цифровых товаров provider_token может быть пустой строкой
	// Согласно документации: "You can pass an empty string as the provider_token parameter"
	if c.providerToken != "" {
		invoice.ProviderToken = c.providerToken
	} else {
		// Для цифровых товаров оставляем пустую строку
		invoice.ProviderToken = ""
		logger.Debug("Создание инвойса для цифровых товаров (provider_token: '')")
	}

	// Если не указана валюта, используем Stars (XTR)
	if invoice.Currency == "" {
		invoice.Currency = "XTR"
	}

	// Проверяем обязательные поля
	if invoice.Title == "" || invoice.Description == "" || invoice.Payload == "" {
		return "", fmt.Errorf("обязательные поля инвойса не заполнены: title, description, payload")
	}

	// Проверяем цены
	if len(invoice.Prices) == 0 {
		return "", fmt.Errorf("инвойс должен содержать хотя бы одну цену")
	}

	params := map[string]interface{}{
		"title":          invoice.Title,
		"description":    invoice.Description,
		"payload":        invoice.Payload,
		"provider_token": invoice.ProviderToken, // Может быть пустой строкой
		"currency":       invoice.Currency,
		"prices":         invoice.Prices,
	}

	// Добавляем опциональные поля только если они не нулевые
	if invoice.MaxTipAmount > 0 {
		params["max_tip_amount"] = invoice.MaxTipAmount
	}
	if len(invoice.SuggestedTipAmounts) > 0 {
		params["suggested_tip_amounts"] = invoice.SuggestedTipAmounts
	}
	if invoice.StartParameter != "" {
		params["start_parameter"] = invoice.StartParameter
	}
	if invoice.PhotoURL != "" {
		params["photo_url"] = invoice.PhotoURL
		params["photo_size"] = invoice.PhotoSize
		params["photo_width"] = invoice.PhotoWidth
		params["photo_height"] = invoice.PhotoHeight
	}
	if invoice.NeedName {
		params["need_name"] = invoice.NeedName
	}
	if invoice.NeedPhoneNumber {
		params["need_phone_number"] = invoice.NeedPhoneNumber
	}
	if invoice.NeedEmail {
		params["need_email"] = invoice.NeedEmail
	}
	if invoice.NeedShippingAddress {
		params["need_shipping_address"] = invoice.NeedShippingAddress
	}
	if invoice.SendPhoneNumberToProvider {
		params["send_phone_number_to_provider"] = invoice.SendPhoneNumberToProvider
	}
	if invoice.SendEmailToProvider {
		params["send_email_to_provider"] = invoice.SendEmailToProvider
	}
	if invoice.IsFlexible {
		params["is_flexible"] = invoice.IsFlexible
	}

	var response telegram.CreateInvoiceResponse
	if err := c.makeRequest(endpoint, params, &response); err != nil {
		return "", fmt.Errorf("ошибка создания инвойса: %v", err)
	}

	if !response.OK {
		return "", fmt.Errorf("ошибка Telegram API: %s", response.Description)
	}

	if response.Result == "" {
		return "", fmt.Errorf("результат создания инвойса пуст")
	}

	logger.Info("Создан инвойс Stars: %s", response.Result)
	return response.Result, nil
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
		Title:                     title,
		Description:               description,
		Payload:                   payload,
		Currency:                  "XTR",
		StartParameter:            payload,
		NeedName:                  false,
		NeedPhoneNumber:           false,
		NeedEmail:                 false,
		NeedShippingAddress:       false,
		SendPhoneNumberToProvider: false,
		SendEmailToProvider:       false,
		IsFlexible:                false,
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
