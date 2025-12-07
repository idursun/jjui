package context

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type CustomCommand interface {
	Binding() key.Binding
	Description(ctx *MainContext) string
	Prepare(ctx *MainContext) tea.Cmd
	IsApplicableTo(item SelectedItem) bool
}

type CustomCommandBase struct {
	Name string
	Key  []string `toml:"key"`
}

func (c CustomCommandBase) Binding() key.Binding {
	keys := strings.Join(c.Key, "|")
	return key.NewBinding(
		key.WithKeys(c.Key...),
		key.WithHelp(keys, c.Name),
	)
}

func LoadCustomCommands(output string) (map[string]CustomCommand, error) {
	type customCommandsToml struct {
		RawCustomCommands map[string]toml.Primitive `toml:"custom_commands"`
	}

	var registry = make(map[string]CustomCommand)

	var metadata toml.MetaData
	var err error

	var customCommands customCommandsToml
	metadata, err = toml.Decode(output, &customCommands)
	if err != nil {
		return nil, err
	}

	for name, primitive := range customCommands.RawCustomCommands {
		var tempMap map[string]interface{}
		if err := metadata.PrimitiveDecode(primitive, &tempMap); err != nil {
			return nil, fmt.Errorf("failed to decode custom command %s: %w", name, err)
		}

		if _, hasLua := tempMap["lua"]; hasLua {
			var cmd CustomLuaCommand
			if err := metadata.PrimitiveDecode(primitive, &cmd); err != nil {
				return nil, fmt.Errorf("failed to decode lua command %s: %w", name, err)
			}
			cmd.Name = name
			registry[name] = cmd
		} else if _, hasRevset := tempMap["revset"]; hasRevset {
			var cmd CustomRevsetCommand
			if err := metadata.PrimitiveDecode(primitive, &cmd); err != nil {
				return nil, fmt.Errorf("failed to decode revset command %s: %w", name, err)
			}
			cmd.Name = name
			registry[name] = cmd
		} else {
			var cmd CustomRunCommand
			if err := metadata.PrimitiveDecode(primitive, &cmd); err != nil {
				return nil, fmt.Errorf("failed to decode run command %s: %w", name, err)
			}
			cmd.Name = name
			registry[name] = cmd
		}
	}
	return registry, nil
}
