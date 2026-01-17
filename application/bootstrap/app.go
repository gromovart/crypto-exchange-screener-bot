// application/bootstrap/app.go
package bootstrap

// Импорты для вспомогательных функций
import (
	layer_manager "crypto-exchange-screener-bot/application/layer_manager"
	"crypto-exchange-screener-bot/internal/infrastructure/config"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Application основное приложение
type Application struct {
	mu sync.RWMutex

	// Компоненты
	config       *config.Config
	layerManager *layer_manager.LayerManager

	// Состояние
	running   bool
	startTime time.Time
	stopChan  chan os.Signal

	// Логирование
	logger *log.Logger
}

// NewApplication создает новое приложение
func NewApplication(cfg *config.Config) (*Application, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	app := &Application{
		config:    cfg,
		stopChan:  make(chan os.Signal, 1),
		startTime: time.Now(),
		logger:    log.New(os.Stdout, "[APP] ", log.LstdFlags),
		running:   false,
	}

	// Настраиваем обработку сигналов
	signal.Notify(app.stopChan, syscall.SIGINT, syscall.SIGTERM)

	return app, nil
}

// Initialize инициализирует приложение
func (app *Application) Initialize() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.running {
		return errors.New("приложение уже инициализировано и запущено")
	}

	app.logger.Println("🚀 Инициализация приложения...")
	app.logger.Printf("📋 Конфигурация: TelegramEnabled=%v, TestMode=%v",
		app.config.TelegramEnabled, app.config.MonitoringTestMode)

	// Создаем LayerManager
	app.logger.Println("🛠️  Создание LayerManager...")
	layerManager := layer_manager.NewLayerManager(app.config)

	// Инициализируем LayerManager
	app.logger.Println("🔧 Инициализация LayerManager...")
	if err := layerManager.Initialize(); err != nil {
		app.logger.Printf("❌ Ошибка инициализации LayerManager: %v", err)
		return fmt.Errorf("не удалось инициализировать LayerManager: %w", err)
	}

	app.layerManager = layerManager
	app.logger.Println("✅ LayerManager создан и инициализирован")

	// Получаем статус для отладки
	if status := layerManager.GetHealthStatus(); status != nil {
		app.logger.Printf("📊 Статус LayerManager: initialized=%v", status["initialized"])
		if layers, ok := status["layers"].(map[string]interface{}); ok {
			app.logger.Printf("📊 Слои: %d", len(layers))
			for name, layerStatus := range layers {
				app.logger.Printf("   - %s: %v", name, layerStatus)
			}
		}
	}

	app.logger.Println("✅ Приложение инициализировано")
	return nil
}

// GetConfig возвращает конфигурацию приложения
func (app *Application) GetConfig() *config.Config {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.config
}

// GetLayerManager возвращает менеджер слоев
func (app *Application) GetLayerManager() *layer_manager.LayerManager {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.layerManager
}

// IsRunning проверяет, запущено ли приложение
func (app *Application) IsRunning() bool {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.running
}

// GetUptime возвращает время работы приложения
func (app *Application) GetUptime() time.Duration {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return time.Since(app.startTime)
}

// GetStartTime возвращает время запуска
func (app *Application) GetStartTime() time.Time {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.startTime
}

// GetLogger возвращает логгер приложения
func (app *Application) GetLogger() *log.Logger {
	return app.logger
}

// Restart перезапускает приложение
func (app *Application) Restart() error {
	app.logger.Println("🔄 Перезапуск приложения...")

	// 1. Останавливаем
	if err := app.Stop(); err != nil {
		return fmt.Errorf("остановка при перезапуске: %w", err)
	}

	// 2. Ждем немного
	time.Sleep(2 * time.Second)

	// 3. Очищаем
	app.Cleanup()

	// 4. Переинициализируем
	if err := app.Initialize(); err != nil {
		return fmt.Errorf("инициализация при перезапуске: %w", err)
	}

	// 5. Запускаем
	if err := app.Run(); err != nil {
		return fmt.Errorf("запуск при перезапуске: %w", err)
	}

	return nil
}

// Wait блокирует до завершения приложения
func (app *Application) Wait() {
	// Создаем канал, который закроется при остановке
	waitChan := make(chan struct{})

	go func() {
		for app.IsRunning() {
			time.Sleep(100 * time.Millisecond)
		}
		close(waitChan)
	}()

	<-waitChan
}

// DebugInfo возвращает отладочную информацию
func (app *Application) DebugInfo() map[string]interface{} {
	status := app.Status()

	// Добавляем дополнительную отладочную информацию
	debugInfo := map[string]interface{}{
		"status":        status,
		"goroutines":    runtime.NumGoroutine(),
		"environment":   getEnvironmentInfo(),
		"dependencies":  getDependencyVersions(),
		"configuration": app.getConfigurationSummary(),
	}

	return debugInfo
}

// getMemoryStats возвращает статистику памяти
func (app *Application) getMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"num_gc":         m.NumGC,
		"goroutines":     runtime.NumGoroutine(),
	}
}

// getEnvironmentInfo возвращает информацию об окружении
func getEnvironmentInfo() map[string]interface{} {
	return map[string]interface{}{
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"cpu_cores":  runtime.NumCPU(),
		"pid":        os.Getpid(),
		"ppid":       os.Getppid(),
		"hostname":   getHostname(),
		"user":       getCurrentUser(),
	}
}

// getConfigurationSummary возвращает краткую информацию о конфигурации
func (app *Application) getConfigurationSummary() map[string]interface{} {
	cfg := app.GetConfig()

	return map[string]interface{}{
		"telegram_enabled": cfg.TelegramEnabled,
		"telegram_chat_id": maskString(cfg.TelegramChatID, 4),
		"update_interval":  cfg.UpdateInterval,
		"log_level":        cfg.LogLevel,
		"rate_limit_delay": cfg.RateLimitDelay,
		"test_mode":        cfg.MonitoringTestMode,
	}
}

// Вспомогательные функции
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getCurrentUser() string {
	user, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return user.Username
}

func maskString(s string, visibleChars int) string {
	if len(s) <= visibleChars {
		return s
	}
	masked := ""
	for i := 0; i < len(s)-visibleChars; i++ {
		masked += "*"
	}
	return masked + s[len(s)-visibleChars:]
}

func getDependencyVersions() map[string]string {
	// Здесь можно добавить информацию о версиях зависимостей
	return map[string]string{
		"go": runtime.Version(),
	}
}
