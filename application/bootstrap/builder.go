// application/bootstrap/builder.go
package bootstrap

import (
	"crypto-exchange-screener-bot/application/layer_manager/layers"
	"crypto-exchange-screener-bot/application/scheduler"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"crypto-exchange-screener-bot/pkg/logger"
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

	// 1. Останавливаем Scheduler
	if app.scheduler != nil {
		app.scheduler.Stop()
	}

	// 2. Останавливаем LayerManager
	if app.layerManager != nil {
		if err := app.layerManager.Stop(); err != nil {
			app.logger.Printf("⚠️  Ошибка остановки LayerManager: %v", err)
		}
	}

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
	app.logger.Println("🔍 Проверка LayerManager...")

	// Инициализируем если еще не инициализировано
	if app.layerManager == nil {
		app.logger.Println("⚠️  LayerManager не инициализирован, инициализируем...")
		if err := app.Initialize(); err != nil {
			app.mu.Unlock()
			app.logger.Printf("❌ Ошибка инициализации приложения: %v", err)
			return fmt.Errorf("инициализация приложения: %w", err)
		}
	} else {
		app.logger.Println("✅ LayerManager уже инициализирован")
	}

	app.running = true
	app.startTime = time.Now()
	app.mu.Unlock()

	app.logger.Println("🚀 Запуск LayerManager...")
	// Запускаем LayerManager
	if err := app.layerManager.Start(); err != nil {
		app.logger.Printf("❌ Ошибка запуска LayerManager: %v", err)
		return fmt.Errorf("запуск LayerManager: %w", err)
	}

	// Запускаем Scheduler
	if err := app.startScheduler(); err != nil {
		app.logger.Printf("⚠️  Ошибка запуска Scheduler: %v", err)
		// Не фатально — продолжаем работу без планировщика
	}

	app.logger.Println("✅ Приложение запущено и работает")
	app.logger.Println("⏳ Ожидание graceful shutdown...")

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
			"telegram_enabled": app.config.Telegram.Enabled,
			"update_interval":  app.config.UpdateInterval,
			"log_level":        app.config.LogLevel,
		},
	}

	// Добавляем статус LayerManager если есть
	if app.layerManager != nil {
		status["layerManager"] = app.layerManager.GetHealthStatus()
	}

	return status
}

// Cleanup очищает ресурсы
func (app *Application) Cleanup() {
	app.mu.Lock()
	defer app.mu.Unlock()

	// LayerManager сам управляет своими ресурсами
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
func (b *AppBuilder) WithLogger(loggerInstance *logger.Logger) *AppBuilder {
	// Упрощаем - не устанавливаем кастомный логгер
	// Application создаст свой собственный логгер
	return b
}

// WithTestMode включает тестовый режим (fluent метод)
func (b *AppBuilder) WithTestMode(enabled bool) *AppBuilder {
	b.options = append(b.options, WithTestMode(enabled))
	return b
}

// WithTelegramBot настраивает Telegram бота (fluent метод)
func (b *AppBuilder) WithTelegramBot(enabled bool, chatID string) *AppBuilder {
	b.options = append(b.options, WithTelegramBot(enabled, chatID))
	return b
}

// WithTelegramBotToken устанавливает токен Telegram бота (fluent метод)
func (b *AppBuilder) WithTelegramBotToken(token string) *AppBuilder {
	b.options = append(b.options, WithTelegramBotToken(token))
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
			// Устанавливаем флаг тестового режима в конфигурации
			app.config.MonitoringTestMode = true
		}
		return nil
	}
}

// WithTelegramBot настраивает Telegram бота
func WithTelegramBot(enabled bool, chatID string) AppOption {
	return func(app *Application) error {
		if enabled {
			app.logger.Printf("Telegram бот включен (чат: %s)", chatID)
			app.config.Telegram.Enabled = true
			if chatID != "" {
				app.config.TelegramChatID = chatID
			}
		}
		return nil
	}
}

// startScheduler инициализирует и запускает планировщик задач.
// Получает *sqlx.DB из InfrastructureFactory, регистрирует все задачи и стартует.
func (app *Application) startScheduler() error {
	// Получаем InfrastructureLayer через LayerRegistry
	layerRaw, ok := app.layerManager.GetLayerRegistry().Get("InfrastructureLayer")
	if !ok {
		return fmt.Errorf("InfrastructureLayer не найден в LayerRegistry")
	}

	infraLayer, ok := layerRaw.(*layers.InfrastructureLayer)
	if !ok {
		return fmt.Errorf("InfrastructureLayer имеет неожиданный тип")
	}

	factory := infraLayer.GetInfrastructureFactory()
	if factory == nil {
		return fmt.Errorf("InfrastructureFactory не инициализирована")
	}

	dbSvc, err := factory.CreateDatabaseService()
	if err != nil {
		return fmt.Errorf("получение DatabaseService: %w", err)
	}

	db := dbSvc.GetDB()
	if db == nil {
		return fmt.Errorf("GetDB вернул nil")
	}

	deps := scheduler.Deps{DB: db}

	// Подключаем SubscriptionService если доступен
	coreLayerRaw, ok := app.layerManager.GetLayerRegistry().Get("CoreLayer")
	if ok {
		if coreLayer, ok := coreLayerRaw.(*layers.CoreLayer); ok {
			if svc, err := coreLayer.GetSubscriptionService(); err == nil {
				if sv, ok := svc.(scheduler.SubscriptionValidator); ok {
					deps.SubscriptionService = sv
					logger.Info("✅ [Scheduler] SubscriptionService подключен к планировщику")
				}
			}
		}
	}

	sched := scheduler.New()
	scheduler.RegisterAll(sched, deps)
	sched.Start()

	app.scheduler = sched
	return nil
}

// WithTelegramBotToken устанавливает токен Telegram бота
func WithTelegramBotToken(token string) AppOption {
	return func(app *Application) error {
		if token != "" {
			app.config.TelegramBotToken = token
			app.logger.Println("Токен Telegram бота установлен")
		}
		return nil
	}
}
