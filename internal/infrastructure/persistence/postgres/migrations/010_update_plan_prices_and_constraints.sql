-- internal/infrastructure/persistence/postgres/migrations/010_update_plan_prices_and_constraints.sql

-- ============================================
-- Миграция 010: Обновление цен в Stars и добавление проверок
-- ============================================

-- 1. Обновляем цены в Stars для существующих планов (если колонки существуют)
DO $$
BEGIN
    -- Проверяем существование колонки stars_price_monthly
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'subscription_plans'
        AND column_name = 'stars_price_monthly'
    ) THEN
        UPDATE subscription_plans
        SET
            stars_price_monthly = CASE
                WHEN code = 'free' THEN 0
                WHEN code = 'basic' THEN 299
                WHEN code = 'pro' THEN 999
                WHEN code = 'enterprise' THEN 2499
                ELSE stars_price_monthly
            END,
            stars_price_yearly = CASE
                WHEN code = 'free' THEN 0
                WHEN code = 'basic' THEN 2999
                WHEN code = 'pro' THEN 9999
                WHEN code = 'enterprise' THEN 24999
                ELSE stars_price_yearly
            END
        WHERE code IN ('free', 'basic', 'pro', 'enterprise');
    END IF;
END $$;

-- 2. Добавляем проверки целостности для цен (если их еще нет)
DO $$
BEGIN
    -- Проверка price_monthly
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'subscription_plans'
        AND constraint_name = 'check_positive_monthly_price'
    ) THEN
        ALTER TABLE subscription_plans
        ADD CONSTRAINT check_positive_monthly_price
            CHECK (price_monthly >= 0);
    END IF;

    -- Проверка price_yearly
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'subscription_plans'
        AND constraint_name = 'check_positive_yearly_price'
    ) THEN
        ALTER TABLE subscription_plans
        ADD CONSTRAINT check_positive_yearly_price
            CHECK (price_yearly >= 0);
    END IF;

    -- Проверка stars_price_monthly (если колонка существует)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'subscription_plans'
        AND column_name = 'stars_price_monthly'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'subscription_plans'
        AND constraint_name = 'check_positive_stars_monthly'
    ) THEN
        ALTER TABLE subscription_plans
        ADD CONSTRAINT check_positive_stars_monthly
            CHECK (stars_price_monthly >= 0);
    END IF;

    -- Проверка stars_price_yearly (если колонка существует)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'subscription_plans'
        AND column_name = 'stars_price_yearly'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'subscription_plans'
        AND constraint_name = 'check_positive_stars_yearly'
    ) THEN
        ALTER TABLE subscription_plans
        ADD CONSTRAINT check_positive_stars_yearly
            CHECK (stars_price_yearly >= 0);
    END IF;
END $$;

-- 3. Создаем индексы для производительности (если их нет)
CREATE INDEX IF NOT EXISTS idx_plans_stars_monthly
    ON subscription_plans(stars_price_monthly);

CREATE INDEX IF NOT EXISTS idx_plans_stars_yearly
    ON subscription_plans(stars_price_yearly);

CREATE INDEX IF NOT EXISTS idx_plans_active
    ON subscription_plans(is_active)
    WHERE is_active = true;

-- 4. Добавляем проверки для таблицы payments (если она существует)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'payments'
    ) THEN
        -- Проверка amount
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE table_name = 'payments'
            AND constraint_name = 'check_positive_amount'
        ) THEN
            ALTER TABLE payments
            ADD CONSTRAINT check_positive_amount
                CHECK (amount >= 0);
        END IF;

        -- Проверка stars_amount
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE table_name = 'payments'
            AND constraint_name = 'check_positive_stars_amount'
        ) THEN
            ALTER TABLE payments
            ADD CONSTRAINT check_positive_stars_amount
                CHECK (stars_amount >= 0);
        END IF;

        -- Проверка fiat_amount
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE table_name = 'payments'
            AND constraint_name = 'check_positive_fiat_amount'
        ) THEN
            ALTER TABLE payments
            ADD CONSTRAINT check_positive_fiat_amount
                CHECK (fiat_amount >= 0);
        END IF;
    END IF;
END $$;

-- 5. Добавляем проверки для таблицы invoices (если она существует)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'invoices'
    ) THEN
        -- Проверка amount_usd
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE table_name = 'invoices'
            AND constraint_name = 'check_positive_amount_usd'
        ) THEN
            ALTER TABLE invoices
            ADD CONSTRAINT check_positive_amount_usd
                CHECK (amount_usd >= 0);
        END IF;

        -- Проверка stars_amount
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE table_name = 'invoices'
            AND constraint_name = 'check_positive_stars_amount_invoice'
        ) THEN
            ALTER TABLE invoices
            ADD CONSTRAINT check_positive_stars_amount_invoice
                CHECK (stars_amount >= 0);
        END IF;

        -- Проверка fiat_amount
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE table_name = 'invoices'
            AND constraint_name = 'check_positive_fiat_amount_invoice'
        ) THEN
            ALTER TABLE invoices
            ADD CONSTRAINT check_positive_fiat_amount_invoice
                CHECK (fiat_amount >= 0);
        END IF;

        -- Проверка expires_at > created_at
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE table_name = 'invoices'
            AND constraint_name = 'check_expiry_date'
        ) THEN
            ALTER TABLE invoices
            ADD CONSTRAINT check_expiry_date
                CHECK (expires_at > created_at);
        END IF;
    END IF;
END $$;

-- 6. Создаем функцию для расчета цены в Stars из USD (если не существует)
CREATE OR REPLACE FUNCTION calculate_stars_from_usd(usd_amount DECIMAL(10,2))
RETURNS INTEGER AS $$
BEGIN
    RETURN FLOOR(usd_amount * 100)::INTEGER;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 7. Создаем функцию для расчета USD из Stars (если не существует)
CREATE OR REPLACE FUNCTION calculate_usd_from_stars(stars_amount INTEGER)
RETURNS DECIMAL(10,2) AS $$
BEGIN
    RETURN (stars_amount::DECIMAL / 100);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 8. Создаем функцию для проверки соответствия цен (если не существует)
CREATE OR REPLACE FUNCTION validate_plan_prices()
RETURNS TRIGGER AS $$
BEGIN
    -- Проверяем только если колонки существуют
    IF NEW.stars_price_monthly IS NOT NULL AND
       NEW.stars_price_monthly != calculate_stars_from_usd(NEW.price_monthly) THEN
        RAISE EXCEPTION 'Цена в Stars не соответствует цене в USD для месячного плана';
    END IF;

    IF NEW.stars_price_yearly IS NOT NULL AND
       NEW.stars_price_yearly != calculate_stars_from_usd(NEW.price_yearly) THEN
        RAISE EXCEPTION 'Цена в Stars не соответствует цене в USD для годового плана';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 9. Создаем триггер для автоматической проверки цен планов (если не существует)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
        WHERE tgname = 'trigger_validate_plan_prices'
    ) THEN
        CREATE TRIGGER trigger_validate_plan_prices
            BEFORE INSERT OR UPDATE ON subscription_plans
            FOR EACH ROW
            EXECUTE FUNCTION validate_plan_prices();
    END IF;
END $$;

-- 10. Создаем или заменяем представление для удобного просмотра планов с ценами
CREATE OR REPLACE VIEW plan_pricing_view AS
SELECT
    sp.id,
    sp.name,
    sp.code,
    sp.description,
    sp.price_monthly,
    sp.price_yearly,
    COALESCE(sp.stars_price_monthly, 0) as stars_price_monthly,
    COALESCE(sp.stars_price_yearly, 0) as stars_price_yearly,
    sp.max_symbols,
    sp.max_signals_per_day,
    sp.features,
    sp.is_active,
    -- Расчет экономии при годовой оплате (если цена месячная > 0)
    CASE
        WHEN sp.price_monthly > 0 THEN
            ROUND((1 - (sp.price_yearly / (sp.price_monthly * 12))) * 100, 1)
        ELSE 0
    END as yearly_savings_percent
FROM subscription_plans sp
WHERE sp.is_active = true;

-- 11. Создаем или заменяем функцию для получения активного плана пользователя
CREATE OR REPLACE FUNCTION get_user_active_plan(user_id_param INTEGER)
RETURNS TABLE(
    plan_id INTEGER,
    plan_name VARCHAR(100),
    plan_code VARCHAR(50),
    status VARCHAR(20),
    period_end TIMESTAMP WITH TIME ZONE,
    days_remaining INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        sp.id,
        sp.name,
        sp.code,
        us.status,
        us.current_period_end,
        EXTRACT(DAY FROM (us.current_period_end - NOW()))::INTEGER
    FROM user_subscriptions us
    JOIN subscription_plans sp ON us.plan_id = sp.id
    WHERE us.user_id = user_id_param
        AND us.status IN ('active', 'trialing')
        AND us.current_period_end > NOW()
    ORDER BY us.current_period_end DESC
    LIMIT 1;

    -- Если нет активной подписки, возвращаем NULL
    IF NOT FOUND THEN
        plan_id := NULL;
        plan_name := NULL;
        plan_code := NULL;
        status := NULL;
        period_end := NULL;
        days_remaining := NULL;
        RETURN;
    END IF;
END;
$$ LANGUAGE plpgsql STABLE;

-- 12. Комментарии к обновленным таблицам (если они существуют)
DO $$
BEGIN
    -- Комментарий к subscription_plans.stars_price_monthly
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'subscription_plans'
        AND column_name = 'stars_price_monthly'
    ) THEN
        COMMENT ON COLUMN subscription_plans.stars_price_monthly IS 'Цена в Telegram Stars за месяц';
    END IF;

    -- Комментарий к subscription_plans.stars_price_yearly
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'subscription_plans'
        AND column_name = 'stars_price_yearly'
    ) THEN
        COMMENT ON COLUMN subscription_plans.stars_price_yearly IS 'Цена в Telegram Stars за год';
    END IF;

    -- Комментарий к user_subscriptions.payment_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_subscriptions'
        AND column_name = 'payment_id'
    ) THEN
        COMMENT ON COLUMN user_subscriptions.payment_id IS 'ID платежа, которым была оплачена подписка';
    END IF;
END $$;

-- 13. Создаем индексы для поиска платежей (если таблица существует)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'payments'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_payments_user_status
            ON payments(user_id, status);
    END IF;
END $$;

-- 14. Создаем индексы для поиска инвойсов (если таблица существует)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'invoices'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_invoices_user_status
            ON invoices(user_id, status);
    END IF;
END $$;

-- 15. Добавляем комментарии к функциям
COMMENT ON FUNCTION calculate_stars_from_usd IS 'Конвертирует сумму в USD в количество Telegram Stars (1 USD = 100 Stars)';
COMMENT ON FUNCTION calculate_usd_from_stars IS 'Конвертирует количество Telegram Stars в сумму в USD (100 Stars = 1 USD)';
COMMENT ON FUNCTION validate_plan_prices IS 'Проверяет соответствие цен в USD и Telegram Stars для планов подписки';
COMMENT ON FUNCTION get_user_active_plan IS 'Возвращает активный план подписки пользователя или NULL если нет активной подписки';
COMMENT ON VIEW plan_pricing_view IS 'Представление для просмотра тарифных планов с расчетами стоимости и экономии';

-- ============================================
-- Завершение миграции
-- ============================================
