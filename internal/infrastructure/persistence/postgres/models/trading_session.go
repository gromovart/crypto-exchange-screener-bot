// internal/infrastructure/persistence/postgres/models/trading_session.go
package models

import "time"

// TradingSession хранит состояние торговой сессии пользователя в БД
type TradingSession struct {
	ID        int       `db:"id"         json:"id"`
	UserID    int       `db:"user_id"    json:"user_id"`
	ChatID    int64     `db:"chat_id"    json:"chat_id"`
	StartedAt time.Time `db:"started_at" json:"started_at"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	IsActive  bool      `db:"is_active"  json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
