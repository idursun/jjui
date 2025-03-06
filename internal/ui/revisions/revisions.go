package revisions

import (
	"bytes"
	"slices"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/abandon"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/evolog"
	"github.com/idursun/jjui/internal/ui/operations/git"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/operations/squash"
	"github.com/idursun/jjui/internal/ui/operations/undo"
	"github.com/idursun/jjui/internal/ui/revset"
)

type viewRange struct {
	start int
	end   int
}

var normalStyle = lipgloss.NewStyle()

type Model struct {
	rows        []jj.GraphRow
	op          operations.Operation
	revsetValue string
	viewRange   *viewRange
	cursor      int
	width       int
	height      int
	context     context.AppContext
	keymap      common.KeyMappings[key.Binding]
}

type updateRevisionsMsg struct {
	rows             []jj.GraphRow
	selectedRevision string
}

func (m *Model) IsFocused() bool {
	if _, ok := m.op.(common.Focusable); ok {
		return true
	}
	return false
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return m.height
}

func (m *Model) SetWidth(w int) {
	m.width = w
}

func (m *Model) SetHeight(h int) {
	m.height = h
}

func (m *Model) ShortHelp() []key.Binding {
	if op, ok := m.op.(help.KeyMap); ok {
		return op.ShortHelp()
	}
	return (&operations.Noop{}).ShortHelp()
}

func (m *Model) FullHelp() [][]key.Binding {
	if op, ok := m.op.(help.KeyMap); ok {
		return op.FullHelp()
	}
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) SelectedRevision() *jj.Commit {
	if m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor].Commit
}

func (m *Model) Init() tea.Cmd {
	return common.Refresh
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	curOp := m.op

	preSelectedRevision := m.SelectedRevision()
	switch msg := msg.(type) {
	case common.CloseViewMsg:
		m.op = operations.Default(m.context)
		m.context.SetSelectedItem(context.SelectedRevision{ChangeId: m.SelectedRevision().ChangeId})
		return m, tea.Batch(common.SelectionChanged, operations.OperationChanged(m.op))
	case operations.SetOperationMsg:
		m.op = msg.Operation
		return m, operations.OperationChanged(m.op)
	case revset.UpdateRevSetMsg:
		m.revsetValue = string(msg)
		if selectedRevision := m.SelectedRevision(); selectedRevision != nil {
			cmds = append(cmds, common.Refresh)
		} else {
			cmds = append(cmds, common.Refresh)
		}
	case common.RefreshMsg:
		return m, m.load(m.revsetValue, msg.SelectedRevision)
	case updateRevisionsMsg:
		if msg.rows != nil {
			currentSelectedRevision := msg.selectedRevision
			if cur := m.SelectedRevision(); currentSelectedRevision == "" && cur != nil {
				currentSelectedRevision = cur.GetChangeId()
			}
			m.rows = msg.rows
			m.cursor = m.selectRevision(currentSelectedRevision)
			if m.cursor == -1 {
				m.cursor = m.selectRevision("@")
			}
			if m.cursor == -1 {
				m.cursor = 0
			}
		}
	}

	if op, ok := m.op.(operations.OperationWithOverlay); ok {
		var cmd tea.Cmd
		if m.op, cmd = op.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keymap.Down):
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		default:
			if op, ok := m.op.(operations.HandleKey); ok {
				cmd = op.HandleKey(msg)
			} else {
				switch {
				case key.Matches(msg, m.keymap.Cancel):
					m.op = operations.Default(m.context)
				case key.Matches(msg, m.keymap.Details.Mode):
					m.op, cmd = details.NewOperation(m.context, m.SelectedRevision())
				case key.Matches(msg, m.keymap.Undo):
					m.op, cmd = undo.NewOperation(m.context)
				case key.Matches(msg, m.keymap.New):
					cmd = m.context.RunCommand(jj.New(m.SelectedRevision().GetChangeId()), common.RefreshAndSelect("@"))
				case key.Matches(msg, m.keymap.Edit):
					cmd = m.context.RunCommand(jj.Edit(m.SelectedRevision().GetChangeId()), common.Refresh)
				case key.Matches(msg, m.keymap.Diffedit):
					changeId := m.SelectedRevision().GetChangeId()
					cmd = m.context.RunInteractiveCommand(jj.DiffEdit(changeId), common.Refresh)
				case key.Matches(msg, m.keymap.Abandon):
					m.op, cmd = abandon.NewOperation(m.context, m.SelectedRevision())
				case key.Matches(msg, m.keymap.Split):
					currentRevision := m.SelectedRevision().GetChangeId()
					return m, m.context.RunInteractiveCommand(jj.Split(currentRevision, []string{}), common.Refresh)
				case key.Matches(msg, m.keymap.Describe):
					currentRevision := m.SelectedRevision().GetChangeId()
					return m, m.context.RunInteractiveCommand(jj.Describe(currentRevision), common.Refresh)
				case key.Matches(msg, m.keymap.Evolog):
					m.op, cmd = evolog.NewOperation(m.context, m.SelectedRevision().GetChangeId(), m.width, m.height)
				case key.Matches(msg, m.keymap.Diff):
					return m, func() tea.Msg {
						output, _ := m.context.RunCommandImmediate(jj.Diff(m.SelectedRevision().GetChangeId(), ""))
						return common.ShowDiffMsg(output)
					}
				case key.Matches(msg, m.keymap.Refresh):
					cmd = common.Refresh
				case key.Matches(msg, m.keymap.Git.Mode):
					m.op = git.NewOperation(m.context)
				case key.Matches(msg, m.keymap.Squash):
					m.op = squash.NewOperation(m.context, m.SelectedRevision().ChangeIdShort)
					if m.cursor < len(m.rows)-1 {
						m.cursor++
					}
				case key.Matches(msg, m.keymap.Rebase.Mode):
					m.op = rebase.NewOperation(m.context, m.SelectedRevision().ChangeIdShort, rebase.SourceRevision, rebase.TargetDestination)
				case key.Matches(msg, m.keymap.Bookmark.Mode):
					m.op = bookmark.NewChooseBookmarkOperation(m.context)
				case key.Matches(msg, m.keymap.Quit):
					return m, tea.Quit
				}
			}
		}
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if op, ok := m.op.(operations.TracksSelectedRevision); ok {
		op.SetSelectedRevision(m.SelectedRevision())
	}
	curSelected := m.SelectedRevision()
	if preSelectedRevision != curSelected {
		cmds = append(cmds, common.SelectionChanged)
		m.context.SetSelectedItem(context.SelectedRevision{ChangeId: curSelected.ChangeId})
	}
	if m.op.Name() != curOp.Name() {
		cmds = append(cmds, operations.OperationChanged(m.op))
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if len(m.rows) == 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "loading")
	}
	topView := ""
	topViewHeight := 0

	if m.op.RenderPosition() == operations.RenderPositionTop {
		topView = m.op.Render()
		topViewHeight = lipgloss.Height(topView)
	}

	h := m.height - topViewHeight
	viewHeight := m.viewRange.end - m.viewRange.start
	if viewHeight != h {
		m.viewRange.end = m.viewRange.start + h
	}

	var w graph.GraphWriter
	selectedLineStart := -1
	selectedLineEnd := -1
	for i, row := range m.rows {
		nodeRenderer := graph.DefaultRowRenderer{
			Palette:       common.DefaultPalette,
			Op:            m.op,
			IsHighlighted: i == m.cursor,
		}

		if i == m.cursor {
			selectedLineStart = w.LineCount()
		}
		w.RenderRow(row, nodeRenderer)
		if i == m.cursor {
			selectedLineEnd = w.LineCount()
		}
		if selectedLineEnd > 0 && w.LineCount() > h && w.LineCount() > m.viewRange.end {
			break
		}
	}

	if selectedLineStart <= m.viewRange.start {
		m.viewRange.start = selectedLineStart
		m.viewRange.end = selectedLineStart + h
	} else if selectedLineEnd > m.viewRange.end {
		m.viewRange.end = selectedLineEnd
		m.viewRange.start = selectedLineEnd - h
	}

	content := w.String(m.viewRange.start, m.viewRange.end)
	content = lipgloss.PlaceHorizontal(m.Width(), lipgloss.Left, content)

	if topViewHeight > 0 {
		return lipgloss.JoinVertical(0, topView, content)
	}
	return normalStyle.MaxWidth(m.width).Render(content)
}

func (m *Model) load(revset string, selectedRevision string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.Log(revset))
		if err != nil {
			return common.UpdateRevisionsFailedMsg(err)
		}
		p := jj.NewParser(bytes.NewReader(output))
		graphLines := p.Parse()
		return updateRevisionsMsg{graphLines, selectedRevision}
	}
}

func (m *Model) selectRevision(revision string) int {
	idx := slices.IndexFunc(m.rows, func(row jj.GraphRow) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return row.Commit.GetChangeId() == revision || row.Commit.ChangeIdShort == revision
	})
	return idx
}

func New(c context.AppContext) Model {
	v := viewRange{start: 0, end: 0}
	keymap := c.KeyMap()
	return Model{
		context:   c,
		keymap:    keymap,
		rows:      nil,
		viewRange: &v,
		op:        operations.Default(c),
		cursor:    0,
		width:     20,
		height:    10,
	}
}

func (m *Model) InOverlayEditing() bool {
	overlayEditingOpNames := []string{"bookmark"}
	return slices.Contains(overlayEditingOpNames, m.op.Name())
}
