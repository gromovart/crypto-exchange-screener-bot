// internal/infrastructure/persistence/postgres/database/service.go
package database

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	postgres_migrations "crypto-exchange-screener-bot/internal/infrastructure/persistence/postgres"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// DatabaseService сервис для работы с базой данных
type DatabaseService struct {
	config   *config.Config
	db       *sqlx.DB
	mu       sync.RWMutex
	state    ServiceState
	migrator *postgres_migrations.Migrator
}

// ServiceState состояние сервиса
type ServiceState string

const (
	StateStopped  ServiceState = "stopped"
	StateStarting ServiceState = "starting"
	StateRunning  ServiceState = "running"
	StateStopping ServiceState = "stopping"
	StateError    ServiceState = "error"
)

// NewDatabaseService создает новый сервис базы данных
func NewDatabaseService(cfg *config.Config) *DatabaseService {
	return &DatabaseService{
		config: cfg,
		state:  StateStopped,
	}
}

// Start запускает сервис базы данных
func (ds *DatabaseService) Start() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.state == StateRunning {
		return fmt.Errorf("database service already running")
	}

	logger.Info("🔄 Starting database service...")
	ds.state = StateStarting

	// Получаем конфигурацию базы данных
	dbConfig := ds.config.GetDatabaseConfig()

	// Формируем DSN строку
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Name,
		dbConfig.SSLMode,
	)

	logger.Info("📡 Connecting to PostgreSQL: %s:%d/%s",
		dbConfig.Host, dbConfig.Port, dbConfig.Name)

	// Подключаемся к базе данных
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		ds.state = StateError
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(dbConfig.MaxConnLifetime)
	db.SetConnMaxIdleTime(dbConfig.MaxConnIdleTime)

	// Проверяем подключение с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		ds.state = StateError
		return fmt.Errorf("failed to ping database: %w", err)
	}

	ds.db = db
	ds.state = StateRunning

	// Логируем успешное подключение
	logger.Info("✅ Successfully connected to PostgreSQL")
	logger.Info("   • Host: %s:%d", dbConfig.Host, dbConfig.Port)
	logger.Info("   • Database: %s", dbConfig.Name)
	logger.Info("   • User: %s", dbConfig.User)
	logger.Info("   • Pool: %d/%d connections",
		dbConfig.MaxIdleConns, dbConfig.MaxOpenConns)

	// Создаем и запускаем мигратор
	ds.migrator = postgres_migrations.NewMigrator(db)

	// Запускаем миграции если включено
	if dbConfig.EnableAutoMigrate {
		if err := ds.runMigrations(dbConfig.MigrationsPath); err != nil {
			logger.Warn("⚠️ Database migrations failed: %v", err)
			// Не останавливаем приложение при ошибке миграций
			// Можно добавить опцию для критичности миграций
		}
	}

	return nil
}

// Stop останавливает сервис базы данных
func (ds *DatabaseService) Stop() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.state != StateRunning {
		return fmt.Errorf("database service is not running")
	}

	logger.Info("🛑 Stopping database service...")
	ds.state = StateStopping

	if ds.db != nil {
		if err := ds.db.Close(); err != nil {
			ds.state = StateError
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}

	ds.db = nil
	ds.migrator = nil
	ds.state = StateStopped
	logger.Info("✅ Database service stopped")

	return nil
}

// runMigrations запускает миграции базы данных
func (ds *DatabaseService) runMigrations(migrationsPath string) error {
	if ds.migrator == nil {
		return fmt.Errorf("migrator not initialized")
	}

	logger.Info("🔄 Running database migrations from: %s", migrationsPath)

	// Загружаем миграции
	if err := ds.migrator.LoadMigrations(migrationsPath); err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Сначала применяем миграции (это создаст таблицу migrations при первом запуске)
	if err := ds.migrator.Migrate(); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Теперь получаем статус (таблица migrations уже должна существовать)
	statuses, err := ds.migrator.Status()
	if err != nil {
		// Даже после Migrate() возможны проблемы, логируем, но не падаем
		logger.Warn("⚠️ Failed to get migration status: %v", err)
		statuses = []postgres_migrations.MigrationStatus{} // Пустой статус
	}

	// Логируем текущий статус
	if len(statuses) > 0 {
		logger.Info("📊 Migration status:")
		for _, status := range statuses {
			statusIcon := "⏳"
			if status.Applied {
				statusIcon = "✅"
			}
			logger.Info("   %s %03d: %s", statusIcon, status.ID, status.Name)
		}
	}

	// Проверяем валидность (теперь таблица должна существовать)
	if err := ds.migrator.Validate(); err != nil {
		logger.Warn("⚠️ Migration validation warning: %v", err)
	}

	logger.Info("✅ Database migrations completed successfully")
	return nil
}

// Migrate выполняет миграции базы данных
func (ds *DatabaseService) Migrate() error {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.state != StateRunning || ds.migrator == nil {
		return fmt.Errorf("database service is not running or migrator not initialized")
	}

	return ds.migrator.Migrate()
}

// Rollback откатывает последнюю миграцию
func (ds *DatabaseService) Rollback() error {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.state != StateRunning || ds.migrator == nil {
		return fmt.Errorf("database service is not running or migrator not initialized")
	}

	return ds.migrator.Rollback()
}

// GetMigrationStatus возвращает статус миграций
func (ds *DatabaseService) GetMigrationStatus() ([]postgres_migrations.MigrationStatus, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.state != StateRunning || ds.migrator == nil {
		return nil, fmt.Errorf("database service is not running or migrator not initialized")
	}

	return ds.migrator.Status()
}

// ValidateMigrations проверяет валидность миграций
func (ds *DatabaseService) ValidateMigrations() error {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.state != StateRunning || ds.migrator == nil {
		return fmt.Errorf("database service is not running or migrator not initialized")
	}

	return ds.migrator.Validate()
}

// GetDB возвращает соединение с базой данных
func (ds *DatabaseService) GetDB() *sqlx.DB {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.db
}

// State возвращает состояние сервиса
func (ds *DatabaseService) State() ServiceState {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.state
}

// HealthCheck проверяет здоровье базы данных
func (ds *DatabaseService) HealthCheck() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.state != StateRunning || ds.db == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ds.db.PingContext(ctx); err != nil {
		logger.Info("⚠️ Database health check failed: %v", err)
		return false
	}

	return true
}

// GetStats возвращает статистику базы данных
func (ds *DatabaseService) GetStats() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	stats := map[string]interface{}{
		"state":     ds.state,
		"connected": ds.db != nil,
	}

	if ds.db != nil {
		stats["open_connections"] = ds.db.Stats().OpenConnections
		stats["in_use"] = ds.db.Stats().InUse
		stats["idle"] = ds.db.Stats().Idle
		stats["wait_count"] = ds.db.Stats().WaitCount
		stats["wait_duration"] = ds.db.Stats().WaitDuration.String()
	}

	// Добавляем статус миграций
	if ds.migrator != nil {
		migrationStats := map[string]interface{}{
			"migrator_initialized": true,
		}

		// Пытаемся получить статус миграций, но не падаем при ошибке
		if statuses, err := ds.migrator.Status(); err == nil {
			var applied, pending int
			for _, status := range statuses {
				if status.Applied {
					applied++
				} else {
					pending++
				}
			}
			migrationStats["migrations_applied"] = applied
			migrationStats["migrations_pending"] = pending
			migrationStats["migrations_total"] = len(statuses)
		}

		stats["migrations"] = migrationStats
	} else {
		stats["migrations"] = map[string]interface{}{
			"migrator_initialized": false,
		}
	}

	return stats
}

// TestConnection тестирует подключение к базе данных
func (ds *DatabaseService) TestConnection() error {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.state != StateRunning || ds.db == nil {
		return fmt.Errorf("database service is not running")
	}

	// Выполняем простой запрос для проверки
	var result int
	err := ds.db.Get(&result, "SELECT 1")
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	return nil
}

// GetDatabaseName возвращает имя текущей базы данных
func (ds *DatabaseService) GetDatabaseName() string {
	if ds.db == nil {
		return ""
	}

	var dbName string
	err := ds.db.Get(&dbName, "SELECT current_database()")
	if err != nil {
		return ""
	}

	return dbName
}

// CreateMigration создает новый файл миграции
func (ds *DatabaseService) CreateMigration(name, description string) (string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.state != StateRunning || ds.migrator == nil {
		return "", fmt.Errorf("database service is not running or migrator not initialized")
	}

	return ds.migrator.CreateMigration(name, description)
}
