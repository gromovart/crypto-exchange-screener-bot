// internal/delivery/telegram/app/bot/handlers/callbacks/watchlist_view/handler.go
package watchlist_view

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

type watchlistViewHandler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

func NewHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &watchlistViewHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_view_handler",
			Command: constants.CallbackWatchlistView,
			Type:    handlers.TypeCallback,
		},
		watchlistService: watchlistService,
	}
}

func (h *watchlistViewHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	watchlist, err := h.watchlistService.GetUserWatchlist(params.User.ID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": "🔙 Назад", "callback_data": constants.CallbackWatchlistMenu}},
		},
	}

	if len(watchlist) == 0 {
		return handlers.HandlerResult{
			Message:  "📋 *Вотчлист пуст*\n\nОтслеживаются все монеты (фильтр не задан).",
			Keyboard: keyboard,
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 *Мой вотчлист* (%d монет):\n\n", len(watchlist)))
	for i, sym := range watchlist {
		sb.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, sym))
		// Telegram сообщение ограничено 4096 символами
		if i == 199 {
			sb.WriteString(fmt.Sprintf("\n_...и ещё %d монет_", len(watchlist)-200))
			break
		}
	}

	return handlers.HandlerResult{
		Message:  sb.String(),
		Keyboard: keyboard,
	}, nil
}
