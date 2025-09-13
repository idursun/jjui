package ace_jump

import (
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	models2 "github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/ace_jump"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var (
	_ operations.Operation       = (*Operation)(nil)
	_ operations.SegmentRenderer = (*Operation)(nil)
	_ view.IViewModel            = (*Operation)(nil)
	_ help.KeyMap                = (*Operation)(nil)
)

type Operation struct {
	*view.ViewNode
	renderer *list.ListRenderer[*models2.RevisionItem]
	context  *context.MainContext
	aceJump  *ace_jump.AceJump
	keymap   config.KeyMappings[key.Binding]
}

func NewOperation(ctx *context.MainContext, renderer *list.ListRenderer[*models2.RevisionItem]) view.IViewModel {
	return &Operation{
		context:  ctx,
		renderer: renderer,
		keymap:   config.Current.GetKeyMap(),
		aceJump:  ace_jump.NewAceJump(),
	}
}

func (o *Operation) RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row *models2.RevisionItem) string {
	style := currentStyle
	if aceIdx := o.aceJumpIndex(segment.Text, row.Row); aceIdx > -1 {
		mid := lipgloss.NewRange(aceIdx, aceIdx+1, style.Reverse(true))
		return lipgloss.StyleRanges(style.Render(segment.Text), mid)
	}
	return ""
}

func (o *Operation) aceJumpIndex(text string, row models2.Row) int {
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

func (o *Operation) Mount(v *view.ViewNode) {
	o.ViewNode = v
	v.Id = "ace jump"
	revisions := view.RevisionsViewId
	v.KeyDelegation = &revisions
}

func (o *Operation) Init() tea.Cmd {
	o.aceJump = o.findAceKeys()
	return nil
}

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, o.keymap.Cancel):
			o.aceJump = nil
			o.ViewManager.StopEditing()
			o.ViewManager.UnregisterView(o.GetId())
			return o, nil
		case key.Matches(msg, o.keymap.Apply):
			o.context.Revisions.Cursor = o.aceJump.First().RowIdx
			o.aceJump = nil
			return o, nil
		default:
			log.Printf("received message: %T", msg)
			if found := o.aceJump.Narrow(msg); found != nil {
				o.context.Revisions.Cursor = found.RowIdx
				o.aceJump = nil
				o.ViewManager.StopEditing()
				o.ViewManager.UnregisterView(o.GetId())
				return o, nil
			}
		}
		return o, nil
	}
	return o, nil
}

func (o *Operation) View() string {
	return ""
}

func (o *Operation) GetId() view.ViewId {
	return "ace jump"
}

func (o *Operation) Render(*models2.Commit, operations.RenderPosition) string {
	return ""
}

func (o *Operation) findAceKeys() *ace_jump.AceJump {
	aj := ace_jump.NewAceJump()
	first, last := o.renderer.FirstRowIndex, o.renderer.LastRowIndex
	if first == -1 || last == -1 {
		return nil // wait until rendered
	}
	for i := range last - first + 1 {
		i += first
		row := o.context.Revisions.Items[i]
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
