package ace_jump

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var (
	_ operations.Operation       = (*Operation)(nil)
	_ operations.SegmentRenderer = (*Operation)(nil)
	_ common.Focusable           = (*Operation)(nil)
	_ common.Editable            = (*Operation)(nil)
)

type Operation struct {
	setCursor   func(int)
	aceJump     *AceJump
	getItemFn   func(index int) parser.Row
	first, last int
	parentOp    any // parent operation to return to after completion
}

func (o *Operation) IsEditing() bool {
	return true
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Name() string {
	return "ace jump"
}

func (o *Operation) Scope() keybindings.Scope {
	return keybindings.Scope(actions.OwnerAceJump)
}

func NewOperation(setCursor func(int), getItemFn func(index int) parser.Row, first, last int, parentOp any) *Operation {
	return &Operation{
		setCursor: setCursor,
		aceJump:   NewAceJump(),
		first:     first,
		last:      last,
		getItemFn: getItemFn,
		parentOp:  parentOp,
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
	lowerText := strings.ToLower(text)
	if lowerText != strings.ToLower(row.Commit.ChangeId) && lowerText != strings.ToLower(row.Commit.CommitId) {
		return -1
	}
	lowerPrefix := strings.ToLower(*aceJumpPrefix)
	if !strings.HasPrefix(lowerText, lowerPrefix) {
		return -1
	}
	idx := len(lowerPrefix)
	if idx == len(lowerText) {
		idx-- // dont move past last character
	}
	return idx
}

func (o *Operation) Init() tea.Cmd {
	o.aceJump = o.findAceKeys()
	return nil
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if found := o.aceJump.Narrow(msg); found != nil {
		o.setCursor(found.RowIdx)
		o.aceJump = nil
		if o.parentOp != nil {
			return common.RestoreOperation(o.parentOp)
		}
		return common.Close
	}
	return nil
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		switch msg.(type) {
		case intents.Cancel:
			o.aceJump = nil
			if o.parentOp != nil {
				return common.RestoreOperation(o.parentOp)
			}
			return common.Close
		case intents.Apply:
			if o.aceJump == nil || o.aceJump.First() == nil {
				return nil
			}
			o.setCursor(o.aceJump.First().RowIdx)
			o.aceJump = nil
			if o.parentOp != nil {
				return common.RestoreOperation(o.parentOp)
			}
			return common.Close
		}
	case tea.KeyMsg:
		return o.HandleKey(msg)
	}
	return nil
}

func (o *Operation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (o *Operation) Render(*jj.Commit, operations.RenderPosition) string {
	return ""
}

func (o *Operation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (o *Operation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
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

func (o *Operation) ResolveAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	return actions.ResolveByScopeStrict(o.Scope(), action, args)
}
