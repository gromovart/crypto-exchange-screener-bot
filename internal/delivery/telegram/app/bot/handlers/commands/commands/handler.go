// internal/delivery/telegram/app/bot/handlers/commands/commands/handler.go
package commands

import (
	"fmt"
	"strings"

	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/constants"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers"
	"crypto-exchange-screener-bot/internal/delivery/telegram/app/bot/handlers/base"
)

// commandsCommandHandler реализация обработчика команды /commands
type commandsCommandHandler struct {
	*base.BaseHandler
}

// NewHandler создает новый обработчик команды /commands
func NewHandler() handlers.Handler {
	return &commandsCommandHandler{
		BaseHandler: &base.BaseHandler{
			Name:    "commands_command_handler",
			Command: "commands",
			Type:    handlers.TypeCommand,
		},
	}
}

// Execute выполняет обработку команды /commands
func (h *commandsCommandHandler) Execute(params handlers.HandlerParams) (handlers.HandlerResult, error) {
	message := h.createCommandsMessage()
	keyboard := h.createCommandsKeyboard()

	return handlers.HandlerResult{
		Message:  message,
		Keyboard: keyboard,
		Metadata: map[string]interface{}{
			"user_id": params.User.ID,
		},
	}, nil
}

// createCommandsMessage создает сообщение со списком команд
func (h *commandsCommandHandler) createCommandsMessage() string {
	commands := []struct {
		cmd         string
		description string
	}{
		{"/start", constants.CommandDescriptions.Start},
		{"/help", constants.CommandDescriptions.Help},
		{"/profile", constants.CommandDescriptions.Profile},
		{"/settings", constants.CommandDescriptions.Settings},
		{"/notifications", constants.CommandDescriptions.Notifications},
		{"/periods", constants.CommandDescriptions.Periods},
		{"/thresholds", constants.CommandDescriptions.Thresholds},
		{"/commands", constants.CommandDescriptions.Commands},
		{"/stats", constants.CommandDescriptions.Stats},
	}

	var sb strings.Builder
	sb.WriteString("📜 *Список доступных команд*\n\n")

	for _, cmd := range commands {
		sb.WriteString(fmt.Sprintf("%s - %s\n", cmd.cmd, cmd.description))
	}

	sb.WriteString("\n📌 *Как использовать:*\n")
	sb.WriteString("• Напишите команду в чат\n")
	sb.WriteString("• Используйте кнопки меню\n")
	sb.WriteString("• Введите `/` для автодополнения\n\n")
	sb.WriteString("ℹ️ Все команды доступны через меню бота в интерфейсе Telegram")

	return sb.String()
}

// createCommandsKeyboard создает клавиатуру для команды /commands
func (h *commandsCommandHandler) createCommandsKeyboard() interface{} {
	return map[string]interface{}{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": constants.CommandButtonTexts.Start, "callback_data": constants.CallbackMenuMain},
				{"text": constants.CommandButtonTexts.Help, "callback_data": constants.CallbackHelp},
			},
			{
				{"text": constants.CommandButtonTexts.Profile, "callback_data": constants.CallbackProfileMain},
				{"text": constants.CommandButtonTexts.Settings, "callback_data": constants.CallbackSettingsMain},
			},
			{
				{"text": constants.CommandButtonTexts.Notifications, "callback_data": constants.CallbackNotificationsMenu},
				{"text": constants.CommandButtonTexts.Periods, "callback_data": constants.CallbackPeriodsMenu},
			},
			{
				{"text": constants.CommandButtonTexts.Thresholds, "callback_data": constants.CallbackThresholdsMenu},
				{"text": constants.CommandButtonTexts.Stats, "callback_data": constants.CallbackStats},
			},
			{
				{"text": constants.CommandButtonTexts.Back, "callback_data": constants.CallbackMenuMain},
			},
		},
	}
}
