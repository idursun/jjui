package ace_jump

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var (
	_ operations.Operation       = (*Operation)(nil)
	_ operations.SegmentRenderer = (*Operation)(nil)
	_ help.KeyMap                = (*Operation)(nil)
	_ view.IHasActionMap         = (*Operation)(nil)
)

type Operation struct {
	cursor      list.IListCursor
	aceJump     *AceJump
	keymap      config.KeyMappings[key.Binding]
	getItemFn   func(index int) parser.Row
	first, last int
}

func (o *Operation) GetActionMap() map[string]actions.Action {
	return config.Current.GetBindings("ace_jump")
}

func (o *Operation) Name() string {
	return "ace jump"
}

func NewOperation(listCursor list.IListCursor, getItemFn func(index int) parser.Row, first, last int) *Operation {
	return &Operation{
		cursor:    listCursor,
		keymap:    config.Current.GetKeyMap(),
		aceJump:   NewAceJump(),
		first:     first,
		last:      last,
		getItemFn: getItemFn,
	}
}

func (o *Operation) RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string {
	style := currentStyle
	if aceIdx := o.aceJumpIndex(segment.Text, row); aceIdx > -1 {
		mid := lipgloss.NewRange(aceIdx, aceIdx+1, style.Reverse(true))
		return lipgloss.StyleRanges(style.Render(segment.Text), mid)
	}
	return ""
}

func (o *Operation) aceJumpIndex(text string, row parser.Row) int {
	aceJumpPrefix := o.aceJump.Prefix()
	if aceJumpPrefix == nil || row.Commit == nil {
		return -1
	}
	if !(text == row.Commit.ChangeId || text == row.Commit.CommitId) {
		return -1
	}
	lowerText, lowerPrefix := strings.ToLower(text), strings.ToLower(*aceJumpPrefix)
	if !strings.HasPrefix(lowerText, lowerPrefix) {
		return -1
	}
	idx := len(lowerPrefix)
	if idx == len(lowerText) {
		idx-- // dont move past last character
	}
	return idx
}

func (o *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		o.keymap.Cancel,
		o.keymap.Apply,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) Init() tea.Cmd {
	o.aceJump = o.findAceKeys()
	return nil
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if found := o.aceJump.Narrow(msg); found != nil {
		o.cursor.SetCursor(found.RowIdx)
		o.aceJump = nil
		return actions.InvokeAction(actions.Action{Id: "close ace_jump"})
	}
	return nil
}

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(actions.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "ace_jump.apply":
			o.cursor.SetCursor(o.aceJump.First().RowIdx)
			o.aceJump = nil
			return o, nil
		}
	}
	if msg, ok := msg.(tea.KeyMsg); ok {
		return o, o.HandleKey(msg)
	}
	return o, nil
}

func (o *Operation) View() string {
	return ""
}

func (o *Operation) Render(*jj.Commit, operations.RenderPosition) string {
	return ""
}

func (o *Operation) findAceKeys() *AceJump {
	aj := NewAceJump()
	if o.first == -1 || o.last == -1 {
		return nil // wait until rendered
	}
	for i := range o.last - o.first + 1 {
		i += o.first
		row := o.getItemFn(i)
		c := row.Commit
		if c == nil {
			continue
		}
		aj.Append(i, c.CommitId, 0)
		if c.Hidden || c.IsConflicting() || c.IsRoot() {
			continue
		}
		aj.Append(i, c.ChangeId, 0)
	}
	return aj
}
