package quick_search

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var (
	_ operations.Operation       = (*Operation)(nil)
	_ operations.SegmentRenderer = (*Operation)(nil)
	_ view.IStatus               = (*Operation)(nil)
	_ view.IHasActionMap         = (*Operation)(nil)
)

type state int

const (
	stateEditing state = iota
	stateApplied
)

type Operation struct {
	cursor     list.IListCursor
	getItemFn  func(index int) parser.Row
	count      int
	context    *context.MainContext
	input      textinput.Model
	searchTerm string
	state      state
}

func (o *Operation) GetActionMap() actions.ActionMap {
	return config.Current.GetBindings("quick_search")
}

func (o *Operation) Name() string {
	return "quick search"
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "quick_search.apply":
			o.searchTerm = o.input.Value()
			o.state = stateApplied
			o.cursor.SetCursor(o.search(0))
			return o, nil
		case "quick_search.cycle":
			o.cursor.SetCursor(o.search(o.cursor.Cursor() + 1))
			return o, nil
		}
	case tea.KeyMsg:
		if msg.String() == "enter" {
			o.searchTerm = o.input.Value()
			o.state = stateApplied
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

func (o *Operation) RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string {
	start, end := segment.FindSubstringRange(o.input.Value())
	if start != -1 {
		mid := lipgloss.NewRange(start, end, currentStyle.Reverse(true))
		return lipgloss.StyleRanges(currentStyle.Render(segment.Text), mid)
	}
	return currentStyle.Render(segment.Text)
}

func (o *Operation) Render(*jj.Commit, operations.RenderPosition) string {
	return ""
}

func (o *Operation) View() string {
	if o.state == stateApplied {
		return ""
	}
	return o.input.View()
}

func (o *Operation) search(startIndex int) int {
	if o.searchTerm == "" {
		return o.cursor.Cursor()
	}

	n := o.count
	for i := startIndex; i < n+startIndex; i++ {
		c := i % n
		row := o.getItemFn(c)
		for _, line := range row.Lines {
			for _, segment := range line.Segments {
				if segment.Text != "" && strings.Contains(segment.Text, o.searchTerm) {
					return c
				}
			}
		}
	}
	return o.cursor.Cursor()
}

func NewOperation(listCursor list.IListCursor, getItemFn func(index int) parser.Row, count int) *Operation {
	i := textinput.New()
	i.Placeholder = "Quick Search"
	i.Focus()
	i.CharLimit = 256
	i.Prompt = "/ "
	i.PromptStyle = i.PromptStyle.Bold(true)
	i.TextStyle = i.TextStyle.Bold(true)
	i.Cursor.SetMode(cursor.CursorStatic)

	return &Operation{
		cursor:    listCursor,
		getItemFn: getItemFn,
		count:     count,
		input:     i,
		state:     stateEditing,
	}
}
