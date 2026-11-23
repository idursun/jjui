package bookmark

import (
	"math"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

var _ operations.Operation = (*SetBookmarkOperation)(nil)
var _ operations.SegmentRenderer = (*SetBookmarkOperation)(nil)
var _ common.Editable = (*SetBookmarkOperation)(nil)
var _ common.ScopeProvider = (*SetBookmarkOperation)(nil)

// unknownDistance is the distance of a bookmark on a commit outside the log.
const unknownDistance = math.MinInt32

type bookmarkCandidate struct {
	name     string
	commitId string
	// distance is the row offset from the target, used for ranking; positive
	// is below it.
	distance  int
	backwards bool
}

type SetBookmarkOperation struct {
	context         *context.MainContext
	revision        string
	logOrder        []*jj.Commit
	name            textinput.Model
	candidates      []bookmarkCandidate
	suggestionIndex int
	// cycleBase is the input a completion cycle started from.
	cycleBase string
	styles    struct {
		text   lipgloss.Style
		dimmed lipgloss.Style
	}
}

func (s *SetBookmarkOperation) IsEditing() bool {
	return true
}

func (s *SetBookmarkOperation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeSetBookmark,
			Leak:    common.LeakNone,
			Handler: s,
		},
	}
}

func (s *SetBookmarkOperation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Cancel:
		return common.Close, true
	case intents.Apply:
		name := s.name.Value()
		chosen := s.chosen()
		if name == "" {
			if chosen == nil {
				return nil, true
			}
			name = chosen.name
		}
		var extraFlags []string
		if chosen != nil && chosen.backwards {
			extraFlags = append(extraFlags, "--allow-backwards")
		}
		return s.context.RunCommand(jj.BookmarkSet(s.revision, name, extraFlags...), common.CloseApplied, common.Refresh), true
	case intents.AutocompleteCycle:
		s.cycleSuggestion(intent.Reverse)
		return nil, true
	}
	return nil, false
}

func (s *SetBookmarkOperation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := s.HandleIntent(msg)
		return cmd
	}
	before := s.name.Value()
	var cmd tea.Cmd
	s.name, cmd = s.name.Update(msg)
	s.name.SetValue(strings.ReplaceAll(s.name.Value(), " ", "-"))
	if s.name.Value() != before {
		s.suggestionIndex = -1
	}
	return cmd
}

func (s *SetBookmarkOperation) Init() tea.Cmd {
	if output, err := s.context.RunCommandImmediate(jj.BookmarkListMovable(s.revision)); err == nil {
		s.candidates = rankCandidates(s.revision, s.logOrder, jj.ParseBookmarkListOutput(string(output)))
		suggestions := make([]string, len(s.candidates))
		for i, c := range s.candidates {
			suggestions[i] = c.name
		}
		s.name.SetSuggestions(suggestions)
	}

	return textinput.Blink
}

// rankCandidates orders bookmarks nearest first: those below the target, then
// those above it, then those outside the log.
func rankCandidates(revision string, logOrder []*jj.Commit, bookmarks []jj.Bookmark) []bookmarkCandidate {
	targetIndex := slices.IndexFunc(logOrder, func(c *jj.Commit) bool {
		return c != nil && c.GetChangeId() == revision
	})
	var candidates []bookmarkCandidate
	for _, b := range bookmarks {
		if b.Name == "" {
			continue
		}
		c := bookmarkCandidate{name: b.Name, distance: unknownDistance, backwards: b.Backwards}
		if b.Local != nil {
			c.commitId = b.Local.CommitId
		}
		if targetIndex >= 0 && c.commitId != "" {
			if i := slices.IndexFunc(logOrder, func(commit *jj.Commit) bool {
				return commit != nil && strings.HasPrefix(commit.CommitId, c.commitId)
			}); i >= 0 {
				c.distance = i - targetIndex
			}
		}
		if c.distance == 0 {
			continue
		}
		// Without the target in the log, bookmarks already on it can't be told
		// apart from backwards ones.
		if targetIndex < 0 && c.backwards {
			continue
		}
		candidates = append(candidates, c)
	}
	slices.SortStableFunc(candidates, func(a, b bookmarkCandidate) int {
		if a.distance == b.distance {
			return strings.Compare(a.name, b.name)
		}
		if a.distance >= 0 && b.distance >= 0 {
			return a.distance - b.distance
		}
		return b.distance - a.distance
	})
	return candidates
}

func (s *SetBookmarkOperation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := s.viewContent()
	w, h := lipgloss.Size(content)
	rect := layout.Rect(box.R.Min.X, box.R.Min.Y, w, h)
	dl.AddDraw(rect, content, 0)
	dl.SetCursorInRect(s.name.Cursor(), rect, 0, 0)
}

func (s *SetBookmarkOperation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeCommitId || commit.GetChangeId() != s.revision {
		return ""
	}
	return s.viewContent() + s.preview() + s.name.Styles().Focused.Text.Render(" ")
}

// preview names the bookmark that would move here.
func (s *SetBookmarkOperation) preview() string {
	chosen := s.chosen()
	if chosen == nil {
		return ""
	}
	var parts []string
	// A typed name is already visible in the input.
	if s.name.Value() == "" {
		parts = append(parts, s.styles.dimmed.Render(" ← "), s.styles.text.Render(chosen.name))
	}
	if chosen.backwards {
		parts = append(parts, s.styles.dimmed.Render(" backwards"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// RenderSegment reverses the bookmark that would move.
func (s *SetBookmarkOperation) RenderSegment(style lipgloss.Style, segment *screen.Segment, row parser.Row) string {
	if row.Commit == nil || row.Commit.GetChangeId() == s.revision || !segment.HasLabel("name") {
		return ""
	}
	name := jj.UnquoteRefName(segment.Text)
	chosen := s.chosen()
	if chosen == nil || chosen.name != name || localBookmarkSegment(row, name) != segment {
		return ""
	}
	return style.Reverse(true).Render(segment.Text)
}

// chosen is the candidate the input names, or the nearest one while it is empty.
func (s *SetBookmarkOperation) chosen() *bookmarkCandidate {
	value := strings.TrimSpace(s.name.Value())
	for i := range s.candidates {
		if value == "" || s.candidates[i].name == value {
			return &s.candidates[i]
		}
	}
	return nil
}

func (s *SetBookmarkOperation) InlineCursor(commit *jj.Commit, pos operations.RenderPosition) *tea.Cursor {
	if pos != operations.RenderBeforeCommitId || commit.GetChangeId() != s.revision {
		return nil
	}
	return s.name.Cursor()
}

func (s *SetBookmarkOperation) Name() string {
	return "set bookmark"
}

func NewSetBookmarkOperation(context *context.MainContext, changeId string, initialValue string, logOrder []*jj.Commit) *SetBookmarkOperation {
	t := textinput.New()
	t.ShowSuggestions = true
	t.CharLimit = 120
	t.Prompt = ""
	t.SetVirtualCursor(false)

	t.SetValue(initialValue)
	t.CursorEnd()
	t.Focus()

	op := &SetBookmarkOperation{
		name:     t,
		revision: changeId,
		context:  context,
		logOrder: logOrder,
		// -1 means no active completion cycle.
		suggestionIndex: -1,
	}
	op.styles.text = theme.DefaultPalette.Get("revisions", "", "text", false).Inline(true)
	op.styles.dimmed = theme.DefaultPalette.Get("revisions", "", "dimmed", false).Inline(true)
	return op
}

func (s *SetBookmarkOperation) viewContent() string {
	styles := s.name.Styles()
	styles.Focused.Text = s.styles.text
	styles.Focused.Prompt = s.styles.text
	styles.Focused.Suggestion = s.styles.dimmed
	styles.Focused.Placeholder = s.styles.dimmed
	styles.Blurred.Text = s.styles.text
	styles.Blurred.Prompt = s.styles.text
	styles.Blurred.Suggestion = s.styles.dimmed
	styles.Blurred.Placeholder = s.styles.dimmed
	s.name.SetStyles(styles)

	return s.name.View()
}

func (s *SetBookmarkOperation) cycleSuggestion(reverse bool) {
	if s.suggestionIndex < 0 {
		s.cycleBase = s.name.Value()
	}
	candidates := s.matchingSuggestions(s.cycleBase)
	if len(candidates) == 0 {
		return
	}
	delta := 1
	if reverse {
		delta = -1
	}
	if s.suggestionIndex < 0 {
		if reverse {
			s.suggestionIndex = len(candidates) - 1
		} else {
			s.suggestionIndex = 0
		}
	} else {
		s.suggestionIndex = (s.suggestionIndex + delta + len(candidates)) % len(candidates)
	}
	s.name.SetValue(candidates[s.suggestionIndex])
	s.name.CursorEnd()
}

func (s *SetBookmarkOperation) matchingSuggestions(input string) []string {
	if len(s.candidates) == 0 {
		return nil
	}
	needle := strings.TrimSpace(input)
	matches := make([]string, 0, len(s.candidates))
	for _, c := range s.candidates {
		if strings.HasPrefix(c.name, needle) {
			matches = append(matches, c.name)
		}
	}
	return matches
}
