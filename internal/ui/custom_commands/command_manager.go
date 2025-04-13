package customcommands

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/context"
	"iter"
)

var commandManager *CommandManager

type CommandManager struct {
	commands []CustomCommand
}

func (cm *CommandManager) Iter(ctx context.AppContext) iter.Seq[CustomCommand] {
	return func(yield func(CustomCommand) bool) {
		for _, command := range cm.commands {
			if !command.applicableTo(ctx.SelectedItem()) {
				continue
			}
			if !yield(command) {
				return
			}
		}
	}
}

func InitCommandManager() *CommandManager {
	var commands []CustomCommand
	for name, def := range config.Current.CustomCommands {
		commands = append(commands, NewCustomCommand(name, def))
	}
	commandManager = &CommandManager{commands: commands}
	return commandManager
}

func Matches(msg tea.KeyMsg) *CustomCommand {
	if commandManager == nil {
		InitCommandManager()
	}
	for _, v := range commandManager.commands {
		if key.Matches(msg, v.key) {
			return &v
		}
	}
	return nil
}
