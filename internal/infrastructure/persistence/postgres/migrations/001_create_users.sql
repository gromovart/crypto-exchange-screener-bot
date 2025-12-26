-- persistence/postgres/migrations/001_create_users.sql
CREATE TABLE users
(
  id SERIAL PRIMARY KEY,
  telegram_id BIGINT UNIQUE NOT NULL,
  username VARCHAR(100),
  first_name VARCHAR(100) NOT NULL,
  last_name VARCHAR(100),
  chat_id VARCHAR(100) UNIQUE NOT NULL,

  -- Настройки уведомлений
  notifications_enabled BOOLEAN DEFAULT TRUE,
  notify_growth BOOLEAN DEFAULT TRUE,
  notify_fall BOOLEAN DEFAULT TRUE,
  notify_continuous BOOLEAN DEFAULT TRUE,
  quiet_hours_start INTEGER DEFAULT 23,
  quiet_hours_end INTEGER DEFAULT 8,

  -- Настройки анализа
  min_growth_threshold DECIMAL(5,2) DEFAULT 2.0,
  min_fall_threshold DECIMAL(5,2) DEFAULT 2.0,
  preferred_periods INTEGER
  [] DEFAULT '{5,15,30}',

    -- Профиль
    language VARCHAR
  (10) DEFAULT 'ru',
    timezone VARCHAR
  (50) DEFAULT 'Europe/Moscow',
    display_mode VARCHAR
  (20) DEFAULT 'compact',

    -- Статус и лимиты
    role VARCHAR
  (20) DEFAULT 'user',
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    subscription_tier VARCHAR
  (20) DEFAULT 'free',
    signals_today INTEGER DEFAULT 0,
    max_signals_per_day INTEGER DEFAULT 50,

    -- Временные метки
    created_at TIMESTAMP
  WITH TIME ZONE DEFAULT NOW
  (),
    updated_at TIMESTAMP
  WITH TIME ZONE DEFAULT NOW
  (),
    last_login_at TIMESTAMP
  WITH TIME ZONE,
    last_signal_at TIMESTAMP
  WITH TIME ZONE,

    -- Индексы
    CONSTRAINT users_telegram_id_key UNIQUE
  (telegram_id),
    CONSTRAINT users_chat_id_key UNIQUE
  (chat_id)
);

  CREATE INDEX idx_users_telegram_id ON users(telegram_id);
  CREATE INDEX idx_users_chat_id ON users(chat_id);
  CREATE INDEX idx_users_is_active ON users(is_active);
  CREATE INDEX idx_users_created_at ON users(created_at);

  -- Триггер для updated_at
  CREATE OR REPLACE FUNCTION update_updated_at_column
  ()
RETURNS TRIGGER AS $$
  BEGIN
    NEW.updated_at = NOW
  ();
  RETURN NEW;
  END;
$$ language 'plpgsql';

  CREATE TRIGGER update_users_updated_at
    BEFORE
  UPDATE ON users
    FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column
  ();