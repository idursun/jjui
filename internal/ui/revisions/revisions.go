package revisions

import (
	"log"
	"strings"

	"github.com/idursun/jjui/internal/ui/ace_jump"
	"github.com/idursun/jjui/internal/ui/operations/duplicate"
	"github.com/idursun/jjui/internal/ui/operations/megamerge"
	"github.com/idursun/jjui/internal/ui/operations/revert"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/operations/describe"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/abandon"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/evolog"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/operations/squash"
)

type Model struct {
	width           int
	height          int
	context         *appContext.RevisionsContext
	keymap          config.KeyMappings[key.Binding]
	output          string
	err             error
	aceJump         *ace_jump.AceJump
	quickSearch     string
	previousOpLogId string
	isLoading       bool
	w               *graph.Renderer
	textStyle       lipgloss.Style
}

type revisionsMsg struct {
	msg tea.Msg
}

// Allow a message to be targetted to this component.
func RevisionsCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return revisionsMsg{msg: msg}
	}
}

func (m *Model) IsFocused() bool {
	if _, ok := m.context.Op.(common.Focusable); ok {
		return true
	}
	return false
}

func (m *Model) InNormalMode() bool {
	if _, ok := m.context.Op.(*operations.Default); ok {
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
	if op, ok := m.context.Op.(help.KeyMap); ok {
		return op.ShortHelp()
	}
	return (&operations.Default{}).ShortHelp()
}

func (m *Model) FullHelp() [][]key.Binding {
	if op, ok := m.context.Op.(help.KeyMap); ok {
		return op.FullHelp()
	}
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) SelectedRevisions() jj.SelectedRevisions {
	var selected []*jj.Commit
	ids := make(map[string]bool)
	items := m.context.GetCheckedItems()
	for _, item := range items {
		ids[item.Commit.CommitId] = true
	}
	for _, row := range m.context.Items {
		if _, ok := ids[row.Commit.CommitId]; ok {
			selected = append(selected, row.Commit)
		}
	}

	if len(selected) == 0 {
		return jj.NewSelectedRevisions(m.context.Current().Commit)
	}
	return jj.NewSelectedRevisions(selected...)
}

func (m *Model) Init() tea.Cmd {
	return common.RefreshAndSelect("@")
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if k, ok := msg.(revisionsMsg); ok {
		msg = k.msg
	}
	switch msg := msg.(type) {
	case common.CloseViewMsg:
		m.context.Op = operations.NewDefault()
		return m, m.updateSelection()
	case common.UpdateRevSetMsg:
		return m, common.Refresh
	case common.QuickSearchMsg:
		m.quickSearch = string(msg)
		m.context.Cursor = m.search(0)
		m.context.Op = operations.NewDefault()
		m.w.ResetViewRange()
		return m, nil
	case common.CommandCompletedMsg:
		m.output = msg.Output
		m.err = msg.Err
		return m, nil
	case common.AutoRefreshMsg:
		id, _ := m.context.RunCommandImmediate(jj.OpLogId(true))
		currentOperationId := string(id)
		log.Println("Previous operation ID:", m.previousOpLogId, "Current operation ID:", currentOperationId)
		if currentOperationId != m.previousOpLogId {
			m.previousOpLogId = currentOperationId
			return m, common.RefreshAndKeepSelections
		}
	case common.RefreshMsg:
		//if !msg.KeepSelections {
		//	m.context.ClearCheckedItems(reflect.TypeFor[appContext.SelectedRevision]())
		//}
		m.context.LoadRows(msg.SelectedRevision)
	case common.JumpToParentMsg:
		if msg.Commit == nil {
			return m, nil
		}
		m.context.JumpToParent(jj.NewSelectedRevisions(msg.Commit))
		return m, m.updateSelection()
	}

	// TODO: This is duplicated at the end of the function, needs refactoring
	if curSelected := m.context.Current(); curSelected != nil {
		if op, ok := m.context.Op.(operations.TracksSelectedRevision); ok {
			op.SetSelectedRevision(curSelected.Commit)
		}
	}

	if len(m.context.Items) == 0 {
		return m, nil
	}

	if cmd, ok := m.updateOperation(msg); ok {
		return m, cmd
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up):
			m.context.Prev()
		case key.Matches(msg, m.keymap.Down):
			m.context.Next()
		case key.Matches(msg, m.keymap.JumpToParent):
			m.context.JumpToParent(m.SelectedRevisions())
		case key.Matches(msg, m.keymap.JumpToChildren):
			immediate, _ := m.context.RunCommandImmediate(jj.GetFirstChild(m.context.Current().Commit))
			index := m.context.SelectRevision(string(immediate))
			if index != -1 {
				m.context.Cursor = index
			}
		case key.Matches(msg, m.keymap.JumpToWorkingCopy):
			workingCopyIndex := m.context.SelectRevision("@")
			if workingCopyIndex != -1 {
				m.context.Cursor = workingCopyIndex
			}
			return m, m.updateSelection()
		case key.Matches(msg, m.keymap.AceJump):
			m.aceJump = m.findAceKeys()
		default:
			if op, ok := m.context.Op.(operations.HandleKey); ok {
				cmd = op.HandleKey(msg)
				break
			}

			switch {
			case key.Matches(msg, m.keymap.ToggleSelect):
				commit := m.context.Current().Commit
				m.context.Current().Toggle()
				immediate, _ := m.context.RunCommandImmediate(jj.GetParent(jj.NewSelectedRevisions(commit)))
				parentIndex := m.context.SelectRevision(string(immediate))
				if parentIndex != -1 {
					m.context.Cursor = parentIndex
				}
			case key.Matches(msg, m.keymap.Cancel):
				m.context.Op = operations.NewDefault()
			case key.Matches(msg, m.keymap.QuickSearchCycle):
				m.context.Cursor = m.search(m.context.Cursor + 1)
				m.w.ResetViewRange()
				return m, nil
			case key.Matches(msg, m.keymap.Details.Mode):
				m.context.DetailsContext.Load(m.context.Current())
				m.context.Op, cmd = details.NewOperation(m.context)
			case key.Matches(msg, m.keymap.InlineDescribe.Mode):
				m.context.Op, cmd = describe.NewOperation(m.context, m.context.Current().Commit.GetChangeId(), m.width)
				return m, cmd
			case key.Matches(msg, m.keymap.New):
				cmd = m.context.RunCommand(jj.New(m.SelectedRevisions()), common.RefreshAndSelect("@"))
			case key.Matches(msg, m.keymap.Commit):
				cmd = m.context.RunInteractiveCommand(jj.CommitWorkingCopy(), common.Refresh)
			case key.Matches(msg, m.keymap.Edit, m.keymap.ForceEdit):
				ignoreImmutable := key.Matches(msg, m.keymap.ForceEdit)
				cmd = m.context.RunCommand(jj.Edit(m.context.Current().Commit.GetChangeId(), ignoreImmutable), common.Refresh)
			case key.Matches(msg, m.keymap.Diffedit):
				changeId := m.context.Current().Commit.GetChangeId()
				cmd = m.context.RunInteractiveCommand(jj.DiffEdit(changeId), common.Refresh)
			case key.Matches(msg, m.keymap.Absorb):
				changeId := m.context.Current().Commit.GetChangeId()
				cmd = m.context.RunCommand(jj.Absorb(changeId), common.Refresh)
			case key.Matches(msg, m.keymap.Abandon):
				selections := m.SelectedRevisions()
				m.context.Op = abandon.NewOperation(m.context, selections)
			case key.Matches(msg, m.keymap.Bookmark.Set):
				m.context.Op, cmd = bookmark.NewSetBookmarkOperation(m.context, m.context.Current().Commit.GetChangeId())
			case key.Matches(msg, m.keymap.Split):
				currentRevision := m.context.Current().Commit.GetChangeId()
				return m, m.context.RunInteractiveCommand(jj.Split(currentRevision, []string{}), common.Refresh)
			case key.Matches(msg, m.keymap.Describe):
				currentRevision := m.context.Current().Commit.GetChangeId()
				return m, m.context.RunInteractiveCommand(jj.Describe(currentRevision), common.Refresh)
			case key.Matches(msg, m.keymap.Evolog.Mode):
				m.context.Op, cmd = evolog.NewOperation(m.context, m.context.Current().Commit, m.width, m.height)
			case key.Matches(msg, m.keymap.Diff):
				return m, func() tea.Msg {
					changeId := m.context.Current().Commit.GetChangeId()
					output, _ := m.context.RunCommandImmediate(jj.Diff(changeId, ""))
					return common.ShowDiffMsg(output)
				}
			case key.Matches(msg, m.keymap.Refresh):
				cmd = common.Refresh
			case key.Matches(msg, m.keymap.Squash.Mode):
				selectedRevisions := m.SelectedRevisions()
				parent, _ := m.context.RunCommandImmediate(jj.GetParent(selectedRevisions))
				parentIdx := m.context.SelectRevision(string(parent))
				if parentIdx != -1 {
					m.context.Cursor = parentIdx
				} else if m.context.Cursor < len(m.context.Items)-1 {
					m.context.Cursor++
				}
				m.context.Op = squash.NewOperation(m.context, selectedRevisions)
			case key.Matches(msg, m.keymap.Revert.Mode):
				m.context.Op = revert.NewOperation(m.context, m.SelectedRevisions(), revert.TargetDestination)
			case key.Matches(msg, m.keymap.Rebase.Mode):
				m.context.Op = rebase.NewOperation(m.context, m.SelectedRevisions(), rebase.SourceRevision, rebase.TargetDestination)
			case key.Matches(msg, m.keymap.Duplicate.Mode):
				m.context.Op = duplicate.NewOperation(m.context, m.SelectedRevisions(), duplicate.TargetDestination)
			case key.Matches(msg, m.keymap.Megamerge):
				m.context.Op = megamerge.NewModel(m.context, m.context.Current().Commit)
			}
		}
	}

	if curSelected := m.context.Current(); curSelected != nil {
		if op, ok := m.context.Op.(operations.TracksSelectedRevision); ok {
			op.SetSelectedRevision(curSelected.Commit)
		}
		return m, tea.Batch(m.updateSelection(), cmd)
	}
	return m, cmd
}

func (m *Model) updateSelection() tea.Cmd {
	//if selectedRevision := m.context.Current().Commit; selectedRevision != nil {
	//	return m.context.SetSelectedItem(appContext.SelectedRevision{
	//		ChangeId: selectedRevision.GetChangeId(),
	//		CommitId: selectedRevision.CommitId,
	//	})
	//}
	return nil
}

func (m *Model) highlightChanges() tea.Msg {
	if m.err != nil || m.output == "" {
		return nil
	}

	changes := strings.Split(m.output, "\n")
	for _, change := range changes {
		if !strings.HasPrefix(change, " ") {
			continue
		}
		line := strings.Trim(change, "\n ")
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) > 0 {
			for i := range m.context.Items {
				row := m.context.Items[i]
				if row.Commit.GetChangeId() == parts[0] {
					row.IsAffected = true
					break
				}
			}
		}
	}
	return nil
}

func (m *Model) View() string {
	if len(m.context.Items) == 0 {
		if m.isLoading {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "loading")
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "(no matching revisions)")
	}

	checkedItems := m.context.GetCheckedItems()
	selections := make(map[string]bool)
	for _, checkedItem := range checkedItems {
		selections[checkedItem.Commit.GetChangeId()] = true
	}

	renderer := graph.NewDefaultRowIterator(m.context.AsRows(), graph.WithWidth(m.width), graph.WithStylePrefix("revisions"), graph.WithSelections(selections))
	renderer.Op = m.context.Op
	renderer.Cursor = m.context.Cursor
	renderer.SearchText = m.quickSearch
	renderer.AceJumpPrefix = m.aceJump.Prefix()

	m.w.SetSize(m.width, m.height)
	if config.Current.UI.Tracer.Enabled {
		start, end := m.w.FirstRowIndex(), m.w.LastRowIndex()+1 // +1 because the last row is inclusive in the view range
		log.Println("Visible row range:", start, end, "Cursor:", m.context.Cursor, "Total rows:", len(m.context.Items))
		renderer.Tracer = parser.NewTracer(m.context.AsRows(), m.context.Cursor, start, end)
	}
	output := m.w.Render(renderer)
	output = m.textStyle.MaxWidth(m.width).Render(output)
	return lipgloss.Place(m.width, m.height, 0, 0, output)
}

func (m *Model) search(startIndex int) int {
	if m.quickSearch == "" {
		return m.context.Cursor
	}

	n := len(m.context.Items)
	for i := startIndex; i < n+startIndex; i++ {
		c := i % n
		row := m.context.Items[c]
		for _, line := range row.Lines {
			for _, segment := range line.Segments {
				if segment.Text != "" && strings.Contains(segment.Text, m.quickSearch) {
					return c
				}
			}
		}
	}
	return m.context.Cursor
}

func (m *Model) GetCommitIds() []string {
	var commitIds []string
	for _, row := range m.context.Items {
		commitIds = append(commitIds, row.Commit.CommitId)
	}
	return commitIds
}

func New(c *appContext.MainContext) Model {
	keymap := config.Current.GetKeyMap()
	w := graph.NewRenderer(20, 10)
	return Model{
		context:   c.Revisions,
		w:         w,
		keymap:    keymap,
		width:     20,
		height:    10,
		textStyle: common.DefaultPalette.Get("revisions text"),
	}
}

func (m *Model) updateOperation(msg tea.Msg) (tea.Cmd, bool) {
	// HACK: Evolog operation with overlay but also change its mode from select to restore.
	// In 'select' mode, they function like standard overlays.
	// 'Restore' mode transforms them into rebase/squash-like operations.
	// This is currently a hack due to the lack of a mechanism to handle mode changes.
	// The 'restore' mode name was added to facilitate this special case.
	// Future refactoring will address mode changes more generically.
	if m.context.Op != nil && (m.context.Op.Name() == "restore" || m.context.Op.Name() == "target") {
		if _, ok := msg.(tea.KeyMsg); ok {
			return nil, false
		}
	}
	var cmd tea.Cmd
	if op, ok := m.context.Op.(operations.OperationWithOverlay); ok {
		m.context.Op, cmd = op.Update(msg)
		return cmd, true
	}
	return nil, false
}
