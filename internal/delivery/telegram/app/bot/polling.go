// internal/delivery/telegram/app/bot/polling.go
package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram"
	"crypto-exchange-screener-bot/pkg/logger"
)

// PollingClient - клиент для polling обновлений
type PollingClient struct {
	bot      *TelegramBot
	offset   int
	running  bool
	stopChan chan struct{}
}

// NewPollingClient создает новый polling клиент
func NewPollingClient(bot *TelegramBot) *PollingClient {
	return &PollingClient{
		bot:      bot,
		offset:   0,
		running:  false,
		stopChan: make(chan struct{}),
	}
}

// Start запускает polling обновлений
func (pc *PollingClient) Start() error {
	if pc.running {
		return fmt.Errorf("polling already running")
	}

	pc.running = true
	logger.Warn("🔄 Starting Telegram bot polling...")

	go pc.pollLoop()

	return nil
}

// Stop останавливает polling обновлений
func (pc *PollingClient) Stop() error {
	if !pc.running {
		return nil
	}

	pc.running = false
	close(pc.stopChan)
	log.Println("🛑 Stopping Telegram bot polling...")

	return nil
}

// pollLoop основной цикл polling
func (pc *PollingClient) pollLoop() {
	for pc.running {
		select {
		case <-pc.stopChan:
			return
		default:
			pc.fetchUpdates()
			time.Sleep(1 * time.Second)
		}
	}
}

// fetchUpdates получает обновления от Telegram API
func (pc *PollingClient) fetchUpdates() {
	// Используем PollingClient с увеличенным таймаутом (35 секунд)
	resp, err := pc.bot.GetPollingClient().GetUpdates(pc.offset, 30)
	if err != nil {
		log.Printf("❌ Error fetching updates: %v", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result []struct {
			UpdateID int `json:"update_id"`
			Message  *struct {
				MessageID int `json:"message_id"`
				From      *struct {
					ID        int64  `json:"id"`
					Username  string `json:"username"`
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
				} `json:"from"`
				Chat *struct {
					ID int64 `json:"id"`
				} `json:"chat"`
				Text              string `json:"text"`
				SuccessfulPayment *struct {
					Currency                string `json:"currency"`
					TotalAmount             int    `json:"total_amount"`
					InvoicePayload          string `json:"invoice_payload"`
					TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
					ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
				} `json:"successful_payment"`
			} `json:"message"`
			CallbackQuery *struct {
				ID   string `json:"id"`
				From *struct {
					ID        int64  `json:"id"`
					Username  string `json:"username"`
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
				} `json:"from"`
				Message *struct {
					MessageID int `json:"message_id"`
					Chat      *struct {
						ID int64 `json:"id"`
					} `json:"chat"`
				} `json:"message"`
				Data string `json:"data"`
			} `json:"callback_query"`
			PreCheckoutQuery *struct {
				ID   string `json:"id"`
				From *struct {
					ID        int64  `json:"id"`
					Username  string `json:"username"`
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
				} `json:"from"`
				Currency       string `json:"currency"`
				TotalAmount    int    `json:"total_amount"`
				InvoicePayload string `json:"invoice_payload"`
			} `json:"pre_checkout_query"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("❌ Error decoding updates: %v", err)
		return
	}

	if !result.OK {
		return
	}

	for _, update := range result.Result {
		pc.processUpdate(update)
		pc.offset = update.UpdateID + 1
	}
}

// processUpdate обрабатывает одно обновление
func (pc *PollingClient) processUpdate(update struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int `json:"message_id"`
		From      *struct {
			ID        int64  `json:"id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"from"`
		Chat *struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text              string `json:"text"`
		SuccessfulPayment *struct {
			Currency                string `json:"currency"`
			TotalAmount             int    `json:"total_amount"`
			InvoicePayload          string `json:"invoice_payload"`
			TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
			ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
		} `json:"successful_payment"`
	} `json:"message"`
	CallbackQuery *struct {
		ID   string `json:"id"`
		From *struct {
			ID        int64  `json:"id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"from"`
		Message *struct {
			MessageID int `json:"message_id"`
			Chat      *struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
		Data string `json:"data"`
	} `json:"callback_query"`
	PreCheckoutQuery *struct {
		ID   string `json:"id"`
		From *struct {
			ID        int64  `json:"id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"from"`
		Currency       string `json:"currency"`
		TotalAmount    int    `json:"total_amount"`
		InvoicePayload string `json:"invoice_payload"`
	} `json:"pre_checkout_query"`
}) {
	// ⭐ ЛОГ ДЛЯ ОТЛАДКИ
	logger.Warn("📩 [POLLING] Получено обновление ID=%d", update.UpdateID)
	logger.Warn("   • Message: %v", update.Message != nil)
	logger.Warn("   • Callback: %v", update.CallbackQuery != nil)
	logger.Warn("   • PreCheckout: %v", update.PreCheckoutQuery != nil)

	// Если есть successful_payment, логируем детали
	if update.Message != nil && update.Message.SuccessfulPayment != nil {
		logger.Warn("💰💰💰 [POLLING] SUCCESSFUL PAYMENT DETECTED!")
		logger.Warn("   • From ID: %d", update.Message.From.ID)
		logger.Warn("   • Amount: %d %s", update.Message.SuccessfulPayment.TotalAmount, update.Message.SuccessfulPayment.Currency)
		logger.Warn("   • Payload: %s", update.Message.SuccessfulPayment.InvoicePayload)
		logger.Warn("   • TelegramChargeID: %s", update.Message.SuccessfulPayment.TelegramPaymentChargeID)
		logger.Warn("   • ProviderChargeID: %s", update.Message.SuccessfulPayment.ProviderPaymentChargeID)
	}

	// Создаем обновление в формате telegram.TelegramUpdate
	middlewareUpdate := &telegram.TelegramUpdate{
		UpdateID: update.UpdateID,
	}

	// Обработка Message
	if update.Message != nil {
		msg := &telegram.Message{
			MessageID: int64(update.Message.MessageID),
			Text:      update.Message.Text,
		}

		// Преобразуем From
		if update.Message.From != nil {
			msg.From = telegram.User{
				ID:        update.Message.From.ID,
				Username:  update.Message.From.Username,
				FirstName: update.Message.From.FirstName,
				LastName:  update.Message.From.LastName,
			}
		}

		// Преобразуем Chat
		if update.Message.Chat != nil {
			msg.Chat = telegram.Chat{
				ID: update.Message.Chat.ID,
			}
		}

		// ⭐ Добавляем SuccessfulPayment если есть
		if update.Message.SuccessfulPayment != nil {
			msg.SuccessfulPayment = &telegram.SuccessfulPayment{
				Currency:                update.Message.SuccessfulPayment.Currency,
				TotalAmount:             update.Message.SuccessfulPayment.TotalAmount,
				InvoicePayload:          update.Message.SuccessfulPayment.InvoicePayload,
				TelegramPaymentChargeID: update.Message.SuccessfulPayment.TelegramPaymentChargeID,
				ProviderPaymentChargeID: update.Message.SuccessfulPayment.ProviderPaymentChargeID,
			}
		}

		middlewareUpdate.Message = msg
	}

	// Обработка CallbackQuery
	if update.CallbackQuery != nil {
		callback := &telegram.CallbackQueryStruct{
			ID:   update.CallbackQuery.ID,
			Data: update.CallbackQuery.Data,
		}

		// Преобразуем From
		if update.CallbackQuery.From != nil {
			callback.From = &struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			}{
				ID:        update.CallbackQuery.From.ID,
				Username:  update.CallbackQuery.From.Username,
				FirstName: update.CallbackQuery.From.FirstName,
				LastName:  update.CallbackQuery.From.LastName,
			}
		}

		// Преобразуем Message
		if update.CallbackQuery.Message != nil {
			callback.Message = &struct {
				MessageID int `json:"message_id"`
				Chat      *struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			}{
				MessageID: update.CallbackQuery.Message.MessageID,
			}

			if update.CallbackQuery.Message.Chat != nil {
				callback.Message.Chat = &struct {
					ID int64 `json:"id"`
				}{
					ID: update.CallbackQuery.Message.Chat.ID,
				}
			}
		}

		middlewareUpdate.CallbackQuery = callback
	}

	// Обработка PreCheckoutQuery
	if update.PreCheckoutQuery != nil {
		preCheckout := &telegram.PreCheckoutQuery{
			ID:             update.PreCheckoutQuery.ID,
			Currency:       update.PreCheckoutQuery.Currency,
			TotalAmount:    update.PreCheckoutQuery.TotalAmount,
			InvoicePayload: update.PreCheckoutQuery.InvoicePayload,
		}

		// Преобразуем From
		if update.PreCheckoutQuery.From != nil {
			preCheckout.From = &struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			}{
				ID:        update.PreCheckoutQuery.From.ID,
				Username:  update.PreCheckoutQuery.From.Username,
				FirstName: update.PreCheckoutQuery.From.FirstName,
				LastName:  update.PreCheckoutQuery.From.LastName,
			}
		}

		middlewareUpdate.PreCheckoutQuery = preCheckout
	}

	// Обрабатываем обновление через бота
	if err := pc.bot.HandleUpdate(middlewareUpdate); err != nil {
		log.Printf("❌ Error handling update %d: %v", update.UpdateID, err)
	}
}
