// internal/infrastructure/persistence/postgres/database/database_service.go
package database

import (
	"context"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// DatabaseService сервис для работы с базой данных
type DatabaseService struct {
	config *config.Config
	db     *sqlx.DB
	mu     sync.RWMutex
	state  ServiceState
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

	// Запускаем миграции если включено
	if dbConfig.EnableAutoMigrate {
		go ds.runMigrations(dbConfig.MigrationsPath)
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
	ds.state = StateStopped
	logger.Info("✅ Database service stopped")

	return nil
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

// runMigrations запускает миграции базы данных
func (ds *DatabaseService) runMigrations(migrationsPath string) {
	logger.Info("🔄 Running database migrations...")

	// Реализация миграций будет добавлена позже
	// Временный заглушка
	logger.Info("⚠️ Database migrations not implemented yet")

	// Здесь можно добавить использование golang-migrate или другого инструмента
	// Например: m, err := migrate.New(migrationsPath, ds.config.GetPostgresDSN())
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
