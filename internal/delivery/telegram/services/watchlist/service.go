// internal/delivery/telegram/services/watchlist/service.go
package watchlist

import (
	"sort"
	"strings"

	"crypto-exchange-screener-bot/internal/core/domain/users"
	storage "crypto-exchange-screener-bot/internal/infrastructure/persistence/redis_storage"
)

type serviceImpl struct {
	userService        *users.Service
	priceStorageGetter func() storage.PriceStorageInterface
}

// NewService создаёт сервис управления вотчлистом.
// priceStorageGetter вызывается лениво при каждом обращении к символам,
// что позволяет создавать сервис до того как CandleSystem запущена.
func NewService(userService *users.Service, priceStorageGetter func() storage.PriceStorageInterface) Service {
	return &serviceImpl{
		userService:        userService,
		priceStorageGetter: priceStorageGetter,
	}
}

func (s *serviceImpl) ToggleSymbol(userID int, symbol string) (bool, error) {
	watchlist, err := s.GetUserWatchlist(userID)
	if err != nil {
		return false, err
	}

	// Ищем символ в списке
	for i, sym := range watchlist {
		if sym == symbol {
			// Удаляем
			updated := append(watchlist[:i], watchlist[i+1:]...)
			return false, s.userService.UpdateWatchlist(userID, updated)
		}
	}

	// Добавляем
	updated := append(watchlist, symbol)
	return true, s.userService.UpdateWatchlist(userID, updated)
}

// ClearFilter — фильтр активен, но пуст → ноль сигналов
func (s *serviceImpl) ClearFilter(userID int) error {
	return s.userService.UpdateWatchlist(userID, []string{})
}

// DisableFilter — отключает фильтр полностью (nil → все сигналы)
func (s *serviceImpl) DisableFilter(userID int) error {
	return s.userService.UpdateWatchlist(userID, nil)
}

func (s *serviceImpl) AddAllToWatchlist(userID int) error {
	all := s.GetAllSymbols()
	return s.userService.UpdateWatchlist(userID, all)
}

func (s *serviceImpl) GetUserWatchlist(userID int) ([]string, error) {
	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user.WatchlistSymbols == nil {
		return []string{}, nil
	}
	return user.WatchlistSymbols, nil
}

func (s *serviceImpl) GetAllSymbols() []string {
	if s.priceStorageGetter == nil {
		return nil
	}
	ps := s.priceStorageGetter()
	if ps == nil {
		return nil
	}
	symbols := ps.GetSymbols()
	sorted := make([]string, len(symbols))
	copy(sorted, symbols)
	sort.Strings(sorted)
	return sorted
}

func (s *serviceImpl) SearchSymbols(query string) []string {
	query = strings.ToUpper(strings.TrimSpace(query))
	if query == "" {
		return s.GetAllSymbols()
	}
	all := s.GetAllSymbols()
	var result []string
	for _, sym := range all {
		if strings.Contains(sym, query) {
			result = append(result, sym)
		}
	}
	return result
}

func (s *serviceImpl) GetSymbolsByLetter(letter string) []string {
	letter = strings.ToUpper(letter)
	all := s.GetAllSymbols()
	var result []string
	for _, sym := range all {
		if strings.HasPrefix(sym, letter) {
			result = append(result, sym)
		}
	}
	return result
}

func (s *serviceImpl) GetAvailableLetters() []string {
	all := s.GetAllSymbols()
	seen := make(map[string]bool)
	for _, sym := range all {
		if len(sym) > 0 {
			letter := string(sym[0])
			seen[letter] = true
		}
	}
	letters := make([]string, 0, len(seen))
	for l := range seen {
		letters = append(letters, l)
	}
	sort.Strings(letters)
	return letters
}

func (s *serviceImpl) PageSymbols(symbols []string, page, pageSize int) ([]string, int) {
	total := len(symbols)
	if total == 0 || pageSize <= 0 {
		return nil, 0
	}
	totalPages := (total + pageSize - 1) / pageSize
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return symbols[start:end], totalPages
}

func (s *serviceImpl) IsInWatchlist(userID int, symbol string) (bool, error) {
	watchlist, err := s.GetUserWatchlist(userID)
	if err != nil {
		return false, err
	}
	for _, sym := range watchlist {
		if sym == symbol {
			return true, nil
		}
	}
	return false, nil
}
