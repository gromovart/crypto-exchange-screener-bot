// internal/delivery/max/bot/handlers/callbacks/watchlist_menu/handler.go
package watchlist_menu

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

// Handler — главный экран вотчлиста в MAX боте
type Handler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

// New создаёт обработчик меню вотчлиста
func New(watchlistService watchlistSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler:      base.New("watchlist_menu", kb.CbWatchlistMenu, handlers.TypeCallback),
		watchlistService: watchlistService,
	}
}

func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	userID := params.User.ID

	watchlist, err := h.watchlistService.GetUserWatchlist(userID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}

	letters := h.watchlistService.GetAvailableLetters()
	total := len(h.watchlistService.GetAllSymbols())

	var msg strings.Builder
	msg.WriteString("📋 Вотчлист монет\n\n")
	if len(watchlist) == 0 {
		msg.WriteString("Сейчас отслеживаются все монеты (вотчлист не задан).\n\n")
	} else {
		msg.WriteString(fmt.Sprintf("Вы отслеживаете: %d монет\n\n", len(watchlist)))
	}
	msg.WriteString(fmt.Sprintf("Всего доступно: %d монет\n\n", total))
	msg.WriteString("Выберите букву для фильтра или используйте поиск:")

	keyboard := buildMenuKeyboard(letters, len(watchlist))
	return handlers.HandlerResult{
		Message:     msg.String(),
		Keyboard:    keyboard,
		EditMessage: params.MessageID != "",
	}, nil
}

func buildMenuKeyboard(letters []string, watchlistLen int) interface{} {
	var rows [][]map[string]string

	rows = append(rows, []map[string]string{
		kb.B("🔍 Найти монету", kb.CbWatchlistSearch),
	})

	const lettersPerRow = 4
	for i := 0; i < len(letters); i += lettersPerRow {
		end := i + lettersPerRow
		if end > len(letters) {
			end = len(letters)
		}
		var row []map[string]string
		for _, l := range letters[i:end] {
			row = append(row, kb.B(l, "watchlist_letter_"+l+"_0"))
		}
		rows = append(rows, row)
	}

	if watchlistLen > 0 {
		rows = append(rows, []map[string]string{
			kb.B("👁 Мой вотчлист", kb.CbWatchlistView),
		})
	}
	rows = append(rows, []map[string]string{
		kb.B("➕ Добавить все монеты", kb.CbWatchlistAddAll),
	})
	if watchlistLen > 0 {
		rows = append(rows, []map[string]string{
			kb.B("🗑️ Очистить вотчлист", kb.CbWatchlistReset),
		})
	}

	rows = append(rows, kb.BackRow(kb.CbMenuMain))
	return kb.Keyboard(rows)
}
