// internal/delivery/max/bot/handlers/callbacks/watchlist_reset/handler.go
package watchlist_reset

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

// Handler — сброс вотчлиста
type Handler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

// New создаёт обработчик сброса вотчлиста
func New(watchlistService watchlistSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler:      base.New("watchlist_reset", kb.CbWatchlistReset, handlers.TypeCallback),
		watchlistService: watchlistService,
	}
}

func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if err := h.watchlistService.ResetWatchlist(params.User.ID); err != nil {
		return handlers.HandlerResult{}, err
	}

	letters := h.watchlistService.GetAvailableLetters()
	total := len(h.watchlistService.GetAllSymbols())

	var rows [][]map[string]string
	rows = append(rows, []map[string]string{kb.B("🔍 Найти монету", kb.CbWatchlistSearch)})
	const lettersPerRow = 8
	for i := 0; i < len(letters); i += lettersPerRow {
		end := i + lettersPerRow
		if end > len(letters) {
			end = len(letters)
		}
		var row []map[string]string
		for _, l := range letters[i:end] {
			row = append(row, kb.B(l, "watchlist_letter_"+l+"_0"))
		}
		rows = append(rows, row)
	}
	rows = append(rows, kb.BackRow(kb.CbMenuMain))

	return handlers.HandlerResult{
		Message:     fmt.Sprintf("✅ Вотчлист сброшен. Теперь отслеживаются все монеты (%d).", total),
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
