package bookmark

import (
	"github.com/charmbracelet/bubbles/textinput"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type SetBookmarkOperation struct {
	context  *context.MainContext
	revision string
	name     textinput.Model
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

func (s *SetBookmarkOperation) View() string {
	return s.name.View()
}

func (s *SetBookmarkOperation) IsFocused() bool {
	return true
}

func (s *SetBookmarkOperation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return s, common.Close
		case "enter":
			return s, s.context.RunCommand(jj.BookmarkSet(s.revision, s.name.Value()), common.Close, common.Refresh)
		}
	}
	var cmd tea.Cmd
	s.name, cmd = s.name.Update(msg)
	s.name.SetValue(strings.ReplaceAll(s.name.Value(), " ", "-"))
	return s, cmd
}

func (s *SetBookmarkOperation) Render(_ *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeCommitId {
		return ""
	}
	return s.name.View() + s.name.TextStyle.Render(" ")
}

func (s *SetBookmarkOperation) Name() string {
	return "bookmark"
}

func NewSetBookmarkOperation(context *context.MainContext, changeId string) (operations.Operation, tea.Cmd) {
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
	return op, op.Init()
}
