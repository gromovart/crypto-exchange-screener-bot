// internal/delivery/telegram/app/bot/handlers/callbacks/watchlist_menu/handler.go
package watchlist_menu

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

const pageSize = 20

type watchlistMenuHandler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

// NewHandler создаёт обработчик главного экрана вотчлиста
func NewHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &watchlistMenuHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_menu_handler",
			Command: constants.CallbackWatchlistMenu,
			Type:    handlers.TypeCallback,
		},
		watchlistService: watchlistService,
	}
}

// Execute отображает главное меню вотчлиста с буквенным фильтром
func (h *watchlistMenuHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	userID := params.User.ID

	watchlist, err := h.watchlistService.GetUserWatchlist(userID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}

	letters := h.watchlistService.GetAvailableLetters()
	total := len(h.watchlistService.GetAllSymbols())

	var msg strings.Builder
	msg.WriteString("📋 *Вотчлист монет*\n\n")
	if len(watchlist) == 0 {
		msg.WriteString("Сейчас отслеживаются *все монеты* (вотчлист не задан).\n\n")
	} else {
		msg.WriteString(fmt.Sprintf("Вы отслеживаете: *%d монет*\n\n", len(watchlist)))
	}
	msg.WriteString(fmt.Sprintf("Всего доступно: %d монет\n\n", total))
	msg.WriteString("Выберите букву для фильтра или воспользуйтесь поиском:")

	keyboard := h.buildKeyboard(letters, len(watchlist))
	return handlers.HandlerResult{
		Message:  msg.String(),
		Keyboard: keyboard,
	}, nil
}

func (h *watchlistMenuHandler) buildKeyboard(letters []string, watchlistLen int) interface{} {
	var rows [][]map[string]string

	// Кнопка поиска
	rows = append(rows, []map[string]string{
		{"text": "🔍 Поиск по названию", "callback_data": constants.CallbackWatchlistSearch},
	})

	// Буквы по 8 в строке
	const lettersPerRow = 8
	for i := 0; i < len(letters); i += lettersPerRow {
		end := i + lettersPerRow
		if end > len(letters) {
			end = len(letters)
		}
		var row []map[string]string
		for _, l := range letters[i:end] {
			row = append(row, map[string]string{
				"text":          l,
				"callback_data": constants.CallbackWatchlistLetterPrefix + l + ":0",
			})
		}
		rows = append(rows, row)
	}

	// Кнопка сброса (только если вотчлист задан)
	if watchlistLen > 0 {
		rows = append(rows, []map[string]string{
			{"text": "🗑️ Сбросить вотчлист (все монеты)", "callback_data": constants.CallbackWatchlistReset},
		})
	}

	// Назад
	rows = append(rows, []map[string]string{
		{"text": constants.ButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
	})

	return map[string]interface{}{"inline_keyboard": rows}
}
