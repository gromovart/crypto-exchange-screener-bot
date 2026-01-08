// internal/infrastructure/persistence/postgres/migrator.go
package postgres

import (
	"crypto-exchange-screener-bot/pkg/logger"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Migrator управляет миграциями базы данных
type Migrator struct {
	db         *sqlx.DB
	migrations map[int]*Migration
	logger     *logger.Logger
}

// Migration представляет одну миграцию
type Migration struct {
	ID          int
	Name        string
	Description string
	SQL         string
	AppliedAt   time.Time
	Checksum    string
	Upgrade     bool
}

// NewMigrator создает новый мигратор
func NewMigrator(db *sqlx.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: make(map[int]*Migration),
		logger:     logger.GetLogger(),
	}
}

// Init инициализирует таблицу миграций
func (m *Migrator) Init() error {
	query := `
	CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		sql_content TEXT NOT NULL,
		applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		checksum VARCHAR(64) NOT NULL,
		upgrade BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_migrations_id ON migrations(id);
	CREATE INDEX IF NOT EXISTS idx_migrations_applied_at ON migrations(applied_at);
	`

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	m.logger.Info("✅ Migrations table initialized")
	return nil
}

// LoadMigrations загружает миграции из директории
func (m *Migrator) LoadMigrations(migrationsDir string) error {
	// Проверяем существование директории
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", migrationsDir)
	}

	m.logger.Info("📂 Loading migrations from: %s", migrationsDir)

	// Сканируем файлы в директории
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Фильтруем и сортируем SQL файлы
	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	// Сортируем по имени (они должны начинаться с числа)
	sort.Strings(migrationFiles)

	// Загружаем каждую миграцию
	for _, filename := range migrationFiles {
		if err := m.loadMigration(migrationsDir, filename); err != nil {
			return fmt.Errorf("failed to load migration %s: %w", filename, err)
		}
	}

	m.logger.Info("✅ Loaded %d migrations", len(m.migrations))
	return nil
}

// loadMigration загружает одну миграцию из файла
func (m *Migrator) loadMigration(dir, filename string) error {
	// Парсим ID и имя из имени файла
	// Формат: 001_create_users.sql
	id, name, err := parseMigrationFilename(filename)
	if err != nil {
		return err
	}

	// Читаем содержимое файла
	path := filepath.Join(dir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Создаем миграцию
	migration := &Migration{
		ID:          id,
		Name:        name,
		Description: extractDescription(string(content)),
		SQL:         string(content),
		Checksum:    calculateChecksum(string(content)),
		Upgrade:     true,
	}

	m.migrations[id] = migration
	m.logger.Debug("📄 Loaded migration: %s (%s)", filename, migration.Description)
	return nil
}

// Status показывает статус миграций
func (m *Migrator) Status() ([]MigrationStatus, error) {
	// Получаем список примененных миграций
	query := `
	SELECT id, name, applied_at, checksum, upgrade
	FROM migrations
	ORDER BY id
	`

	rows, err := m.db.Query(query)
	if err != nil {
		// Проверяем, является ли ошибка "relation does not exist"
		if strings.Contains(err.Error(), "relation \"migrations\" does not exist") ||
			strings.Contains(err.Error(), "does not exist") {
			// Таблица migrations не существует - возвращаем пустой статус
			m.logger.Debug("Migrations table does not exist, returning empty status")
			return []MigrationStatus{}, nil
		}
		return nil, fmt.Errorf("failed to query migrations status: %w", err)
	}
	defer rows.Close()

	// Собираем статус
	var statuses []MigrationStatus
	appliedMigrations := make(map[int]*MigrationRecord)

	for rows.Next() {
		var record MigrationRecord
		var appliedAt sql.NullTime
		err := rows.Scan(&record.ID, &record.Name, &appliedAt, &record.Checksum, &record.Upgrade)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		if appliedAt.Valid {
			record.AppliedAt = appliedAt.Time
		}
		appliedMigrations[record.ID] = &record
	}

	// Создаем полный список статусов для всех загруженных миграций
	for id := 1; id <= len(m.migrations); id++ {
		status := MigrationStatus{
			ID:   id,
			Name: m.migrations[id].Name,
		}

		if record, exists := appliedMigrations[id]; exists {
			status.Applied = true
			status.AppliedAt = record.AppliedAt
			status.Upgrade = record.Upgrade

			// Проверяем контрольную сумму
			expectedChecksum := m.migrations[id].Checksum
			if record.Checksum != expectedChecksum {
				status.Status = "checksum_mismatch"
				status.Message = fmt.Sprintf("Checksum mismatch: expected %s, got %s",
					expectedChecksum, record.Checksum)
			} else {
				status.Status = "applied"
			}
		} else {
			status.Applied = false
			status.Status = "pending"
			status.Message = "Migration not applied"
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Migrate применяет все непройденные миграции
func (m *Migrator) Migrate() error {
	m.logger.Info("🚀 Starting database migrations...")

	// Проверяем, что таблица миграций существует
	if err := m.Init(); err != nil {
		return fmt.Errorf("failed to init migrations table: %w", err)
	}

	// Получаем список примененных миграций
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Применяем миграции по порядку
	var appliedCount int
	for id := 1; id <= len(m.migrations); id++ {
		migration, exists := m.migrations[id]
		if !exists {
			m.logger.Warn("⚠️ Migration ID %d not found in loaded migrations", id)
			continue
		}

		// Проверяем, применена ли уже миграция
		if record, applied := applied[id]; applied {
			// Проверяем контрольную сумму
			if record.Checksum != migration.Checksum {
				m.logger.Warn("⚠️ Checksum mismatch for migration %d: %s", id, migration.Name)
				m.logger.Warn("   Expected: %s", migration.Checksum)
				m.logger.Warn("   Got:      %s", record.Checksum)
				return fmt.Errorf("checksum mismatch for migration %d: %s", id, migration.Name)
			}
			m.logger.Debug("✅ Migration already applied: %s", migration.Name)
			continue
		}

		// Применяем миграцию
		if err := m.applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %s: %w", id, migration.Name, err)
		}

		appliedCount++
	}

	if appliedCount > 0 {
		m.logger.Info("✅ Applied %d new migrations", appliedCount)
	} else {
		m.logger.Info("✅ Database is up to date")
	}

	return nil
}

// Rollback откатывает последнюю миграцию
func (m *Migrator) Rollback() error {
	m.logger.Info("↩️ Rolling back last migration...")

	// Получаем последнюю примененную миграцию
	query := `
	SELECT id, name, sql_content, upgrade
	FROM migrations
	WHERE upgrade = TRUE
	ORDER BY id DESC
	LIMIT 1
	`

	var lastMigration MigrationRecord
	err := m.db.QueryRow(query).Scan(&lastMigration.ID, &lastMigration.Name,
		&lastMigration.SQLContent, &lastMigration.Upgrade)
	if err == sql.ErrNoRows {
		m.logger.Info("ℹ️ No migrations to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	// Для безопасности, не откатываем создание таблицы пользователей
	if lastMigration.ID == 1 {
		return fmt.Errorf("cannot rollback initial users table migration")
	}

	// Пытаемся найти SQL для отката (должен быть в том же файле с префиксом -- DOWN)
	migration := m.migrations[lastMigration.ID]
	rollbackSQL := extractRollbackSQL(migration.SQL)

	if rollbackSQL == "" {
		return fmt.Errorf("no rollback SQL found for migration %d: %s", lastMigration.ID, migration.Name)
	}

	// Выполняем откат в транзакции
	tx, err := m.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Выполняем SQL отката
	m.logger.Info("↩️ Rolling back: %s", migration.Name)
	if _, err := tx.Exec(rollbackSQL); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Удаляем запись о миграции
	deleteQuery := `DELETE FROM migrations WHERE id = $1`
	if _, err := tx.Exec(deleteQuery, lastMigration.ID); err != nil {
		return fmt.Errorf("failed to delete migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	m.logger.Info("✅ Successfully rolled back migration: %s", migration.Name)
	return nil
}

// CreateMigration создает новый файл миграции
func (m *Migrator) CreateMigration(name, description string) (string, error) {
	// Генерируем следующий ID
	nextID := len(m.migrations) + 1

	// Создаем имя файла
	filename := fmt.Sprintf("%03d_%s.sql", nextID, strings.ToLower(strings.ReplaceAll(name, " ", "_")))
	filepath := filepath.Join("migrations", filename)

	// Создаем шаблон миграции
	template := fmt.Sprintf(`-- Migration: %s
-- Description: %s
-- Created: %s

-- UP Migration
/*
  SQL для применения миграции
  Здесь пишите CREATE TABLE, ALTER TABLE, INSERT и т.д.
*/

-- Example:
-- CREATE TABLE new_table (
--     id SERIAL PRIMARY KEY,
--     name VARCHAR(100) NOT NULL
-- );

-- DOWN Migration (опционально)
/*
  SQL для отката миграции
  Должен быть обратным к UP миграции
*/

-- Example:
-- DROP TABLE IF EXISTS new_table;
`,
		name, description, time.Now().Format("2006-01-02 15:04:05"))

	// Записываем в файл
	if err := os.WriteFile(filepath, []byte(template), 0644); err != nil {
		return "", fmt.Errorf("failed to create migration file: %w", err)
	}

	m.logger.Info("📝 Created new migration template: %s", filepath)
	return filepath, nil
}

// Validate проверяет целостность миграций
func (m *Migrator) Validate() error {
	m.logger.Info("🔍 Validating migrations...")

	// Проверяем, что все миграции загружены последовательно
	if len(m.migrations) == 0 {
		return fmt.Errorf("no migrations loaded")
	}

	maxID := 0
	for id := range m.migrations {
		if id > maxID {
			maxID = id
		}
		// Проверяем, что нет пропущенных ID
		if _, exists := m.migrations[id-1]; id > 1 && !exists {
			return fmt.Errorf("missing migration with ID %d", id-1)
		}
	}

	// Получаем список примененных миграций
	applied, err := m.getAppliedMigrations()
	if err != nil {
		// Если таблица migrations не существует, это нормально для первого запуска
		// Просто пропускаем проверку контрольных сумм
		m.logger.Debug("Skipping validation check - migrations table may not exist yet")
		return nil
	}

	// Если нет примененных миграций, пропускаем проверку
	if len(applied) == 0 {
		m.logger.Debug("No applied migrations to validate")
		return nil
	}

	// Проверяем контрольные суммы
	var errors []string
	for id, record := range applied {
		if migration, exists := m.migrations[id]; exists {
			if record.Checksum != migration.Checksum {
				errors = append(errors,
					fmt.Sprintf("Migration %d (%s): checksum mismatch", id, migration.Name))
			}
		} else {
			errors = append(errors,
				fmt.Sprintf("Migration %d applied but not found in files", id))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("migration validation failed:\n  %s", strings.Join(errors, "\n  "))
	}

	m.logger.Info("✅ All migrations validated successfully")
	return nil
}

// Вспомогательные методы

func (m *Migrator) getAppliedMigrations() (map[int]*MigrationRecord, error) {
	query := `
	SELECT id, name, applied_at, checksum, sql_content, upgrade
	FROM migrations
	ORDER BY id
	`

	rows, err := m.db.Query(query)
	if err != nil {
		// Проверяем, является ли ошибка "relation does not exist"
		if strings.Contains(err.Error(), "relation \"migrations\" does not exist") ||
			strings.Contains(err.Error(), "does not exist") {
			// Таблица migrations не существует - это нормально для первого запуска
			m.logger.Debug("Migrations table does not exist, treating as empty")
			return make(map[int]*MigrationRecord), nil
		}
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]*MigrationRecord)
	for rows.Next() {
		var record MigrationRecord
		var appliedAt sql.NullTime
		err := rows.Scan(&record.ID, &record.Name, &appliedAt,
			&record.Checksum, &record.SQLContent, &record.Upgrade)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		if appliedAt.Valid {
			record.AppliedAt = appliedAt.Time
		}
		applied[record.ID] = &record
	}

	return applied, nil
}

func (m *Migrator) applyMigration(migration *Migration) error {
	m.logger.Info("📤 Applying migration: %s", migration.Name)

	// Начинаем транзакцию
	tx, err := m.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Выполняем SQL миграции
	if _, err := tx.Exec(migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Сохраняем запись о миграции
	query := `
	INSERT INTO migrations (id, name, description, sql_content, checksum, upgrade)
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(query,
		migration.ID,
		migration.Name,
		migration.Description,
		migration.SQL,
		migration.Checksum,
		migration.Upgrade,
	)
	if err != nil {
		return fmt.Errorf("failed to save migration record: %w", err)
	}

	// Фиксируем транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	m.logger.Info("✅ Applied migration: %s", migration.Name)
	return nil
}

// Структуры для статуса

type MigrationStatus struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Applied   bool      `json:"applied"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Upgrade   bool      `json:"upgrade"`
}

type MigrationRecord struct {
	ID         int       `db:"id"`
	Name       string    `db:"name"`
	AppliedAt  time.Time `db:"applied_at"`
	Checksum   string    `db:"checksum"`
	SQLContent string    `db:"sql_content"`
	Upgrade    bool      `db:"upgrade"`
}

// Вспомогательные функции

func parseMigrationFilename(filename string) (int, string, error) {
	// Убираем расширение .sql
	base := strings.TrimSuffix(filename, ".sql")

	// Разделяем по первому подчеркиванию
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid migration filename format: %s (expected: 001_name.sql)", filename)
	}

	// Парсим ID
	var id int
	if _, err := fmt.Sscanf(parts[0], "%d", &id); err != nil {
		return 0, "", fmt.Errorf("invalid migration ID in filename: %s", filename)
	}

	// Имя миграции
	name := strings.ReplaceAll(parts[1], "_", " ")

	return id, name, nil
}

func extractDescription(sql string) string {
	// Ищем комментарий с описанием
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-- Description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "-- Description:"))
		}
	}
	return "No description"
}

func calculateChecksum(content string) string {
	// Простая контрольная сумма (в продакшене лучше использовать sha256)
	// Для простоты используем длину содержимого
	return fmt.Sprintf("%d", len(content))
}

func extractRollbackSQL(sql string) string {
	// Ищем секцию DOWN Migration
	lines := strings.Split(sql, "\n")
	var inDownSection bool
	var rollbackLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "-- DOWN Migration") {
			inDownSection = true
			continue
		}

		if inDownSection {
			// Пропускаем комментарии
			if strings.HasPrefix(trimmed, "--") || trimmed == "" {
				continue
			}
			// Если нашли следующую секцию, выходим
			if strings.Contains(trimmed, "--") && strings.Contains(strings.ToUpper(trimmed), "MIGRATION") {
				break
			}
			rollbackLines = append(rollbackLines, line)
		}
	}

	if len(rollbackLines) == 0 {
		return ""
	}

	return strings.Join(rollbackLines, "\n")
}
