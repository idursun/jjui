package leader

import (
	"maps"
	"slices"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case initMsg:
		m.shown = contextEnabled(m.context, m.context.Leader)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.cancel):
			m.shown = nil
			return common.Close
		}
		for c := range maps.Values(m.shown) {
			if key.Matches(msg, *c.Bind) {
				if len(c.Nest) > 0 {
					m.shown = contextEnabled(m.context, c.Nest)
					return nil
				}
				m.shown = nil
				cmds := sendCmds(c.Send)
				return tea.Batch(
					common.Close,
					tea.Sequence(cmds...),
				)
			}
		}
	}
	return nil
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
	var cmds []tea.Cmd
	send := func(k tea.Key) {
		cmds = append(cmds, func() tea.Msg {
			return tea.KeyMsg(k)
		})
	}
	for _, s := range strings {
		if k, ok := keyNames[s]; ok {
			send(k)
		} else {
			for _, r := range s {
				send(tea.Key{
					Type:  tea.KeyRunes,
					Runes: []rune{r},
				})
			}
		}
	}
	return cmds
}

// From bubbletea's key.go. So that we can identify by their string.
// Notable Exception: tea.KeyRunes. Because we create them.
var keyTypes = []tea.KeyType{
	// Control keys.
	tea.KeyTab,
	tea.KeyEnter,
	tea.KeyEsc,
	tea.KeyBackspace,

	tea.KeyCtrlAt,
	tea.KeyCtrlA,
	tea.KeyCtrlB,
	tea.KeyCtrlC,
	tea.KeyCtrlD,
	tea.KeyCtrlE,
	tea.KeyCtrlF,
	tea.KeyCtrlG,
	tea.KeyCtrlH,
	tea.KeyCtrlJ,
	tea.KeyCtrlK,
	tea.KeyCtrlL,
	tea.KeyCtrlN,
	tea.KeyCtrlO,
	tea.KeyCtrlP,
	tea.KeyCtrlQ,
	tea.KeyCtrlR,
	tea.KeyCtrlS,
	tea.KeyCtrlT,
	tea.KeyCtrlU,
	tea.KeyCtrlV,
	tea.KeyCtrlW,
	tea.KeyCtrlX,
	tea.KeyCtrlY,
	tea.KeyCtrlZ,

	tea.KeyCtrlCloseBracket,
	tea.KeyCtrlCaret,
	tea.KeyCtrlUnderscore,
	tea.KeyCtrlBackslash,

	// Other keys.
	tea.KeyUp,
	tea.KeyDown,
	tea.KeyRight,
	tea.KeySpace,
	tea.KeyLeft,
	tea.KeyShiftTab,
	tea.KeyHome,
	tea.KeyEnd,
	tea.KeyCtrlHome,
	tea.KeyCtrlEnd,
	tea.KeyShiftHome,
	tea.KeyShiftEnd,
	tea.KeyCtrlShiftHome,
	tea.KeyCtrlShiftEnd,
	tea.KeyPgUp,
	tea.KeyPgDown,
	tea.KeyCtrlPgUp,
	tea.KeyCtrlPgDown,
	tea.KeyDelete,
	tea.KeyInsert,
	tea.KeyCtrlUp,
	tea.KeyCtrlDown,
	tea.KeyCtrlRight,
	tea.KeyCtrlLeft,
	tea.KeyShiftUp,
	tea.KeyShiftDown,
	tea.KeyShiftRight,
	tea.KeyShiftLeft,
	tea.KeyCtrlShiftUp,
	tea.KeyCtrlShiftDown,
	tea.KeyCtrlShiftLeft,
	tea.KeyCtrlShiftRight,
	tea.KeyF1,
	tea.KeyF2,
	tea.KeyF3,
	tea.KeyF4,
	tea.KeyF5,
	tea.KeyF6,
	tea.KeyF7,
	tea.KeyF8,
	tea.KeyF9,
	tea.KeyF10,
	tea.KeyF11,
	tea.KeyF12,
	tea.KeyF13,
	tea.KeyF14,
	tea.KeyF15,
	tea.KeyF16,
	tea.KeyF17,
	tea.KeyF18,
	tea.KeyF19,
	tea.KeyF20,
}

func keysFromTypes() map[string]tea.Key {
	m := map[string]tea.Key{}
	set := func(t tea.KeyType) {
		m[t.String()] = tea.Key{
			Type: t,
		}
	}
	for _, t := range keyTypes {
		set(t)
	}
	return m
}

var keyNames = keysFromTypes()
