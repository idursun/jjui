package details

import (
	"bufio"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

type updateCommitStatusMsg struct {
	summary       string
	selectedFiles []string
}

var (
	_ operations.Operation                = (*Operation)(nil)
	_ operations.EmbeddedOperation        = (*Operation)(nil)
	_ operations.CheckedItemsSynchronizer = (*Operation)(nil)
	_ common.Focusable                    = (*Operation)(nil)
	_ common.Editable                     = (*Operation)(nil)
	_ common.Overlay                      = (*Operation)(nil)
	_ dispatch.ScopeProvider              = (*Operation)(nil)
)

type Operation struct {
	*DetailsList
	context      *context.MainContext
	Current      *jj.Commit
	revision     *jj.Commit
	confirmation *confirmation.Model
	styles       styles
}

func (s *Operation) IsOverlay() bool {
	return true
}

func (s *Operation) IsFocused() bool {
	return true
}

func (s *Operation) IsEditing() bool {
	return s.confirmation != nil
}

func (s *Operation) Scopes() []dispatch.Scope {
	var ret []dispatch.Scope
	if s.confirmation != nil {
		ret = append(ret, dispatch.Scope{
			Name:    actions.ScopeDetailsConfirmation,
			Leak:    dispatch.LeakNone,
			Handler: s,
		})
	}
	ret = append(ret, dispatch.Scope{
		Name:    actions.ScopeDetails,
		Leak:    dispatch.LeakGlobal,
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
	case updateCommitStatusMsg:
		items := s.createListItems(msg.summary, msg.selectedFiles)
		s.context.ClearCheckedItems(reflect.TypeFor[context.SelectedFile]())

		for _, it := range items {
			if it.selected {
				sel := context.SelectedFile{
					ChangeId: s.revision.GetChangeId(),
					CommitId: s.revision.CommitId,
					File:     it.fileName,
				}
				s.context.AddCheckedItem(sel)
			}
		}
		s.setItems(items)

		return s.updateSelection()
	default:
		oldCursor := s.cursor
		var cmds []tea.Cmd
		cmds = append(cmds, s.internalUpdate(msg))
		if s.cursor != oldCursor {
			cmds = append(cmds, s.updateSelection())
		}
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
			s.SyncCheckedItems()
		case msg.Ctrl:
			s.setCursor(msg.Index)
			if current := s.current(); current != nil {
				current.selected = !current.selected
				checkedFile := context.SelectedFile{
					ChangeId: s.revision.GetChangeId(),
					CommitId: s.revision.CommitId,
					File:     current.fileName,
				}
				if current.selected {
					s.context.AddCheckedItem(checkedFile)
				} else {
					s.context.RemoveCheckedItem(checkedFile)
				}
			}
		default:
			s.setCursor(msg.Index)
		}
		return s.updateSelection()
	case FileListScrollMsg:
		if msg.Horizontal {
			return nil
		}
		s.Scroll(msg.Delta)
		return nil
	case tea.KeyMsg:
		if s.confirmation != nil {
			return s.confirmation.Update(msg)
		}
		return nil
	case intents.Intent:
		cmd, _ := s.HandleIntent(msg)
		return cmd
	}
	return nil
}

func (s *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	oldCursor := s.cursor
	cmd, handled := s.handleIntentInner(intent)
	if handled && s.cursor != oldCursor {
		if selCmd := s.updateSelection(); selCmd != nil {
			return tea.Batch(cmd, selCmd), true
		}
	}
	return cmd, handled
}

func (s *Operation) updateSelection() tea.Cmd {
	current := s.current()
	if current == nil {
		return nil
	}
	return s.context.SetSelectedItem(context.SelectedFile{
		ChangeId: s.revision.GetChangeId(),
		CommitId: s.revision.CommitId,
		File:     current.fileName,
	})
}

func (s *Operation) handleIntentInner(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Apply:
		if s.confirmation != nil {
			return s.confirmation.Update(intent), true
		}
		return nil, true
	case intents.Cancel:
		if s.confirmation != nil {
			return s.confirmation.Update(intent), true
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
			output, _ := s.context.RunCommandImmediate(jj.Diff(s.revision.GetChangeId(), selected.fileName))
			return intents.DiffShow{Content: string(output)}
		}, true
	case intents.DetailsSplit:
		selectedFiles := s.getSelectedFiles(true)
		s.selectedHint = "stays as is"
		s.unselectedHint = "moves to the new revision"
		model := confirmation.New(
			[]string{"Are you sure you want to split the selected files?"},
			confirmation.WithStylePrefix("revisions"),
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
		return func() tea.Msg {
			return intents.OpenSquash{
				Selected: jj.NewSelectedRevisions(s.revision),
				Files:    s.getSelectedFiles(true),
			}
		}, true
	case intents.DetailsRestore:
		selectedFiles := s.getSelectedFiles(true)
		s.selectedHint = "gets restored"
		s.unselectedHint = "stays as is"
		model := confirmation.New(
			[]string{"Are you sure you want to restore the selected files?"},
			confirmation.WithStylePrefix("revisions"),
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
		s.selectedHint = "might get absorbed into parents"
		s.unselectedHint = "stays as is"
		model := confirmation.New(
			[]string{"Are you sure you want to absorb changes from the selected files?"},
			confirmation.WithStylePrefix("revisions"),
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
			isChecked := !current.selected
			current.selected = isChecked

			checkedFile := context.SelectedFile{
				ChangeId: s.revision.GetChangeId(),
				CommitId: s.revision.CommitId,
				File:     current.fileName,
			}
			if isChecked {
				s.context.AddCheckedItem(checkedFile)
			} else {
				s.context.RemoveCheckedItem(checkedFile)
			}

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
				if !s.files[i].selected {
					s.files[i].selected = true
					s.context.AddCheckedItem(context.SelectedFile{
						ChangeId: s.revision.GetChangeId(),
						CommitId: s.revision.CommitId,
						File:     intent.File,
					})
				}
				break
			}
		}
		return nil, true
	}
	return nil, false
}

func (s *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	background := lipgloss.NewStyle().Background(s.styles.Text.GetBackground())
	dl.AddFill(box.R, ' ', background, 0)
	s.renderIntoRect(dl, box.R)
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	s.Current = commit
	if commit == nil {
		return nil
	}
	if s.revision == nil || s.revision.GetChangeId() != commit.GetChangeId() {
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
	if s.Len() == 0 {
		return 1
	}
	confirmationHeight := 0
	if s.confirmation != nil {
		confirmationHeight = lipgloss.Height(s.confirmation.View())
	}
	return s.Len() + confirmationHeight
}

func (s *Operation) renderIntoRect(dl *render.DisplayContext, rect layout.Rectangle) int {
	if s.Len() == 0 {
		// Render "No changes" message
		content := s.styles.Dimmed.Render("No changes")
		dl.AddDraw(layout.Rect(rect.Min.X, rect.Min.Y, rect.Dx(), 1), content, 0)
		return 1
	}

	confirmationHeight := 0
	if s.confirmation != nil {
		confirmationHeight = lipgloss.Height(s.confirmation.View())
	}

	availableListHeight := max(rect.Dy()-confirmationHeight, 0)

	// Calculate available height
	height := min(availableListHeight, s.Len())

	// Render the file list to DisplayContext
	// viewRect is already absolute, so don't reapply the parent screen offset.
	viewRect := layout.Box{R: layout.Rect(rect.Min.X, rect.Min.Y, rect.Dx(), height)}
	s.RenderFileList(dl, viewRect)

	if s.confirmation != nil && confirmationHeight > 0 && height < rect.Dy() {
		confirmRect := layout.Rect(rect.Min.X, rect.Min.Y+height, rect.Dx(), confirmationHeight)
		s.confirmation.ViewRect(dl, layout.Box{R: confirmRect})
	}

	return height + confirmationHeight
}

func (s *Operation) Name() string {
	return "details"
}

func (s *Operation) SyncCheckedItems() {
	s.context.ClearCheckedItems(reflect.TypeFor[context.SelectedFile]())
	for _, f := range s.files {
		if f.selected {
			s.context.AddCheckedItem(context.SelectedFile{
				ChangeId: s.revision.GetChangeId(),
				CommitId: s.revision.CommitId,
				File:     f.fileName,
			})
		}
	}
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
		selectedFiles = append(selectedFiles, s.current().fileName)
		return selectedFiles
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
		var status status
		switch file[0] {
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
		fileName := file[2:]

		actualFileName := fileName
		if (status == Renamed || status == Copied) && strings.Contains(actualFileName, "{") {
			re := regexp.MustCompile(`\{[^}]*? => \s*([^}]*?)\s*\}`)
			actualFileName = path.Clean(re.ReplaceAllString(actualFileName, "$1"))
		}
		items = append(items, &item{
			status:   status,
			name:     fileName,
			fileName: actualFileName,
			selected: slices.ContainsFunc(selectedFiles, func(s string) bool { return s == actualFileName }),
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
	s := styles{
		Added:    common.DefaultPalette.Get("revisions details added"),
		Deleted:  common.DefaultPalette.Get("revisions details deleted"),
		Modified: common.DefaultPalette.Get("revisions details modified"),
		Renamed:  common.DefaultPalette.Get("revisions details renamed"),
		Copied:   common.DefaultPalette.Get("revisions details copied"),
		Selected: common.DefaultPalette.Get("revisions details selected"),
		Dimmed:   common.DefaultPalette.Get("revisions details dimmed"),
		Text:     common.DefaultPalette.Get("revisions details text"),
		Conflict: common.DefaultPalette.Get("revisions details conflict"),
	}

	l := NewDetailsList(s)
	op := &Operation{
		DetailsList: l,
		context:     context,
		revision:    selected,
		styles:      s,
	}
	return op
}
