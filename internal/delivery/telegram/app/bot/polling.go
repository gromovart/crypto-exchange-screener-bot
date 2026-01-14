// internal/delivery/telegram/app/bot/polling.go
package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/middlewares"
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
	log.Println("🔄 Starting Telegram bot polling...")

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
				Text string `json:"text"`
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
		Text string `json:"text"`
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
}) {
	// Преобразуем в формат middlewares.TelegramUpdate
	middlewareUpdate := &middlewares.TelegramUpdate{
		UpdateID: update.UpdateID,
	}

	if update.Message != nil {
		middlewareUpdate.Message = &struct {
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
			Text string `json:"text"`
		}{
			MessageID: update.Message.MessageID,
			Text:      update.Message.Text,
		}

		if update.Message.From != nil {
			middlewareUpdate.Message.From = &struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			}{
				ID:        update.Message.From.ID,
				Username:  update.Message.From.Username,
				FirstName: update.Message.From.FirstName,
				LastName:  update.Message.From.LastName,
			}
		}

		if update.Message.Chat != nil {
			middlewareUpdate.Message.Chat = &struct {
				ID int64 `json:"id"`
			}{
				ID: update.Message.Chat.ID,
			}
		}
	}

	if update.CallbackQuery != nil {
		middlewareUpdate.CallbackQuery = &struct {
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
		}{
			ID:   update.CallbackQuery.ID,
			Data: update.CallbackQuery.Data,
		}

		if update.CallbackQuery.From != nil {
			middlewareUpdate.CallbackQuery.From = &struct {
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
	}

	// Обрабатываем обновление через бота
	if err := pc.bot.HandleUpdate(middlewareUpdate); err != nil {
		log.Printf("❌ Error handling update %d: %v", update.UpdateID, err)
	}
}
