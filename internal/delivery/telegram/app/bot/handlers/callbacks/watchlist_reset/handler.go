// internal/delivery/telegram/app/bot/handlers/callbacks/watchlist_reset/handler.go
package watchlist_reset

import (
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

	return handlers.HandlerResult{
		Message: "✅ Вотчлист очищен. Теперь отслеживаются *все монеты*.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{{"text": "📋 Открыть вотчлист", "callback_data": constants.CallbackWatchlistMenu}},
				{{"text": "🔙 Главное меню", "callback_data": constants.CallbackMenuMain}},
			},
		},
	}, nil
}
