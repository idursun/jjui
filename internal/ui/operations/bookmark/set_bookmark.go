package bookmark

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

var _ view.IViewModel = (*SetBookmarkOperation)(nil)
var _ help.KeyMap = (*SetBookmarkOperation)(nil)

type SetBookmarkOperation struct {
	*view.ViewNode
	context  *context.MainContext
	keymap   config.KeyMappings[key.Binding]
	revision *models.RevisionItem
	name     textinput.Model
}

func (s *SetBookmarkOperation) ShortHelp() []key.Binding {
	return []key.Binding{
		s.keymap.Cancel,
		s.keymap.Apply,
	}
}

func (s *SetBookmarkOperation) FullHelp() [][]key.Binding {
	//TODO implement me
	panic("implement me")
}

func (s *SetBookmarkOperation) GetId() view.ViewId {
	return "set bookmark"
}

func (s *SetBookmarkOperation) Mount(v *view.ViewNode) {
	s.ViewNode = v
	v.Id = "set bookmark"
}

func (s *SetBookmarkOperation) Init() tea.Cmd {
	if output, err := s.context.RunCommandImmediate(jj.Args(jj.BookmarkListMovableArgs{Revision: *s.revision})); err == nil {
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

func (s *SetBookmarkOperation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, s.keymap.Cancel):
			s.ViewManager.UnregisterView(s.Id)
			return s, nil
		case key.Matches(msg, s.keymap.Apply):
			s.ViewManager.UnregisterView(s.Id)
			return s, s.context.RunCommand(jj.Args(jj.BookmarkSetArgs{Revision: *s.revision, Bookmark: s.name.Value()}), common.Refresh)
		}
	}
	var cmd tea.Cmd
	s.name, cmd = s.name.Update(msg)
	s.name.SetValue(strings.ReplaceAll(s.name.Value(), " ", "-"))
	return s, cmd
}

func (s *SetBookmarkOperation) Render(_ *models.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeCommitId {
		return ""
	}
	return s.name.View() + s.name.TextStyle.Render(" ")
}

func NewSetBookmarkOperation(context *context.MainContext, revision *models.RevisionItem) view.IViewModel {
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
		keymap:   config.Current.GetKeyMap(),
		revision: revision,
		context:  context,
	}
	return op
}
