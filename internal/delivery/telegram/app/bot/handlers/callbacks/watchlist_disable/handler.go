// internal/delivery/telegram/app/bot/handlers/callbacks/watchlist_disable/handler.go
package watchlist_disable

import (
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

type watchlistDisableHandler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

func NewHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &watchlistDisableHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_disable_handler",
			Command: constants.CallbackWatchlistDisable,
			Type:    handlers.TypeCallback,
		},
		watchlistService: watchlistService,
	}
}

func (h *watchlistDisableHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// DisableFilter: nil → фильтр отключён → все сигналы
	if err := h.watchlistService.DisableFilter(params.User.ID); err != nil {
		return handlers.HandlerResult{}, err
	}
	return handlers.HandlerResult{
		Message: "📡 Фильтр отключён. Приходят сигналы по *всем монетам*.",
		Keyboard: map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{{"text": "📋 Настроить фильтр", "callback_data": constants.CallbackWatchlistMenu}},
				{{"text": "🔙 Главное меню", "callback_data": constants.CallbackMenuMain}},
			},
		},
	}, nil
}
