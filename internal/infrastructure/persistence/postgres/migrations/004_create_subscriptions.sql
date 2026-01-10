-- persistence/postgres/migrations/004_create_subscriptions.sql

CREATE TABLE subscription_plans
(
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    price_monthly DECIMAL(10,2) DEFAULT 0,
    price_yearly DECIMAL(10,2) DEFAULT 0,
    max_symbols INTEGER DEFAULT 100,
    max_signals_per_day INTEGER DEFAULT 100,
    features JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP
    WITH TIME ZONE DEFAULT NOW
    ()
);

    CREATE TABLE user_subscriptions
    (
        id SERIAL PRIMARY KEY,
        user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
        plan_id INTEGER REFERENCES subscription_plans(id),
        stripe_subscription_id VARCHAR(100),
        status VARCHAR(20) DEFAULT 'pending',
        current_period_start TIMESTAMP
        WITH TIME ZONE,
    current_period_end TIMESTAMP
        WITH TIME ZONE,
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP
        WITH TIME ZONE DEFAULT NOW
        (),
    updated_at TIMESTAMP
        WITH TIME ZONE DEFAULT NOW
        (),

    -- Обычный UNIQUE constraint
    UNIQUE
        (user_id, plan_id),

    CONSTRAINT valid_status CHECK
        (status IN
        (
        'pending', 'active', 'trialing', 'past_due',
        'canceled', 'expired', 'incomplete'
    ))
);

        -- Partial unique index для активных/триальных подписок
        CREATE UNIQUE INDEX idx_unique_active_user_plan
    ON user_subscriptions(user_id, plan_id)
    WHERE status IN ('active', 'trialing');

        -- Индексы
        CREATE INDEX idx_user_subscriptions_user_id ON user_subscriptions(user_id);
        CREATE INDEX idx_user_subscriptions_status ON user_subscriptions(status);
        CREATE INDEX idx_user_subscriptions_period_end ON user_subscriptions(current_period_end);
        CREATE INDEX idx_user_subscriptions_stripe_id ON user_subscriptions(stripe_subscription_id);

        -- Триггер для updated_at
        CREATE TRIGGER update_subscriptions_updated_at
    BEFORE
        UPDATE ON user_subscriptions
    FOR EACH ROW
        EXECUTE FUNCTION update_updated_at_column
        ();

        -- Добавляем дефолтные тарифные планы
        INSERT INTO subscription_plans
            (name, code, description, price_monthly, price_yearly, max_symbols, max_signals_per_day, features)
        VALUES
            ('Free', 'free', 'Бесплатный тариф для начала работы', 0, 0, 50, 50, '{"notifications": true, "basic_analytics": true, "community_access": true}'),
            ('Basic', 'basic', 'Для активных трейдеров', 9.99, 99.99, 200, 500, '{"notifications": true, "advanced_analytics": true, "priority_support": true, "custom_thresholds": true}'),
            ('Pro', 'pro', 'Профессиональный трейдинг', 29.99, 299.99, 500, 2000, '{"notifications": true, "advanced_analytics": true, "priority_support": true, "custom_thresholds": true, "api_access": true, "whitelabel": true, "custom_indicators": true}'),
            ('Enterprise', 'enterprise', 'Корпоративное решение', 99.99, 999.99, -1, -1, '{"notifications": true, "advanced_analytics": true, "dedicated_support": true, "custom_thresholds": true, "api_access": true, "whitelabel": true, "custom_indicators": true, "sla": true, "custom_integrations": true}');