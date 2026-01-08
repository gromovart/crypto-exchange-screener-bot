package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Config struct {
	Host           string `mapstructure:"DB_HOST"`
	Port           int    `mapstructure:"DB_PORT"`
	User           string `mapstructure:"DB_USER"`
	Password       string `mapstructure:"DB_PASSWORD"`
	Database       string `mapstructure:"DB_NAME"`
	SSLMode        string `mapstructure:"DB_SSLMODE"`
	MaxConns       int    `mapstructure:"DB_MAX_CONNS"`
	MaxIdle        int    `mapstructure:"DB_MAX_IDLE"`
	MigrationsPath string `mapstructure:"DB_MIGRATIONS_PATH"`
}

func DefaultConfig() *Config {
	return &Config{
		Host:           "localhost",
		Port:           5432,
		User:           "cryptobot",
		Password:       "password",
		Database:       "cryptobot_db",
		SSLMode:        "disable",
		MaxConns:       25,
		MaxIdle:        10,
		MigrationsPath: "internal/infrastructure/persistence/postgres/migrations",
	}
}

func Connect(cfg *Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Настройки пула соединений
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Проверка подключения
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	log.Println("✅ Connected to PostgreSQL")

	// Выполняем миграции
	if cfg.MigrationsPath != "" {
		if err := RunMigrations(db.DB, cfg.MigrationsPath); err != nil {
			log.Printf("⚠️ Failed to run migrations: %v", err)
			// Не падаем, если миграции не удались, но логируем ошибку
		}
	}

	return db, nil
}

func RunMigrations(db *sql.DB, migrationsPath string) error {
	// Создаем абсолютный путь
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	log.Printf("📂 Running migrations from: %s", absPath)

	// Создаем мигратор
	sqlxDB := sqlx.NewDb(db, "postgres")
	migrator := NewMigrator(sqlxDB)

	// Инициализируем таблицу миграций
	if err := migrator.Init(); err != nil {
		return fmt.Errorf("failed to init migrations table: %w", err)
	}

	// Загружаем миграции из директории
	if err := migrator.LoadMigrations(absPath); err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Применяем миграции
	if err := migrator.Migrate(); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Валидируем миграции
	if err := migrator.Validate(); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
	}

	log.Println("✅ Database migrations completed successfully")
	return nil
}
