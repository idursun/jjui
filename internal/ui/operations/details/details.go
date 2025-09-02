package details

import (
	"bufio"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/common/models"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
)

type styles struct {
	Added    lipgloss.Style
	Deleted  lipgloss.Style
	Modified lipgloss.Style
	Renamed  lipgloss.Style
	Selected lipgloss.Style
	Dimmed   lipgloss.Style
	Text     lipgloss.Style
	Conflict lipgloss.Style
}

type DetailsList struct {
	*list.CheckableList[*models.RevisionFileItem]
	renderer            *list.ListRenderer[*models.RevisionFileItem]
	selectedHint        string
	unselectedHint      string
	isVirtuallySelected bool
	styles              styles
}

func (d *DetailsList) showHint() bool {
	return d.selectedHint != "" || d.unselectedHint != ""
}

func (d *DetailsList) RenderItem(w io.Writer, index int) {
	item := d.Items[index]
	var style lipgloss.Style
	switch item.Status {
	case models.Added:
		style = d.styles.Added
	case models.Deleted:
		style = d.styles.Deleted
	case models.Modified:
		style = d.styles.Modified
	case models.Renamed:
		style = d.styles.Renamed
	}

	if index == d.Cursor {
		style = style.Bold(true).Background(d.styles.Selected.GetBackground())
	} else {
		style = style.Background(d.styles.Text.GetBackground())
	}

	title := item.Title()
	if item.IsChecked() {
		title = "✓" + title
	} else {
		title = " " + title
	}

	hint := ""
	if d.showHint() {
		hint = d.unselectedHint
		if item.IsChecked() || (d.isVirtuallySelected && index == d.Cursor) {
			hint = d.selectedHint
			title = "✓" + item.Title()
		}
	}

	_, _ = fmt.Fprint(w, style.PaddingRight(1).Render(title))
	if item.Conflict {
		_, _ = fmt.Fprint(w, d.styles.Conflict.Render("conflict "))
	}
	if hint != "" {
		_, _ = fmt.Fprint(w, d.styles.Dimmed.Render(hint))
	}
	_, _ = fmt.Fprintln(w)
}

func (d *DetailsList) GetItemHeight(int) int {
	return 1
}

type Model struct {
	*common.Sizeable
	*DetailsList
	context      *context.MainContext
	revision     *jj.Commit
	mode         mode
	confirmation *confirmation.Model
	keyMap       config.KeyMappings[key.Binding]
}

func (m Model) ShortHelp() []key.Binding {
	if m.confirmation != nil {
		return m.confirmation.ShortHelp()
	}
	return []key.Binding{
		m.keyMap.Cancel,
		m.keyMap.Details.Diff,
		m.keyMap.Details.ToggleSelect,
		m.keyMap.Details.Split,
		m.keyMap.Details.Squash,
		m.keyMap.Details.Restore,
		m.keyMap.Details.Absorb,
		m.keyMap.Details.RevisionsChangingFile,
	}
}

func (m Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

type updateCommitStatusMsg struct {
	summary string
}

func New(context *context.MainContext, revision *jj.Commit) Model {
	keyMap := config.Current.GetKeyMap()

	s := styles{
		Added:    common.DefaultPalette.Get("revisions details added"),
		Deleted:  common.DefaultPalette.Get("revisions details deleted"),
		Modified: common.DefaultPalette.Get("revisions details modified"),
		Renamed:  common.DefaultPalette.Get("revisions details renamed"),
		Selected: common.DefaultPalette.Get("revisions details selected"),
		Dimmed:   common.DefaultPalette.Get("revisions details dimmed"),
		Text:     common.DefaultPalette.Get("revisions details text"),
		Conflict: common.DefaultPalette.Get("revisions details conflict"),
	}

	size := common.NewSizeable(5, 5)
	dl := &DetailsList{
		CheckableList: list.NewCheckableList[*models.RevisionFileItem](),
		styles:        s,
	}
	dl.renderer = list.NewRenderer[*models.RevisionFileItem](dl.List, dl.RenderItem, dl.GetItemHeight, common.NewSizeable(5, 5))
	return Model{
		Sizeable:    size,
		DetailsList: dl,
		revision:    revision,
		mode:        viewMode,
		context:     context,
		keyMap:      keyMap,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.load(m.revision.GetChangeId()), tea.WindowSize())
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmation != nil {
			model, cmd := m.confirmation.Update(msg)
			m.confirmation = model
			return m, cmd
		}
		switch {
		case key.Matches(msg, m.keyMap.Up):
			m.CursorUp()
			return m, m.context.SetSelectedItem(context.SelectedFile{
				ChangeId: m.revision.GetChangeId(),
				CommitId: m.revision.CommitId,
				File:     m.Current().FileName,
			})
		case key.Matches(msg, m.keyMap.Down):
			m.CursorDown()
			return m, m.context.SetSelectedItem(context.SelectedFile{
				ChangeId: m.revision.GetChangeId(),
				CommitId: m.revision.CommitId,
				File:     m.Current().FileName,
			})
		case key.Matches(msg, m.keyMap.Cancel), key.Matches(msg, m.keyMap.Details.Close):
			return m, common.Close
		case key.Matches(msg, m.keyMap.Details.Diff):
			selected := m.Current()
			if selected == nil {
				return m, nil
			}
			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.Diff(m.revision.GetChangeId(), selected.FileName))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keyMap.Details.Split):
			selectedFiles, isVirtuallySelected := m.getSelectedFiles()
			m.isVirtuallySelected = isVirtuallySelected
			m.selectedHint = "stays as is"
			m.unselectedHint = "moves to the new revision"
			model := confirmation.New(
				[]string{"Are you sure you want to split the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					tea.Batch(m.context.RunInteractiveCommand(jj.Split(m.revision.GetChangeId(), selectedFiles), common.Refresh), common.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			m.confirmation = model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.Squash):
			m.mode = squashTargetMode
			return m, common.JumpToParent(m.revision)
		case key.Matches(msg, m.keyMap.Details.Restore):
			selectedFiles, isVirtuallySelected := m.getSelectedFiles()
			m.isVirtuallySelected = isVirtuallySelected
			m.selectedHint = "gets restored"
			m.unselectedHint = "stays as is"
			model := confirmation.New(
				[]string{"Are you sure you want to restore the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					m.context.RunCommand(jj.Restore(m.revision.GetChangeId(), selectedFiles), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			m.confirmation = model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.Absorb):
			selectedFiles, isVirtuallySelected := m.getSelectedFiles()
			m.isVirtuallySelected = isVirtuallySelected
			m.selectedHint = "might get absorbed into parents"
			m.unselectedHint = "stays as is"

			model := confirmation.New(
				[]string{"Are you sure you want to absorb changes from the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					m.context.RunCommand(jj.Absorb(m.revision.GetChangeId(), selectedFiles...), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			m.confirmation = model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.ToggleSelect):
			current := m.Current()
			current.Toggle()
			m.CursorDown()
			return m, m.context.SetSelectedItem(context.SelectedFile{
				ChangeId: m.revision.GetChangeId(),
				CommitId: m.revision.CommitId,
				File:     current.FileName,
			})
		case key.Matches(msg, m.keyMap.Details.RevisionsChangingFile):
			if current := m.Current(); current != nil {
				return m, tea.Batch(common.Close, common.UpdateRevSet(fmt.Sprintf("files(%s)", jj.EscapeFileName(current.FileName))))
			}
			return m, nil
		}
	case confirmation.CloseMsg:
		m.confirmation = nil
		m.selectedHint = ""
		m.unselectedHint = ""
		return m, nil
	case common.RefreshMsg:
		return m, m.load(m.revision.GetChangeId())
	case updateCommitStatusMsg:
		items := m.createListItems(msg.summary)
		m.SetItems(items)
		m.Cursor = 0
		return m, nil
	case tea.WindowSizeMsg:
		m.SetHeight(msg.Height)
	}
	return m, nil
}

func (m Model) createListItems(content string) []*models.RevisionFileItem {
	items := make([]*models.RevisionFileItem, 0)
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
		var status models.Status
		switch file[0] {
		case 'A':
			status = models.Added
		case 'D':
			status = models.Deleted
		case 'M':
			status = models.Modified
		case 'R':
			status = models.Renamed
		}
		fileName := file[2:]

		actualFileName := fileName
		if status == models.Renamed && strings.Contains(actualFileName, "{") {
			for strings.Contains(actualFileName, "{") {
				start := strings.Index(actualFileName, "{")
				end := strings.Index(actualFileName, "}")
				if end == -1 {
					break
				}
				replacement := actualFileName[start+1 : end]
				parts := strings.Split(replacement, " => ")
				replacement = parts[1]
				actualFileName = path.Clean(actualFileName[:start] + replacement + actualFileName[end+1:])
			}
		}
		items = append(items, &models.RevisionFileItem{
			Checkable: &models.Checkable{Checked: false},
			Status:    status,
			Name:      fileName,
			FileName:  actualFileName,
			Conflict:  conflicts[index],
		})
		index++
	}

	return items
}

func (m Model) getSelectedFiles() ([]string, bool) {
	selectedFiles := make([]string, 0)
	checkedItems := m.GetCheckedItems()

	if len(checkedItems) == 0 {
		current := m.Current()
		return []string{current.FileName}, true
	}

	for _, f := range checkedItems {
		if f.IsChecked() {
			selectedFiles = append(selectedFiles, f.FileName)
		}
	}
	return selectedFiles, false
}

func (m Model) View() string {
	confirmationView := ""
	ch := 0
	if m.confirmation != nil {
		confirmationView = m.confirmation.View()
		ch = lipgloss.Height(confirmationView)
	}
	if len(m.Items) == 0 {
		return m.styles.Dimmed.Render("No changes\n")
	}
	m.renderer.SetHeight(min(m.Height-5-ch, len(m.Items)))
	filesView := m.renderer.Render()

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
	return lipgloss.Place(w, h, 0, 0, view, lipgloss.WithWhitespaceBackground(m.styles.Text.GetBackground()))
}

func (m Model) load(revision string) tea.Cmd {
	output, err := m.context.RunCommandImmediate(jj.Snapshot())
	if err == nil {
		output, err = m.context.RunCommandImmediate(jj.Status(revision))
		if err == nil {
			return func() tea.Msg {
				summary := string(output)
				return updateCommitStatusMsg{summary}
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
