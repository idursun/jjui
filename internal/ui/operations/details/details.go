package details

import (
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/context"
)

type status uint8

var (
	Added    status = 0
	Deleted  status = 1
	Modified status = 2
	Renamed  status = 3
)

type item struct {
	status   status
	name     string
	fileName string
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
	case Renamed:
		status = "R"
	}
	return fmt.Sprintf("%s %s", status, f.name)
}
func (f item) Description() string { return "" }
func (f item) FilterValue() string { return f.name }

type itemDelegate struct {
	selectedHint        string
	unselectedHint      string
	isVirtuallySelected bool
}

func (i itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(item)
	if !ok {
		return
	}
	var style lipgloss.Style
	switch item.status {
	case Added:
		style = common.DefaultPalette.Get("details added")
	case Deleted:
		style = common.DefaultPalette.Get("details deleted")
	case Modified:
		style = common.DefaultPalette.Get("details modified")
	case Renamed:
		style = common.DefaultPalette.Get("details renamed")
	}
	if index == m.Index() {
		style = style.Bold(true).Background(common.DefaultPalette.Get("details selected").GetBackground())
	}

	title := item.Title()
	if item.selected {
		title = "✓" + title
	} else {
		title = " " + title
	}

	hint := ""
	if i.showHint() {
		hint = i.unselectedHint
		if item.selected || (i.isVirtuallySelected && index == m.Index()) {
			hint = i.selectedHint
			title = "✓" + item.Title()
		}
	}

	fmt.Fprint(w, style.PaddingRight(1).Render(title), " ", common.DefaultPalette.Get("details dimmed").Render(hint))
}

func (i itemDelegate) Height() int                         { return 1 }
func (i itemDelegate) Spacing() int                        { return 0 }
func (i itemDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (i itemDelegate) showHint() bool {
	return i.selectedHint != "" || i.unselectedHint != ""
}

type Model struct {
	revision     string
	files        list.Model
	height       int
	confirmation tea.Model
	context      *context.MainContext
	keyMap       config.KeyMappings[key.Binding]
}

type updateCommitStatusMsg struct {
	summary       []string
	selectedFiles []string
}

func New(context *context.MainContext, revision string) tea.Model {
	keyMap := config.Current.GetKeyMap()
	l := list.New(nil, itemDelegate{}, 0, 0)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.KeyMap.CursorUp = keyMap.Up
	l.KeyMap.CursorDown = keyMap.Down
	return Model{
		revision: revision,
		files:    l,
		context:  context,
		keyMap:   keyMap,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.load(m.revision), tea.WindowSize())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case key.Matches(msg, m.keyMap.Details.Diff):
			selected, ok := m.files.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.Diff(m.revision, selected.fileName))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keyMap.Details.Split):
			selectedFiles, isVirtuallySelected := m.getSelectedFiles()
			m.files.SetDelegate(itemDelegate{
				isVirtuallySelected: isVirtuallySelected,
				selectedHint:        "stays as is",
				unselectedHint:      "moves to the new revision",
			})
			model := confirmation.New("Are you sure you want to split the selected files?")

			model.AddOption("Yes", tea.Batch(m.context.RunInteractiveCommand(jj.Split(m.revision, selectedFiles), common.Refresh), common.Close), key.NewBinding(key.WithKeys("y")))
			model.AddOption("No", confirmation.Close, key.NewBinding(key.WithKeys("n", "esc")))
			m.confirmation = &model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.Restore):
			selectedFiles, isVirtuallySelected := m.getSelectedFiles()
			m.files.SetDelegate(itemDelegate{
				isVirtuallySelected: isVirtuallySelected,
				selectedHint:        "gets restored",
				unselectedHint:      "stays as is",
			})
			model := confirmation.New("Are you sure you want to restore the selected files?")
			model.AddOption("Yes", m.context.RunCommand(jj.Restore(m.revision, selectedFiles), common.Refresh, confirmation.Close), key.NewBinding(key.WithKeys("y")))
			model.AddOption("No", confirmation.Close, key.NewBinding(key.WithKeys("n", "esc")))
			m.confirmation = &model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.Absorb):
			selectedFiles, isVirtuallySelected := m.getSelectedFiles()
			m.files.SetDelegate(itemDelegate{
				isVirtuallySelected: isVirtuallySelected,
				selectedHint:        "might get absorbed into parents",
				unselectedHint:      "stays as is",
			})
			model := confirmation.New("Are you sure you want to absorb changes from the selected files?")
			model.AddOption("Yes", m.context.RunCommand(jj.Absorb(m.revision, selectedFiles...), common.Refresh, confirmation.Close), key.NewBinding(key.WithKeys("y")))
			model.AddOption("No", confirmation.Close, key.NewBinding(key.WithKeys("n", "esc")))
			m.confirmation = &model
			return m, m.confirmation.Init()
		case key.Matches(msg, m.keyMap.Details.ToggleSelect):
			if oldItem, ok := m.files.SelectedItem().(item); ok {
				oldItem.selected = !oldItem.selected
				oldIndex := m.files.Index()
				m.files.CursorDown()

				curItem := m.files.SelectedItem().(item)
				return m, tea.Batch(m.files.SetItem(oldIndex, oldItem), m.context.SetSelectedItem(context.SelectedFile{ChangeId: m.revision, File: curItem.fileName}))
			}
			return m, nil
		case key.Matches(msg, m.keyMap.Details.RevisionsChangingFile):
			if item, ok := m.files.SelectedItem().(item); ok {
				return m, tea.Batch(common.Close, common.UpdateRevSet(fmt.Sprintf("files(%s)", item.fileName)))
			}
		default:
			if len(m.files.Items()) > 0 {
				var cmd tea.Cmd
				m.files, cmd = m.files.Update(msg)
				curItem := m.files.SelectedItem().(item)
				return m, tea.Batch(cmd, m.context.SetSelectedItem(context.SelectedFile{ChangeId: m.revision, File: curItem.fileName}))
			}
		}
	case confirmation.CloseMsg:
		m.confirmation = nil
		m.files.SetDelegate(itemDelegate{})
		return m, nil
	case common.RefreshMsg:
		return m, m.load(m.revision)
	case updateCommitStatusMsg:
		items := m.createListItems(msg.summary, msg.selectedFiles)
		var selectionChangedCmd tea.Cmd
		if len(items) > 0 {
			selectionChangedCmd = m.context.SetSelectedItem(context.SelectedFile{ChangeId: m.revision, File: items[0].(item).fileName})
		}
		if len(msg.selectedFiles) > 0 {

		}
		return m, tea.Batch(selectionChangedCmd, m.files.SetItems(items))
	case tea.WindowSizeMsg:
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) createListItems(content []string, selectedFiles []string) []list.Item {
	items := make([]list.Item, 0)
	for _, file := range content {
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
		}
		fileName := file[2:]

		actualFileName := fileName
		if status == Renamed && strings.Contains(actualFileName, "{") {
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
		items = append(items, item{
			status:   status,
			name:     fileName,
			fileName: actualFileName,
			selected: slices.ContainsFunc(selectedFiles, func(s string) bool { return s == actualFileName }),
		})
	}
	return items
}

func (m Model) getSelectedFiles() ([]string, bool) {
	selectedFiles := make([]string, 0)
	if len(m.files.Items()) == 0 {
		return selectedFiles, false
	}

	isVirtuallySelected := false
	for _, f := range m.files.Items() {
		if f.(item).selected {
			selectedFiles = append(selectedFiles, f.(item).fileName)
			isVirtuallySelected = false
		}
	}
	if len(selectedFiles) == 0 {
		selectedFiles = append(selectedFiles, m.files.SelectedItem().(item).fileName)
		return selectedFiles, true
	}
	return selectedFiles, isVirtuallySelected
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

func (m Model) load(revision string) tea.Cmd {
	output, err := m.context.RunCommandImmediate(jj.Snapshot())
	if err == nil {
		output, err = m.context.RunCommandImmediate(jj.Status(revision))
		if err == nil {
			return func() tea.Msg {
				summary := strings.Split(strings.TrimSpace(string(output)), "\n")
				selectedFiles, isVirtuallySelected := m.getSelectedFiles()
				if isVirtuallySelected {
					selectedFiles = []string{}
				}
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
