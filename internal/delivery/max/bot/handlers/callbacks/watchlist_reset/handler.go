// internal/delivery/max/bot/handlers/callbacks/watchlist_reset/handler.go
package watchlist_reset

import (
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

	rows := [][]map[string]string{
		{kb.B("📋 Открыть вотчлист", kb.CbWatchlistMenu)},
		kb.BackRow(kb.CbMenuMain),
	}

	return handlers.HandlerResult{
		Message:     "✅ Вотчлист очищен. Теперь отслеживаются все монеты.",
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
