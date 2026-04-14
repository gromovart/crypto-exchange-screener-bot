// internal/delivery/telegram/app/bot/handlers/callbacks/watchlist_search/handler.go
// Устанавливает состояние FSM "watchlist_search", чтобы следующее текстовое
// сообщение пользователя было интерпретировано как поисковый запрос.
package watchlist_search

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// StateWatchlistSearch — ключ FSM-состояния поиска монет
const StateWatchlistSearch = "watchlist_search"

type watchlistSearchHandler struct {
	*base.BaseHandler
	userService *users.Service
}

// NewHandler создаёт обработчик инициации поиска по вотчлисту
func NewHandler(userService *users.Service) handlers.Handler {
	return &watchlistSearchHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_search_handler",
			Command: constants.CallbackWatchlistSearch,
			Type:    handlers.TypeCallback,
		},
		userService: userService,
	}
}

func (h *watchlistSearchHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	// Устанавливаем состояние поиска
	if err := h.userService.SetUserState(params.User.ID, StateWatchlistSearch); err != nil {
		return handlers.HandlerResult{}, err
	}

	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{{"text": "🔙 Отмена", "callback_data": constants.CallbackWatchlistMenu}},
		},
	}
	return handlers.HandlerResult{
		Message:  "🔍 Введите название монеты (например: BTC или USDT).\n\nПоиск ведётся по вхождению в тикер.",
		Keyboard: keyboard,
	}, nil
}
