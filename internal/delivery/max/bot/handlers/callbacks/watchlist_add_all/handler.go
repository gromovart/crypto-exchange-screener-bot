// internal/delivery/max/bot/handlers/callbacks/watchlist_add_all/handler.go
package watchlist_add_all

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

// Handler — добавление всех монет в вотчлист
type Handler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

// New создаёт обработчик добавления всех монет
func New(watchlistService watchlistSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler:      base.New("watchlist_add_all", kb.CbWatchlistAddAll, handlers.TypeCallback),
		watchlistService: watchlistService,
	}
}

func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if err := h.watchlistService.AddAllToWatchlist(params.User.ID); err != nil {
		return handlers.HandlerResult{}, err
	}

	total := len(h.watchlistService.GetAllSymbols())

	rows := [][]map[string]string{
		{kb.B("📋 Открыть вотчлист", kb.CbWatchlistMenu)},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     fmt.Sprintf("✅ Все %d монет добавлены в вотчлист.\n\nТеперь вы можете убирать ненужные монеты вручную.", total),
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
