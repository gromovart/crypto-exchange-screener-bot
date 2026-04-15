// internal/delivery/max/bot/handlers/callbacks/watchlist_disable/handler.go
package watchlist_disable

import (
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

type Handler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

func New(watchlistService watchlistSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler:      base.New("watchlist_disable", kb.CbWatchlistDisable, handlers.TypeCallback),
		watchlistService: watchlistService,
	}
}

func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// DisableFilter: nil → фильтр отключён → все сигналы
	if err := h.watchlistService.DisableFilter(params.User.ID); err != nil {
		return handlers.HandlerResult{}, err
	}
	rows := [][]map[string]string{
		{kb.B("📋 Настроить фильтр", kb.CbWatchlistMenu)},
		kb.BackRow(kb.CbMenuMain),
	}
	return handlers.HandlerResult{
		Message:     "📡 Фильтр отключён. Приходят сигналы по всем монетам.",
		Keyboard:    kb.Keyboard(rows),
		EditMessage: params.MessageID != "",
	}, nil
}
