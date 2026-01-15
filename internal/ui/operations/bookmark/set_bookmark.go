package bookmark

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*SetBookmarkOperation)(nil)
var _ common.Editable = (*SetBookmarkOperation)(nil)

type SetBookmarkOperation struct {
	context  *context.MainContext
	revision string
	name     textinput.Model
}

func (s *SetBookmarkOperation) IsEditing() bool {
	return true
}

func (s *SetBookmarkOperation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return common.Close
		case "enter":
			return s.context.RunCommand(jj.BookmarkSet(s.revision, s.name.Value()), common.Close, common.Refresh)
		}
	}
	var cmd tea.Cmd
	s.name, cmd = s.name.Update(msg)
	s.name.SetValue(strings.ReplaceAll(s.name.Value(), " ", "-"))
	return cmd
}

func (s *SetBookmarkOperation) Init() tea.Cmd {
	if output, err := s.context.RunCommandImmediate(jj.BookmarkListMovable(s.revision)); err == nil {
		bookmarks := jj.ParseBookmarkListOutput(string(output))
		var suggestions []string
		for _, b := range bookmarks {
			if b.Name != "" && !b.Backwards {
				suggestions = append(suggestions, b.Name)
			}
		}
		s.name.SetSuggestions(suggestions)
	}

	return textinput.Blink
}

func (s *SetBookmarkOperation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := s.viewContent()
	w, h := lipgloss.Size(content)
	rect := cellbuf.Rect(box.R.Min.X, box.R.Min.Y, w, h)
	dl.AddDraw(rect, content, 0)
}

func (s *SetBookmarkOperation) IsFocused() bool {
	return true
}

func (s *SetBookmarkOperation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeCommitId || commit.GetChangeId() != s.revision {
		return ""
	}
	return s.viewContent() + s.name.TextStyle.Render(" ")
}

func (s *SetBookmarkOperation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (s *SetBookmarkOperation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (s *SetBookmarkOperation) Name() string {
	return "bookmark"
}

func NewSetBookmarkOperation(context *context.MainContext, changeId string) *SetBookmarkOperation {
	dimmedStyle := common.DefaultPalette.Get("revisions dimmed").Inline(true)
	textStyle := common.DefaultPalette.Get("revisions text").Inline(true)
	t := textinput.New()
	t.Width = 0
	t.ShowSuggestions = true
	t.CharLimit = 120
	t.Prompt = ""
	t.TextStyle = textStyle
	t.PromptStyle = t.TextStyle
	t.Cursor.TextStyle = t.TextStyle
	t.CompletionStyle = dimmedStyle
	t.PlaceholderStyle = t.CompletionStyle
	t.SetValue("")
	t.Focus()

	op := &SetBookmarkOperation{
		name:     t,
		revision: changeId,
		context:  context,
	}
	return op
}

func (s *SetBookmarkOperation) viewContent() string {
	return s.name.View()
}
