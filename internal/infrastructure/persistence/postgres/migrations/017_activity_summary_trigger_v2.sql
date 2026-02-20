-- 017_activity_summary_trigger_v2.sql
-- Улучшенный триггер для activity_summary

CREATE OR REPLACE FUNCTION trigger_update_activity_summary()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO activity_summary (
        date, user_id, total_activities,
        login_count, signal_count, api_request_count,
        growth_signals, fall_signals,
        telegram_activities,
        first_activity_time, last_activity_time,
        updated_at
    ) VALUES (
        DATE(NEW.created_at), NEW.user_id, 1,
        CASE WHEN NEW.activity_type LIKE 'user_login%' THEN 1 ELSE 0 END,
        CASE WHEN NEW.activity_type LIKE 'signal%' THEN 1 ELSE 0 END,
        CASE WHEN NEW.activity_type LIKE 'api%' THEN 1 ELSE 0 END,
        CASE WHEN NEW.activity_type = 'signal_received' AND NEW.details->>'signal_type' = 'growth' THEN 1 ELSE 0 END,
        CASE WHEN NEW.activity_type = 'signal_received' AND NEW.details->>'signal_type' = 'fall' THEN 1 ELSE 0 END,
        1,
        NEW.created_at, NEW.created_at, NOW()
    )
    ON CONFLICT (user_id, date) DO UPDATE SET
        total_activities   = activity_summary.total_activities + 1,
        login_count        = activity_summary.login_count +
            CASE WHEN NEW.activity_type LIKE 'user_login%' THEN 1 ELSE 0 END,
        signal_count       = activity_summary.signal_count +
            CASE WHEN NEW.activity_type LIKE 'signal%' THEN 1 ELSE 0 END,
        api_request_count  = activity_summary.api_request_count +
            CASE WHEN NEW.activity_type LIKE 'api%' THEN 1 ELSE 0 END,
        growth_signals     = activity_summary.growth_signals +
            CASE WHEN NEW.activity_type = 'signal_received' AND NEW.details->>'signal_type' = 'growth' THEN 1 ELSE 0 END,
        fall_signals       = activity_summary.fall_signals +
            CASE WHEN NEW.activity_type = 'signal_received' AND NEW.details->>'signal_type' = 'fall' THEN 1 ELSE 0 END,
        telegram_activities = activity_summary.telegram_activities + 1,
        last_activity_time = NEW.created_at,
        updated_at         = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_update_activity_summary ON user_activities;

CREATE TRIGGER trg_update_activity_summary
    AFTER INSERT ON user_activities
    FOR EACH ROW
    WHEN (NEW.user_id > 0)
    EXECUTE FUNCTION trigger_update_activity_summary();
