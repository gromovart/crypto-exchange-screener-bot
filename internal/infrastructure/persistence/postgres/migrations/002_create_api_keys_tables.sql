-- persistence/postgres/migrations/002_create_api_keys.sql

-- Таблица для хранения API ключей пользователей
CREATE TABLE user_api_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange VARCHAR(50) NOT NULL,
    api_key_encrypted TEXT NOT NULL,
    api_secret_encrypted TEXT NOT NULL,
    label VARCHAR(100),
    permissions JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Ограничения
    CONSTRAINT valid_exchange CHECK (exchange IN ('bybit', 'binance', 'kucoin', 'okx', 'gateio')),
    CONSTRAINT unique_user_exchange UNIQUE (user_id, exchange)
);

-- Создаем partial unique index для активных ключей
CREATE UNIQUE INDEX idx_unique_active_user_exchange
    ON user_api_keys(user_id, exchange)
    WHERE is_active = TRUE;

-- Таблица для логов использования API ключей
CREATE TABLE api_key_usage_logs (
    id SERIAL PRIMARY KEY,
    api_key_id INTEGER NOT NULL REFERENCES user_api_keys(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    endpoint VARCHAR(200),
    request_body JSONB,
    response_status INTEGER,
    response_body JSONB,
    ip_address INET,
    user_agent TEXT,
    latency_ms INTEGER,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Таблица для хранения разрешений API ключей
CREATE TABLE api_key_permissions (
    id SERIAL PRIMARY KEY,
    api_key_id INTEGER NOT NULL REFERENCES user_api_keys(id) ON DELETE CASCADE,
    permission VARCHAR(100) NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    granted_by INTEGER REFERENCES users(id),

    -- Ограничения
    CONSTRAINT unique_api_key_permission UNIQUE (api_key_id, permission)
);

-- Таблица для ротации API ключей
CREATE TABLE api_key_rotation_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange VARCHAR(50) NOT NULL,
    old_key_id INTEGER REFERENCES user_api_keys(id),
    new_key_id INTEGER NOT NULL REFERENCES user_api_keys(id),
    rotated_by INTEGER REFERENCES users(id),
    rotation_reason VARCHAR(200),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для производительности
CREATE INDEX idx_user_api_keys_user_id ON user_api_keys(user_id);
CREATE INDEX idx_user_api_keys_exchange ON user_api_keys(exchange);
CREATE INDEX idx_user_api_keys_is_active ON user_api_keys(is_active);
CREATE INDEX idx_user_api_keys_expires_at ON user_api_keys(expires_at);
CREATE INDEX idx_user_api_keys_last_used_at ON user_api_keys(last_used_at);

CREATE INDEX idx_api_key_usage_logs_api_key_id ON api_key_usage_logs(api_key_id);
CREATE INDEX idx_api_key_usage_logs_created_at ON api_key_usage_logs(created_at);
CREATE INDEX idx_api_key_usage_logs_action ON api_key_usage_logs(action);

CREATE INDEX idx_api_key_permissions_api_key_id ON api_key_permissions(api_key_id);
CREATE INDEX idx_api_key_permissions_permission ON api_key_permissions(permission);

CREATE INDEX idx_api_key_rotation_history_user_id ON api_key_rotation_history(user_id);
CREATE INDEX idx_api_key_rotation_history_exchange ON api_key_rotation_history(exchange);
CREATE INDEX idx_api_key_rotation_history_created_at ON api_key_rotation_history(created_at);

-- Триггер для обновления updated_at
CREATE TRIGGER update_user_api_keys_updated_at
    BEFORE UPDATE ON user_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Триггер для логирования изменений API ключей
CREATE OR REPLACE FUNCTION log_api_key_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        -- Логируем деактивацию ключа
        IF OLD.is_active = TRUE AND NEW.is_active = FALSE THEN
            INSERT INTO api_key_usage_logs
                (api_key_id, action, endpoint, ip_address, user_agent)
            VALUES (
                NEW.id,
                'key_deactivated',
                'system',
                '0.0.0.0',
                'system_trigger'
            );
        END IF;

        -- Логируем использование ключа
        IF NEW.last_used_at IS DISTINCT FROM OLD.last_used_at THEN
            INSERT INTO api_key_usage_logs
                (api_key_id, action, endpoint, ip_address, user_agent)
            VALUES (
                NEW.id,
                'key_used',
                'system',
                '0.0.0.0',
                'system_trigger'
            );
        END IF;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO api_key_usage_logs
            (api_key_id, action, endpoint, ip_address, user_agent)
        VALUES (
            NEW.id,
            'key_created',
            'system',
            '0.0.0.0',
            'system_trigger'
        );
    END IF;

    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER log_api_key_changes_trigger
    AFTER INSERT OR UPDATE ON user_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION log_api_key_changes();

-- Функция для шифрования API ключей
CREATE OR REPLACE FUNCTION encrypt_api_key(
    plain_text TEXT,
    encryption_key TEXT
) RETURNS TEXT AS $$
BEGIN
    -- В реальной реализации используйте pgcrypto
    -- RETURN pgp_sym_encrypt(plain_text, encryption_key);
    RETURN 'encrypted:' || plain_text; -- Заглушка для примера
END;
$$ language 'plpgsql' SECURITY DEFINER;

-- Функция для дешифрования API ключей
CREATE OR REPLACE FUNCTION decrypt_api_key(
    encrypted_text TEXT,
    encryption_key TEXT
) RETURNS TEXT AS $$
BEGIN
    -- В реальной реализации используйте pgcrypto
    -- RETURN pgp_sym_decrypt(encrypted_text, encryption_key);
    RETURN REPLACE(encrypted_text, 'encrypted:', ''); -- Заглушка для примера
END;
$$ language 'plpgsql' SECURITY DEFINER;

-- Функция для проверки разрешений API ключа
CREATE OR REPLACE FUNCTION check_api_key_permission(
    p_api_key_id INTEGER,
    p_permission VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    has_permission BOOLEAN;
BEGIN
    -- Проверяем, есть ли разрешение
    SELECT EXISTS (
        SELECT 1
        FROM api_key_permissions
        WHERE api_key_id = p_api_key_id
          AND permission = p_permission
    ) INTO has_permission;

    -- Также проверяем разрешения в JSON поле permissions
    IF NOT has_permission THEN
        SELECT (
            permissions->'allow' IS NOT NULL
            AND permissions->'allow' ? p_permission
        ) INTO has_permission
        FROM user_api_keys
        WHERE id = p_api_key_id;
    END IF;

    RETURN COALESCE(has_permission, FALSE);
END;
$$ language 'plpgsql';

-- Функция для ротации API ключей
CREATE OR REPLACE FUNCTION rotate_api_key(
    p_user_id INTEGER,
    p_exchange VARCHAR,
    p_new_api_key TEXT,
    p_new_api_secret TEXT,
    p_rotation_reason VARCHAR DEFAULT 'security_rotation'
) RETURNS INTEGER AS $$
DECLARE
    v_old_key_id INTEGER;
    v_new_key_id INTEGER;
    v_encryption_key TEXT := current_setting('app.encryption_key', TRUE);
BEGIN
    -- Получаем старый активный ключ
    SELECT id INTO v_old_key_id
    FROM user_api_keys
    WHERE user_id = p_user_id
      AND exchange = p_exchange
      AND is_active = TRUE;

    -- Деактивируем старый ключ
    IF v_old_key_id IS NOT NULL THEN
        UPDATE user_api_keys
        SET is_active = FALSE,
            updated_at = NOW()
        WHERE id = v_old_key_id;
    END IF;

    -- Создаем новый ключ
    INSERT INTO user_api_keys (
        user_id, exchange,
        api_key_encrypted, api_secret_encrypted,
        label, is_active, created_at
    ) VALUES (
        p_user_id, p_exchange,
        encrypt_api_key(p_new_api_key, v_encryption_key),
        encrypt_api_key(p_new_api_secret, v_encryption_key),
        'Rotated ' || CURRENT_TIMESTAMP::DATE,
        TRUE, NOW()
    ) RETURNING id INTO v_new_key_id;

    -- Копируем разрешения со старого ключа
    IF v_old_key_id IS NOT NULL THEN
        INSERT INTO api_key_permissions (api_key_id, permission, granted_at)
        SELECT v_new_key_id, permission, NOW()
        FROM api_key_permissions
        WHERE api_key_id = v_old_key_id;
    END IF;

    -- Логируем ротацию
    INSERT INTO api_key_rotation_history (
        user_id, exchange, old_key_id, new_key_id, rotation_reason
    ) VALUES (
        p_user_id, p_exchange, v_old_key_id, v_new_key_id, p_rotation_reason
    );

    RETURN v_new_key_id;
END;
$$ language 'plpgsql';

-- Функция для очистки старых логов
CREATE OR REPLACE FUNCTION cleanup_old_api_logs()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM api_key_usage_logs
    WHERE created_at < NOW() - INTERVAL '90 days'
    RETURNING COUNT(*) INTO deleted_count;

    RETURN deleted_count;
END;
$$ language 'plpgsql';

-- Функция для получения статистики использования API ключей
CREATE OR REPLACE FUNCTION get_api_key_stats(
    p_user_id INTEGER DEFAULT NULL,
    p_exchange VARCHAR DEFAULT NULL
) RETURNS TABLE (
    total_keys BIGINT,
    active_keys BIGINT,
    expired_keys BIGINT,
    total_requests BIGINT,
    avg_latency_ms NUMERIC,
    error_rate NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(DISTINCT uk.id)::BIGINT as total_keys,
        COUNT(DISTINCT CASE WHEN uk.is_active THEN uk.id END)::BIGINT as active_keys,
        COUNT(DISTINCT CASE WHEN uk.expires_at < NOW() THEN uk.id END)::BIGINT as expired_keys,
        COUNT(al.id)::BIGINT as total_requests,
        COALESCE(AVG(al.latency_ms), 0)::NUMERIC as avg_latency_ms,
        COALESCE(
            COUNT(CASE WHEN al.response_status >= 400 OR al.error_message IS NOT NULL THEN 1 END) * 100.0 /
            NULLIF(COUNT(al.id), 0),
            0
        )::NUMERIC as error_rate
    FROM user_api_keys uk
    LEFT JOIN api_key_usage_logs al ON uk.id = al.api_key_id
    WHERE (p_user_id IS NULL OR uk.user_id = p_user_id)
      AND (p_exchange IS NULL OR uk.exchange = p_exchange)
      AND al.created_at >= NOW() - INTERVAL '30 days';
END;
$$ language 'plpgsql';

-- Создаем представление для удобного доступа к API ключам
CREATE VIEW api_keys_view AS
SELECT
    uk.id,
    uk.user_id,
    u.telegram_id,
    u.first_name,
    u.last_name,
    uk.exchange,
    uk.label,
    uk.permissions,
    uk.is_active,
    uk.last_used_at,
    uk.expires_at,
    uk.created_at,
    uk.updated_at,
    COUNT(al.id) as total_requests,
    COUNT(DISTINCT DATE(al.created_at)) as active_days,
    MAX(al.created_at) as last_request_at
FROM user_api_keys uk
JOIN users u ON uk.user_id = u.id
LEFT JOIN api_key_usage_logs al ON uk.id = al.api_key_id
GROUP BY uk.id, u.id;

-- Добавляем комментарии к таблицам
COMMENT ON TABLE user_api_keys IS 'Таблица для хранения API ключей пользователей для различных бирж';
COMMENT ON COLUMN user_api_keys.api_key_encrypted IS 'Зашифрованный API ключ';
COMMENT ON COLUMN user_api_keys.api_secret_encrypted IS 'Зашифрованный секретный ключ';
COMMENT ON COLUMN user_api_keys.permissions IS 'JSON с разрешениями ключа в формате {"allow": ["read", "trade"], "deny": ["withdraw"]}';

COMMENT ON TABLE api_key_usage_logs IS 'Таблица для логов использования API ключей';
COMMENT ON COLUMN api_key_usage_logs.latency_ms IS 'Время выполнения запроса в миллисекундах';

COMMENT ON TABLE api_key_permissions IS 'Таблица для хранения разрешений API ключей';

COMMENT ON TABLE api_key_rotation_history IS 'Таблица истории ротации API ключей';

-- Создаем пользовательские типы если нужно
CREATE TYPE exchange_type AS ENUM ('bybit', 'binance', 'kucoin', 'okx', 'gateio');
CREATE TYPE api_key_permission_type AS ENUM (
    'read_only',
    'trade',
    'withdraw',
    'margin',
    'futures',
    'spot',
    'wallet',
    'sub_account'
);