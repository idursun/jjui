package oplog

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/common/models"
	"github.com/idursun/jjui/internal/ui/context"
)

type updateOpLogMsg struct {
	Rows []*models.OperationLogItem
}

type OpLogList struct {
	*list.List[*models.OperationLogItem]
	renderer      *list.ListRenderer[*models.OperationLogItem]
	selectedStyle lipgloss.Style
	textStyle     lipgloss.Style
}

func (o *OpLogList) RenderItem(w io.Writer, index int) {
	row := o.Items[index]
	isHighlighted := index == o.Cursor

	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			if isHighlighted {
				fmt.Fprint(&lw, segment.Style.Inherit(o.selectedStyle).Render(segment.Text))
			} else {
				fmt.Fprint(&lw, segment.Style.Inherit(o.textStyle).Render(segment.Text))
			}
		}
		line := lw.String()
		if isHighlighted {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(o.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(o.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(o.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(o.textStyle.GetBackground())))
		}
		fmt.Fprint(w, "\n")
	}
}

func (o *OpLogList) GetItemHeight(index int) int {
	return len(o.Items[index].Lines)
}

type Model struct {
	*common.Sizeable
	*OpLogList
	context *context.MainContext
	keymap  config.KeyMappings[key.Binding]
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

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateOpLogMsg:
		m.Items = msg.Rows
		m.Cursor = 0
		m.renderer.Reset()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
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
				output, _ := m.context.RunCommandImmediate(jj.OpShow(m.Current().OperationId))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keymap.OpLog.Restore):
			return m, tea.Batch(common.Close, m.context.RunCommand(jj.OpRestore(m.Current().OperationId), common.Refresh))
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
		output, err := m.context.RunCommandImmediate(jj.OpLog(config.Current.OpLog.Limit))
		if err != nil {
			panic(err)
		}

		rows := parseRows(bytes.NewReader(output))
		return updateOpLogMsg{Rows: rows}
	}
}

func New(ctx *context.MainContext, width int, height int) *Model {
	ctx.ActiveList = context.ListOplog
	size := common.NewSizeable(width, height)

	keyMap := config.Current.GetKeyMap()
	l := ctx.OpLog
	ol := &OpLogList{
		List:          l,
		selectedStyle: common.DefaultPalette.Get("oplog selected"),
		textStyle:     common.DefaultPalette.Get("oplog text"),
	}
	ol.renderer = list.NewRenderer[*models.OperationLogItem](l, ol.RenderItem, ol.GetItemHeight, size)
	return &Model{
		OpLogList: ol,
		Sizeable:  size,
		context:   ctx,
		keymap:    keyMap,
	}
}
