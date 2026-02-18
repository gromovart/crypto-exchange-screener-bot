-- internal/infrastructure/persistence/postgres/migrations/012_auto_free_subscription_on_register.sql

-- ============================================
-- Миграция 012: Автоматическое создание бесплатной подписки при регистрации
-- ============================================

-- 1. Функция для создания бесплатной подписки при регистрации пользователя
CREATE OR REPLACE FUNCTION create_free_subscription_on_register()
RETURNS TRIGGER AS $$
DECLARE
    free_plan_id INTEGER;
    free_plan_name VARCHAR(100);
    free_plan_code VARCHAR(50);
BEGIN
    -- Получаем ID и данные бесплатного плана
    SELECT id, name, code INTO free_plan_id, free_plan_name, free_plan_code
    FROM subscription_plans
    WHERE code = 'free'
    LIMIT 1;

    -- Если план найден, создаем бесплатную подписку на 24 часа
    IF free_plan_id IS NOT NULL THEN
        INSERT INTO user_subscriptions (
            user_id,
            plan_id,
            status,
            current_period_start,
            current_period_end,
            cancel_at_period_end,
            metadata
        ) VALUES (
            NEW.id,
            free_plan_id,
            'active',
            NOW(),
            NOW() + INTERVAL '1 day',
            false,
            jsonb_build_object(
                'type', 'welcome_trial',
                'duration_hours', 24,
                'created_at', NOW(),
                'source', 'registration_trigger',
                'plan_name', free_plan_name,
                'plan_code', free_plan_code
            )
        );

        RAISE NOTICE '✅ Создана бесплатная подписка для нового пользователя % на 24 часа', NEW.id;
    ELSE
        RAISE WARNING '⚠️ План "free" не найден, бесплатная подписка не создана для пользователя %', NEW.id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. Создаем триггер на вставку нового пользователя
DROP TRIGGER IF EXISTS trigger_free_subscription_on_register ON users;
CREATE TRIGGER trigger_free_subscription_on_register
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION create_free_subscription_on_register();

-- 3. Добавляем комментарии
COMMENT ON FUNCTION create_free_subscription_on_register IS
'Создает бесплатную подписку на 24 часа при регистрации нового пользователя';

COMMENT ON TRIGGER trigger_free_subscription_on_register ON users IS
'Триггер для автоматического создания бесплатной подписки при регистрации';

-- ============================================
-- Завершение миграции
-- ============================================
