// internal/delivery/telegram/app/bot/handlers/callbacks/watchlist_toggle/handler.go
// Обрабатывает нажатие кнопки монеты (watchlist_toggle:{SYMBOL}) и
// показ страницы монет по букве (watchlist_letter:{LETTER}:{PAGE}).
package watchlist_toggle

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

const pageSize = 20

type watchlistToggleHandler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

// NewHandler создаёт обработчик toggle + letter filter
func NewHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &watchlistToggleHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_toggle_handler",
			Command: constants.CallbackWatchlistTogglePrefix + "*",
			Type:    handlers.TypeCallback,
		},
		watchlistService: watchlistService,
	}
}

// NewSearchResultHandler создаёт обработчик для отображения результатов поиска.
// Вызывается из bot.go когда пользователь находится в состоянии watchlist_search.
func NewSearchResultHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &watchlistToggleHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_search_result_handler",
			Command: "watchlist_search_result",
			Type:    handlers.TypeMessage,
		},
		watchlistService: watchlistService,
	}
}

// ExecuteSearch выполняет поиск по запросу и возвращает страницу результатов
func ExecuteSearch(watchlistService watchlistSvc.Service, userID int, query string) (handlers.HandlerResult, error) {
	h := &watchlistToggleHandler{watchlistService: watchlistService}
	results := watchlistService.SearchSymbols(query)

	watchlist, err := watchlistService.GetUserWatchlist(userID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}
	inWatchlist := make(map[string]bool, len(watchlist))
	for _, s := range watchlist {
		inWatchlist[s] = true
	}

	if len(results) == 0 {
		return handlers.HandlerResult{
			Message: "❌ Монеты по запросу *" + query + "* не найдены.\n\nПопробуйте другой запрос.",
			Keyboard: map[string]interface{}{
				"inline_keyboard": [][]map[string]string{
					{{"text": "🔍 Новый поиск", "callback_data": constants.CallbackWatchlistSearch}},
					{{"text": "🔙 Назад", "callback_data": constants.CallbackWatchlistMenu}},
				},
			},
		}, nil
	}

	// letter = "s"+query кодирует контекст поиска в callback-данных,
	// чтобы пагинация и toggle возвращали результаты поиска, а не все монеты.
	letter := "s" + strings.ToUpper(strings.TrimSpace(query))
	items, totalPages := watchlistService.PageSymbols(results, 0, pageSize)

	return handlers.HandlerResult{
		Message:  fmt.Sprintf("🔍 Результаты поиска *%s*: найдено %d монет", query, len(results)),
		Keyboard: h.buildSymbolKeyboard(items, inWatchlist, letter, 0, totalPages),
	}, nil
}

// NewLetterHandler создаёт обработчик буквенного фильтра
func NewLetterHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &watchlistToggleHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "watchlist_letter_handler",
			Command: constants.CallbackWatchlistLetterPrefix + "*",
			Type:    handlers.TypeCallback,
		},
		watchlistService: watchlistService,
	}
}

func (h *watchlistToggleHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	data := params.Data
	userID := params.User.ID

	switch {
	case strings.HasPrefix(data, constants.CallbackWatchlistTogglePrefix):
		return h.handleToggle(userID, data)
	case strings.HasPrefix(data, constants.CallbackWatchlistLetterPrefix):
		return h.handleLetterPage(userID, data)
	}
	return handlers.HandlerResult{Message: "Неизвестная операция"}, nil
}

// handleToggle переключает символ в вотчлисте и обновляет страницу
func (h *watchlistToggleHandler) handleToggle(userID int, data string) (handlers.HandlerResult, error) {
	// data = "watchlist_toggle:{SYMBOL}:{LETTER}:{PAGE}"
	parts := strings.SplitN(strings.TrimPrefix(data, constants.CallbackWatchlistTogglePrefix), ":", 3)
	if len(parts) < 1 {
		return handlers.HandlerResult{Message: "Ошибка формата"}, nil
	}
	symbol := parts[0]
	letter := ""
	page := 0
	if len(parts) >= 2 {
		letter = parts[1]
	}
	if len(parts) >= 3 {
		page, _ = strconv.Atoi(parts[2])
	}

	added, err := h.watchlistService.ToggleSymbol(userID, symbol)
	if err != nil {
		return handlers.HandlerResult{}, err
	}

	var notice string
	if added {
		notice = fmt.Sprintf("✅ %s добавлен в вотчлист", symbol)
	} else {
		notice = fmt.Sprintf("❌ %s удалён из вотчлиста", symbol)
	}

	// Перерисовываем страницу с монетами
	return h.buildLetterPage(userID, letter, page, notice)
}

// handleLetterPage отображает страницу монет на букву
func (h *watchlistToggleHandler) handleLetterPage(userID int, data string) (handlers.HandlerResult, error) {
	// data = "watchlist_letter:{LETTER}:{PAGE}"
	rest := strings.TrimPrefix(data, constants.CallbackWatchlistLetterPrefix)
	parts := strings.SplitN(rest, ":", 2)
	letter := ""
	page := 0
	if len(parts) >= 1 {
		letter = parts[0]
	}
	if len(parts) >= 2 {
		page, _ = strconv.Atoi(parts[1])
	}
	return h.buildLetterPage(userID, letter, page, "")
}

func (h *watchlistToggleHandler) buildLetterPage(userID int, letter string, page int, notice string) (handlers.HandlerResult, error) {
	var symbols []string
	var title string
	if strings.HasPrefix(letter, "s") {
		// "s"+query — контекст поиска
		query := letter[1:]
		symbols = h.watchlistService.SearchSymbols(query)
		title = fmt.Sprintf("🔍 Поиск *%s*", query)
	} else if letter != "" {
		symbols = h.watchlistService.GetSymbolsByLetter(letter)
		title = fmt.Sprintf("📋 Монеты на букву *%s*", letter)
	} else {
		symbols = h.watchlistService.GetAllSymbols()
		title = "📋 Все монеты"
	}

	items, totalPages := h.watchlistService.PageSymbols(symbols, page, pageSize)

	watchlist, err := h.watchlistService.GetUserWatchlist(userID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}
	inWatchlist := make(map[string]bool, len(watchlist))
	for _, s := range watchlist {
		inWatchlist[s] = true
	}

	var msg strings.Builder
	msg.WriteString(title)
	if notice != "" {
		msg.WriteString("\n\n" + notice)
	}
	msg.WriteString(fmt.Sprintf("\n\nСтраница %d из %d (%d монет):", page+1, totalPages, len(symbols)))

	keyboard := h.buildSymbolKeyboard(items, inWatchlist, letter, page, totalPages)
	return handlers.HandlerResult{
		Message:  msg.String(),
		Keyboard: keyboard,
	}, nil
}

func (h *watchlistToggleHandler) buildSymbolKeyboard(items []string, inWatchlist map[string]bool, letter string, page, totalPages int) interface{} {
	var rows [][]map[string]string

	// По 2 кнопки в строке
	for i := 0; i < len(items); i += 2 {
		var row []map[string]string
		for j := i; j < i+2 && j < len(items); j++ {
			sym := items[j]
			icon := "⬜️"
			if inWatchlist[sym] {
				icon = "✅"
			}
			// callback: watchlist_toggle:{SYMBOL}:{LETTER}:{PAGE}
			cb := fmt.Sprintf("%s%s:%s:%d", constants.CallbackWatchlistTogglePrefix, sym, letter, page)
			row = append(row, map[string]string{"text": icon + " " + sym, "callback_data": cb})
		}
		rows = append(rows, row)
	}

	// Пагинация
	var navRow []map[string]string
	if page > 0 {
		navRow = append(navRow, map[string]string{
			"text":          "◀️",
			"callback_data": fmt.Sprintf("%s%s:%d", constants.CallbackWatchlistLetterPrefix, letter, page-1),
		})
	}
	if page < totalPages-1 {
		navRow = append(navRow, map[string]string{
			"text":          "▶️",
			"callback_data": fmt.Sprintf("%s%s:%d", constants.CallbackWatchlistLetterPrefix, letter, page+1),
		})
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	// Назад к меню вотчлиста
	rows = append(rows, []map[string]string{
		{"text": "🔙 Назад", "callback_data": constants.CallbackWatchlistMenu},
	})

	return map[string]interface{}{"inline_keyboard": rows}
}
