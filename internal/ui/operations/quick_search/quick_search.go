package quick_search

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var (
	_ view.IViewModel            = (*Operation)(nil)
	_ operations.Operation       = (*Operation)(nil)
	_ operations.SegmentRenderer = (*Operation)(nil)
	_ help.KeyMap                = (*Operation)(nil)
)

type state int

const (
	stateEditing state = iota
	stateApplied
)

type Operation struct {
	*view.ViewNode
	context    *context.MainContext
	input      textinput.Model
	keymap     config.KeyMappings[key.Binding]
	searchTerm string
	state      state
	list       *list.List[*models.RevisionItem]
}

func (o *Operation) ShortHelp() []key.Binding {
	switch o.state {
	case stateEditing:
		return []key.Binding{
			o.keymap.Cancel,
			o.keymap.Apply,
		}
	case stateApplied:
		return []key.Binding{
			o.keymap.Cancel,
			o.keymap.QuickSearchCycle,
		}
	}
	return nil
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) GetId() view.ViewId {
	return "quick search"
}

func (o *Operation) Mount(v *view.ViewNode) {
	o.ViewNode = v
	v.Id = o.GetId()
	o.input.Width = v.Parent.Width - 4
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, o.keymap.Cancel):
			o.ViewManager.StopEditing()
			o.ViewManager.UnregisterView(o.Id)
			return o, nil
		case key.Matches(msg, o.keymap.Apply):
			o.searchTerm = o.input.Value()
			o.state = stateApplied
			o.list.SetCursor(o.search(0))
			revisionsViewId := view.RevisionsViewId
			o.KeyDelegation = &revisionsViewId
			o.ViewManager.StopEditing()
			return o, nil
		case key.Matches(msg, o.keymap.QuickSearchCycle) && o.state == stateApplied:
			o.list.SetCursor(o.search(o.list.Cursor + 1))
			return o, nil
		}
	}

	if o.state == stateEditing {
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(msg)
		o.searchTerm = o.input.Value()
		return o, cmd
	}
	return o, nil
}

func (o *Operation) RenderSegment(currenStyle lipgloss.Style, segment *screen.Segment, row *models.RevisionItem) string {
	style := currenStyle
	start, end := segment.FindSubstringRange(o.input.Value())
	if start != -1 {
		mid := lipgloss.NewRange(start, end, style.Reverse(true))
		return lipgloss.StyleRanges(style.Render(segment.Text), mid)
	}
	return segment.Text
}

func (o *Operation) Render(*models.Commit, operations.RenderPosition) string {
	return ""
}

func (o *Operation) View() string {
	if o.state == stateApplied {
		return ""
	}
	return o.input.View()
}

func (o *Operation) search(startIndex int) int {
	list := o.list

	if o.searchTerm == "" {
		return list.Cursor
	}

	n := len(list.Items)
	for i := startIndex; i < n+startIndex; i++ {
		c := i % n
		row := list.Items[c]
		for _, line := range row.Lines {
			for _, segment := range line.Segments {
				if segment.Text != "" && strings.Contains(segment.Text, o.searchTerm) {
					return c
				}
			}
		}
	}
	return list.Cursor
}

func NewOperation(list *list.List[*models.RevisionItem]) view.IViewModel {
	i := textinput.New()
	i.Placeholder = "Quick Search"
	i.Focus()
	i.CharLimit = 256
	i.Prompt = "/ "
	i.PromptStyle = i.PromptStyle.Bold(true)
	i.TextStyle = i.TextStyle.Bold(true)
	i.Cursor.SetMode(cursor.CursorStatic)

	return &Operation{
		list:   list,
		keymap: config.Current.GetKeyMap(),
		input:  i,
		state:  stateEditing,
	}
}
