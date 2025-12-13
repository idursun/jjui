package actiondispatch

import (
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"

	tea "github.com/charmbracelet/bubbletea"
)

type handler func(ctx *context.MainContext) tea.Cmd

var builtin = map[string]handler{
	"revisions.up": func(ctx *context.MainContext) tea.Cmd {
		return intents.Invoke(intents.Navigate{Delta: -1})
	},
	"revisions.down": func(ctx *context.MainContext) tea.Cmd {
		return intents.Invoke(intents.Navigate{Delta: 1})
	},
	"revisions.navigate_to_working_copy": func(ctx *context.MainContext) tea.Cmd {
		return intents.Invoke(intents.Navigate{Target: intents.TargetWorkingCopy})
	},
	"revisions.navigate_to_children": func(ctx *context.MainContext) tea.Cmd {
		return intents.Invoke(intents.Navigate{Target: intents.TargetChild})
	},
	"revisions.navigate_to_parent": func(ctx *context.MainContext) tea.Cmd {
		return intents.Invoke(intents.Navigate{Target: intents.TargetParent})
	},
	"preview.toggle": func(ctx *context.MainContext) tea.Cmd {
		return func() tea.Msg { return common.TogglePreviewMsg{} }
	},
}

// Cmd returns a built-in action command for the given name, if it exists.
func Cmd(name string, ctx *context.MainContext) tea.Cmd {
	if h, ok := builtin[name]; ok {
		return h(ctx)
	}
	return nil
}
