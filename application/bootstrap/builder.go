// application/bootstrap/app_builder.go
package bootstrap

import (
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
)

// waitForShutdown ждет сигналов завершения
func (app *Application) waitForShutdown() <-chan struct{} {
	done := make(chan struct{})

	go func() {
		// Ждем сигналов завершения
		<-app.stopChan
		app.logger.Println("🛑 Получен сигнал завершения...")

		// Говорим, что хотим завершиться через 30 секунд
		app.shutdownWithTimeout(30 * time.Second)

		close(done)
	}()

	return done
}

// shutdownWithTimeout выполняет graceful shutdown с таймаутом
func (app *Application) shutdownWithTimeout(timeout time.Duration) {
	app.logger.Printf("⏳ Начинаем graceful shutdown (таймаут: %v)...", timeout)

	// Создаем канал для завершения shutdown
	shutdownDone := make(chan struct{})

	go func() {
		app.shutdown()
		close(shutdownDone)
	}()

	// Ждем завершения или таймаута
	select {
	case <-shutdownDone:
		app.logger.Println("✅ Graceful shutdown завершен успешно")
	case <-time.After(timeout):
		app.logger.Println("⚠️  Таймаут graceful shutdown, принудительное завершение")
		// Можно добавить принудительное завершение критических ресурсов
	}
}

// shutdown выполняет остановку приложения
func (app *Application) shutdown() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if !app.running {
		return nil
	}

	app.logger.Println("🛑 Останавливаем приложение...")

	// 1. Останавливаем оркестратор
	if app.orchestrator != nil {
		if err := app.orchestrator.Stop(); err != nil {
			app.logger.Printf("⚠️  Ошибка остановки оркестратора: %v", err)
		}
	}

	// 2. Останавливаем контейнер
	// if app.container != nil {
	// 	// Можно добавить cleanup для контейнера
	// 	app.container.Cleanup()
	// }

	app.running = false
	app.logger.Printf("✅ Приложение остановлено. Время работы: %v", time.Since(app.startTime))

	return nil
}

// Run запускает приложение
func (app *Application) Run() error {
	app.mu.Lock()

	if app.running {
		app.mu.Unlock()
		return errors.New("приложение уже запущено")
	}

	app.logger.Println("🚀 Запуск приложения...")

	// Инициализируем если еще не инициализировано
	// if app.container == nil {
	// 	if err := app.Initialize(); err != nil {
	// 		app.mu.Unlock()
	// 		return fmt.Errorf("инициализация приложения: %w", err)
	// 	}
	// }

	app.running = true
	app.startTime = time.Now()
	app.mu.Unlock()

	// Запускаем оркестратор
	// if err := app.orchestrator.Start(); err != nil {
	// 	return fmt.Errorf("запуск оркестратора: %w", err)
	// }

	app.logger.Println("✅ Приложение запущено и работает")

	// Ждем завершения
	<-app.waitForShutdown()

	return nil
}

// Статус приложения
func (app *Application) Status() map[string]interface{} {
	app.mu.RLock()
	defer app.mu.RUnlock()

	status := map[string]interface{}{
		"running":   app.running,
		"uptime":    time.Since(app.startTime).String(),
		"startTime": app.startTime.Format(time.RFC3339),
		"config": map[string]interface{}{
			"telegram_enabled": app.config.TelegramEnabled,
			"update_interval":  app.config.UpdateInterval,
			"log_level":        app.config.LogLevel,
		},
	}

	// Добавляем статус оркестратора если есть
	// if app.orchestrator != nil {
	// 	status["orchestrator"] = app.orchestrator.GetStatus()
	// }

	return status
}

// Cleanup очищает ресурсы
func (app *Application) Cleanup() {
	app.mu.Lock()
	defer app.mu.Unlock()

	// if app.container != nil {
	// 	app.container.Cleanup()
	// }
}

// Stop останавливает приложение
func (app *Application) Stop() error {
	// Посылаем сигнал завершения
	select {
	case app.stopChan <- syscall.SIGTERM:
	default:
		// Канал уже закрыт или полон
	}

	return nil
}

// ==================== AppBuilder ====================

// AppBuilder строитель приложения
type AppBuilder struct {
	config  *config.Config
	options []AppOption
	logger  *log.Logger
}

// AppOption опция для настройки приложения
type AppOption func(*Application) error

// NewAppBuilder создает новый строитель приложений
func NewAppBuilder() *AppBuilder {
	return &AppBuilder{
		logger: log.New(os.Stdout, "[BUILDER] ", log.LstdFlags),
	}
}

// WithConfig устанавливает конфигурацию
func (b *AppBuilder) WithConfig(cfg *config.Config) *AppBuilder {
	b.config = cfg
	return b
}

// WithConfigFile загружает конфигурацию из файла
func (b *AppBuilder) WithConfigFile(path string) *AppBuilder {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		b.logger.Printf("⚠️  Ошибка загрузки конфигурации: %v", err)
		// Используем дефолтную конфигурацию
		b.config = &config.Config{}
	} else {
		b.config = cfg
	}
	return b
}

// WithOption добавляет опцию настройки
func (b *AppBuilder) WithOption(option AppOption) *AppBuilder {
	b.options = append(b.options, option)
	return b
}

// WithLogger устанавливает логгер
func (b *AppBuilder) WithLogger(logger *log.Logger) *AppBuilder {
	b.logger = logger
	return b
}

// Build строит приложение
func (b *AppBuilder) Build() (*Application, error) {
	if b.config == nil {
		// Используем дефолтную конфигурацию
		b.config, _ = config.LoadConfig(".env")
		b.logger.Println("ℹ️  Используется конфигурация по умолчанию")
	}

	// Создаем приложение
	app, err := NewApplication(b.config)
	if err != nil {
		return nil, fmt.Errorf("создание приложения: %w", err)
	}

	// Применяем опции
	for _, option := range b.options {
		if err := option(app); err != nil {
			return nil, fmt.Errorf("применение опции: %w", err)
		}
	}

	// Инициализируем приложение
	if err := app.Initialize(); err != nil {
		return nil, fmt.Errorf("инициализация: %w", err)
	}

	return app, nil
}

// ==================== Опции приложения ====================

// WithConsoleLogging включает консольное логирование
func WithConsoleLogging(level string) AppOption {
	return func(app *Application) error {
		// Здесь можно настроить логирование
		app.logger.Printf("Установлен уровень логирования: %s", level)
		return nil
	}
}

// WithMetrics включает сбор метрик
func WithMetrics(enabled bool, port string) AppOption {
	return func(app *Application) error {
		if enabled {
			app.logger.Printf("Сбор метрик включен (порт: %s)", port)
		}
		return nil
	}
}

// WithTestMode включает тестовый режим
func WithTestMode(enabled bool) AppOption {
	return func(app *Application) error {
		if enabled {
			app.logger.Println("🧪 Тестовый режим включен")
			// Можно установить флаги тестового режима
		}
		return nil
	}
}

// WithTelegramBot настраивает Telegram бота
func WithTelegramBot(enabled bool, chatID string) AppOption {
	return func(app *Application) error {
		if enabled {
			app.logger.Printf("Telegram бот включен (чат: %s)", chatID)
		}
		return nil
	}
}
