package revisions

import (
	"log"
	"strings"

	"github.com/idursun/jjui/internal/ui/ace_jump"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/common/models"
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
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/abandon"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/evolog"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/operations/squash"
)

type RevisionList struct {
	*list.CheckableList[*models.RevisionItem]
	renderer      *list.ListRenderer[*models.RevisionItem]
	op            operations.Operation
	aceJump       *ace_jump.AceJump
	quickSearch   string
	dimmedStyle   lipgloss.Style
	checkStyle    lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	Tracer        parser.LaneTracer
}

type Model struct {
	*common.Sizeable
	*appContext.RevisionsContext
	*RevisionList
	context         *appContext.MainContext
	keymap          config.KeyMappings[key.Binding]
	output          string
	err             error
	previousOpLogId string
	isLoading       bool
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
	if f, ok := m.op.(common.Focusable); ok {
		return f.IsFocused()
	}
	return false
}

func (m *Model) InNormalMode() bool {
	if _, ok := m.op.(*operations.Default); ok {
		return true
	}
	return false
}

func (m *Model) ShortHelp() []key.Binding {
	if op, ok := m.op.(help.KeyMap); ok {
		return op.ShortHelp()
	}
	return (&operations.Default{}).ShortHelp()
}

func (m *Model) FullHelp() [][]key.Binding {
	if op, ok := m.op.(help.KeyMap); ok {
		return op.FullHelp()
	}
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) SelectedRevision() *jj.Commit {
	if current := m.Current(); current != nil {
		return current.Commit
	}
	return nil
}

func (m *Model) SelectedRevisions() jj.SelectedRevisions {
	checked := m.GetCheckedItems()
	if len(checked) == 0 {
		checked = append(checked, m.Current())
	}

	var selected []*jj.Commit
	for _, item := range checked {
		selected = append(selected, item.Commit)
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
		m.op = operations.NewDefault()
		return m, m.updateSelection()
	case common.QuickSearchMsg:
		m.quickSearch = string(msg)
		m.Cursor = m.search(0)
		m.op = operations.NewDefault()
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
		if !msg.KeepSelections {
			m.context.Revisions.Revisions.ClearCheckedItems()
			//m.context.ClearCheckedItems(reflect.TypeFor[appContext.SelectedRevision]())
		}
		m.isLoading = true
		cmd, _ := m.updateOperation(msg)
		if config.Current.Revisions.LogBatching {
			return m, tea.Batch(m.LoadStreaming(m.context.CurrentRevset, msg.SelectedRevision), cmd)
		} else {
			return m, tea.Batch(m.Load(m.context.CurrentRevset, msg.SelectedRevision), cmd)
		}
	case appContext.UpdateRevisionsMsg:
		m.isLoading = false
		m.updateGraphRows(msg.Rows, msg.SelectedRevision)
		return m, tea.Batch(m.highlightChanges, m.updateSelection(), func() tea.Msg {
			return common.UpdateRevisionsSuccessMsg{}
		})
	case appContext.AppendRowsBatchMsg:
		currentSelectedRevision := m.SelectedRevision()
		m.SetItems(msg.Rows)

		if m.Cursor == -1 && currentSelectedRevision != nil {
			m.Cursor = m.SelectRevision(currentSelectedRevision.GetChangeId())
		}

		if (m.Cursor < 0 || m.Cursor >= len(m.Items)) && len(m.Items) > 0 {
			m.Cursor = 0
		}

		cmds := []tea.Cmd{m.highlightChanges, m.updateSelection()}
		return m, tea.Batch(cmds...)
	case common.JumpToParentMsg:
		if msg.Commit == nil {
			return m, nil
		}
		m.JumpToParent(jj.NewSelectedRevisions(msg.Commit))
		return m, m.updateSelection()
	}

	// TODO: This is duplicated at the end of the function, needs refactoring
	if curSelected := m.SelectedRevision(); curSelected != nil {
		if op, ok := m.op.(operations.TracksSelectedRevision); ok {
			op.SetSelectedRevision(curSelected)
		}
	}

	if len(m.Items) == 0 {
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
			if m.Cursor > 0 {
				m.Cursor--
			}
		case key.Matches(msg, m.keymap.Down):
			m.Revisions.CursorDown()
		case key.Matches(msg, m.keymap.JumpToParent):
			m.JumpToParent(m.SelectedRevisions())
		case key.Matches(msg, m.keymap.JumpToChildren):
			immediate, _ := m.context.RunCommandImmediate(jj.GetFirstChild(m.SelectedRevision()))
			index := m.SelectRevision(string(immediate))
			if index != -1 {
				m.Cursor = index
			}
		case key.Matches(msg, m.keymap.JumpToWorkingCopy):
			workingCopyIndex := m.SelectRevision("@")
			if workingCopyIndex != -1 {
				m.Cursor = workingCopyIndex
			}
			return m, m.updateSelection()
		case key.Matches(msg, m.keymap.AceJump):
			m.aceJump = m.findAceKeys()
		default:
			if op, ok := m.op.(operations.HandleKey); ok {
				cmd = op.HandleKey(msg)
				break
			}

			switch {
			case key.Matches(msg, m.keymap.ToggleSelect):
				current := m.Current()
				current.Toggle()
				commit := current.Commit
				//changeId := commit.GetChangeId()
				//item := appContext.SelectedRevision{ChangeId: changeId, CommitId: commit.CommitId}
				//m.context.ToggleCheckedItem(item)
				m.JumpToParent(jj.NewSelectedRevisions(commit))
			case key.Matches(msg, m.keymap.Cancel):
				m.op = operations.NewDefault()
			case key.Matches(msg, m.keymap.QuickSearchCycle):
				m.Cursor = m.search(m.Cursor + 1)
				return m, nil
			case key.Matches(msg, m.keymap.Details.Mode):
				m.op, cmd = details.NewOperation(m.context, m.SelectedRevision())
			case key.Matches(msg, m.keymap.InlineDescribe.Mode):
				m.op, cmd = describe.NewOperation(m.context, m.SelectedRevision().GetChangeId(), m.Width)
				return m, cmd
			case key.Matches(msg, m.keymap.New):
				cmd = m.context.RunCommand(jj.New(m.SelectedRevisions()), common.RefreshAndSelect("@"))
			case key.Matches(msg, m.keymap.Commit):
				cmd = m.context.RunInteractiveCommand(jj.CommitWorkingCopy(), common.Refresh)
			case key.Matches(msg, m.keymap.Edit, m.keymap.ForceEdit):
				ignoreImmutable := key.Matches(msg, m.keymap.ForceEdit)
				cmd = m.context.RunCommand(jj.Edit(m.SelectedRevision().GetChangeId(), ignoreImmutable), common.Refresh)
			case key.Matches(msg, m.keymap.Diffedit):
				changeId := m.SelectedRevision().GetChangeId()
				cmd = m.context.RunInteractiveCommand(jj.DiffEdit(changeId), common.Refresh)
			case key.Matches(msg, m.keymap.Absorb):
				changeId := m.SelectedRevision().GetChangeId()
				cmd = m.context.RunCommand(jj.Absorb(changeId), common.Refresh)
			case key.Matches(msg, m.keymap.Abandon):
				selections := m.SelectedRevisions()
				m.op = abandon.NewOperation(m.context, selections)
			case key.Matches(msg, m.keymap.Bookmark.Set):
				m.op, cmd = bookmark.NewSetBookmarkOperation(m.context, m.SelectedRevision().GetChangeId())
			case key.Matches(msg, m.keymap.Split):
				currentRevision := m.SelectedRevision().GetChangeId()
				return m, m.context.RunInteractiveCommand(jj.Split(currentRevision, []string{}), common.Refresh)
			case key.Matches(msg, m.keymap.Describe):
				selections := m.SelectedRevisions()
				return m, m.context.RunInteractiveCommand(jj.Describe(selections), common.Refresh)
			case key.Matches(msg, m.keymap.Evolog.Mode):
				m.op, cmd = evolog.NewOperation(m.context, m.SelectedRevision(), m.Width, m.Height)
			case key.Matches(msg, m.keymap.Diff):
				return m, func() tea.Msg {
					changeId := m.SelectedRevision().GetChangeId()
					output, _ := m.context.RunCommandImmediate(jj.Diff(changeId, ""))
					return common.ShowDiffMsg(output)
				}
			case key.Matches(msg, m.keymap.Refresh):
				cmd = common.Refresh
			case key.Matches(msg, m.keymap.Squash.Mode):
				selectedRevisions := m.SelectedRevisions()
				parent, _ := m.context.RunCommandImmediate(jj.GetParent(selectedRevisions))
				parentIdx := m.SelectRevision(string(parent))
				if parentIdx != -1 {
					m.Cursor = parentIdx
				} else if m.Cursor < len(m.Items)-1 {
					m.Cursor++
				}
				m.op = squash.NewOperation(m.context, selectedRevisions)
			case key.Matches(msg, m.keymap.Revert.Mode):
				m.op = revert.NewOperation(m.context, m.SelectedRevisions(), revert.TargetDestination)
			case key.Matches(msg, m.keymap.Rebase.Mode):
				m.op = rebase.NewOperation(m.context, m.SelectedRevisions(), rebase.SourceRevision, rebase.TargetDestination)
			case key.Matches(msg, m.keymap.Duplicate.Mode):
				m.op = duplicate.NewOperation(m.context, m.SelectedRevisions(), duplicate.TargetDestination)
			case key.Matches(msg, m.keymap.Megamerge):
				m.op = megamerge.NewModel(m.context, m.SelectedRevision())
			}
		}
	}

	if curSelected := m.SelectedRevision(); curSelected != nil {
		if op, ok := m.op.(operations.TracksSelectedRevision); ok {
			op.SetSelectedRevision(curSelected)
		}
		return m, tea.Batch(m.updateSelection(), cmd)
	}
	return m, cmd
}

func (m *Model) updateSelection() tea.Cmd {
	if selectedRevision := m.SelectedRevision(); selectedRevision != nil {
		return m.context.SetSelectedItem(appContext.SelectedRevision{
			ChangeId: selectedRevision.GetChangeId(),
			CommitId: selectedRevision.CommitId,
		})
	}
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
			for i := range m.Items {
				row := m.Items[i]
				if row.Commit.GetChangeId() == parts[0] {
					row.IsAffected = true
					break
				}
			}
		}
	}
	return nil
}

func (m *Model) updateGraphRows(rows []*models.RevisionItem, selectedRevision string) {
	if rows == nil {
		rows = []*models.RevisionItem{}
	}

	currentSelectedRevision := selectedRevision
	if cur := m.SelectedRevision(); currentSelectedRevision == "" && cur != nil {
		currentSelectedRevision = cur.GetChangeId()
	}
	m.SetItems(rows)

	if len(m.Items) > 0 {
		m.Cursor = m.SelectRevision(currentSelectedRevision)
		if m.Cursor == -1 {
			m.Cursor = m.SelectRevision("@")
		}
		if m.Cursor == -1 {
			m.Cursor = 0
		}
	} else {
		m.Cursor = 0
	}
}

func (m *Model) View() string {
	if len(m.Items) == 0 {
		if m.isLoading {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "loading")
		}
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "(no matching revisions)")
	}

	selections := make(map[string]bool)
	checked := m.GetCheckedItems()
	for _, item := range checked {
		selections[item.Commit.GetChangeId()] = true
	}

	output := m.renderer.Render()
	output = m.textStyle.MaxWidth(m.Width).Render(output)
	return lipgloss.Place(m.Width, m.Height, 0, 0, output)
}

func (m *Model) search(startIndex int) int {
	if m.quickSearch == "" {
		return m.Cursor
	}

	n := len(m.Items)
	for i := startIndex; i < n+startIndex; i++ {
		c := i % n
		row := m.Items[c]
		for _, line := range row.Lines {
			for _, segment := range line.Segments {
				if segment.Text != "" && strings.Contains(segment.Text, m.quickSearch) {
					return c
				}
			}
		}
	}
	return m.Cursor
}

func (m *Model) CurrentOperation() operations.Operation {
	return m.op
}

func (m *Model) GetCommitIds() []string {
	var commitIds []string
	for _, row := range m.Items {
		commitIds = append(commitIds, row.Commit.CommitId)
	}
	return commitIds
}

func New(c *appContext.MainContext) Model {
	keymap := config.Current.GetKeyMap()
	l := c.Revisions.Revisions
	size := common.NewSizeable(20, 10)

	rl := &RevisionList{
		CheckableList: l,
		op:            operations.NewDefault(),
		aceJump:       nil,
		textStyle:     common.DefaultPalette.Get("revisions text"),
		selectedStyle: common.DefaultPalette.Get("revisions selected").Inline(true),
		dimmedStyle:   common.DefaultPalette.Get("revisions dimmed"),
		checkStyle:    common.DefaultPalette.Get("revisions success").Inline(true),
		Tracer:        parser.NewNoopTracer(),
	}
	rl.renderer = list.NewRenderer[*models.RevisionItem](l.List, rl.RenderItem, rl.GetItemHeight, size)
	return Model{
		RevisionsContext: c.Revisions,
		Sizeable:         size,
		RevisionList:     rl,
		context:          c,
		keymap:           keymap,
	}
}

func (m *Model) updateOperation(msg tea.Msg) (tea.Cmd, bool) {
	// HACK: Evolog operation with overlay but also change its mode from select to restore.
	// In 'select' mode, they function like standard overlays.
	// 'Restore' mode transforms them into rebase/squash-like operations.
	// This is currently a hack due to the lack of a mechanism to handle mode changes.
	// The 'restore' mode name was added to facilitate this special case.
	// Future refactoring will address mode changes more generically.
	if m.op != nil && (m.op.Name() == "restore" || m.op.Name() == "target") {
		if _, ok := msg.(tea.KeyMsg); ok {
			return nil, false
		}
	}
	var cmd tea.Cmd
	if op, ok := m.op.(operations.OperationWithOverlay); ok {
		m.op, cmd = op.Update(msg)
		return cmd, true
	}
	return nil, false
}
