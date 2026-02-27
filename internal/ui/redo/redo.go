package redo

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	confirmation *confirmation.Model
}

func (m *Model) StackedActionOwner() string {
	return actions.OwnerRedo
}

func (m *Model) Init() tea.Cmd {
	return m.confirmation.Init()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case intents.Apply:
		return m.confirmation.Update(confirmation.SelectOptionMsg{Index: 0})
	case intents.Cancel:
		return m.confirmation.Update(confirmation.SelectOptionMsg{Index: 1})
	}
	return m.confirmation.Update(msg)
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	v := m.confirmation.View()
	w, h := lipgloss.Size(v)
	pw, ph := box.R.Dx(), box.R.Dy()
	sx := box.R.Min.X + max((pw-w)/2, 0)
	sy := box.R.Min.Y + max((ph-h)/2, 0)
	frame := layout.Rect(sx, sy, w, h)
	m.confirmation.ViewRect(dl, layout.Box{R: frame})
}

func NewModel(context *context.MainContext) *Model {
	output, _ := context.RunCommandImmediate(jj.OpLog(1))
	lastOperation := lipgloss.NewStyle().PaddingBottom(1).Render(string(output))
	model := confirmation.New(
		[]string{lastOperation, "Are you sure you want to redo last change?"},
		confirmation.WithStylePrefix("redo"),
		confirmation.WithZIndex(render.ZDialogs),
		confirmation.WithOption("Yes", context.RunCommand(jj.Redo(), common.Refresh, common.Close), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", common.Close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
	)
	model.Styles.Border = common.DefaultPalette.GetBorder("redo border", lipgloss.NormalBorder()).Padding(1)
	return &Model{
		confirmation: model,
	}
}
