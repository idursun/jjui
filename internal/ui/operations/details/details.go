package details

import (
	"bufio"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
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

var (
	_ operations.Operation = (*Operation)(nil)
	_ common.Focusable     = (*Operation)(nil)
	_ common.Overlay       = (*Operation)(nil)
)

type Operation struct {
	*DetailsList
	context           *context.MainContext
	Current           *jj.Commit
	keymap            config.KeyMappings[key.Binding]
	targetMarkerStyle lipgloss.Style
	revision          *jj.Commit
	confirmation      *confirmation.Model
	keyMap            config.KeyMappings[key.Binding]
	styles            styles
}

func (s *Operation) IsOverlay() bool {
	return true
}

func (s *Operation) IsFocused() bool {
	return true
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

		// Set selection to current cursor position
		var selectionChangedCmd tea.Cmd
		if current := s.current(); current != nil {
			selectionChangedCmd = s.context.SetSelectedItem(context.SelectedFile{
				ChangeId: s.revision.GetChangeId(),
				CommitId: s.revision.CommitId,
				File:     current.fileName,
			})
		}
		return selectionChangedCmd
	default:
		oldCursor := s.cursor
		var cmds []tea.Cmd
		cmds = append(cmds, s.internalUpdate(msg))
		if s.cursor != oldCursor {
			cmds = append(cmds, s.context.SetSelectedItem(context.SelectedFile{
				ChangeId: s.revision.GetChangeId(),
				CommitId: s.revision.CommitId,
				File:     s.current().fileName,
			}))
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
		s.setCursor(msg.Index)
		return s.context.SetSelectedItem(context.SelectedFile{
			ChangeId: s.revision.GetChangeId(),
			CommitId: s.revision.CommitId,
			File:     s.current().fileName,
		})
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
		switch {
		case key.Matches(msg, s.keyMap.Up):
			s.cursorUp()
			return nil
		case key.Matches(msg, s.keyMap.Down):
			s.cursorDown()
			return nil
		case key.Matches(msg, s.keyMap.Cancel), key.Matches(msg, s.keyMap.Details.Close):
			return common.Close
		case key.Matches(msg, s.keyMap.Quit): // handle global quit after cancel
			return tea.Quit
		case key.Matches(msg, s.keyMap.Refresh):
			return common.Refresh
		case key.Matches(msg, s.keyMap.Details.Diff):
			selected := s.current()
			if selected == nil {
				return nil
			}
			return func() tea.Msg {
				output, _ := s.context.RunCommandImmediate(jj.Diff(s.revision.GetChangeId(), selected.fileName))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, s.keyMap.Details.Split, s.keyMap.Details.SplitParallel):
			isParallel := key.Matches(msg, s.keyMap.Details.SplitParallel)
			selectedFiles := s.getSelectedFiles(true)
			s.selectedHint = "stays as is"
			s.unselectedHint = "moves to the new revision"
			model := confirmation.New(
				[]string{"Are you sure you want to split the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					tea.Batch(s.context.RunInteractiveCommand(jj.Split(s.revision.GetChangeId(), selectedFiles, isParallel), common.Refresh), common.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			s.confirmation = model
			return s.confirmation.Init()
		case key.Matches(msg, s.keyMap.Details.Squash):
			return intents.Invoke(intents.StartSquash{
				Selected: jj.NewSelectedRevisions(s.revision),
				Files:    s.getSelectedFiles(true),
			})
		case key.Matches(msg, s.keyMap.Details.Restore):
			selectedFiles := s.getSelectedFiles(true)
			selected := s.current()
			s.selectedHint = "gets restored"
			s.unselectedHint = "stays as is"
			model := confirmation.New(
				[]string{"Are you sure you want to restore the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					s.context.RunCommand(jj.Restore(s.revision.GetChangeId(), selectedFiles), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("Interactive",
					tea.Batch(s.context.RunInteractiveCommand(jj.RestoreInteractive(s.revision.GetChangeId(), selected.fileName), common.Refresh), common.Close),
					key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "interactive"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			s.confirmation = model
			return s.confirmation.Init()
		case key.Matches(msg, s.keyMap.Details.Absorb):
			selectedFiles := s.getSelectedFiles(true)
			s.selectedHint = "might get absorbed into parents"
			s.unselectedHint = "stays as is"
			model := confirmation.New(
				[]string{"Are you sure you want to absorb changes from the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					s.context.RunCommand(jj.Absorb(s.revision.GetChangeId(), selectedFiles...), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			s.confirmation = model
			return s.confirmation.Init()
		case key.Matches(msg, s.keyMap.Details.ToggleSelect):
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

				s.cursorDown()
			}
			return nil
		case key.Matches(msg, s.keyMap.Details.RevisionsChangingFile):
			if current := s.current(); current != nil {
				return tea.Batch(common.Close, common.UpdateRevSet(fmt.Sprintf("files(%s)", jj.EscapeFileName(current.fileName))))
			}
		}
	}
	return nil
}

func (s *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := s.viewContent(box.R.Dx(), box.R.Dy())
	content = lipgloss.Place(
		box.R.Dx(),
		box.R.Dy(),
		lipgloss.Left,
		lipgloss.Top,
		content,
		lipgloss.WithWhitespaceBackground(s.styles.Text.GetBackground()),
	)
	dl.AddDraw(box.R, content, 0)
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	s.Current = commit
	return nil
}

func (s *Operation) ShortHelp() []key.Binding {
	if s.confirmation != nil {
		return s.confirmation.ShortHelp()
	}
	return []key.Binding{
		s.keyMap.Cancel,
		s.keyMap.Details.Diff,
		s.keyMap.Details.ToggleSelect,
		s.keyMap.Details.Split,
		s.keyMap.Details.SplitParallel,
		s.keyMap.Details.Squash,
		s.keyMap.Details.Restore,
		s.keyMap.Details.Absorb,
		s.keyMap.Details.RevisionsChangingFile,
	}
}

func (s *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{s.ShortHelp()}
}

func (s *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	// RenderToDisplayContext handles the actual rendering
	// This method is only called as a fallback when DesiredHeight returns 0,
	// which only happens when !isSelected || pos != After - the same conditions
	// that would make this return "" anyway.
	return ""
}

// DesiredHeight returns the desired height for the operation
func (s *Operation) DesiredHeight(commit *jj.Commit, pos operations.RenderPosition) int {
	isSelected := s.Current != nil && s.Current.GetChangeId() == commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return 0
	}
	if s.Len() == 0 {
		return 1 // "No changes" message
	}
	confirmationHeight := 0
	if s.confirmation != nil {
		confirmationHeight = lipgloss.Height(s.confirmation.View())
	}
	return s.Len() + confirmationHeight
}

// RenderToDisplayContext renders the file list directly to the DisplayContext
func (s *Operation) RenderToDisplayContext(dl *render.DisplayContext, commit *jj.Commit, pos operations.RenderPosition, rect cellbuf.Rectangle, screenOffset cellbuf.Position) int {
	isSelected := s.Current != nil && s.Current.GetChangeId() == commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return 0
	}

	if s.Len() == 0 {
		// Render "No changes" message
		content := s.styles.Dimmed.Render("No changes")
		dl.AddDraw(cellbuf.Rect(rect.Min.X, rect.Min.Y, rect.Dx(), 1), content, 0)
		return 1
	}

	confirmationHeight := 0
	if s.confirmation != nil {
		confirmationHeight = lipgloss.Height(s.confirmation.View())
	}

	availableListHeight := rect.Dy() - confirmationHeight
	if availableListHeight < 0 {
		availableListHeight = 0
	}

	// Calculate available height
	height := min(availableListHeight, s.Len())

	// Render the file list to DisplayContext
	// viewRect is already absolute, so don't reapply the parent screen offset.
	viewRect := layout.Box{R: cellbuf.Rect(rect.Min.X, rect.Min.Y, rect.Dx(), height)}
	s.RenderFileList(dl, viewRect, cellbuf.Pos(0, 0))

	if s.confirmation != nil && confirmationHeight > 0 && height < rect.Dy() {
		confirmRect := cellbuf.Rect(rect.Min.X, rect.Min.Y+height, rect.Dx(), confirmationHeight)
		s.confirmation.ViewRect(dl, layout.Box{R: confirmRect})
	}

	return height + confirmationHeight
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

	start := strings.IndexByte(content, '$')
	scanner = bufio.NewScanner(strings.NewReader(content[start+1:]))
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
	keyMap := config.Current.GetKeyMap()

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
		DetailsList:       l,
		context:           context,
		revision:          selected,
		keyMap:            keyMap,
		styles:            s,
		keymap:            config.Current.GetKeyMap(),
		targetMarkerStyle: common.DefaultPalette.Get("revisions details target_marker"),
	}
	return op
}

func (s *Operation) viewContent(width, maxHeight int) string {
	confirmationView := ""
	ch := 0
	if s.confirmation != nil {
		confirmationView = s.confirmation.View()
		ch = lipgloss.Height(confirmationView)
	}
	if s.Len() == 0 {
		return s.styles.Dimmed.Render("No changes")
	}
	if width <= 0 {
		width = 80 // sensible default
	}
	height := min(maxHeight-5-ch, s.Len())
	if height < 0 {
		height = 0
	}
	dl := render.NewDisplayContext()
	viewRect := layout.Box{R: cellbuf.Rect(0, 0, width, height)}
	if height > 0 {
		s.RenderFileList(dl, viewRect, cellbuf.Pos(0, 0))
	}
	filesView := strings.TrimRight(dl.RenderToString(width, height), "\n")
	if confirmationView != "" {
		return lipgloss.JoinVertical(lipgloss.Top, filesView, confirmationView)
	}
	return filesView
}
