-- Migration: Remove quiet hours columns
-- Description: Удаление колонок тихих часов из таблицы users
-- Created: 2026-02-18

-- UP Migration
ALTER TABLE users DROP COLUMN IF EXISTS quiet_hours_start;
ALTER TABLE users DROP COLUMN IF EXISTS quiet_hours_end;

-- DOWN Migration (на случай отката)
-- ALTER TABLE users ADD COLUMN quiet_hours_start INTEGER DEFAULT 0;
-- ALTER TABLE users ADD COLUMN quiet_hours_end INTEGER DEFAULT 0;
-- COMMENT ON COLUMN users.quiet_hours_start IS 'Начало тихих часов (час)';
-- COMMENT ON COLUMN users.quiet_hours_end IS 'Конец тихих часов (час)';
