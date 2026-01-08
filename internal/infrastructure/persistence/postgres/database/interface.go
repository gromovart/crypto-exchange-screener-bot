// internal/infrastructure/persistence/postgres/database/interface.go
package database

// Database интерфейс для работы с базой данных
type Database interface {
	// Общие методы
	Exec(query string, args ...interface{}) error
	QueryRow(query string, args ...interface{}) (map[string]interface{}, error)
	Query(query string, args ...interface{}) ([]map[string]interface{}, error)

	// Транзакции
	Begin() (Transaction, error)

	// Закрытие
	Close() error

	// Пинг
	Ping() error
}

// Transaction интерфейс транзакции
type Transaction interface {
	Commit() error
	Rollback() error
	Exec(query string, args ...interface{}) error
	QueryRow(query string, args ...interface{}) (map[string]interface{}, error)
	Query(query string, args ...interface{}) ([]map[string]interface{}, error)
}
