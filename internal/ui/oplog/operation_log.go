package oplog

import (
	"bytes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

type updateOpLogMsg struct {
	Rows []row
}

var _ list.IList = (*Model)(nil)
var _ list.IListCursor = (*Model)(nil)
var _ common.ContextProvider = (*Model)(nil)
var _ view.IHasActionMap = (*Model)(nil)

type Model struct {
	*common.Sizeable
	context       *context.MainContext
	renderer      *list.ListRenderer
	rows          []row
	cursor        int
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

func (m *Model) Cursor() int {
	return m.cursor
}

func (m *Model) SetCursor(index int) {
	if index < 0 || index >= len(m.rows) {
		return
	}
	m.cursor = index
	m.context.Router.ContinueAction("@oplog.cursor")
}

func (m *Model) Read(value string) string {
	switch value {
	case jj.OperationIdPlaceholder:
		if len(m.rows) > 0 {
			return m.rows[m.cursor].OperationId
		}
	}
	return ""
}

func (m *Model) GetActionMap() actions.ActionMap {
	return config.Current.GetBindings("oplog")
}

func (m *Model) Len() int {
	if m.rows == nil {
		return 0
	}
	return len(m.rows)
}

func (m *Model) GetItemRenderer(index int) list.IItemRenderer {
	item := m.rows[index]
	style := m.textStyle
	if index == m.cursor {
		style = m.selectedStyle
	}
	return &itemRenderer{
		row:   item,
		style: style,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.load()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "oplog.up":
			m.SetCursor(m.cursor - 1)
			return m, nil
		case "oplog.down":
			m.SetCursor(m.cursor + 1)
			return m, nil
		case "oplog.diff":
			return m, actions.InvokeAction(actions.Action{
				Id: "diff.show",
				Args: map[string]any{
					"jj": jj.OpShow(m.rows[m.cursor].OperationId),
				},
			})
		case "oplog.restore":
			return m, tea.Batch(m.context.RunCommand(jj.OpRestore(m.rows[m.cursor].OperationId), common.Refresh))
		}
	case updateOpLogMsg:
		m.rows = msg.Rows
		m.SetCursor(0)
		m.renderer.Reset()
	}
	return m, nil
}

func (m *Model) View() string {
	if m.rows == nil {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "loading")
	}

	m.renderer.Reset()
	m.renderer.SetWidth(m.Width)
	m.renderer.SetHeight(m.Height)
	content := m.renderer.Render(m.cursor)
	content = lipgloss.PlaceHorizontal(m.Width, lipgloss.Left, content)
	return m.textStyle.MaxWidth(m.Width).Render(content)
}

func (m *Model) load() tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.OpLog(config.Current.OpLog.Limit))
		if err != nil {
			panic(err)
		}

		rows := parseRows(bytes.NewReader(output))
		return updateOpLogMsg{Rows: rows}
	}
}

func New(context *context.MainContext, width int, height int) tea.Model {
	m := &Model{
		Sizeable:      &common.Sizeable{Width: width, Height: height},
		context:       context,
		rows:          nil,
		cursor:        -1,
		textStyle:     common.DefaultPalette.Get("oplog text"),
		selectedStyle: common.DefaultPalette.Get("oplog selected"),
	}
	m.renderer = list.NewRenderer(m, common.NewSizeable(width, height))
	return m
}
