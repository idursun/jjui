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
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
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
	height            int
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
		var selectionChangedCmd tea.Cmd
		s.context.ClearCheckedItems(reflect.TypeFor[context.SelectedFile]())
		if len(items) > 0 {
			var first context.SelectedItem
			for _, it := range items {
				sel := context.SelectedFile{
					ChangeId: s.revision.GetChangeId(),
					CommitId: s.revision.CommitId,
					File:     it.fileName,
				}
				if first == nil {
					first = sel
				}
				if it.selected {
					s.context.AddCheckedItem(sel)
				}
			}
			selectionChangedCmd = s.context.SetSelectedItem(first)
		}
		s.setItems(items)
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
			return func() tea.Msg {
				return common.StartSquashOperationMsg{Revision: s.revision, Files: s.getSelectedFiles(true)}
			}
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

func (s *Operation) View() string {
	confirmationView := ""
	ch := 0
	if s.confirmation != nil {
		confirmationView = s.confirmation.View()
		ch = lipgloss.Height(confirmationView)
	}
	if s.Len() == 0 {
		return s.styles.Dimmed.Render("No changes\n")
	}
	s.SetHeight(min(s.height-5-ch, s.Len()))
	filesView := s.renderer.Render(s.cursor)

	view := lipgloss.JoinVertical(lipgloss.Top, filesView, confirmationView)
	// We are trimming spaces from each line to prevent visual artefacts
	// Empty lines use the default background colour, and it looks bad if the user has a custom background colour
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(view))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		lines = append(lines, line)
	}
	view = strings.Join(lines, "\n")
	w, h := lipgloss.Size(view)
	return lipgloss.Place(w, h, 0, 0, view, lipgloss.WithWhitespaceBackground(s.styles.Text.GetBackground()))
}

func (s *Operation) SetSelectedRevision(commit *jj.Commit) {
	s.Current = commit
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
	isSelected := s.Current != nil && s.Current.GetChangeId() == commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return s.View()
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
	var conflicts []bool
	if scanner.Scan() {
		conflictsLine := strings.Split(scanner.Text(), " ")
		for _, c := range conflictsLine {
			conflicts = append(conflicts, c == "true")
		}
	} else {
		return items
	}

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

func NewOperation(context *context.MainContext, selected *jj.Commit, height int) *Operation {
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

	l := NewDetailsList(s, common.NewSizeable(0, height))
	op := &Operation{
		DetailsList:       l,
		context:           context,
		revision:          selected,
		keyMap:            keyMap,
		styles:            s,
		height:            height,
		keymap:            config.Current.GetKeyMap(),
		targetMarkerStyle: common.DefaultPalette.Get("revisions details target_marker"),
	}
	return op
}
