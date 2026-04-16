// internal/delivery/telegram/services/watchlist/interface.go
package watchlist

// Service интерфейс сервиса управления вотчлистом
type Service interface {
	// ToggleSymbol добавляет/убирает символ из вотчлиста.
	// Возвращает true если символ был добавлен, false если удалён.
	ToggleSymbol(userID int, symbol string) (bool, error)

	// ClearFilter очищает список монет ([] = фильтр активен, но пуст → ноль сигналов)
	ClearFilter(userID int) error

	// DisableFilter отключает фильтр полностью (nil → все сигналы, как по умолчанию)
	DisableFilter(userID int) error

	// AddAllToWatchlist добавляет все доступные монеты явным списком
	AddAllToWatchlist(userID int) error

	// GetUserWatchlist возвращает текущий вотчлист пользователя
	GetUserWatchlist(userID int) ([]string, error)

	// IsFilterDisabled возвращает true если фильтр отключён (nil → все сигналы)
	IsFilterDisabled(userID int) (bool, error)

	// GetAllSymbols возвращает все доступные символы
	GetAllSymbols() []string

	// SearchSymbols фильтрует символы по подстроке (case-insensitive)
	SearchSymbols(query string) []string

	// GetSymbolsByLetter возвращает символы начинающиеся на букву
	GetSymbolsByLetter(letter string) []string

	// GetAvailableLetters возвращает буквы, на которые начинаются символы
	GetAvailableLetters() []string

	// PageSymbols разбивает список символов на страницы
	PageSymbols(symbols []string, page, pageSize int) (items []string, totalPages int)

	// IsInWatchlist проверяет, есть ли символ в вотчлисте пользователя
	IsInWatchlist(userID int, symbol string) (bool, error)
}
