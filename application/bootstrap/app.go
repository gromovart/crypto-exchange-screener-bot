// application/bootstrap/app_builder.go
package bootstrap

// import (
// 	"crypto-exchange-screener-bot/application/composition"
// 	"crypto-exchange-screener-bot/application/services"
// 	"crypto-exchange-screener-bot/internal/infrastructure/config"
// )

// // AppBuilder строит приложение
// type AppBuilder struct {
// 	config    *config.Config
// 	container *composition.Container
// }

// func NewAppBuilder() *AppBuilder {
// 	return &AppBuilder{}
// }

// func (b *AppBuilder) WithConfig(cfg *config.Config) *AppBuilder {
// 	b.config = cfg
// 	return b
// }

// func (b *AppBuilder) Build() (*Application, error) {
// 	// 1. Создаем контейнер зависимостей
// 	container, err := composition.NewContainer(b.config)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 2. Создаем оркестратор
// 	orchestrator := services.NewOrchestrator(container.DataManager)

// 	app := &Application{
// 		container:    container,
// 		orchestrator: orchestrator,
// 		config:       b.config,
// 	}

// 	return app, nil
// }

// // Application - основное приложение
// type Application struct {
// 	container    *composition.Container
// 	orchestrator *services.Orchestrator
// 	config       *config.Config
// 	running      bool
// }

// func (app *Application) Run() error {
// 	if app.running {
// 		return nil
// 	}

// 	// Запускаем оркестратор
// 	if err := app.orchestrator.Start(); err != nil {
// 		return err
// 	}

// 	app.running = true
// 	return nil
// }

// func (app *Application) Stop() error {
// 	if !app.running {
// 		return nil
// 	}

// 	app.orchestrator.Stop()
// 	app.running = false
// 	return nil
// }
