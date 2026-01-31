-- ============================================
-- Миграция 008: Таблица платежей
-- ============================================

-- 1. Таблица payments (платежи)
CREATE TABLE payments
(
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id INTEGER REFERENCES user_subscriptions(id) ON DELETE SET NULL,
    invoice_id BIGINT, -- будет ссылаться на invoices в миграции 009

    -- Основная информация (совместимость с текущей реализацией)
    external_id VARCHAR(255),               -- TelegramPaymentID
    amount DECIMAL(10,2) NOT NULL,         -- Сумма в USD
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    stars_amount INTEGER NOT NULL,          -- Количество Stars
    fiat_amount INTEGER NOT NULL,           -- Сумма в центах (для совместимости)

    -- Тип и статус
    payment_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    provider VARCHAR(50) NOT NULL,

    -- Детали платежа
    description TEXT,
    payload TEXT,                           -- Payload из инвойса
    metadata JSONB DEFAULT '{}',

    -- Временные метки
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    paid_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,

    -- Индексы
    CONSTRAINT valid_payment_type CHECK (payment_type IN ('stars', 'crypto', 'bank_card')),
    CONSTRAINT valid_payment_status CHECK (status IN (
        'pending', 'processing', 'completed', 'failed',
        'refunded', 'expired', 'cancelled'
    )),
    CONSTRAINT valid_currency CHECK (currency IN ('USD', 'EUR', 'RUB'))
);

-- Индексы для payments
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_subscription_id ON payments(subscription_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_payment_type ON payments(payment_type);
CREATE INDEX idx_payments_external_id ON payments(external_id);
CREATE INDEX idx_payments_created_at ON payments(created_at);
CREATE INDEX idx_payments_paid_at ON payments(paid_at);
CREATE INDEX idx_payments_payload ON payments(payload);

-- 2. Обновляем subscription_plans - добавляем цены в Stars
ALTER TABLE subscription_plans
ADD COLUMN IF NOT EXISTS stars_price_monthly INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS stars_price_yearly INTEGER DEFAULT 0;

-- 3. Обновляем user_subscriptions - добавляем связь с платежами
ALTER TABLE user_subscriptions
ADD COLUMN IF NOT EXISTS payment_id BIGINT REFERENCES payments(id) ON DELETE SET NULL;

-- 4. Триггер для updated_at payments
CREATE OR REPLACE FUNCTION update_payments_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_payments_updated_at();

-- 5. Комментарии к таблице payments
COMMENT ON TABLE payments IS 'Таблица платежей пользователей (совместима с текущей реализацией)';
COMMENT ON COLUMN payments.external_id IS 'TelegramPaymentID из системы Telegram Stars';
COMMENT ON COLUMN payments.fiat_amount IS 'Сумма в центах для совместимости с текущей реализацией';
COMMENT ON COLUMN payments.payload IS 'Payload из инвойса для связи с транзакцией';
COMMENT ON COLUMN subscription_plans.stars_price_monthly IS 'Цена в Telegram Stars (месячная)';
COMMENT ON COLUMN subscription_plans.stars_price_yearly IS 'Цена в Telegram Stars (годовая)';
COMMENT ON COLUMN user_subscriptions.payment_id IS 'Ссылка на платеж, которым была оплачена подписка';
