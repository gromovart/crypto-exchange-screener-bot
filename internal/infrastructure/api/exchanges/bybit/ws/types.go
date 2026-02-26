// internal/infrastructure/api/exchanges/bybit/ws/types.go
package ws

// LiquidationMsg — входящее WS-сообщение с данными ликвидации
type LiquidationMsg struct {
	Topic string          `json:"topic"`
	Type  string          `json:"type"`
	Data  LiquidationData `json:"data"`
}

// LiquidationData — данные одной ликвидации
// Side: "Buy"  — ликвидирован шорт (принудительная покупка)
// Side: "Sell" — ликвидирован лонг (принудительная продажа)
type LiquidationData struct {
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	Size        string `json:"size"`        // объём в базовой монете
	Price       string `json:"price"`       // цена исполнения
	UpdatedTime int64  `json:"updatedTime"` // Unix ms
}

// wsSubscribeMsg — исходящее сообщение подписки
type wsSubscribeMsg struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}

// wsPingMsg — heartbeat-пинг
type wsPingMsg struct {
	Op string `json:"op"`
}

// wsResponseMsg — ответ сервера (op: "pong" / "subscribe")
type wsResponseMsg struct {
	Op      string `json:"op"`
	ConnID  string `json:"conn_id,omitempty"`
	ReqID   string `json:"req_id,omitempty"`
	Success bool   `json:"success,omitempty"`
	RetMsg  string `json:"ret_msg,omitempty"`
}
