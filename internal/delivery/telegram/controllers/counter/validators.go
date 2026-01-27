// internal/delivery/telegram/controllers/counter/validators.go
package counter

import (
	periodPkg "crypto-exchange-screener-bot/pkg/period"
	"fmt"
)

// ValidateEventData валидирует структуру данных события
// ValidateEventData валидирует структуру данных события
func ValidateEventData(eventData interface{}) error {
	dataMap, ok := eventData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("данные события должны быть map[string]interface{}, получен %T", eventData)
	}

	// Обязательные поля
	requiredFields := []string{"symbol", "direction", "change_percent", "period"}
	for _, field := range requiredFields {
		if _, exists := dataMap[field]; !exists {
			return fmt.Errorf("отсутствует обязательное поле: %s", field)
		}
	}

	// Проверка типов
	if symbol, ok := dataMap["symbol"].(string); !ok || symbol == "" {
		return fmt.Errorf("поле symbol должно быть непустой строкой")
	}

	if direction, ok := dataMap["direction"].(string); !ok || (direction != "growth" && direction != "fall") {
		return fmt.Errorf("поле direction должно быть 'growth' или 'fall'")
	}

	if _, ok := dataMap["change_percent"].(float64); !ok {
		return fmt.Errorf("поле change_percent должно быть числом float64")
	}

	// ИСПРАВЛЕНИЕ: Валидация формата периода
	if period, ok := dataMap["period"].(string); !ok || period == "" {
		return fmt.Errorf("поле period должно быть непустой строкой")
	} else if !periodPkg.IsValidPeriod(period) {
		return fmt.Errorf("невалидный формат периода: %s. Допустимые значения: %v",
			period, periodPkg.GetStandardPeriods())
	}

	return nil
}

// ValidateCounterParams валидирует параметры счетчика после преобразования
func ValidateCounterParams(params interface{}) error {
	counterParams, ok := params.(map[string]interface{})
	if !ok {
		return fmt.Errorf("параметры должны быть map[string]interface{}")
	}

	// Проверка обязательных полей после преобразования
	if symbol, ok := counterParams["symbol"].(string); !ok || symbol == "" {
		return fmt.Errorf("невалидный символ: %v", counterParams["symbol"])
	}

	if _, ok := counterParams["change_percent"].(float64); !ok {
		return fmt.Errorf("невалидное изменение: %v", counterParams["change_percent"])
	}

	return nil
}
