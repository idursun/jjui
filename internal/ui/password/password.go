package password

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

var _ common.Model = (*Model)(nil)

type Model struct {
	*common.ViewNode
	textinput  textinput.Model
	passwordCh chan<- []byte

	context *appContext.MainContext
	styles  styles
}

type styles struct {
	border lipgloss.Style
}

func New(msg common.TogglePasswordMsg, parent *common.ViewNode) *Model {
	styles := styles{
		border: common.DefaultPalette.GetBorder("password border", lipgloss.NormalBorder()).Padding(1),
	}
	ti := textinput.New()
	ti.Prompt = msg.Prompt
	ti.EchoMode = textinput.EchoPassword
	ti.PromptStyle = common.DefaultPalette.Get("password title")
	ti.Focus()

	return &Model{
		ViewNode:   &common.ViewNode{Width: 0, Height: 0, Parent: parent},
		styles:     styles,
		textinput:  ti,
		passwordCh: msg.Password,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.TogglePasswordMsg:
		close(m.passwordCh)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return func() tea.Msg {
				return common.TogglePasswordMsg{}
			}
		case tea.KeyEnter:
			m.passwordCh <- []byte(m.textinput.Value())
			return func() tea.Msg {
				return common.TogglePasswordMsg{}
			}
		default:
			var cmd tea.Cmd
			m.textinput, cmd = m.textinput.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m *Model) View() string {
	pw, ph := m.Parent.Width, m.Parent.Height
	v := m.styles.border.Width(pw - 2).Render(m.textinput.View())
	w, h := lipgloss.Size(v)
	sx := (pw - w) / 2
	sy := (ph - h) / 2
	m.SetFrame(cellbuf.Rect(sx, sy, w, h))
	return v
}
