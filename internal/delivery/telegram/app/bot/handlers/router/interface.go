// internal/delivery/telegram/app/bot/handlers/router/interface.go
package router

// Router интерфейс маршрутизатора хэндлеров
type Router interface {
	RegisterHandler(handler Handler)                   // автоматическая регистрация
	RegisterCommand(command string, handler Handler)   // явная регистрация команды
	RegisterCallback(callback string, handler Handler) // явная регистрация callback
	Handle(command string, params HandlerParams) (HandlerResult, error)
	GetHandler(command string) (Handler, bool)
	GetCommands() []string
}
