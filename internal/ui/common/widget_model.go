package common

import tea "github.com/charmbracelet/bubbletea/v2"

type WidgetModel interface {
	tea.Model
	tea.ViewModel
}

type miniModel struct {
}

var _ WidgetModel = miniModel{}

func (m miniModel) Init() tea.Cmd {
	//TODO implement me
	panic("implement me")
}

func (m miniModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	//TODO implement me
	panic("implement me")
}

func (m miniModel) View() string {
	//TODO implement me
	panic("implement me")
}
