-- persistence/postgres/migrations/001_create_users.sql
CREATE TABLE users
(
    -- Основная информация
    id           SERIAL PRIMARY KEY,
    telegram_id  BIGINT UNIQUE                NOT NULL,
    username     VARCHAR(100),
    first_name   VARCHAR(100)                 NOT NULL,
    last_name    VARCHAR(100),
    chat_id      VARCHAR(100) UNIQUE          NOT NULL,

    -- Контактная информация
    email        VARCHAR(255),
    phone        VARCHAR(50),

    -- Настройки уведомлений (хранятся в отдельных полях для упрощения запросов)
    notifications_enabled BOOLEAN             DEFAULT TRUE,
    notify_growth         BOOLEAN             DEFAULT TRUE,
    notify_fall           BOOLEAN             DEFAULT TRUE,
    notify_continuous     BOOLEAN             DEFAULT TRUE,
    quiet_hours_start     INTEGER             DEFAULT 23,
    quiet_hours_end       INTEGER             DEFAULT 8,

    -- Настройки анализа
    min_growth_threshold  DECIMAL(5, 2)       DEFAULT 2.00,
    min_fall_threshold    DECIMAL(5, 2)       DEFAULT 2.00,
    preferred_periods     INTEGER[]           DEFAULT '{5,15,30}',
    min_volume_filter     DECIMAL(15, 2)      DEFAULT 100000.00,
    exclude_patterns      VARCHAR(100)[]      DEFAULT '{}',

    -- Настройки отображения
    language              VARCHAR(10)         DEFAULT 'ru',
    timezone              VARCHAR(50)         DEFAULT 'Europe/Moscow',
    display_mode          VARCHAR(20)         DEFAULT 'compact',

    -- Статус и лимиты
    role                  VARCHAR(20)         DEFAULT 'user',
    is_active             BOOLEAN             DEFAULT TRUE,
    is_verified           BOOLEAN             DEFAULT FALSE,
    subscription_tier     VARCHAR(20)         DEFAULT 'free',
    signals_today         INTEGER             DEFAULT 0,
    max_signals_per_day   INTEGER             DEFAULT 50,

    -- Временные метки
    created_at            TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at            TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login_at         TIMESTAMP WITH TIME ZONE,
    last_signal_at        TIMESTAMP WITH TIME ZONE,

    -- Ограничения
    CONSTRAINT users_telegram_id_key UNIQUE (telegram_id),
    CONSTRAINT users_chat_id_key UNIQUE (chat_id),
    CONSTRAINT valid_role CHECK (role IN ('user', 'premium', 'admin')),
    CONSTRAINT valid_tier CHECK (subscription_tier IN ('free', 'basic', 'pro')),
    CONSTRAINT valid_quiet_hours CHECK (quiet_hours_start >= 0 AND quiet_hours_start <= 23 AND
                                        quiet_hours_end >= 0 AND quiet_hours_end <= 23),
    CONSTRAINT valid_thresholds CHECK (min_growth_threshold >= 0 AND min_fall_threshold >= 0)
);

-- Индексы для ускорения запросов
CREATE INDEX idx_users_telegram_id ON users (telegram_id);
CREATE INDEX idx_users_chat_id ON users (chat_id);
CREATE INDEX idx_users_is_active ON users (is_active);
CREATE INDEX idx_users_role ON users (role);
CREATE INDEX idx_users_subscription_tier ON users (subscription_tier);
CREATE INDEX idx_users_created_at ON users (created_at);
CREATE INDEX idx_users_last_login_at ON users (last_login_at);

-- Функция и триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE
    ON users
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Функция для сброса дневных счетчиков
CREATE OR REPLACE FUNCTION reset_daily_signals_counters()
    RETURNS VOID AS
$$
BEGIN
    UPDATE users
    SET signals_today = 0
    WHERE signals_today > 0;
END;
$$ language 'plpgsql';

-- Функция для получения профиля пользователя
CREATE OR REPLACE FUNCTION get_user_profile(p_user_id INTEGER)
    RETURNS TABLE
            (
                user_id            INTEGER,
                telegram_id        BIGINT,
                username           VARCHAR,
                first_name         VARCHAR,
                last_name          VARCHAR,
                email              VARCHAR,
                role               VARCHAR,
                subscription_tier  VARCHAR,
                is_active          BOOLEAN,
                is_verified        BOOLEAN,
                signals_today      INTEGER,
                max_signals_per_day INTEGER,
                created_at         TIMESTAMP,
                last_login_at      TIMESTAMP
            )
AS
$$
BEGIN
    RETURN QUERY
        SELECT u.id,
               u.telegram_id,
               u.username,
               u.first_name,
               u.last_name,
               u.email,
               u.role,
               u.subscription_tier,
               u.is_active,
               u.is_verified,
               u.signals_today,
               u.max_signals_per_day,
               u.created_at,
               u.last_login_at
        FROM users u
        WHERE u.id = p_user_id;
END;
$$ language 'plpgsql';

-- Функция для проверки доступности уведомлений
CREATE OR REPLACE FUNCTION can_receive_notifications(p_user_id INTEGER)
    RETURNS BOOLEAN AS
$$
DECLARE
    v_is_active BOOLEAN;
    v_notifications_enabled BOOLEAN;
BEGIN
    SELECT is_active, notifications_enabled
    INTO v_is_active, v_notifications_enabled
    FROM users
    WHERE id = p_user_id;

    RETURN COALESCE(v_is_active, FALSE) AND COALESCE(v_notifications_enabled, FALSE);
END;
$$ language 'plpgsql';

-- Функция для проверки тихих часов
CREATE OR REPLACE FUNCTION is_in_quiet_hours(p_user_id INTEGER, p_check_hour INTEGER DEFAULT NULL)
    RETURNS BOOLEAN AS
$$
DECLARE
    v_quiet_start INTEGER;
    v_quiet_end   INTEGER;
    v_current_hour INTEGER;
BEGIN
    -- Если час не указан, используем текущий
    IF p_check_hour IS NULL THEN
        v_current_hour := EXTRACT(HOUR FROM NOW() AT TIME ZONE 'UTC');
    ELSE
        v_current_hour := p_check_hour;
    END IF;

    -- Получаем настройки тихих часов пользователя
    SELECT quiet_hours_start, quiet_hours_end
    INTO v_quiet_start, v_quiet_end
    FROM users
    WHERE id = p_user_id;

    -- Если тихие часы не настроены
    IF v_quiet_start = 0 AND v_quiet_end = 0 THEN
        RETURN FALSE;
    END IF;

    -- Обработка случая когда start > end (например, 23-8)
    IF v_quiet_start > v_quiet_end THEN
        RETURN v_current_hour >= v_quiet_start OR v_current_hour < v_quiet_end;
    END IF;

    RETURN v_current_hour >= v_quiet_start AND v_current_hour < v_quiet_end;
END;
$$ language 'plpgsql';

-- Функция для проверки достижения дневного лимита
CREATE OR REPLACE FUNCTION has_reached_daily_limit(p_user_id INTEGER)
    RETURNS BOOLEAN AS
$$
DECLARE
    v_signals_today     INTEGER;
    v_max_signals_per_day INTEGER;
BEGIN
    SELECT signals_today, max_signals_per_day
    INTO v_signals_today, v_max_signals_per_day
    FROM users
    WHERE id = p_user_id;

    RETURN COALESCE(v_signals_today, 0) >= COALESCE(v_max_signals_per_day, 50);
END;
$$ language 'plpgsql';

-- Комментарии к таблице и полям
COMMENT ON TABLE users IS 'Таблица пользователей системы';

COMMENT ON COLUMN users.telegram_id IS 'ID пользователя в Telegram';
COMMENT ON COLUMN users.chat_id IS 'ID чата Telegram';
COMMENT ON COLUMN users.notifications_enabled IS 'Включены ли уведомления';
COMMENT ON COLUMN users.quiet_hours_start IS 'Начало тихих часов (0-23)';
COMMENT ON COLUMN users.quiet_hours_end IS 'Конец тихих часов (0-23)';
COMMENT ON COLUMN users.min_growth_threshold IS 'Минимальный порог роста для уведомления (%)';
COMMENT ON COLUMN users.min_fall_threshold IS 'Минимальный порог падения для уведомления (%)';
COMMENT ON COLUMN users.preferred_periods IS 'Предпочтительные периоды анализа в минутах';
COMMENT ON COLUMN users.min_volume_filter IS 'Минимальный объем фильтрации (USDT)';
COMMENT ON COLUMN users.exclude_patterns IS 'Паттерны исключения символов';
COMMENT ON COLUMN users.signals_today IS 'Количество полученных сигналов за сегодня';
COMMENT ON COLUMN users.max_signals_per_day IS 'Максимальное количество сигналов в день';

-- Предварительное заполнение администратора (опционально)
-- INSERT INTO users (telegram_id, username, first_name, chat_id, role, subscription_tier, max_signals_per_day)
-- VALUES (123456789, 'admin_username', 'Admin', '123456789', 'admin', 'pro', 1000)
-- ON CONFLICT (telegram_id) DO NOTHING;