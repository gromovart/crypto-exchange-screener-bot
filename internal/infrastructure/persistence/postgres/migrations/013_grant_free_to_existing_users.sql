-- internal/infrastructure/persistence/postgres/migrations/013_grant_free_to_existing_users.sql

-- ============================================
-- Миграция 013: Раздача бесплатных подписок существующим пользователям
-- ============================================

DO $$
DECLARE
    free_plan_id INTEGER;
    free_plan_name VARCHAR(100);
    free_plan_code VARCHAR(50);
    user_record RECORD;
    existing_subscription INTEGER;
    created_count INTEGER := 0;
    skipped_count INTEGER := 0;
BEGIN
    -- Получаем данные бесплатного плана
    SELECT id, name, code INTO free_plan_id, free_plan_name, free_plan_code
    FROM subscription_plans
    WHERE code = 'free'
    LIMIT 1;

    IF free_plan_id IS NULL THEN
        RAISE EXCEPTION '❌ План "free" не найден в таблице subscription_plans';
    END IF;

    RAISE NOTICE '✅ Найден free план: ID=%, Name=%, Code=%', free_plan_id, free_plan_name, free_plan_code;
    RAISE NOTICE '🔄 Начинаем раздачу бесплатных подписок существующим пользователям...';

    -- Перебираем всех пользователей
    FOR user_record IN SELECT id FROM users LOOP
        -- Проверяем, есть ли уже активная подписка
        SELECT id INTO existing_subscription
        FROM user_subscriptions
        WHERE user_id = user_record.id
          AND status IN ('active', 'trialing')
          AND (current_period_end IS NULL OR current_period_end > NOW())
        LIMIT 1;

        -- Если нет активной подписки, создаем бесплатную на 24 часа
        IF existing_subscription IS NULL THEN
            INSERT INTO user_subscriptions (
                user_id,
                plan_id,
                status,
                current_period_start,
                current_period_end,
                cancel_at_period_end,
                metadata
            ) VALUES (
                user_record.id,
                free_plan_id,
                'active',
                NOW(),
                NOW() + INTERVAL '1 day',
                false,
                jsonb_build_object(
                    'type', 'welcome_trial',
                    'duration_hours', 24,
                    'granted_at', NOW(),
                    'granted_by', 'migration_013',
                    'plan_name', free_plan_name,
                    'plan_code', free_plan_code
                )
            );
            created_count := created_count + 1;
        ELSE
            skipped_count := skipped_count + 1;
        END IF;
    END LOOP;

    RAISE NOTICE '✅ Создано бесплатных подписок: %', created_count;
    RAISE NOTICE '⏭️ Пропущено пользователей (уже есть подписка): %', skipped_count;
END $$;

-- ============================================
-- Завершение миграции
-- ============================================
