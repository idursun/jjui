package context

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
)

// CustomLuaCommand executes a Lua script via the capability bridge.
type CustomLuaCommand struct {
	CustomCommandBase
	Script string `toml:"lua"`
}

func (c CustomLuaCommand) IsApplicableTo(item SelectedItem) bool {
	return true
}

func (c CustomLuaCommand) Description(ctx *MainContext) string {
	return fmt.Sprintf("lua: %s", c.Script)
}

func (c CustomLuaCommand) Prepare(ctx *MainContext) tea.Cmd {
	return func() tea.Msg {
		return common.RunLuaScriptMsg{Script: c.Script}
	}
}
