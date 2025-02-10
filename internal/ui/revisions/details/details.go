package details

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
)

var (
	AddedStyle    = lipgloss.NewStyle().Foreground(common.Green)
	DeletedStyle  = lipgloss.NewStyle().Foreground(common.Red)
	ModifiedStyle = lipgloss.NewStyle().Foreground(common.Cyan)
	HintStyle     = lipgloss.NewStyle().Foreground(common.DarkBlack).PaddingLeft(1)
)

type status uint8

var (
	Added    status = 0
	Deleted  status = 1
	Modified status = 2
)

var (
	cancel  = key.NewBinding(key.WithKeys("esc", "h"))
	mark    = key.NewBinding(key.WithKeys("m", " "))
	restore = key.NewBinding(key.WithKeys("r"))
	split   = key.NewBinding(key.WithKeys("s"))
	up      = key.NewBinding(key.WithKeys("up", "k"))
	down    = key.NewBinding(key.WithKeys("down", "j"))
	diff    = key.NewBinding(key.WithKeys("d"))
)

type item struct {
	status   status
	name     string
	selected bool
}

func (f item) Title() string {
	status := "M"
	switch f.status {
	case Added:
		status = "A"
	case Deleted:
		status = "D"
	case Modified:
		status = "M"
	}
	return fmt.Sprintf("%s %s", status, f.name)
}
func (f item) Description() string { return "" }
func (f item) FilterValue() string { return f.name }

type itemDelegate struct {
	selectedHint   string
	unselectedHint string
}

func (i itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(item)
	if !ok {
		return
	}
	var style lipgloss.Style
	switch item.status {
	case Added:
		style = AddedStyle
	case Deleted:
		style = DeletedStyle
	case Modified:
		style = ModifiedStyle
	}
	if index == m.Index() {
		style = style.Bold(true).Background(common.DarkBlack)
	}
	title := item.Title()
	hint := ""
	if item.selected {
		title = "✓" + title
		hint = i.selectedHint
	} else {
		title = " " + title
		hint = i.unselectedHint
	}
	fmt.Fprint(w, style.PaddingRight(1).Render(title), HintStyle.Render(hint))
}

func (i itemDelegate) Height() int                         { return 1 }
func (i itemDelegate) Spacing() int                        { return 0 }
func (i itemDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

type Model struct {
	revision     string
	files        list.Model
	height       int
	confirmation *confirmation.Model
	common.UICommands
}

func New(revision string, commands common.UICommands) tea.Model {
	l := list.New(nil, itemDelegate{}, 0, 0)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.KeyMap.CursorUp = up
	l.KeyMap.CursorDown = down
	return Model{
		revision:   revision,
		files:      l,
		UICommands: commands,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Status(m.revision), tea.WindowSize())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmation != nil {
			model, cmd := m.confirmation.Update(msg)
			m.confirmation = &model
			return m, cmd
		}
		switch {
		case key.Matches(msg, cancel):
			return m, common.Close
		case key.Matches(msg, diff):
			v := m.files.SelectedItem().(item).name
			return m, m.UICommands.GetDiff(m.revision, v)
		case key.Matches(msg, split):
			selectedFiles := m.getSelectedFiles()
			m.files.SetDelegate(itemDelegate{
				selectedHint:   "stays as is",
				unselectedHint: "moves to the new revision",
			})
			model := confirmation.New("Are you sure you want to split the selected files?")
			model.AddOption("Yes", tea.Batch(m.UICommands.Split(m.revision, selectedFiles), common.Close))
			model.AddOption("No", confirmation.Close)
			m.confirmation = &model
			return m, m.confirmation.Init()
		case key.Matches(msg, restore):
			selectedFiles := m.getSelectedFiles()
			m.files.SetDelegate(itemDelegate{
				selectedHint:   "gets restored",
				unselectedHint: "stays as is",
			})
			model := confirmation.New("Are you sure you want to restore the selected files?")
			model.AddOption("Yes", tea.Batch(m.UICommands.Restore(m.revision, selectedFiles), common.Close))
			model.AddOption("No", confirmation.Close)
			m.confirmation = &model
			return m, m.confirmation.Init()
		case key.Matches(msg, mark):
			if item, ok := m.files.SelectedItem().(item); ok {
				item.selected = !item.selected
				oldIndex := m.files.Index()
				m.files.CursorDown()
				return m, m.files.SetItem(oldIndex, item)
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.files, cmd = m.files.Update(msg)
			return m, cmd
		}
	case confirmation.CloseMsg:
		m.confirmation = nil
		m.files.SetDelegate(itemDelegate{})
		return m, nil
	case common.RefreshMsg:
		return m, m.Status(m.revision)
	case common.UpdateCommitStatusMsg:
		items := make([]list.Item, 0)
		for _, file := range msg {
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
			}
			items = append(items, item{
				status: status,
				name:   file[2:],
			})
		}
		m.files.SetItems(items)
	case tea.WindowSizeMsg:
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) getSelectedFiles() []string {
	selectedFiles := make([]string, 0)
	for _, f := range m.files.Items() {
		if f.(item).selected {
			selectedFiles = append(selectedFiles, f.(item).name)
		}
	}
	if len(selectedFiles) == 0 {
		selectedFiles = append(selectedFiles, m.files.SelectedItem().(item).name)
	}
	return selectedFiles
}

func (m Model) View() string {
	confirmationView := ""
	ch := 0
	if m.confirmation != nil {
		confirmationView = m.confirmation.View()
		ch = lipgloss.Height(confirmationView)
	}
	m.files.SetHeight(min(m.height-5-ch, len(m.files.Items())))
	filesView := m.files.View()
	return lipgloss.JoinVertical(lipgloss.Top, filesView, confirmationView)
}
