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
	// ClearFilter: фильтр активен, список пуст → ноль сигналов
	if err := h.watchlistService.ClearFilter(params.User.ID); err != nil {
		return handlers.HandlerResult{}, err
	}
	return handlers.HandlerResult{
		Message: "🗑️ Фильтр очищен.\n\nСигналов не будет, пока не добавите монеты.\n\nЧтобы получать все сигналы — нажмите *«📡 Все сигналы»*.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{{"text": "📡 Все сигналы", "callback_data": constants.CallbackWatchlistDisable}},
				{{"text": "📋 Открыть фильтр", "callback_data": constants.CallbackWatchlistMenu}},
				{{"text": "🔙 Главное меню", "callback_data": constants.CallbackMenuMain}},
			},
		},
	}, nil
}
