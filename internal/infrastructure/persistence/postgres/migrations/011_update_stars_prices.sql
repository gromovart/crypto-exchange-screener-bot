-- internal/infrastructure/persistence/postgres/migrations/011_update_stars_prices.sql

-- ============================================
-- Миграция 011: Обновление цен в Stars на 500/1000/2500
-- ============================================

-- 1. Временно отключаем триггер проверки цен
ALTER TABLE subscription_plans DISABLE TRIGGER trigger_validate_plan_prices;

-- 2. Обновляем цены в Stars для планов
UPDATE subscription_plans
SET
    stars_price_monthly = CASE
        WHEN code = 'free' THEN 0
        WHEN code = 'basic' THEN 500      -- ⭐ 500 Stars
        WHEN code = 'pro' THEN 1000       -- ⭐ 1000 Stars
        WHEN code = 'enterprise' THEN 2500 -- ⭐ 2500 Stars
        ELSE stars_price_monthly
    END,
    stars_price_yearly = CASE
        WHEN code = 'free' THEN 0
        WHEN code = 'basic' THEN 5000     -- ⭐ 5000 Stars
        WHEN code = 'pro' THEN 10000      -- ⭐ 10000 Stars
        WHEN code = 'enterprise' THEN 25000 -- ⭐ 25000 Stars
        ELSE stars_price_yearly
    END,
    -- Обновляем цены в USD в соответствии с курсом 36.23
    price_monthly = CASE
        WHEN code = 'basic' THEN 13.80    -- 500 / 36.23 = 13.80
        WHEN code = 'pro' THEN 27.60      -- 1000 / 36.23 = 27.60
        WHEN code = 'enterprise' THEN 69.00 -- 2500 / 36.23 = 69.00
        ELSE price_monthly
    END,
    price_yearly = CASE
        WHEN code = 'basic' THEN 138.00   -- 5000 / 36.23 = 138.00
        WHEN code = 'pro' THEN 276.00     -- 10000 / 36.23 = 276.00
        WHEN code = 'enterprise' THEN 690.00 -- 25000 / 36.23 = 690.00
        ELSE price_yearly
    END
WHERE code IN ('basic', 'pro', 'enterprise', 'free');

-- 3. Включаем триггер обратно
ALTER TABLE subscription_plans ENABLE TRIGGER trigger_validate_plan_prices;

-- 4. Проверяем результаты
DO $$
DECLARE
    plan_rec RECORD;
BEGIN
    RAISE NOTICE 'Проверка обновленных цен:';
    FOR plan_rec IN
SELECT code, price_monthly, stars_price_monthly,
  price_yearly, stars_price_yearly
FROM subscription_plans
WHERE code IN ('basic', 'pro', 'enterprise')
ORDER BY id
    LOOP
        RAISE NOTICE
'План %: $% = % Stars (месяц), $% = % Stars (год)',
            plan_rec.code,
            plan_rec.price_monthly,
            plan_rec.stars_price_monthly,
            plan_rec.price_yearly,
            plan_rec.stars_price_yearly;
END LOOP;
END $$;

-- ============================================
-- Завершение миграции
-- ============================================
