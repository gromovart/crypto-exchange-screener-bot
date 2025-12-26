-- persistence/postgres/migrations/005_create_sessions.sql
CREATE TABLE user_sessions
(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token VARCHAR(255) UNIQUE NOT NULL,
  device_info JSONB DEFAULT '{}',
  ip_address INET,
  user_agent TEXT,
  expires_at TIMESTAMP
  WITH TIME ZONE NOT NULL,
    data JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    revoked_at TIMESTAMP
  WITH TIME ZONE,
    revoked_reason VARCHAR
  (100),
    created_at TIMESTAMP
  WITH TIME ZONE DEFAULT NOW
  (),
    updated_at TIMESTAMP
  WITH TIME ZONE DEFAULT NOW
  (),
    last_activity TIMESTAMP
  WITH TIME ZONE DEFAULT NOW
  ()
);

  CREATE TABLE session_activities
  (
    id SERIAL PRIMARY KEY,
    session_id UUID REFERENCES user_sessions(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    details JSONB DEFAULT '{}',
    ip_address INET,
    created_at TIMESTAMP
    WITH TIME ZONE DEFAULT NOW
    ()
);

    -- Индексы
    CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
    CREATE INDEX idx_user_sessions_token ON user_sessions(token);
    CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
    CREATE INDEX idx_user_sessions_is_active ON user_sessions(is_active);
    CREATE INDEX idx_user_sessions_last_activity ON user_sessions(last_activity);
    CREATE INDEX idx_user_sessions_ip_address ON user_sessions(ip_address);

    CREATE INDEX idx_session_activities_session_id ON session_activities(session_id);
    CREATE INDEX idx_session_activities_created_at ON session_activities(created_at);
    CREATE INDEX idx_session_activities_activity_type ON session_activities(activity_type);

    -- Триггер для updated_at
    CREATE TRIGGER update_user_sessions_updated_at
    BEFORE
    UPDATE ON user_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column
    ();

    -- Функция для автоматического обновления last_activity
    CREATE OR REPLACE FUNCTION update_session_last_activity
    ()
RETURNS TRIGGER AS $$
    BEGIN
    NEW.last_activity = NOW
    ();
    RETURN NEW;
    END;
$$ language 'plpgsql';

    CREATE TRIGGER update_session_last_activity
    BEFORE
    UPDATE ON user_sessions
    FOR EACH ROW
    WHEN
    (OLD.* IS DISTINCT FROM NEW.*)
    EXECUTE FUNCTION update_session_last_activity
    ();

    -- Функция для очистки старых сессий
    CREATE OR REPLACE FUNCTION cleanup_old_sessions
    ()
RETURNS void AS $$
    BEGIN
      DELETE FROM user_sessions
    WHERE expires_at < NOW() - INTERVAL
      '90 days'
       OR
      (is_active = FALSE AND revoked_at < NOW
      () - INTERVAL '30 days');
    END;
    $$ language 'plpgsql';