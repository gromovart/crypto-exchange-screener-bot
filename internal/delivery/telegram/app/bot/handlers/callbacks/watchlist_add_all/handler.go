// internal/delivery/telegram/app/bot/handlers/callbacks/watchlist_add_all/handler.go
package watchlist_add_all

import (
	"fmt"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

type watchlistAddAllHandler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

// NewHandler создаёт обработчик добавления всех монет в вотчлист
func NewHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &watchlistAddAllHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_add_all_handler",
			Command: constants.CallbackWatchlistAddAll,
			Type:    handlers.TypeCallback,
		},
		watchlistService: watchlistService,
	}
}

func (h *watchlistAddAllHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if err := h.watchlistService.AddAllToWatchlist(params.User.ID); err != nil {
		return handlers.HandlerResult{}, err
	}

	total := len(h.watchlistService.GetAllSymbols())

	return handlers.HandlerResult{
		Message: fmt.Sprintf(
			"✅ Все *%d монет* добавлены в вотчлист.\n\nТеперь вы можете убирать ненужные монеты вручную.",
			total,
		),
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{{"text": "📋 Открыть вотчлист", "callback_data": constants.CallbackWatchlistMenu}},
				{{"text": "🔙 Главное меню", "callback_data": constants.CallbackMenuMain}},
			},
		},
	}, nil
}
