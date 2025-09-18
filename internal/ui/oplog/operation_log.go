package oplog

import (
	"bytes"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

type updateOpLogMsg struct {
	Rows []*models.OperationLogItem
}

var _ view.IViewModel = (*Model)(nil)
var _ list.IListProvider = (*Model)(nil)

type Model struct {
	*OpLogList
	*view.ViewNode
	context *context.MainContext
	keymap  config.KeyMappings[key.Binding]
}

func (m *Model) CurrentItem() models.IItem {
	return m.OpLogList.Current()
}

func (m *Model) CheckedItems() []models.IItem {
	return nil
}

func (m *Model) GetId() view.ViewId {
	return "oplog"
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Height = v.ViewManager.Height
	v.Width = v.ViewManager.Width
	m.renderer.Sizeable = v.Sizeable
	v.Id = m.GetId()
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{m.keymap.Up, m.keymap.Down, m.keymap.Cancel, m.keymap.Diff, m.keymap.OpLog.Restore}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return m.load()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateOpLogMsg:
		m.Items = msg.Rows
		m.Cursor = 0
		m.renderer.Reset()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
		case key.Matches(msg, m.keymap.Preview.Mode):
			var cmds []tea.Cmd
			if previewView := m.ViewManager.GetView("preview"); previewView != nil {
				previewView.Visible = !previewView.Visible
				if previewView.Visible {
					cmds = append(cmds, previewView.Model.Init())
				}
			}
			m.ViewManager.Layout()
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keymap.Up):
			if m.Cursor > 0 {
				m.Cursor--
			}
		case key.Matches(msg, m.keymap.Down):
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
		case key.Matches(msg, m.keymap.Diff):
			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.OpShowArgs{Operation: *m.Current()}.GetArgs())
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keymap.OpLog.Restore):
			return m, tea.Batch(common.Close, m.context.RunCommand(jj.Args(jj.OpRestoreArgs{Operation: *m.Current()}), common.Refresh))
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if m.Items == nil {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "loading")
	}

	content := m.renderer.Render()
	content = lipgloss.PlaceHorizontal(m.Width, lipgloss.Left, content)
	return m.textStyle.MaxWidth(m.Width).Render(content)
}

func (m *Model) load() tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.Args(jj.OpLogArgs{
			NoGraph:         false,
			Limit:           config.Current.OpLog.Limit,
			GlobalArguments: jj.GlobalArguments{IgnoreWorkingCopy: true, Color: "always"},
		}))

		if err != nil {
			panic(err)
		}

		rows := parseRows(bytes.NewReader(output))
		return updateOpLogMsg{Rows: rows}
	}
}

func New(ctx *context.MainContext) view.IViewModel {
	size := view.NewSizeable(80, 20)

	keyMap := config.Current.GetKeyMap()
	l := ctx.OpLog
	ol := &OpLogList{
		List:          l,
		selectedStyle: common.DefaultPalette.Get("oplog selected"),
		textStyle:     common.DefaultPalette.Get("oplog text"),
	}
	ol.renderer = list.NewRenderer[*models.OperationLogItem](l, ol, size)
	m := &Model{
		OpLogList: ol,
		context:   ctx,
		keymap:    keyMap,
	}
	return m
}
