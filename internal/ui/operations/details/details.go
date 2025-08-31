package details

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/context/models"
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

type Model struct {
	context             *context.DetailsContext
	revision            *models.RevisionItem
	mode                mode
	height              int
	confirmation        *confirmation.Model
	keyMap              config.KeyMappings[key.Binding]
	styles              styles
	selectedHint        string
	unselectedHint      string
	isVirtuallySelected bool
}

func (m *Model) ShortHelp() []key.Binding {
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

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func New(context *context.DetailsContext, revision *models.RevisionItem) *Model {
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
	return &Model{
		revision: revision,
		mode:     viewMode,
		context:  context,
		keyMap:   keyMap,
		styles:   s,
	}
}

func (m *Model) Init() tea.Cmd {
	m.context.Load(m.revision)
	return tea.WindowSize()
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmation != nil {
			model, cmd := m.confirmation.Update(msg)
			m.confirmation = model
			return m, cmd
		}
		switch {
		case key.Matches(msg, m.keyMap.Cancel), key.Matches(msg, m.keyMap.Details.Close):
			return m, common.Close
		case key.Matches(msg, m.keyMap.Up):
			m.context.Prev()
		case key.Matches(msg, m.keyMap.Down):
			m.context.Next()
		case key.Matches(msg, m.keyMap.Details.Diff):
			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.Diff(m.revision.Commit.GetChangeId(), m.context.Current().FileName))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keyMap.Details.Split):
			selectedFiles, isVirtuallySelected := m.getSelectedFiles()
			m.isVirtuallySelected = isVirtuallySelected
			m.selectedHint = "stays as is"
			m.unselectedHint = "moves to the new revisions"
			model := confirmation.New(
				[]string{"Are you sure you want to split the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					tea.Batch(m.context.RunInteractiveCommand(jj.Split(m.revision.Commit.GetChangeId(), selectedFiles), common.Refresh), common.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			m.confirmation = model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.Squash):
			m.mode = squashTargetMode
			return m, nil
			//FIXME: jump to parent
			//return m, common.JumpToParent(m.revision)
		case key.Matches(msg, m.keyMap.Details.Restore):
			selectedFiles, isVirtuallySelected := m.getSelectedFiles()
			m.isVirtuallySelected = isVirtuallySelected
			m.selectedHint = "gets restored"
			m.unselectedHint = "stays as is"
			model := confirmation.New(
				[]string{"Are you sure you want to restore the selected files?"},
				confirmation.WithStylePrefix("revisions"),
				confirmation.WithOption("Yes",
					m.context.RunCommand(jj.Restore(m.revision.Commit.GetChangeId(), selectedFiles), common.Refresh, confirmation.Close),
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
					m.context.RunCommand(jj.Absorb(m.revision.Commit.GetChangeId(), selectedFiles...), common.Refresh, confirmation.Close),
					key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
				confirmation.WithOption("No",
					confirmation.Close,
					key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
			)
			m.confirmation = model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.ToggleSelect):
			m.context.Current().Toggle()
			m.context.Next()
			return m, nil
		case key.Matches(msg, m.keyMap.Details.RevisionsChangingFile):
			if current := m.context.Current(); current != nil {
				return m, tea.Batch(common.Close, common.UpdateRevSet(fmt.Sprintf("files(%s)", jj.EscapeFileName(current.FileName))))
			}
		}
	case confirmation.CloseMsg:
		m.confirmation = nil
		return m, nil
	case tea.WindowSizeMsg:
		m.height = msg.Height
	}
	return m, nil
}

func (m *Model) getSelectedFiles() ([]string, bool) {
	selectedFiles := make([]string, 0)
	checkedItems := m.context.GetCheckedItems()
	for _, checkedItem := range checkedItems {
		selectedFiles = append(selectedFiles, checkedItem.Name)
	}
	isVirtuallySelected := false
	if len(selectedFiles) == 0 {
		current := m.context.Current()
		if current != nil {
			isVirtuallySelected = true
			selectedFiles = append(selectedFiles, current.Name)
		}
	}
	return selectedFiles, isVirtuallySelected
}

func (m *Model) View() string {
	confirmationView := ""
	ch := 0
	if m.confirmation != nil {
		confirmationView = m.confirmation.View()
		ch = lipgloss.Height(confirmationView)
	}
	var sw strings.Builder
	for i, item := range m.context.Items {
		m.Render(&sw, i, item)
		sw.WriteString("\n")
	}
	height := min(m.height-5-ch, len(m.context.Items))
	filesView := lipgloss.PlaceVertical(height, 0, sw.String())

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

func (m *Model) Render(w io.Writer, index int, item *models.RevisionFile) {
	var style lipgloss.Style
	switch item.Status {
	case models.Added:
		style = m.styles.Added
	case models.Deleted:
		style = m.styles.Deleted
	case models.Modified:
		style = m.styles.Modified
	case models.Renamed:
		style = m.styles.Renamed
	}

	if index == m.context.Cursor() {
		style = style.Bold(true).Background(m.styles.Selected.GetBackground())
	} else {
		style = style.Background(m.styles.Text.GetBackground())
	}

	status := "M"
	switch item.Status {
	case models.Added:
		status = "A"
	case models.Deleted:
		status = "D"
	case models.Modified:
		status = "M"
	case models.Renamed:
		status = "R"
	}

	title := fmt.Sprintf("%s %s", status, item.Name)
	if item.IsChecked() {
		title = "✓" + title
	} else {
		title = " " + title
	}

	hint := ""
	if m.showHint() {
		hint = m.unselectedHint
		if item.IsChecked() || (m.isVirtuallySelected && index == m.context.Cursor()) {
			hint = m.selectedHint
			title = "✓" + title
		}
	}

	fmt.Fprint(w, style.PaddingRight(1).Render(title))
	if item.Conflict {
		fmt.Fprint(w, m.styles.Conflict.Render("conflict "))
	}
	if hint != "" {
		fmt.Fprint(w, m.styles.Dimmed.Render(hint))
	}
}

func (m *Model) showHint() bool {
	return m.selectedHint != "" || m.unselectedHint != ""
}
