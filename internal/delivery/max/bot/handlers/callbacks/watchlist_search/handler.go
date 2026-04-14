// internal/delivery/max/bot/handlers/callbacks/watchlist_search/handler.go
// Устанавливает состояние FSM "watchlist_search"
package watchlist_search

import (
	"crypto-exchange-screener-bot/internal/core/domain/users"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
)

// StateWatchlistSearch — ключ FSM состояния
const StateWatchlistSearch = "watchlist_search"

// Handler устанавливает состояние поиска
type Handler struct {
	*base.BaseHandler
	userService *users.Service
}

// New создаёт обработчик инициации поиска
func New(userService *users.Service) handlers.Handler {
	return &Handler{
		BaseHandler: base.New("watchlist_search", kb.CbWatchlistSearch, handlers.TypeCallback),
		userService: userService,
	}
}

func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	if err := h.userService.SetUserState(params.User.ID, StateWatchlistSearch); err != nil {
		return handlers.HandlerResult{}, err
	}

	return handlers.HandlerResult{
		Message: "🔍 Введите название монеты (например: BTC или USDT).\n\nПоиск ведётся по вхождению в тикер.",
		Keyboard: kb.Keyboard([][]map[string]string{
			{kb.B("🔙 Отмена", kb.CbWatchlistMenu)},
		}),
		EditMessage: params.MessageID != "",
	}, nil
}
