// internal/delivery/telegram/app/bot/handlers/callbacks/watchlist_reset/handler.go
package watchlist_reset

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

type watchlistResetHandler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

// NewHandler создаёт обработчик сброса вотчлиста
func NewHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &watchlistResetHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_reset_handler",
			Command: constants.CallbackWatchlistReset,
			Type:    handlers.TypeCallback,
		},
		watchlistService: watchlistService,
	}
}

func (h *watchlistResetHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if err := h.watchlistService.ResetWatchlist(params.User.ID); err != nil {
		return handlers.HandlerResult{}, err
	}

	letters := h.watchlistService.GetAvailableLetters()
	total := len(h.watchlistService.GetAllSymbols())

	keyboard := buildMenuKeyboard(letters)
	return handlers.HandlerResult{
		Message:  "✅ Вотчлист сброшен. Теперь отслеживаются *все монеты* (" + itoa(total) + ").",
		Keyboard: keyboard,
	}, nil
}

func buildMenuKeyboard(letters []string) interface{} {
	var rows [][]map[string]string
	rows = append(rows, []map[string]string{
		{"text": "🔍 Поиск по названию", "callback_data": constants.CallbackWatchlistSearch},
	})
	const lettersPerRow = 8
	for i := 0; i < len(letters); i += lettersPerRow {
		end := i + lettersPerRow
		if end > len(letters) {
			end = len(letters)
		}
		var row []map[string]string
		for _, l := range letters[i:end] {
			row = append(row, map[string]string{
				"text":          l,
				"callback_data": constants.CallbackWatchlistLetterPrefix + l + ":0",
			})
		}
		rows = append(rows, row)
	}
	rows = append(rows, []map[string]string{
		{"text": "🔙 Назад", "callback_data": constants.CallbackMenuMain},
	})
	return map[string]interface{}{"inline_keyboard": rows}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
