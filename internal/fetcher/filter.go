package fetcher

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/internal/types"
	"strings"
	"time"
)

// SignalFilter - фильтр сигналов
type SignalFilter struct {
	config        *config.Config
	signalsPerMin map[string]int
	lastReset     time.Time
}

// NewSignalFilter создает новый фильтр сигналов
func NewSignalFilter(cfg *config.Config) *SignalFilter {
	return &SignalFilter{
		config:        cfg,
		signalsPerMin: make(map[string]int),
		lastReset:     time.Now(),
	}
}

// ApplyFilters применяет все фильтры к сигналу
func (sf *SignalFilter) ApplyFilters(signal types.GrowthSignal) bool {
	// Сбрасываем счетчик если прошла минута
	if time.Since(sf.lastReset) > time.Minute {
		sf.signalsPerMin = make(map[string]int)
		sf.lastReset = time.Now()
	}

	// Проверяем минимальную уверенность
	if signal.Confidence < sf.config.SignalFilters.MinConfidence {
		return false
	}

	// Проверяем фильтр по имени символа
	if !sf.checkSymbolFilter(signal.Symbol) {
		return false
	}

	// Проверяем паттерны включения
	if len(sf.config.SignalFilters.IncludePatterns) > 0 {
		if !sf.matchesPatterns(signal.Symbol, sf.config.SignalFilters.IncludePatterns) {
			return false
		}
	}

	// Проверяем паттерны исключения
	if len(sf.config.SignalFilters.ExcludePatterns) > 0 {
		if sf.matchesPatterns(signal.Symbol, sf.config.SignalFilters.ExcludePatterns) {
			return false
		}
	}

	// Проверяем ограничение по количеству сигналов в минуту
	if sf.config.SignalFilters.MaxSignalsPerMin > 0 {
		key := signal.Symbol + "_" + signal.Direction
		if sf.signalsPerMin[key] >= sf.config.SignalFilters.MaxSignalsPerMin {
			return false
		}
		sf.signalsPerMin[key]++
	}

	return true
}

// checkSymbolFilter проверяет фильтр символов
func (sf *SignalFilter) checkSymbolFilter(symbol string) bool {
	// Если фильтр символов не задан или установлен в "all", пропускаем все
	if sf.config.SymbolFilter == "" || strings.ToUpper(strings.TrimSpace(sf.config.SymbolFilter)) == "ALL" {
		return true
	}

	// Преобразуем символ к формату без USDT для сравнения
	baseSymbol := strings.TrimSuffix(strings.ToUpper(symbol), "USDT")
	filter := strings.ToUpper(sf.config.SymbolFilter)

	// Разбиваем фильтр на части
	filterParts := strings.Split(filter, ",")

	// Проверяем совпадение с базовым символом
	for _, part := range filterParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Если часть фильтра - это "all"
		if part == "ALL" {
			return true
		}

		// Если часть фильтра - это полный символ (с USDT)
		if strings.HasSuffix(part, "USDT") && part == strings.ToUpper(symbol) {
			return true
		}

		// Если часть фильтра - это базовый символ (без USDT)
		if !strings.HasSuffix(part, "USDT") && part == baseSymbol {
			return true
		}
	}

	// Также проверяем, содержит ли фильтр символ целиком (для простых случаев)
	if strings.Contains(filter, baseSymbol) || strings.Contains(filter, strings.ToUpper(symbol)) {
		return true
	}

	return false
}

// matchesPatterns проверяет соответствие паттернам
func (sf *SignalFilter) matchesPatterns(symbol string, patterns []string) bool {
	for _, pattern := range patterns {
		// Поддерживаем wildcard *
		if strings.Contains(pattern, "*") {
			// Заменяем * на .* для regex-like сопоставления
			pattern := strings.Replace(pattern, "*", ".*", -1)
			if strings.HasPrefix(pattern, ".*") {
				if strings.HasSuffix(symbol, pattern[2:]) {
					return true
				}
			} else if strings.HasSuffix(pattern, ".*") {
				if strings.HasPrefix(symbol, pattern[:len(pattern)-2]) {
					return true
				}
			}
		} else {
			// Точное совпадение
			if strings.EqualFold(symbol, pattern) {
				return true
			}
		}
	}
	return false
}
