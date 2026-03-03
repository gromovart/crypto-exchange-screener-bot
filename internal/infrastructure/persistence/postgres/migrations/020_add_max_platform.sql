-- 020_add_max_platform.sql
-- Добавляем поддержку мессенджера MAX: отдельный пользователь MAX может
-- существовать независимо или быть привязан к существующему Telegram-аккаунту.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS max_user_id          BIGINT       DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS max_chat_id          VARCHAR(100) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS link_code            VARCHAR(10)  DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS link_code_expires_at TIMESTAMPTZ  DEFAULT NULL;

-- Уникальный индекс: один MAX-аккаунт → одна запись в users
CREATE UNIQUE INDEX IF NOT EXISTS users_max_user_id_idx
    ON users (max_user_id)
    WHERE max_user_id IS NOT NULL;

-- Уникальный индекс для быстрого поиска по коду привязки
CREATE UNIQUE INDEX IF NOT EXISTS users_link_code_idx
    ON users (link_code)
    WHERE link_code IS NOT NULL;
