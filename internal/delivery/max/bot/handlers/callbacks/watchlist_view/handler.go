// internal/delivery/max/bot/handlers/callbacks/watchlist_view/handler.go
package watchlist_view

import (
	"fmt"
	"strings"

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
		BaseHandler:      base.New("watchlist_view", kb.CbWatchlistView, handlers.TypeCallback),
		watchlistService: watchlistService,
	}
}

func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	watchlist, err := h.watchlistService.GetUserWatchlist(params.User.ID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}

	rows := [][]map[string]string{
		{kb.B("🔙 Назад", kb.CbWatchlistMenu)},
	}
	keyboard := kb.Keyboard(rows)

	if len(watchlist) == 0 {
		return handlers.HandlerResult{
			Message:     "📋 Вотчлист пуст\n\nОтслеживаются все монеты (фильтр не задан).",
			Keyboard:    keyboard,
			EditMessage: params.MessageID != "",
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 Мой вотчлист (%d монет):\n\n", len(watchlist)))
	// MAX ограничивает длину сообщения — показываем первые 100
	limit := len(watchlist)
	if limit > 100 {
		limit = 100
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, watchlist[i]))
	}
	if len(watchlist) > 100 {
		sb.WriteString(fmt.Sprintf("\n...и ещё %d монет", len(watchlist)-100))
	}

	return handlers.HandlerResult{
		Message:     sb.String(),
		Keyboard:    keyboard,
		EditMessage: params.MessageID != "",
	}, nil
}
