-- Добавляем поле platform для разделения сессий по мессенджерам
ALTER TABLE trading_sessions
    ADD COLUMN IF NOT EXISTS platform VARCHAR(20) NOT NULL DEFAULT 'telegram';

-- Удаляем старый уникальный индекс (один активный на пользователя)
DROP INDEX IF EXISTS idx_trading_sessions_active_user;

-- Новый уникальный индекс: один активный на пользователя × платформу
-- Можно иметь одновременно активную сессию в Telegram и в MAX
CREATE UNIQUE INDEX IF NOT EXISTS idx_trading_sessions_active_user_platform
    ON trading_sessions (user_id, platform) WHERE is_active = TRUE;
