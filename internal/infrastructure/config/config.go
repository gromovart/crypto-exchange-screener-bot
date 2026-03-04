// internal/infrastructure/config/config.go
package config

// Пакет config предоставляет централизованную конфигурацию приложения.
//
// Структура пакета:
// - types.go: определение структур конфигурации
// - loader.go: загрузка конфигурации из переменных окружения
// - validators.go: валидация конфигурации
// - helpers.go: вспомогательные функции для чтения переменных окружения
// - methods.go: методы для работы с конфигурацией
//
// Использование:
//   cfg, err := config.LoadConfig(".env")
//   if err != nil {
//       log.Fatal(err)
//   }
//   cfg.PrintSummary()
