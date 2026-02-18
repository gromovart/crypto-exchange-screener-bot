-- ============================================
-- Миграция 009: Таблица инвойсов и связи
-- ============================================

-- 1. Таблица invoices (инвойсы)
CREATE TABLE invoices
(
  id SERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  plan_id VARCHAR(50) NOT NULL,

  -- Основная информация
  external_id VARCHAR(255),
  title VARCHAR(255) NOT NULL,
  description TEXT,

  -- Сумма и валюта (совместимость со StarsInvoice)
  amount_usd DECIMAL(10,2) NOT NULL,
  stars_amount INTEGER NOT NULL,
  fiat_amount INTEGER NOT NULL DEFAULT 0,
  -- Сумма в центах
  currency VARCHAR(3) NOT NULL DEFAULT 'USD',
  -- Валюта

  -- Статус и провайдер
  status VARCHAR(20) NOT NULL DEFAULT 'created',
  provider VARCHAR(20) NOT NULL,

  -- Ссылки и данные
  invoice_url TEXT NOT NULL,
  payload TEXT,
  metadata JSONB DEFAULT '{}',

  -- Временные метки
  created_at TIMESTAMP
  WITH TIME ZONE DEFAULT NOW
  (),
    updated_at TIMESTAMP
  WITH TIME ZONE DEFAULT NOW
  (),
    expires_at TIMESTAMP
  WITH TIME ZONE NOT NULL,
    paid_at TIMESTAMP
  WITH TIME ZONE,

    -- Индексы
    CONSTRAINT valid_invoice_status CHECK
  (status IN
  ('created', 'pending', 'paid', 'expired', 'cancelled', 'failed')),
    CONSTRAINT valid_invoice_provider CHECK
  (provider IN
  ('telegram', 'stripe', 'manual')),
    CONSTRAINT valid_currency_invoice CHECK
  (currency IN
  ('USD', 'EUR', 'RUB'))
);

  -- Индексы для invoices
  CREATE INDEX idx_invoices_user_id ON invoices(user_id);
  CREATE INDEX idx_invoices_plan_id ON invoices(plan_id);
  CREATE INDEX idx_invoices_status ON invoices(status);
  CREATE INDEX idx_invoices_provider ON invoices(provider);
  CREATE INDEX idx_invoices_external_id ON invoices(external_id);
  CREATE INDEX idx_invoices_created_at ON invoices(created_at);
  CREATE INDEX idx_invoices_expires_at ON invoices(expires_at);
  CREATE INDEX idx_invoices_payload ON invoices(payload);

  -- 2. Обновляем таблицу payments - добавляем foreign key к invoices
  ALTER TABLE payments
ADD CONSTRAINT fk_payments_invoice_id
FOREIGN KEY (invoice_id)
REFERENCES invoices(id)
ON DELETE SET NULL;

  -- 3. Триггер для updated_at invoices
  CREATE OR REPLACE FUNCTION update_invoices_updated_at
  ()
RETURNS TRIGGER AS $$
  BEGIN
    NEW.updated_at = NOW
  ();
  RETURN NEW;
  END;
$$ LANGUAGE plpgsql;

  CREATE TRIGGER update_invoices_updated_at
    BEFORE
  UPDATE ON invoices
    FOR EACH ROW
  EXECUTE FUNCTION update_invoices_updated_at
  ();

-- 4. Обновляем комментарии
COMMENT ON TABLE invoices IS 'Таблица инвойсов (счетов на оплату)';
COMMENT ON COLUMN invoices.fiat_amount IS 'Сумма в центах для совместимости с текущей реализацией StarsInvoice';
COMMENT ON COLUMN invoices.payload IS 'Payload для deep link и связи с транзакциями';
COMMENT ON COLUMN invoices.external_id IS 'Внешний ID инвойса (например, из Telegram Stars API)';

  -- 5. Обновляем subscription_plans - добавляем примерные цены в Stars для дефолтных планов
  UPDATE subscription_plans
SET
    stars_price_monthly = CASE
        WHEN code = 'basic' THEN 299      -- $2.99 в Stars
        WHEN code = 'pro' THEN 999        -- $9.99 в Stars
        WHEN code = 'enterprise' THEN 2499 -- $24.99 в Stars
        ELSE 0
    END,
    stars_price_yearly = CASE
        WHEN code = 'basic' THEN 2999     -- $29.99 в Stars
        WHEN code = 'pro' THEN 9999       -- $99.99 в Stars
        WHEN code = 'enterprise' THEN 24999 -- $249.99 в Stars
        ELSE 0
    END
WHERE code IN ('basic', 'pro', 'enterprise');

  -- 6. Создаем вспомогательное представление для просмотра платежей с инвойсами
  CREATE OR REPLACE VIEW payment_invoice_view AS
  SELECT
    p.id as payment_id,
    p.external_id as telegram_payment_id,
    p.stars_amount,
    p.amount as usd_amount,
    p.status as payment_status,
    p.created_at as payment_created,
    p.paid_at,
    i.id as invoice_id,
    i.plan_id,
    i.title as invoice_title,
    i.status as invoice_status,
    i.provider,
    u.id as user_id,
    u.telegram_id,
    u.username
  FROM payments p
    LEFT JOIN invoices i ON p.invoice_id = i.id
    LEFT JOIN users u ON p.user_id = u.id;

  COMMENT ON VIEW payment_invoice_view IS 'Представление для просмотра платежей с связанными инвойсами и пользователями';
