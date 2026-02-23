CREATE TABLE IF NOT EXISTS trading_sessions (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id    BIGINT  NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Только одна активная сессия на пользователя
CREATE UNIQUE INDEX IF NOT EXISTS idx_trading_sessions_active_user
    ON trading_sessions (user_id) WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_trading_sessions_expires_at
    ON trading_sessions (expires_at);
