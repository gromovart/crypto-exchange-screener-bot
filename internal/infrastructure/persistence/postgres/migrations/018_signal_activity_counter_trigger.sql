-- 018_signal_activity_counter_trigger.sql
-- Триггер: инкремент signals_today при вставке записи в signal_activities с action='received'
-- Сброс: функция для ежедневного сброса счётчиков (вызывается внешним планировщиком)

-- Функция-триггер: инкрементирует signals_today при получении сигнала
CREATE OR REPLACE FUNCTION increment_user_signals_today()
RETURNS TRIGGER AS $$
BEGIN
    -- Считаем только фактически доставленные сигналы
    IF NEW.action = 'received' THEN
        UPDATE users
        SET signals_today = signals_today + 1,
            updated_at    = NOW()
        WHERE id = NEW.user_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер на INSERT в signal_activities
DROP TRIGGER IF EXISTS trg_increment_signals_today ON signal_activities;

CREATE TRIGGER trg_increment_signals_today
    AFTER INSERT ON signal_activities
    FOR EACH ROW
    EXECUTE FUNCTION increment_user_signals_today();

-- Функция сброса счётчиков (вызывать в полночь через pg_cron или приложение)
CREATE OR REPLACE FUNCTION reset_daily_signals()
RETURNS INTEGER AS $$
DECLARE
    affected INTEGER;
BEGIN
    UPDATE users
    SET signals_today = 0,
        updated_at    = NOW()
    WHERE signals_today > 0;

    GET DIAGNOSTICS affected = ROW_COUNT;
    RETURN affected;
END;
$$ LANGUAGE plpgsql;
