package bookmark

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/idursun/jjui/internal/ui/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ operations.Operation = (*SetBookmarkOperation)(nil)
var _ view.IHasActionMap = (*SetBookmarkOperation)(nil)

type SetBookmarkOperation struct {
	context  *context.MainContext
	revision string
	name     textinput.Model
}

func (s *SetBookmarkOperation) GetActionMap() map[string]common.Action {
	return map[string]common.Action{
		"esc":   {Id: "close set_bookmark", Args: nil},
		"enter": {Id: "set_bookmark.accept", Args: nil},
	}
}

func (s *SetBookmarkOperation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.InvokeActionMsg:
		switch msg.Action.Id {
		case "set_bookmark.accept":
			return s, s.context.RunCommand(jj.BookmarkSet(s.revision, s.name.Value()), common.Close, common.Refresh)
		}
	}
	var cmd tea.Cmd
	s.name, cmd = s.name.Update(msg)
	s.name.SetValue(strings.ReplaceAll(s.name.Value(), " ", "-"))
	return s, cmd
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

func (s *SetBookmarkOperation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeCommitId || commit.GetChangeId() != s.revision {
		return ""
	}
	return s.name.View() + s.name.TextStyle.Render(" ")
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
