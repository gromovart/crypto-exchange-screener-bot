-- persistence/postgres/migrations/003_create_activities.sql

-- Таблица для логов активности пользователей
CREATE TABLE user_activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50), -- user, session, api_key, subscription и т.д.
    entity_id INTEGER, -- ID связанной сущности
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    location VARCHAR(100), -- город/страна из IP
    severity VARCHAR(20) DEFAULT 'info', -- info, warning, error, security
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX idx_user_activities_user_id ON user_activities(user_id);
CREATE INDEX idx_user_activities_activity_type ON user_activities(activity_type);
CREATE INDEX idx_user_activities_created_at ON user_activities(created_at);
CREATE INDEX idx_user_activities_entity ON user_activities(entity_type, entity_id);
CREATE INDEX idx_user_activities_severity ON user_activities(severity);

-- Таблица для аудита важных действий (отдельно для безопасности)
CREATE TABLE security_audit_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    action_type VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id INTEGER,
    old_values JSONB,
    new_values JSONB,
    changed_fields TEXT[], -- какие поля изменились
    ip_address INET NOT NULL,
    user_agent TEXT,
    session_id VARCHAR(100),
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для аудита безопасности
CREATE INDEX idx_security_audit_user_id ON security_audit_logs(user_id);
CREATE INDEX idx_security_audit_action_type ON security_audit_logs(action_type);
CREATE INDEX idx_security_audit_resource ON security_audit_logs(resource_type, resource_id);
CREATE INDEX idx_security_audit_created_at ON security_audit_logs(created_at);
CREATE INDEX idx_security_audit_success ON security_audit_logs(success);

-- Таблица для логов входа пользователей
CREATE TABLE login_attempts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    telegram_id BIGINT,
    username VARCHAR(100),
    ip_address INET NOT NULL,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(200),
    two_factor_used BOOLEAN DEFAULT FALSE,
    location VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для анализа входов
CREATE INDEX idx_login_attempts_user_id ON login_attempts(user_id);
CREATE INDEX idx_login_attempts_ip_address ON login_attempts(ip_address);
CREATE INDEX idx_login_attempts_success ON login_attempts(success);
CREATE INDEX idx_login_attempts_created_at ON login_attempts(created_at);
CREATE INDEX idx_login_attempts_telegram_id ON login_attempts(telegram_id);

-- Таблица для отслеживания активности сигналов
CREATE TABLE signal_activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    signal_id VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    signal_type VARCHAR(20) NOT NULL, -- growth, fall, continuous
    change_percent DECIMAL(10,4) NOT NULL,
    confidence DECIMAL(5,2),
    action VARCHAR(50) NOT NULL, -- received, viewed, clicked, ignored, muted
    source VARCHAR(50), -- telegram, email, webhook
    device_info JSONB DEFAULT '{}',
    reaction_time_ms INTEGER, -- время реакции пользователя в мс
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Уникальность пользователь + сигнал + действие
    CONSTRAINT unique_signal_action UNIQUE (user_id, signal_id, action)
);

-- Индексы для анализа сигналов
CREATE INDEX idx_signal_activities_user_id ON signal_activities(user_id);
CREATE INDEX idx_signal_activities_signal_id ON signal_activities(signal_id);
CREATE INDEX idx_signal_activities_symbol ON signal_activities(symbol);
CREATE INDEX idx_signal_activities_created_at ON signal_activities(created_at);
CREATE INDEX idx_signal_activities_action ON signal_activities(action);
CREATE INDEX idx_signal_activities_signal_type ON signal_activities(signal_type);

-- Таблица для логов API запросов от пользователей
CREATE TABLE api_request_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    api_key_id INTEGER REFERENCES user_api_keys(id) ON DELETE SET NULL,
    method VARCHAR(10) NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    request_body JSONB,
    response_status INTEGER NOT NULL,
    response_body JSONB,
    response_time_ms INTEGER NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,
    rate_limit_key VARCHAR(100),
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для мониторинга API
CREATE INDEX idx_api_request_logs_user_id ON api_request_logs(user_id);
CREATE INDEX idx_api_request_logs_api_key_id ON api_request_logs(api_key_id);
CREATE INDEX idx_api_request_logs_endpoint ON api_request_logs(endpoint);
CREATE INDEX idx_api_request_logs_response_status ON api_request_logs(response_status);
CREATE INDEX idx_api_request_logs_created_at ON api_request_logs(created_at);
CREATE INDEX idx_api_request_logs_method ON api_request_logs(method);

-- Таблица для логов подписок и платежей
CREATE TABLE subscription_activities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id INTEGER,
    action_type VARCHAR(50) NOT NULL, -- created, updated, canceled, renewed, payment_failed
    old_plan VARCHAR(50),
    new_plan VARCHAR(50),
    amount DECIMAL(10,2),
    currency VARCHAR(10) DEFAULT 'USD',
    payment_method VARCHAR(50),
    payment_id VARCHAR(100),
    failure_reason VARCHAR(200),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для анализа подписок
CREATE INDEX idx_subscription_activities_user_id ON subscription_activities(user_id);
CREATE INDEX idx_subscription_activities_subscription_id ON subscription_activities(subscription_id);
CREATE INDEX idx_subscription_activities_action_type ON subscription_activities(action_type);
CREATE INDEX idx_subscription_activities_created_at ON subscription_activities(created_at);

-- Таблица для логов изменений настроек пользователей
CREATE TABLE settings_change_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    setting_type VARCHAR(50) NOT NULL, -- notifications, thresholds, display, etc.
    setting_name VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_via VARCHAR(50), -- telegram, web, api, admin
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для отслеживания настроек
CREATE INDEX idx_settings_change_logs_user_id ON settings_change_logs(user_id);
CREATE INDEX idx_settings_change_logs_setting_type ON settings_change_logs(setting_type);
CREATE INDEX idx_settings_change_logs_created_at ON settings_change_logs(created_at);
CREATE INDEX idx_settings_change_logs_setting_name ON settings_change_logs(setting_name);

-- Таблица для логов административных действий
CREATE TABLE admin_activities (
    id SERIAL PRIMARY KEY,
    admin_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    action_type VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id INTEGER,
    old_values JSONB,
    new_values JSONB,
    reason TEXT,
    ip_address INET NOT NULL,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для аудита админки
CREATE INDEX idx_admin_activities_admin_id ON admin_activities(admin_id);
CREATE INDEX idx_admin_activities_target_user_id ON admin_activities(target_user_id);
CREATE INDEX idx_admin_activities_action_type ON admin_activities(action_type);
CREATE INDEX idx_admin_activities_created_at ON admin_activities(created_at);

-- Таблица для агрегированной статистики активности
CREATE TABLE activity_summary (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Общая активность
    total_activities INTEGER DEFAULT 0,
    login_count INTEGER DEFAULT 0,
    signal_count INTEGER DEFAULT 0,
    api_request_count INTEGER DEFAULT 0,

    -- Активность по типам сигналов
    growth_signals INTEGER DEFAULT 0,
    fall_signals INTEGER DEFAULT 0,
    continuous_signals INTEGER DEFAULT 0,

    -- Активность по устройствам
    mobile_activities INTEGER DEFAULT 0,
    desktop_activities INTEGER DEFAULT 0,
    telegram_activities INTEGER DEFAULT 0,
    web_activities INTEGER DEFAULT 0,

    -- Временные метрики
    first_activity_time TIMESTAMP WITH TIME ZONE,
    last_activity_time TIMESTAMP WITH TIME ZONE,
    avg_response_time_ms INTEGER,

    -- Уникальные значения
    unique_symbols TEXT[], -- уникальные символы за день
    unique_endpoints TEXT[], -- уникальные API endpoints

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Одна запись на пользователя в день
    CONSTRAINT unique_user_date UNIQUE (user_id, date)
);

-- Индексы для отчетов
CREATE INDEX idx_activity_summary_date ON activity_summary(date);
CREATE INDEX idx_activity_summary_user_id ON activity_summary(user_id);

-- Таблица для аномальной активности (система обнаружения аномалий)
CREATE TABLE anomaly_detection_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    anomaly_type VARCHAR(100) NOT NULL, -- brute_force, rate_limit, suspicious_location, unusual_behavior
    severity VARCHAR(20) NOT NULL, -- low, medium, high, critical
    description TEXT NOT NULL,
    indicators JSONB NOT NULL, -- показатели, вызвавшие аномалию
    ip_address INET,
    user_agent TEXT,
    triggered_rules TEXT[], -- какие правила сработали
    action_taken VARCHAR(100), -- blocked, warned, notified, etc.
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by INTEGER REFERENCES users(id),
    resolution_notes TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Индексы для мониторинга аномалий
CREATE INDEX idx_anomaly_detection_user_id ON anomaly_detection_logs(user_id);
CREATE INDEX idx_anomaly_detection_anomaly_type ON anomaly_detection_logs(anomaly_type);
CREATE INDEX idx_anomaly_detection_severity ON anomaly_detection_logs(severity);
CREATE INDEX idx_anomaly_detection_resolved ON anomaly_detection_logs(resolved);
CREATE INDEX idx_anomaly_detection_created_at ON anomaly_detection_logs(created_at);

-- Таблица для геолокации активности
CREATE TABLE activity_geolocation (
    id SERIAL PRIMARY KEY,
    ip_address INET NOT NULL,
    country_code VARCHAR(2),
    country_name VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    latitude DECIMAL(10,6),
    longitude DECIMAL(10,6),
    isp VARCHAR(200),
    asn VARCHAR(50),
    threat_score INTEGER, -- оценка угрозы от IP
    first_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    activity_count INTEGER DEFAULT 1,
    user_count INTEGER DEFAULT 1,

    -- Уникальный IP
    CONSTRAINT unique_ip_address UNIQUE (ip_address)
);

-- Индексы для геоанализа
CREATE INDEX idx_activity_geolocation_country ON activity_geolocation(country_code);
CREATE INDEX idx_activity_geolocation_city ON activity_geolocation(city);
CREATE INDEX idx_activity_geolocation_threat_score ON activity_geolocation(threat_score);
CREATE INDEX idx_activity_geolocation_last_seen ON activity_geolocation(last_seen);

-- Триггеры и функции

-- Функция для проверки, что пользователь является администратором
CREATE OR REPLACE FUNCTION check_admin_role()
RETURNS TRIGGER AS $$
BEGIN
    -- Проверяем, что пользователь с admin_id имеет роль 'admin'
    IF NOT EXISTS (
        SELECT 1 FROM users
        WHERE id = NEW.admin_id
            AND role = 'admin'
    ) THEN
        RAISE EXCEPTION 'Пользователь с ID % не является администратором', NEW.admin_id;
    END IF;

    RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггер для проверки роли администратора перед вставкой
CREATE TRIGGER check_admin_role_trigger
    BEFORE INSERT OR UPDATE ON admin_activities
    FOR EACH ROW
    EXECUTE FUNCTION check_admin_role();

-- Триггер для обновления updated_at
CREATE TRIGGER update_activity_summary_updated_at
    BEFORE UPDATE ON activity_summary
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Функция для логирования активности пользователя
CREATE OR REPLACE FUNCTION log_user_activity(
    p_user_id INTEGER,
    p_activity_type VARCHAR,
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
BEGIN
    -- Определяем локацию по IP (можно интегрировать с MaxMind)
    IF p_ip_address IS NOT NULL THEN
        SELECT city || ', ' || country_name INTO v_location
        FROM activity_geolocation
        WHERE ip_address = p_ip_address
        LIMIT 1;
    END IF;

    -- Логируем активность
    INSERT INTO user_activities (
        user_id, activity_type, entity_type, entity_id,
        details, ip_address, user_agent, location, severity
    ) VALUES (
        p_user_id, p_activity_type, p_entity_type, p_entity_id,
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

-- Функция для обновления агрегированной статистики
CREATE OR REPLACE FUNCTION update_activity_summary(
    p_user_id INTEGER,
    p_activity_type VARCHAR
) RETURNS VOID AS $$
DECLARE
    v_today DATE := CURRENT_DATE;
BEGIN
    INSERT INTO activity_summary (
        date, user_id, total_activities,
        login_count, signal_count, api_request_count,
        updated_at
    ) VALUES (
        v_today, p_user_id, 1,
        CASE WHEN p_activity_type LIKE 'login%' THEN 1 ELSE 0 END,
        CASE WHEN p_activity_type LIKE 'signal%' THEN 1 ELSE 0 END,
        CASE WHEN p_activity_type LIKE 'api%' THEN 1 ELSE 0 END,
        NOW()
    )
    ON CONFLICT (user_id, date) DO UPDATE SET
        total_activities = activity_summary.total_activities + 1,
        login_count = activity_summary.login_count +
            CASE WHEN p_activity_type LIKE 'login%' THEN 1 ELSE 0 END,
        signal_count = activity_summary.signal_count +
            CASE WHEN p_activity_type LIKE 'signal%' THEN 1 ELSE 0 END,
        api_request_count = activity_summary.api_request_count +
            CASE WHEN p_activity_type LIKE 'api%' THEN 1 ELSE 0 END,
        updated_at = EXCLUDED.updated_at;
END;
$$ language 'plpgsql';

-- Функция для логирования сигналов
CREATE OR REPLACE FUNCTION log_signal_activity(
    p_user_id INTEGER,
    p_signal_id VARCHAR,
    p_symbol VARCHAR,
    p_signal_type VARCHAR,
    p_change_percent DECIMAL,
    p_confidence DECIMAL,
    p_action VARCHAR,
    p_source VARCHAR DEFAULT 'telegram'
) RETURNS INTEGER AS $$
DECLARE
    v_activity_id INTEGER;
BEGIN
    INSERT INTO signal_activities (
        user_id, signal_id, symbol, signal_type,
        change_percent, confidence, action, source
    ) VALUES (
        p_user_id, p_signal_id, p_symbol, p_signal_type,
        p_change_percent, p_confidence, p_action, p_source
    )
    ON CONFLICT (user_id, signal_id, action) DO UPDATE SET
        reaction_time_ms = EXTRACT(EPOCH FROM (NOW() - signal_activities.created_at)) * 1000,
        updated_at = NOW()
    RETURNING id INTO v_activity_id;

    -- Также логируем в общую таблицу активности
    PERFORM log_user_activity(
        p_user_id,
        'signal_' || p_action,
        'signal',
        NULL,
        jsonb_build_object(
            'signal_id', p_signal_id,
            'symbol', p_symbol,
            'type', p_signal_type,
            'change_percent', p_change_percent,
            'confidence', p_confidence,
            'source', p_source
        )
    );

    RETURN v_activity_id;
END;
$$ language 'plpgsql';

-- Функция для логирования попыток входа
CREATE OR REPLACE FUNCTION log_login_attempt(
    p_ip_address INET,          -- обязательный, без DEFAULT - ставим первым
    p_success BOOLEAN,          -- обязательный, без DEFAULT - ставим вторым
    p_user_id INTEGER DEFAULT NULL,
    p_telegram_id BIGINT DEFAULT NULL,
    p_username VARCHAR DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_failure_reason VARCHAR DEFAULT NULL,
    p_two_factor_used BOOLEAN DEFAULT FALSE
) RETURNS INTEGER AS $$
DECLARE
    v_location VARCHAR;
    v_log_id INTEGER;
BEGIN
    -- Определяем локацию
    SELECT city || ', ' || country_name INTO v_location
    FROM activity_geolocation
    WHERE ip_address = p_ip_address
    LIMIT 1;

    INSERT INTO login_attempts (
        user_id, telegram_id, username, ip_address,
        user_agent, success, failure_reason,
        two_factor_used, location
    ) VALUES (
        p_user_id, p_telegram_id, p_username, p_ip_address,
        p_user_agent, p_success, p_failure_reason,
        p_two_factor_used, v_location
    ) RETURNING id INTO v_log_id;

    -- Логируем в общую таблицу активности
    PERFORM log_user_activity(
        p_user_id,
        CASE
            WHEN p_success THEN 'login_success'
            ELSE 'login_failed'
        END,
        'user',
        p_user_id,
        jsonb_build_object(
            'telegram_id', p_telegram_id,
            'username', p_username,
            'failure_reason', p_failure_reason,
            'two_factor', p_two_factor_used,
            'location', v_location
        ),
        p_ip_address,
        p_user_agent,
        CASE WHEN p_success THEN 'info' ELSE 'warning' END
    );

    -- Проверяем на подозрительную активность
    PERFORM check_suspicious_login_activity(p_ip_address, p_user_id);

    RETURN v_log_id;
END;
$$ language 'plpgsql';

-- Функция для проверки подозрительной активности входа
CREATE OR REPLACE FUNCTION check_suspicious_login_activity(
    p_ip_address INET,
    p_user_id INTEGER DEFAULT NULL
) RETURNS VOID AS $$
DECLARE
    v_failed_attempts INTEGER;
    v_unique_users INTEGER;
    v_last_hour TIMESTAMP := NOW() - INTERVAL '1 hour';
BEGIN
    -- Считаем неудачные попытки за последний час
    SELECT COUNT(*) INTO v_failed_attempts
    FROM login_attempts
    WHERE ip_address = p_ip_address
      AND success = FALSE
      AND created_at >= v_last_hour;

    -- Если больше 5 неудачных попыток - подозрительно
    IF v_failed_attempts >= 5 THEN
        -- Считаем уникальных пользователей
        SELECT COUNT(DISTINCT user_id) INTO v_unique_users
        FROM login_attempts
        WHERE ip_address = p_ip_address
          AND created_at >= v_last_hour;

        -- Если много разных пользователей - брутфорс
        IF v_unique_users >= 3 THEN
            INSERT INTO anomaly_detection_logs (
                user_id, anomaly_type, severity, description,
                indicators, ip_address, triggered_rules
            ) VALUES (
                p_user_id, 'brute_force_attack', 'high',
                'Multiple failed login attempts for different users from same IP',
                jsonb_build_object(
                    'failed_attempts', v_failed_attempts,
                    'unique_users', v_unique_users,
                    'time_window', '1 hour'
                ),
                p_ip_address,
                ARRAY['brute_force_detection']
            );
        END IF;
    END IF;
END;
$$ language 'plpgsql';

-- Функция для получения статистики активности пользователя
CREATE OR REPLACE FUNCTION get_user_activity_stats(
    p_user_id INTEGER,
    p_days INTEGER DEFAULT 30
) RETURNS TABLE (
    date DATE,
    total_activities INTEGER,
    signal_count INTEGER,
    api_requests INTEGER,
    avg_response_time_ms INTEGER,
    favorite_symbol VARCHAR,
    most_active_hour INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        asum.date,
        asum.total_activities,
        asum.signal_count,
        asum.api_request_count,
        asum.avg_response_time_ms,
        (
            SELECT mode() WITHIN GROUP (ORDER BY symbol)
            FROM signal_activities sa
            WHERE sa.user_id = p_user_id
              AND DATE(sa.created_at) = asum.date
            LIMIT 1
        ) as favorite_symbol,
        EXTRACT(HOUR FROM asum.first_activity_time)::INTEGER as most_active_hour
    FROM activity_summary asum
    WHERE asum.user_id = p_user_id
      AND asum.date >= CURRENT_DATE - (p_days || ' days')::INTERVAL
    ORDER BY asum.date DESC;
END;
$$ language 'plpgsql';

-- Функция для очистки старых логов
CREATE OR REPLACE FUNCTION cleanup_old_activity_logs(
    p_retention_days INTEGER DEFAULT 90
) RETURNS TABLE (
    table_name VARCHAR,
    deleted_count BIGINT
) AS $$
DECLARE
    v_tables TEXT[] := ARRAY[
        'user_activities',
        'login_attempts',
        'signal_activities',
        'api_request_logs',
        'subscription_activities',
        'settings_change_logs',
        'admin_activities'
    ];
    v_table TEXT;
    v_deleted BIGINT;
BEGIN
    FOREACH v_table IN ARRAY v_tables
    LOOP
        EXECUTE format(
            'DELETE FROM %I WHERE created_at < NOW() - INTERVAL ''1 day'' * $1 RETURNING COUNT(*)',
            v_table
        ) USING p_retention_days INTO v_deleted;

        table_name := v_table;
        deleted_count := v_deleted;
        RETURN NEXT;
    END LOOP;
END;
$$ language 'plpgsql';

-- Функция для архивации логов
CREATE OR REPLACE FUNCTION archive_activity_logs(
    p_archive_date DATE
) RETURNS INTEGER AS $$
DECLARE
    v_archived_count INTEGER := 0;
BEGIN
    -- Архивируем старые логи в отдельную таблицу
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS user_activities_archive_%s
        (LIKE user_activities INCLUDING ALL)
    ', TO_CHAR(p_archive_date, 'YYYY_MM'));

    EXECUTE format('
        WITH moved AS (
            DELETE FROM user_activities
            WHERE created_at < $1
            RETURNING *
        )
        INSERT INTO user_activities_archive_%s
        SELECT * FROM moved
    ', TO_CHAR(p_archive_date, 'YYYY_MM')) USING p_archive_date;

    GET DIAGNOSTICS v_archived_count = ROW_COUNT;

    RETURN v_archived_count;
END;
$$ language 'plpgsql';

-- Представления для отчетов

-- Представление для ежедневной активности
CREATE VIEW daily_activity_report AS
SELECT
    DATE(ua.created_at) as activity_date,
    COUNT(*) as total_activities,
    COUNT(DISTINCT ua.user_id) as active_users,
    COUNT(CASE WHEN ua.activity_type LIKE 'login%' THEN 1 END) as login_count,
    COUNT(CASE WHEN ua.activity_type LIKE 'signal%' THEN 1 END) as signal_count,
    COUNT(CASE WHEN ua.severity = 'error' THEN 1 END) as error_count,
    COUNT(CASE WHEN ua.severity = 'security' THEN 1 END) as security_count,
    ARRAY_AGG(DISTINCT ua.activity_type) as activity_types
FROM user_activities ua
GROUP BY DATE(ua.created_at)
ORDER BY activity_date DESC;

-- Представление для пользовательской активности
CREATE VIEW user_activity_summary AS
SELECT
    u.id as user_id,
    u.telegram_id,
    u.first_name,
    u.last_name,
    u.role,
    u.created_at as user_created,
    COUNT(ua.id) as total_activities,
    MAX(ua.created_at) as last_activity,
    COUNT(DISTINCT DATE(ua.created_at)) as active_days,
    COUNT(CASE WHEN ua.activity_type LIKE 'signal%' THEN 1 END) as signal_interactions,
    COUNT(CASE WHEN ua.activity_type LIKE 'api%' THEN 1 END) as api_calls,
    ARRAY_AGG(DISTINCT ag.country_name) FILTER (WHERE ag.country_name IS NOT NULL) as countries
FROM users u
LEFT JOIN user_activities ua ON u.id = ua.user_id
LEFT JOIN activity_geolocation ag ON ua.ip_address = ag.ip_address
GROUP BY u.id
ORDER BY total_activities DESC;

-- Представление для анализа сигналов
CREATE VIEW signal_activity_analysis AS
SELECT
    sa.symbol,
    sa.signal_type,
    COUNT(*) as total_activities,
    COUNT(DISTINCT sa.user_id) as unique_users,
    AVG(sa.change_percent) as avg_change_percent,
    AVG(sa.confidence) as avg_confidence,
    COUNT(CASE WHEN sa.action = 'viewed' THEN 1 END) as viewed_count,
    COUNT(CASE WHEN sa.action = 'clicked' THEN 1 END) as clicked_count,
    COUNT(CASE WHEN sa.action = 'ignored' THEN 1 END) as ignored_count,
    AVG(sa.reaction_time_ms) as avg_reaction_time_ms
FROM signal_activities sa
GROUP BY sa.symbol, sa.signal_type
ORDER BY total_activities DESC;

-- Представление для безопасности
CREATE VIEW security_monitoring AS
SELECT
    DATE(adl.created_at) as date,
    adl.anomaly_type,
    adl.severity,
    COUNT(*) as anomaly_count,
    ARRAY_AGG(DISTINCT adl.ip_address) as suspicious_ips,
    ARRAY_AGG(DISTINCT u.telegram_id) as affected_users,
    STRING_AGG(DISTINCT adl.action_taken, ', ') as actions_taken
FROM anomaly_detection_logs adl
LEFT JOIN users u ON adl.user_id = u.id
WHERE adl.resolved = FALSE
GROUP BY DATE(adl.created_at), adl.anomaly_type, adl.severity
ORDER BY date DESC, severity DESC;

-- Добавляем комментарии
COMMENT ON TABLE user_activities IS 'Основная таблица для логов активности пользователей';
COMMENT ON TABLE security_audit_logs IS 'Таблица для аудита важных действий (безопасность)';
COMMENT ON TABLE login_attempts IS 'Таблица для логов попыток входа пользователей';
COMMENT ON TABLE signal_activities IS 'Таблица для отслеживания взаимодействия пользователей с сигналами';
COMMENT ON TABLE api_request_logs IS 'Таблица для логов API запросов от пользователей';
COMMENT ON TABLE subscription_activities IS 'Таблица для логов подписок и платежей';
COMMENT ON TABLE settings_change_logs IS 'Таблица для логов изменений настроек пользователей';
COMMENT ON TABLE admin_activities IS 'Таблица для логов административных действий';
COMMENT ON TABLE activity_summary IS 'Таблица для агрегированной статистики активности (оптимизация отчетов)';
COMMENT ON TABLE anomaly_detection_logs IS 'Таблица для логов аномальной активности (система обнаружения угроз)';
COMMENT ON TABLE activity_geolocation IS 'Таблица для геолокации активности по IP адресам';

-- Создаем ограничения для безопасности
ALTER TABLE admin_activities ENABLE ROW LEVEL SECURITY;

-- Создаем политики безопасности для административных логов
CREATE POLICY admin_activities_admin_only ON admin_activities
    USING (EXISTS (
        SELECT 1 FROM users
        WHERE users.id = current_setting('app.user_id')::INTEGER
          AND users.role = 'admin'
    ));

-- Создаем роль для чтения логов
CREATE ROLE activity_monitor;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO activity_monitor;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO activity_monitor;

-- Индекс для полнотекстового поиска в деталях активности
CREATE INDEX idx_user_activities_details_gin ON user_activities USING GIN (details jsonb_path_ops);