package customcommands

import (
	"iter"
	"sync"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/context"
)

var (
	commandManager     *CommandManager
	commandManagerOnce sync.Once
)

type CommandManager struct {
	commands []CustomCommand
}

func (cm *CommandManager) IterApplicable(ctx context.AppContext) iter.Seq[CustomCommand] {
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

func (cm *CommandManager) Iter() iter.Seq[CustomCommand] {
	return func(yield func(CustomCommand) bool) {
		for _, command := range cm.commands {
			if !yield(command) {
				return
			}
		}
	}
}

func GetCommandManager() *CommandManager {
	commandManagerOnce.Do(func() {
		var commands []CustomCommand
		for name, def := range config.Current.CustomCommands {
			commands = append(commands, newCustomCommand(name, def))
		}
		commandManager = &CommandManager{commands: commands}
	})
	return commandManager
}

func Matches(msg tea.KeyMsg) *CustomCommand {
	for _, v := range GetCommandManager().commands {
		if key.Matches(msg, v.Key) {
			return &v
		}
	}
	return nil
}
