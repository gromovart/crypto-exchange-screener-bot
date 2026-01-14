// internal/delivery/telegram/services/counter/interface.go
package counter

import (
	"crypto-exchange-screener-bot/internal/types"
)

type Service interface {
	Exec(params interface{}) (interface{}, error)
}

// CounterParams параметры для Exec
type CounterParams struct {
	Event types.Event `json:"event"`
}

// CounterResult результат Exec
type CounterResult struct {
	Processed bool   `json:"processed"`
	Message   string `json:"message,omitempty"`
}
