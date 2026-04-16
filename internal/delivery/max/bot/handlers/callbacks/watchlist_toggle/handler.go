// internal/delivery/max/bot/handlers/callbacks/watchlist_toggle/handler.go
// Обрабатывает: watchlist_toggle_{SYMBOL}_{LETTER}_{PAGE}  и  watchlist_letter_{LETTER}_{PAGE}
package watchlist_toggle

import (
	"fmt"
	"strconv"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/max/bot/handlers/base"
	kb "crypto-exchange-screener-bot/internal/delivery/max/bot/keyboard"
	watchlistSvc "crypto-exchange-screener-bot/internal/delivery/telegram/services/watchlist"
)

const pageSize = 20

// Handler обрабатывает toggle и letter-page callbacks для MAX бота
type Handler struct {
	*base.BaseHandler
	watchlistService watchlistSvc.Service
}

// New создаёт обработчик watchlist_toggle_{SYMBOL}
func New(watchlistService watchlistSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler:      base.New("watchlist_toggle", kb.CbWatchlistToggleWildcard, handlers.TypeCallback),
		watchlistService: watchlistService,
	}
}

// NewLetterHandler создаёт обработчик watchlist_letter_{LETTER}
func NewLetterHandler(watchlistService watchlistSvc.Service) handlers.Handler {
	return &Handler{
		BaseHandler:      base.New("watchlist_letter", kb.CbWatchlistLetterWildcard, handlers.TypeCallback),
		watchlistService: watchlistService,
	}
}

// ExecuteSearch выполняет поиск и возвращает результаты (вызывается из bot.go)
func ExecuteSearch(watchlistService watchlistSvc.Service, userID int, query string) (handlers.HandlerResult, error) {
	h := &Handler{watchlistService: watchlistService}
	results := watchlistService.SearchSymbols(query)

	filterDisabled, err := watchlistService.IsFilterDisabled(userID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}
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
			Message: fmt.Sprintf("❌ Монеты по запросу «%s» не найдены.\n\nПопробуйте другой запрос.", query),
			Keyboard: kb.Keyboard([][]map[string]string{
				{kb.B("🔍 Новый поиск", kb.CbWatchlistSearch)},
				kb.BackRow(kb.CbWatchlistMenu),
			}),
		}, nil
	}

	// letter = "s"+query кодирует контекст поиска в callback-данных
	letter := "s" + strings.ToUpper(strings.TrimSpace(query))
	items, totalPages := watchlistService.PageSymbols(results, 0, pageSize)
	return handlers.HandlerResult{
		Message:  fmt.Sprintf("🔍 Результаты поиска «%s»: найдено %d монет", query, len(results)),
		Keyboard: h.buildSymbolKeyboard(items, inWatchlist, filterDisabled, letter, 0, totalPages),
	}, nil
}

func (h *Handler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	data := params.Data
	userID := params.User.ID

	switch {
	case strings.HasPrefix(data, "watchlist_toggle_"):
		return h.handleToggle(userID, data, params.MessageID)
	case strings.HasPrefix(data, "watchlist_letter_"):
		return h.handleLetterPage(userID, data, params.MessageID)
	}
	return handlers.HandlerResult{Message: "Неизвестная операция"}, nil
}

// watchlist_toggle_{SYMBOL}_{LETTER}_{PAGE}
func (h *Handler) handleToggle(userID int, data string, messageID string) (handlers.HandlerResult, error) {
	rest := strings.TrimPrefix(data, "watchlist_toggle_")
	// Последние два сегмента — letter и page; символ может содержать буквы и цифры
	parts := strings.Split(rest, "_")
	if len(parts) < 3 {
		// Fallback: всё — символ, нет letter/page
		symbol := rest
		_, err := h.watchlistService.ToggleSymbol(userID, symbol)
		if err != nil {
			return handlers.HandlerResult{}, err
		}
		return h.buildLetterPage(userID, "", 0, messageID)
	}

	page, _ := strconv.Atoi(parts[len(parts)-1])
	letter := parts[len(parts)-2]
	symbol := strings.Join(parts[:len(parts)-2], "_")

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
	_ = notice // встроим в заголовок страницы

	return h.buildLetterPage(userID, letter, page, messageID)
}

// watchlist_letter_{LETTER}_{PAGE}
func (h *Handler) handleLetterPage(userID int, data string, messageID string) (handlers.HandlerResult, error) {
	rest := strings.TrimPrefix(data, "watchlist_letter_")
	parts := strings.Split(rest, "_")
	letter := ""
	page := 0
	if len(parts) >= 1 {
		letter = parts[0]
	}
	if len(parts) >= 2 {
		page, _ = strconv.Atoi(parts[1])
	}
	return h.buildLetterPage(userID, letter, page, messageID)
}

func (h *Handler) buildLetterPage(userID int, letter string, page int, messageID string) (handlers.HandlerResult, error) {
	var symbols []string
	var title string
	if strings.HasPrefix(letter, "s") {
		// "s"+query — контекст поиска
		query := letter[1:]
		symbols = h.watchlistService.SearchSymbols(query)
		title = fmt.Sprintf("🔍 Поиск «%s» (стр. %%d/%%d):", query)
	} else if letter != "" {
		symbols = h.watchlistService.GetSymbolsByLetter(letter)
		title = fmt.Sprintf("📋 Монеты на букву %s (стр. %%d/%%d):", letter)
	} else {
		symbols = h.watchlistService.GetAllSymbols()
		title = "📋 Все монеты (стр. %%d/%%d):"
	}

	items, totalPages := h.watchlistService.PageSymbols(symbols, page, pageSize)
	title = fmt.Sprintf(title, page+1, totalPages)

	filterDisabled, err := h.watchlistService.IsFilterDisabled(userID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}
	watchlist, err := h.watchlistService.GetUserWatchlist(userID)
	if err != nil {
		return handlers.HandlerResult{}, err
	}
	inWatchlist := make(map[string]bool, len(watchlist))
	for _, s := range watchlist {
		inWatchlist[s] = true
	}

	return handlers.HandlerResult{
		Message:     title,
		Keyboard:    h.buildSymbolKeyboard(items, inWatchlist, filterDisabled, letter, page, totalPages),
		EditMessage: messageID != "",
	}, nil
}

func (h *Handler) buildSymbolKeyboard(items []string, inWatchlist map[string]bool, filterDisabled bool, letter string, page, totalPages int) interface{} {
	var rows [][]map[string]string

	for i := 0; i < len(items); i += 2 {
		var row []map[string]string
		for j := i; j < i+2 && j < len(items); j++ {
			sym := items[j]
			icon := "❌"
			// filterDisabled (nil) → все монеты отслеживаются → ✅
			if filterDisabled || inWatchlist[sym] {
				icon = "✅"
			}
			cb := fmt.Sprintf("watchlist_toggle_%s_%s_%d", sym, letter, page)
			row = append(row, kb.B(icon+" "+sym, cb))
		}
		rows = append(rows, row)
	}

	// Пагинация
	var navRow []map[string]string
	if page > 0 {
		navRow = append(navRow, kb.B("◀️", fmt.Sprintf("watchlist_letter_%s_%d", letter, page-1)))
	}
	if page < totalPages-1 {
		navRow = append(navRow, kb.B("▶️", fmt.Sprintf("watchlist_letter_%s_%d", letter, page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, []map[string]string{kb.B("🔙 Назад", kb.CbWatchlistMenu)})
	return kb.Keyboard(rows)
}
