package password

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	textinput  textinput.Model
	passwordCh chan<- []byte
	styles     styles
}

type styles struct {
	border lipgloss.Style
}

func New(msg common.TogglePasswordMsg) *Model {
	styles := styles{
		border: common.DefaultPalette.GetBorder("password border", lipgloss.NormalBorder()).Padding(1),
	}
	ti := textinput.New()
	ti.Prompt = msg.Prompt
	ti.EchoMode = textinput.EchoPassword
	ti.PromptStyle = common.DefaultPalette.Get("password title")
	ti.Focus()

	return &Model{
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

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	pw, ph := box.R.Dx(), box.R.Dy()
	v := m.styles.border.Width(max(pw-2, 0)).Render(m.textinput.View())
	w, h := lipgloss.Size(v)
	sx := box.R.Min.X + max((pw-w)/2, 0)
	sy := box.R.Min.Y + max((ph-h)/2, 0)
	rect := cellbuf.Rect(sx, sy, w, h)
	dl.AddDraw(rect, v, 300)
}
