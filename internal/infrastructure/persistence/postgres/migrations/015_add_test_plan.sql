-- Migration: add test plan
-- Description: Добавление тестового тарифа для проверки оплаты (только dev окружение)

INSERT INTO subscription_plans (
    name, code, description,
    price_monthly, price_yearly,
    stars_price_monthly, stars_price_yearly,
    max_symbols, max_signals_per_day,
    features, is_active
) VALUES (
    'Test',
    'test',
    'Тестовый тариф для проверки оплаты (только dev)',
    0.05, 0.05,
    2, 2,
    5, 10,
    '{"test_plan": true, "dev_only": true}',
    true
) ON CONFLICT (code) DO NOTHING;
