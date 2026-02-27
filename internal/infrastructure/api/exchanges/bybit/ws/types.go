// internal/infrastructure/api/exchanges/bybit/ws/types.go
package ws

// AllLiquidationMsg — входящее WS-сообщение нового топика allLiquidation.{symbol}
// (старый топик liquidation.{symbol} задепрекейтил Bybit)
type AllLiquidationMsg struct {
	Topic string               `json:"topic"`
	Type  string               `json:"type"`
	Ts    int64                `json:"ts"` // системный timestamp ms
	Data  []AllLiquidationData `json:"data"`
}

// AllLiquidationData — данные одной ликвидации в новом формате
// S: "Buy"  — ликвидирован лонг (Buy-позиция принудительно закрыта)
// S: "Sell" — ликвидирован шорт (Sell-позиция принудительно закрыта)
type AllLiquidationData struct {
	T      int64  `json:"T"` // timestamp ликвидации, ms
	Symbol string `json:"s"` // символ
	Side   string `json:"S"` // сторона позиции: "Buy"=лонг, "Sell"=шорт
	Size   string `json:"v"` // объём в базовой монете
	Price  string `json:"p"` // цена ликвидации
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
