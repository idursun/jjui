package view

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/context"
)

var _ tea.Model = (*model)(nil)
var _ IHasActionMap = (*model)(nil)

type model struct {
	items   []string
	current int
	scope   string
	context *context.MainContext
}

func (m model) GetActionMap() map[string]actions.Action {
	return config.Current.GetBindings(m.scope)
}

func NewSimpleList(ctx *context.MainContext, scope string, items []string) tea.Model {
	return model{
		context: ctx,
		scope:   scope,
		items:   items,
		current: 0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "list.down":
			if m.current < len(m.items) {
				m.current++
			}
			return m, nil
		case "list.up":
			if m.current > 0 {
				m.current--
			}
			return m, nil
		case "list.select":
			m.context.Set("$selected", m.items[m.current])
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() string {
	var w strings.Builder
	for i, item := range m.items {
		prefix := "  "
		if i == m.current {
			prefix = "> "
		}
		w.WriteString(prefix + item + "\n")
	}
	return w.String()
}
