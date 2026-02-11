package ui

import (
	"strings"

	"github.com/pterm/pterm"
)

type Command struct {
	Name string
	Args string
}

type CommandDef struct {
	Name        string
	Description string
}

var AvailableCommands = []CommandDef{
	{Name: "/help", Description: "Показать список команд"},
	{Name: "/model", Description: "Сменить модель Gemini"},
	{Name: "/config", Description: "Открыть настройки"},
	{Name: "/clear", Description: "Очистить экран"},
	{Name: "/exit", Description: "Выйти из программы"},
}

func CommandNames() []string {
	names := make([]string, len(AvailableCommands))
	for i, cmd := range AvailableCommands {
		names[i] = cmd.Name
	}
	return names
}

func ParseCommand(input string) (Command, bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return Command{}, false
	}
	parts := strings.SplitN(input, " ", 2)
	cmd := Command{Name: strings.ToLower(parts[0])}
	if len(parts) > 1 {
		cmd.Args = strings.TrimSpace(parts[1])
	}
	return cmd, true
}

func PrintCommands() {
	pterm.Println(pterm.Gray("Доступные команды:"))
	for _, cmd := range AvailableCommands {
		pterm.Println(pterm.Cyan("  "+cmd.Name) + pterm.Gray("  "+cmd.Description))
	}
	pterm.Println()
}
