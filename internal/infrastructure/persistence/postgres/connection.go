// persistence/postgres/connection.go (исправленный)
package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Config struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     int    `mapstructure:"DB_PORT"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	Database string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"DB_SSLMODE"`
	MaxConns int    `mapstructure:"DB_MAX_CONNS"`
	MaxIdle  int    `mapstructure:"DB_MAX_IDLE"`
}

func DefaultConfig() *Config {
	return &Config{
		Host:     "localhost",
		Port:     5432,
		User:     "cryptobot",
		Password: "password",
		Database: "cryptobot_db",
		SSLMode:  "disable",
		MaxConns: 25,
		MaxIdle:  10,
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
	return db, nil
}

func RunMigrations(db *sql.DB, migrationsPath string) error {
	// Используем golang-migrate для миграций
	// Для простоты можно использовать встроенные миграции
	return nil
}
