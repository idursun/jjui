package diff

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IViewModel = (*Model)(nil)

type updateDiffContentMsg string

type Model struct {
	*view.ViewNode
	view        viewport.Model
	keymap      config.KeyMappings[key.Binding]
	commandArgs jj.CommandArgs
	context     *context.MainContext
}

func (m *Model) GetId() view.ViewId {
	return "diff"
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	m.view.Width = v.Width
	m.view.Height = v.Height
	v.ViewOpts.Sizeable.SetWidth(m.view.Width)
	v.ViewOpts.Sizeable.SetHeight(m.view.Height)
	v.ViewOpts.Id = m.GetId()
}

func (m *Model) ShortHelp() []key.Binding {
	vkm := m.view.KeyMap
	return []key.Binding{
		vkm.Up, vkm.Down, vkm.HalfPageDown, vkm.HalfPageUp, vkm.PageDown, vkm.PageUp,
		m.keymap.Cancel}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	if m.commandArgs != nil {
		return func() tea.Msg {
			output, _ := m.context.RunCommandImmediate(m.commandArgs)
			return updateDiffContentMsg(output)
		}
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateDiffContentMsg:
		content := strings.ReplaceAll(string(msg), "\r", "")
		if content == "" {
			content = "(empty)"
		}
		m.view.SetContent(content)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
		}
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	m.view.Height = m.Sizeable.Height
	m.view.Width = m.Sizeable.Width
	return m.view.View()
}

type Option func(*Model)

func WithCommand(args jj.CommandArgs) Option {
	return func(m *Model) {
		m.commandArgs = args
	}
}

func WithOutput(output string) Option {
	return func(m *Model) {
		if output == "" {
			output = "(empty)"
		}
		m.view.SetContent(output)
	}
}

func New(ctx *context.MainContext, opts ...Option) view.IViewModel {
	v := viewport.New(0, 0)
	v.SetContent("(empty)")
	m := &Model{
		context: ctx,
		view:    v,
		keymap:  config.Current.GetKeyMap(),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}
