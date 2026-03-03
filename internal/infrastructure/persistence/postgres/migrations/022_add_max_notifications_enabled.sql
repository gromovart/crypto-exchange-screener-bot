-- Флаг уведомлений, специфичный для MAX-мессенджера.
-- Независим от notifications_enabled (Telegram).
-- Включается при старте торговой сессии в MAX.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS max_notifications_enabled BOOLEAN NOT NULL DEFAULT FALSE;
