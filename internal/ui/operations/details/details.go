package details

import (
	"bufio"
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

type updateCommitStatusMsg struct {
	summary       string
	selectedFiles []string
}

type filterState uint8

const (
	filterOff filterState = iota
	filterEditing
	filterApplied
)

var (
	_ operations.Operation         = (*Operation)(nil)
	_ operations.EmbeddedOperation = (*Operation)(nil)
	_ common.Focusable             = (*Operation)(nil)
	_ common.Editable              = (*Operation)(nil)
	_ common.Overlay               = (*Operation)(nil)
	_ common.ScopeProvider         = (*Operation)(nil)
	_ common.SelectionProvider     = (*Operation)(nil)
)

type Operation struct {
	*DetailsList
	context      *context.MainContext
	Current      *jj.Commit
	revision     *jj.Commit
	confirmation *confirmation.Model
	filterInput  textinput.Model
	filterState  filterState
}

func (s *Operation) IsOverlay() bool {
	return true
}

func (s *Operation) IsFocused() bool {
	return true
}

func (s *Operation) IsEditing() bool {
	return s.confirmation != nil || s.filterState == filterEditing
}

func (s *Operation) Scopes() []common.Scope {
	var ret []common.Scope
	if s.confirmation != nil {
		ret = append(ret, common.Scope{
			Name:    actions.ScopeDetailsConfirmation,
			Leak:    common.LeakNone,
			Handler: s,
		})
	}
	if s.filterState != filterOff {
		leak := common.LeakAll
		if s.filterState == filterEditing {
			leak = common.LeakNone
		}
		ret = append(ret, common.Scope{
			Name:    actions.ScopeDetails + ".filter",
			Leak:    leak,
			Handler: s,
		})
	}
	ret = append(ret, common.Scope{
		Name:    actions.ScopeDetails,
		Leak:    common.LeakGlobal,
		Handler: s,
	})
	return ret
}

func (s *Operation) Init() tea.Cmd {
	return s.load(s.revision.GetChangeId())
}

func (s *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case confirmation.CloseMsg:
		s.confirmation = nil
		s.selectedHint = ""
		s.unselectedHint = ""
		return nil
	case common.RefreshMsg:
		return s.load(s.revision.GetChangeId())
	case common.SelectionChangedMsg:
		selected, ok := msg.Item.(common.SelectedRevision)
		if !ok {
			return nil
		}
		return s.setSelectedRevision(&jj.Commit{ChangeId: selected.ChangeId, CommitId: selected.CommitId})
	case updateCommitStatusMsg:
		items := s.createListItems(msg.summary, msg.selectedFiles)
		s.setItems(items)
		return nil
	default:
		var cmds []tea.Cmd
		cmds = append(cmds, s.internalUpdate(msg))
		return tea.Batch(cmds...)
	}
}

func (s *Operation) internalUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case confirmation.SelectOptionMsg:
		if s.confirmation != nil {
			return s.confirmation.Update(msg)
		}
		return nil
	case FileClickedMsg:
		switch {
		case msg.Alt:
			prevCursor := s.cursor
			s.setCursor(msg.Index)
			s.rangeSelect(prevCursor, msg.Index)
		case msg.Ctrl:
			s.setCursor(msg.Index)
			if current := s.current(); current != nil {
				current.selected = !current.selected
			}
		default:
			s.setCursor(msg.Index)
		}
		return nil
	case FileListScrollMsg:
		if msg.Horizontal {
			return nil
		}
		s.Scroll(msg.Delta)
		return nil
	case tea.KeyMsg, tea.PasteMsg:
		if s.confirmation != nil {
			return s.confirmation.Update(msg)
		}
		if s.filterState == filterEditing {
			var cmd tea.Cmd
			s.filterInput, cmd = s.filterInput.Update(msg)
			s.setFilter(s.filterInput.Value(), true)
			return cmd
		}
		return nil
	case intents.Intent:
		cmd, _ := s.HandleIntent(msg)
		return cmd
	}
	return nil
}

func (s *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Apply:
		if s.confirmation != nil {
			return s.confirmation.Update(intent), true
		}
		if s.filterState == filterEditing {
			s.applyFilter()
		}
		return nil, true
	case intents.Cancel:
		if s.confirmation != nil {
			return s.confirmation.Update(intent), true
		}
		if s.filterState != filterOff {
			s.clearFilter()
		}
		return nil, true
	case intents.OptionSelect:
		if s.confirmation != nil {
			return s.confirmation.Update(intent), true
		}
		return nil, true
	case intents.DetailsNavigate:
		s.navigate(intent.Delta, intent.IsPage)
		return nil, true
	case intents.DetailsClose:
		return common.Close, true
	case intents.DetailsOpenFilter:
		return s.openFilter(), true
	case intents.DetailsApplyFilter:
		if s.filterState == filterEditing {
			s.applyFilter()
		}
		return nil, true
	case intents.DetailsCancelFilter:
		if s.filterState != filterOff {
			s.clearFilter()
		}
		return nil, true
	case intents.Quit:
		return common.Quit(), true
	case intents.Refresh:
		return common.Refresh, true
	case intents.DetailsDiff:
		selected := s.current()
		if selected == nil {
			return nil, true
		}
		return func() tea.Msg {
			args := jj.Diff(s.revision.GetChangeId(), "")
			output, _ := s.context.RunCommandImmediate(jj.Diff(s.revision.GetChangeId(), selected.fileName))
			return intents.DiffShow{Content: string(output), Args: args}
		}, true
	case intents.DetailsSplit:
		selectedFiles := s.getSelectedFiles(true)
		if len(selectedFiles) == 0 {
			return nil, true
		}
		s.selectedHint = "stays as is"
		s.unselectedHint = "moves to the new revision"
		model := confirmation.New(
			[]string{"Are you sure you want to split the selected files?"},
			confirmation.WithStyleScope("revisions"),
			confirmation.WithOption("Yes",
				tea.Batch(s.context.RunInteractiveCommand(jj.Split(s.revision.GetChangeId(), selectedFiles, intent.IsParallel, false), common.Refresh), common.Close),
				key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
			confirmation.WithOption("Interactive",
				tea.Batch(s.context.RunInteractiveCommand(jj.Split(s.revision.GetChangeId(), selectedFiles, intent.IsParallel, true), common.Refresh), common.Close),
				key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "interactive"))),
			confirmation.WithOption("No",
				confirmation.Close,
				key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		)
		s.confirmation = model
		return s.confirmation.Init(), true
	case intents.DetailsSquash:
		selectedFiles := s.getSelectedFiles(true)
		if len(selectedFiles) == 0 {
			return nil, true
		}
		return func() tea.Msg {
			return intents.OpenSquash{
				Selected: jj.NewSelectedRevisions(s.revision),
				Files:    selectedFiles,
			}
		}, true
	case intents.DetailsRestore:
		selectedFiles := s.getSelectedFiles(true)
		if len(selectedFiles) == 0 {
			return nil, true
		}
		s.selectedHint = "gets restored"
		s.unselectedHint = "stays as is"
		model := confirmation.New(
			[]string{"Are you sure you want to restore the selected files?"},
			confirmation.WithStyleScope("revisions"),
			confirmation.WithOption("Yes",
				tea.Batch(s.context.RunCommand(jj.Restore(s.revision.GetChangeId(), selectedFiles, false), common.Refresh), confirmation.Close),
				key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
			confirmation.WithOption("Interactive",
				tea.Batch(s.context.RunInteractiveCommand(jj.Restore(s.revision.GetChangeId(), selectedFiles, true), common.Refresh), common.Close),
				key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "interactive"))),
			confirmation.WithOption("No",
				confirmation.Close,
				key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		)
		s.confirmation = model
		return s.confirmation.Init(), true
	case intents.DetailsAbsorb:
		selectedFiles := s.getSelectedFiles(true)
		if len(selectedFiles) == 0 {
			return nil, true
		}
		s.selectedHint = "might get absorbed into parents"
		s.unselectedHint = "stays as is"
		model := confirmation.New(
			[]string{"Are you sure you want to absorb changes from the selected files?"},
			confirmation.WithStyleScope("revisions"),
			confirmation.WithOption("Yes",
				s.context.RunCommand(jj.Absorb(s.revision.GetChangeId(), nil, selectedFiles...), common.Refresh, confirmation.Close),
				key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
			confirmation.WithOption("No",
				confirmation.Close,
				key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		)
		s.confirmation = model
		return s.confirmation.Init(), true
	case intents.DetailsToggleSelect:
		if current := s.current(); current != nil {
			current.selected = !current.selected
			s.navigate(1, false)
		}
		return nil, true
	case intents.DetailsRevisionsChangingFile:
		if current := s.current(); current != nil {
			return tea.Batch(common.Close, common.UpdateRevSet(fmt.Sprintf("files(%s)", jj.EscapeFileName(current.fileName)))), true
		}
		return nil, true
	case intents.DetailsSelectFile:
		for i := range s.files {
			if s.files[i].fileName == intent.File {
				s.files[i].selected = true
				break
			}
		}
		return nil, true
	}
	return nil, false
}

func (s *Operation) Selection() common.SelectionSnapshot {
	var snapshot common.SelectionSnapshot
	current := s.current()
	if current != nil {
		snapshot.Highlighted = s.selectedFile(current.fileName)
	}
	for _, file := range s.files {
		if file.selected {
			snapshot.Checked = append(snapshot.Checked, s.selectedFile(file.fileName))
		}
	}
	return snapshot
}

func (s *Operation) selectedFile(file string) context.SelectedFile {
	return context.SelectedFile{
		ChangeId: s.revision.GetChangeId(),
		CommitId: s.revision.CommitId,
		File:     file,
	}
}

func (s *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	textStyle := common.DefaultPalette.Get("revisions", "details", "text", false)
	background := lipgloss.NewStyle().Background(textStyle.GetBackground())
	dl.AddFill(box.R, ' ', background, 0)
	s.renderIntoRect(dl, box.R)
}

func (s *Operation) setSelectedRevision(commit *jj.Commit) tea.Cmd {
	sameCurrent := s.Current.Equal(commit)
	sameRevision := s.revision.Equal(commit)
	if sameCurrent && sameRevision {
		return nil
	}
	s.Current = commit
	if commit == nil {
		return nil
	}
	if !sameRevision {
		s.revision = commit
		return s.load(commit.GetChangeId())
	}
	return nil
}

func (s *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	return ""
}

func (s *Operation) CanEmbed(commit *jj.Commit, pos operations.RenderPosition) bool {
	isSelected := s.Current != nil && s.Current.GetChangeId() == commit.GetChangeId()
	return isSelected && pos == operations.RenderPositionAfter
}

func (s *Operation) EmbeddedHeight(commit *jj.Commit, pos operations.RenderPosition, _ int) int {
	if !s.CanEmbed(commit, pos) {
		return 0
	}
	contentHeight := max(s.VisibleLen(), 1)
	if s.filterState != filterOff {
		contentHeight++
	}
	confirmationHeight := 0
	if s.confirmation != nil {
		confirmationHeight = lipgloss.Height(s.confirmation.View())
	}
	return contentHeight + confirmationHeight
}

func (s *Operation) renderIntoRect(dl *render.DisplayContext, rect layout.Rectangle) int {
	confirmationHeight := 0
	if s.confirmation != nil {
		confirmationHeight = lipgloss.Height(s.confirmation.View())
	}

	availableContentHeight := max(rect.Dy()-confirmationHeight, 0)
	contentY := rect.Min.Y
	filterHeight := 0
	if s.filterState != filterOff && availableContentHeight > 0 {
		s.renderFilterInput(dl, layout.Rect(rect.Min.X, contentY, rect.Dx(), 1))
		filterHeight = 1
		contentY++
		availableContentHeight--
	}

	visibleLen := s.VisibleLen()
	listHeight := min(availableContentHeight, max(visibleLen, 1))

	if listHeight > 0 {
		if visibleLen == 0 {
			dimmedStyle := common.DefaultPalette.Get("revisions", "details", "dimmed", false)
			message := "No matching files"
			if s.Len() == 0 {
				message = "No changes"
			}
			dl.AddDraw(layout.Rect(rect.Min.X, contentY, rect.Dx(), 1), dimmedStyle.Render(message), 0)
		} else {
			// viewRect is already absolute, so don't reapply the parent screen offset.
			viewRect := layout.Box{R: layout.Rect(rect.Min.X, contentY, rect.Dx(), listHeight)}
			s.RenderFileList(dl, viewRect)
		}
	}

	contentHeight := filterHeight + listHeight
	if s.confirmation != nil && confirmationHeight > 0 && contentHeight < rect.Dy() {
		confirmRect := layout.Rect(rect.Min.X, rect.Min.Y+contentHeight, rect.Dx(), confirmationHeight)
		s.confirmation.ViewRect(dl, layout.Box{R: confirmRect})
	}

	return contentHeight + confirmationHeight
}

func (s *Operation) renderFilterInput(dl *render.DisplayContext, rect layout.Rectangle) {
	textStyle := common.DefaultPalette.Get("revisions", "details", "text", false)
	dimmedStyle := common.DefaultPalette.Get("revisions", "details", "dimmed", false)
	styles := s.filterInput.Styles()
	styles.Focused.Prompt = dimmedStyle
	styles.Focused.Text = textStyle
	styles.Blurred.Prompt = dimmedStyle
	styles.Blurred.Text = textStyle
	s.filterInput.SetStyles(styles)
	s.filterInput.SetWidth(max(rect.Dx(), 0))
	dl.AddDraw(rect, s.filterInput.View(), 0)
	if s.filterState == filterEditing {
		dl.SetCursorInRect(s.filterInput.Cursor(), rect, 0, 0)
	}
}

func (s *Operation) openFilter() tea.Cmd {
	s.filterState = filterEditing
	s.setFilter(s.filterInput.Value(), true)
	s.filterInput.Focus()
	s.filterInput.CursorEnd()
	return textinput.Blink
}

func (s *Operation) applyFilter() {
	if strings.TrimSpace(s.filterInput.Value()) == "" {
		s.clearFilter()
		return
	}
	s.filterState = filterApplied
	s.filterInput.Blur()
	s.setFilter(s.filterInput.Value(), true)
}

func (s *Operation) clearFilter() {
	s.filterState = filterOff
	s.filterInput.Reset()
	s.filterInput.Blur()
	s.setFilter("", false)
}

func (s *Operation) Name() string {
	return "details"
}

func (s *Operation) getSelectedFiles(allowVirtualSelection bool) []string {
	selectedFiles := make([]string, 0)
	if len(s.files) == 0 {
		return selectedFiles
	}

	for _, f := range s.files {
		if f.selected {
			selectedFiles = append(selectedFiles, f.fileName)
		}
	}
	if len(selectedFiles) == 0 && allowVirtualSelection {
		if current := s.current(); current != nil {
			selectedFiles = append(selectedFiles, current.fileName)
		}
	}
	return selectedFiles
}

func (s *Operation) createListItems(content string, selectedFiles []string) []*item {
	var items []*item
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Split(bufio.ScanWords)
	var conflicts []bool
	for scanner.Scan() {
		field := scanner.Text()
		if field == "$" {
			break
		}
		conflicts = append(conflicts, field == "true")
	}

	_, after, _ := strings.Cut(content, "$")
	scanner = bufio.NewScanner(strings.NewReader(after))
	index := 0
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file == "" {
			continue
		}
		summary, ok := jj.ParseSummaryFile(file)
		if !ok {
			continue
		}
		var status status
		switch summary.Status {
		case 'A':
			status = Added
		case 'D':
			status = Deleted
		case 'M':
			status = Modified
		case 'R':
			status = Renamed
		case 'C':
			status = Copied
		}
		items = append(items, &item{
			status:   status,
			name:     summary.Name,
			fileName: summary.FileName,
			selected: slices.ContainsFunc(selectedFiles, func(s string) bool { return s == summary.FileName }),
			conflict: conflicts[index],
		})
		index++
	}
	return items
}

func (s *Operation) load(revision string) tea.Cmd {
	output, err := s.context.RunCommandImmediate(jj.Snapshot())
	if err == nil {
		output, err = s.context.RunCommandImmediate(jj.Status(revision))
		if err == nil {
			return func() tea.Msg {
				summary := string(output)
				selectedFiles := s.getSelectedFiles(false)
				return updateCommitStatusMsg{summary, selectedFiles}
			}
		}
	}
	return func() tea.Msg {
		return common.CommandCompletedMsg{
			Output: string(output),
			Err:    err,
		}
	}
}

func NewOperation(context *context.MainContext, selected *jj.Commit) *Operation {
	l := NewDetailsList()
	filterInput := textinput.New()
	filterInput.Prompt = "/ "
	filterInput.CharLimit = 0
	filterInput.SetVirtualCursor(false)
	op := &Operation{
		DetailsList: l,
		context:     context,
		revision:    selected,
		Current:     selected,
		filterInput: filterInput,
	}
	return op
}
