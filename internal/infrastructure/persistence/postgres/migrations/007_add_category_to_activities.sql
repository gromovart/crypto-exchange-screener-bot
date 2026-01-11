-- 007_add_category_to_activities.sql
-- Добавляем колонку category в таблицу user_activities

-- Шаг 1: Добавляем колонку category (допускаем NULL для существующих записей)
ALTER TABLE user_activities ADD COLUMN category VARCHAR(50);

-- Шаг 2: Обновляем существующие записи с категориями по умолчанию на основе activity_type
UPDATE user_activities SET category =
    CASE
        WHEN activity_type IN ('user_login', 'user_logout') THEN 'authentication'
        WHEN activity_type LIKE 'profile%' OR activity_type LIKE 'settings%' THEN 'user_actions'
        WHEN activity_type LIKE 'signal%' THEN 'trading'
        WHEN activity_type = 'api_call' THEN 'analytics'
        WHEN activity_type = 'security_event' THEN 'security'
        WHEN activity_type = 'system_event' THEN 'system'
        WHEN activity_type = 'subscription_event' THEN 'billing'
        WHEN activity_type = 'error_occurred' THEN 'system'
        ELSE 'user_actions'
    END;

-- Шаг 3: Создаем индекс для ускорения поиска по категории
CREATE INDEX idx_user_activities_category ON user_activities(category);

-- Шаг 4: Обновляем представления, которые используют user_activities
DROP VIEW IF EXISTS daily_activity_report;
CREATE VIEW daily_activity_report AS
SELECT
    DATE(ua.created_at) as activity_date,
    ua.category,
    COUNT(*) as total_activities,
    COUNT(DISTINCT ua.user_id) as active_users,
    COUNT(CASE WHEN ua.activity_type LIKE 'login%' THEN 1 END) as login_count,
    COUNT(CASE WHEN ua.activity_type LIKE 'signal%' THEN 1 END) as signal_count,
    COUNT(CASE WHEN ua.severity = 'error' THEN 1 END) as error_count,
    COUNT(CASE WHEN ua.severity = 'security' THEN 1 END) as security_count,
    ARRAY_AGG(DISTINCT ua.activity_type) as activity_types
FROM user_activities ua
GROUP BY DATE(ua.created_at), ua.category
ORDER BY activity_date DESC, category;

-- Шаг 5: Обновляем функцию log_user_activity
CREATE OR REPLACE FUNCTION log_user_activity(
    p_user_id INTEGER,
    p_activity_type VARCHAR,
    p_category VARCHAR DEFAULT NULL,
    p_entity_type VARCHAR DEFAULT NULL,
    p_entity_id INTEGER DEFAULT NULL,
    p_details JSONB DEFAULT '{}',
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_severity VARCHAR DEFAULT 'info'
) RETURNS INTEGER AS $$
DECLARE
    v_activity_id INTEGER;
    v_location VARCHAR;
    v_category_to_use VARCHAR;
BEGIN
    -- Определяем категорию если не указана
    IF p_category IS NULL THEN
        v_category_to_use :=
            CASE
                WHEN p_activity_type IN ('user_login', 'user_logout') THEN 'authentication'
                WHEN p_activity_type LIKE 'profile%' OR p_activity_type LIKE 'settings%' THEN 'user_actions'
                WHEN p_activity_type LIKE 'signal%' THEN 'trading'
                WHEN p_activity_type = 'api_call' THEN 'analytics'
                WHEN p_activity_type = 'security_event' THEN 'security'
                WHEN p_activity_type = 'system_event' THEN 'system'
                WHEN p_activity_type = 'subscription_event' THEN 'billing'
                WHEN p_activity_type = 'error_occurred' THEN 'system'
                ELSE 'user_actions'
            END;
    ELSE
        v_category_to_use := p_category;
    END IF;

    -- Определяем локацию по IP
    IF p_ip_address IS NOT NULL THEN
        SELECT city || ', ' || country_name INTO v_location
        FROM activity_geolocation
        WHERE ip_address = p_ip_address
        LIMIT 1;
    END IF;

    -- Логируем активность
    INSERT INTO user_activities (
        user_id, activity_type, category, entity_type, entity_id,
        details, ip_address, user_agent, location, severity
    ) VALUES (
        p_user_id, p_activity_type, v_category_to_use, p_entity_type, p_entity_id,
        p_details, p_ip_address, p_user_agent, v_location, p_severity
    ) RETURNING id INTO v_activity_id;

    -- Обновляем агрегированную статистику
    PERFORM update_activity_summary(p_user_id, p_activity_type);

    -- Обновляем геолокацию
    IF p_ip_address IS NOT NULL THEN
        INSERT INTO activity_geolocation (
            ip_address, activity_count, last_seen
        ) VALUES (
            p_ip_address, 1, NOW()
        )
        ON CONFLICT (ip_address) DO UPDATE SET
            activity_count = activity_geolocation.activity_count + 1,
            last_seen = EXCLUDED.last_seen;
    END IF;

    RETURN v_activity_id;
END;
$$ language 'plpgsql' SECURITY DEFINER;

-- Шаг 6: Добавляем комментарий к колонке
COMMENT ON COLUMN user_activities.category IS 'Категория активности: authentication, user_actions, trading, system, security, billing, analytics';

-- Шаг 7: Проверяем целостность данных
DO $$
BEGIN
    -- Проверяем что нет NULL значений в category после обновления
    IF EXISTS (SELECT 1 FROM user_activities WHERE category IS NULL) THEN
        RAISE NOTICE 'Есть записи с NULL в category, обновляем...';
        UPDATE user_activities
        SET category = 'user_actions'
        WHERE category IS NULL;
    END IF;

    -- Проверяем что индекс создан
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'user_activities'
        AND indexname = 'idx_user_activities_category'
    ) THEN
        RAISE EXCEPTION 'Индекс idx_user_activities_category не создан';
    END IF;

    RAISE NOTICE 'Миграция 007_add_category_to_activities успешно выполнена';
END $$;