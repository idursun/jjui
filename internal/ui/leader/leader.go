package leader

import (
	"maps"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type Model struct {
	cancel  key.Binding
	shown   context.LeaderMap
	context *context.MainContext
}

func New(ctx *context.MainContext) *Model {
	keyMap := config.Current.GetKeyMap()
	m := &Model{
		context: ctx,
		cancel:  keyMap.Cancel,
	}
	return m
}

func (m *Model) ShortHelp() []key.Binding {
	bindings := []key.Binding{m.cancel}
	for m := range maps.Values(m.shown) {
		bindings = append(bindings, *m.Bind)
	}
	return bindings
}

func (m *Model) FullHelp() [][]key.Binding {
	bindings := slices.Collect(slices.Chunk(m.ShortHelp(), 6))
	return bindings
}

type initMsg struct{}

func InitCmd() tea.Msg {
	return initMsg{}
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initMsg:
		m.shown = contextEnabled(m.context, m.context.Leader)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.cancel):
			m.shown = nil
			return m, common.Close
		}
		for c := range maps.Values(m.shown) {
			if key.Matches(msg, *c.Bind) {
				if len(c.Nest) > 0 {
					m.shown = contextEnabled(m.context, c.Nest)
					return m, nil
				}
				m.shown = nil
				cmds := sendCmds(c.Send)
				return m, tea.Batch(
					common.Close,
					tea.Sequence(cmds...),
				)
			}
		}
	}
	return m, nil
}

func contextEnabled(ctx *context.MainContext, bnds context.LeaderMap) context.LeaderMap {
	bnds = maps.Clone(bnds)
	replacementKeys := slices.Collect(maps.Keys(ctx.CreateReplacements()))
	maps.DeleteFunc(bnds, func(k string, v *context.Leader) bool {
		if v == nil {
			return true
		}
		for _, key := range v.Context {
			if !slices.Contains(replacementKeys, key) {
				return true
			}
		}
		return false
	})
	return bnds
}

func sendCmds(strings []string) []tea.Cmd {
	var keyPresses []tea.KeyPressMsg
	for _, s := range strings {
		if k, ok := parseKey(s); ok {
			keyPresses = append(keyPresses, k)
		} else {
			for _, r := range s {
				keyPresses = append(keyPresses, tea.KeyPressMsg{
					Code: r,
					Text: string(r),
				})
			}
		}
	}
	var cmds []tea.Cmd
	for _, k := range keyPresses {
		cmds = append(cmds, func() tea.Msg {
			return k
		})
	}
	return cmds
}

func parseKey(keystroke string) (tea.KeyPressMsg, bool) {
	var keyPress tea.KeyPressMsg

	parts := strings.Split(keystroke, "+")
	for i, part := range parts {
		switch part {
		case "ctrl":
			keyPress.Mod |= tea.ModCtrl
		case "alt":
			keyPress.Mod |= tea.ModAlt
		case "shift":
			keyPress.Mod |= tea.ModShift
		case "enter":
			keyPress.Code = tea.KeyEnter
		case "tab":
			keyPress.Code = tea.KeyTab
		case "backspace":
			keyPress.Code = tea.KeyBackspace
		case "delete":
			keyPress.Code = tea.KeyDelete
		case "space":
			keyPress.Code = tea.KeySpace
		default:
			if i == len(parts)-1 && len(part) == 1 {
				keyPress.Code = rune(part[0])
				keyPress.Text = part
			} else {
				return keyPress, false
			}
		}
	}
	return keyPress, true
}
