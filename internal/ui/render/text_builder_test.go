package render

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/ui/layout"
)

type testClickMsg struct {
	ID int
}

func TestTextBuilder_Write(t *testing.T) {
	dl := NewDisplayContext()

	dl.Text(0, 0, 0).
		Write("Hello").
		Done()

	draws := make([]Draw, len(dl.draws))
	for i, op := range dl.draws {
		draws[i] = op.Draw
	}
	if len(draws) != 1 {
		t.Fatalf("expected 1 draw, got %d", len(draws))
	}

	if draws[0].Content != "Hello" {
		t.Errorf("expected content 'Hello', got '%s'", draws[0].Content)
	}
}

func TestTextBuilder_Styled(t *testing.T) {
	dl := NewDisplayContext()
	style := lipgloss.NewStyle().Bold(true)

	dl.Text(0, 0, 0).
		Styled("Bold", style).
		Done()

	draws := make([]Draw, len(dl.draws))
	for i, op := range dl.draws {
		draws[i] = op.Draw
	}
	if len(draws) != 1 {
		t.Fatalf("expected 1 draw, got %d", len(draws))
	}

	// Content should contain the text (styling may or may not add ANSI codes depending on terminal)
	if draws[0].Content != style.Render("Bold") {
		t.Errorf("expected styled content '%s', got '%s'", style.Render("Bold"), draws[0].Content)
	}
}

func TestTextBuilder_Clickable(t *testing.T) {
	dl := NewDisplayContext()

	dl.Text(0, 0, 0).
		Clickable("Click", lipgloss.Style{}, testClickMsg{ID: 1}).
		Done()

	draws := make([]Draw, len(dl.draws))
	for i, op := range dl.draws {
		draws[i] = op.Draw
	}
	if len(draws) != 1 {
		t.Fatalf("expected 1 draw, got %d", len(draws))
	}

	interactions := make([]InteractionOp, len(dl.interactions))
	for i, op := range dl.interactions {
		interactions[i] = op.InteractionOp
	}
	if len(interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(interactions))
	}

	if interactions[0].Type != InteractionClick {
		t.Errorf("expected InteractionClick type, got %v", interactions[0].Type)
	}

	msg, ok := interactions[0].Msg.(testClickMsg)
	if !ok {
		t.Fatalf("expected testClickMsg, got %T", interactions[0].Msg)
	}
	if msg.ID != 1 {
		t.Errorf("expected ID 1, got %d", msg.ID)
	}
}

func TestTextBuilder_MultipleSegments(t *testing.T) {
	dl := NewDisplayContext()

	dl.Text(0, 0, 0).
		Write("A").
		Clickable("B", lipgloss.Style{}, testClickMsg{ID: 1}).
		Write("C").
		Done()

	draws := make([]Draw, len(dl.draws))
	for i, op := range dl.draws {
		draws[i] = op.Draw
	}
	if len(draws) != 3 {
		t.Fatalf("expected 3 draws, got %d", len(draws))
	}

	// Check positions
	if draws[0].Rect.Min.X != 0 {
		t.Errorf("expected first segment at x=0, got x=%d", draws[0].Rect.Min.X)
	}
	if draws[1].Rect.Min.X != 1 {
		t.Errorf("expected second segment at x=1, got x=%d", draws[1].Rect.Min.X)
	}
	if draws[2].Rect.Min.X != 2 {
		t.Errorf("expected third segment at x=2, got x=%d", draws[2].Rect.Min.X)
	}

	// Check only one interaction (for "B")
	interactions := make([]InteractionOp, len(dl.interactions))
	for i, op := range dl.interactions {
		interactions[i] = op.InteractionOp
	}
	if len(interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(interactions))
	}
}

func TestTextBuilder_BackdropSwallowsClick(t *testing.T) {
	dl := NewDisplayContext()

	backdropRect := layout.Rect(0, 0, 50, 10)
	dl.AddBackdrop(backdropRect, 10)

	dl.Text(5, 5, 20).
		Clickable("Click", lipgloss.Style{}, testClickMsg{ID: 1}).
		Done()

	hitMsg := tea.MouseClickMsg{X: 5, Y: 5, Button: tea.MouseLeft}
	result, handled := dl.ProcessMouseEvent(hitMsg)
	if !handled {
		t.Fatal("expected mouse event to be handled")
	}
	msg, ok := result.(testClickMsg)
	if !ok {
		t.Fatalf("expected testClickMsg, got %T", result)
	}
	if msg.ID != 1 {
		t.Errorf("expected ID 1, got %d", msg.ID)
	}

	missMsg := tea.MouseClickMsg{X: 40, Y: 5, Button: tea.MouseLeft}
	result2, handled2 := dl.ProcessMouseEvent(missMsg)
	if !handled2 {
		t.Fatal("expected backdrop to swallow the click (handled=true)")
	}
	if result2 != nil {
		t.Errorf("expected nil message from backdrop, got %v", result2)
	}
}

func TestTextBuilder_EmptyText(t *testing.T) {
	dl := NewDisplayContext()

	dl.Text(0, 0, 0).
		Write("").
		Write("Hello").
		Done()

	draws := make([]Draw, len(dl.draws))
	for i, op := range dl.draws {
		draws[i] = op.Draw
	}
	// Empty string should be skipped
	if len(draws) != 1 {
		t.Fatalf("expected 1 draw (empty skipped), got %d", len(draws))
	}

	if draws[0].Content != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", draws[0].Content)
	}
}

func TestTextBuilder_NewLineAndMeasure(t *testing.T) {
	dl := NewDisplayContext()

	tb := dl.Text(0, 0, 0).
		Write("A").
		NewLine().
		Write("BB")

	width, height := tb.Measure()
	if width != 2 || height != 2 {
		t.Fatalf("expected width=2 height=2, got width=%d height=%d", width, height)
	}

	tb.Done()
	draws := make([]Draw, len(dl.draws))
	for i, op := range dl.draws {
		draws[i] = op.Draw
	}
	if len(draws) != 2 {
		t.Fatalf("expected 2 draws, got %d", len(draws))
	}

	if draws[0].Rect.Min.X != 0 || draws[0].Rect.Min.Y != 0 {
		t.Errorf("expected first segment at 0,0 got %d,%d", draws[0].Rect.Min.X, draws[0].Rect.Min.Y)
	}
	if draws[1].Rect.Min.X != 0 || draws[1].Rect.Min.Y != 1 {
		t.Errorf("expected second segment at 0,1 got %d,%d", draws[1].Rect.Min.X, draws[1].Rect.Min.Y)
	}
}

func TestTextBuilder_Space(t *testing.T) {
	dl := NewDisplayContext()

	dl.Text(0, 0, 0).
		Write("A").
		Space(2).
		Write("B").
		Done()

	draws := make([]Draw, len(dl.draws))
	for i, op := range dl.draws {
		draws[i] = op.Draw
	}
	if len(draws) != 3 {
		t.Fatalf("expected 3 draws, got %d", len(draws))
	}

	if draws[2].Rect.Min.X != 3 {
		t.Errorf("expected last segment at x=3, got x=%d", draws[2].Rect.Min.X)
	}
}

func renderPlainLines(dl *DisplayContext, width, height int) []string {
	lines := strings.Split(ansi.Strip(dl.RenderToString(width, height)), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return lines
}

func TestTextBuilder_TextInTruncatesWithEllipsis(t *testing.T) {
	tests := []struct {
		name  string
		build func(tb *TextBuilder)
		want  string
	}{
		{"fits", func(tb *TextBuilder) { tb.Write("abc") }, "abc"},
		{"cuts inside a segment", func(tb *TextBuilder) { tb.Write("ab").Write("cdefgh") }, "abcd…"},
		{"marks a segment that fills the line exactly", func(tb *TextBuilder) { tb.Write("abcde").Write("f") }, "abcd…"},
		{"counts columns, not bytes", func(tb *TextBuilder) { tb.Write("日本語です") }, "日本…"},
		{"keeps cutting after a newline", func(tb *TextBuilder) { tb.Write("abcdefg").NewLine().Write("hi") }, "abcd…\nhi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dl := NewDisplayContext()
			tb := dl.TextIn(layout.Rect(0, 0, 5, 2), 0)
			tt.build(tb)
			tb.Done()

			got := strings.TrimRight(strings.Join(renderPlainLines(dl, 8, 2), "\n"), "\n")
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestTextBuilder_TextInKeepsEllipsisInSegmentStyle(t *testing.T) {
	dl := NewDisplayContext()
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	dl.TextIn(layout.Rect(0, 0, 4, 1), 0).Styled("abcdef", style).Done()

	if len(dl.draws) != 1 {
		t.Fatalf("expected 1 draw, got %d", len(dl.draws))
	}
	if want := style.Render("abc…"); dl.draws[0].Content != want {
		t.Fatalf("expected %q, got %q", want, dl.draws[0].Content)
	}
}

func TestTextBuilder_TextInDropsLinesBelowRect(t *testing.T) {
	dl := NewDisplayContext()

	tb := dl.TextIn(layout.Rect(2, 1, 10, 1), 0).
		Write("one").
		NewLine().
		Clickable("two", lipgloss.NewStyle(), testClickMsg{ID: 2})

	if _, height := tb.Measure(); height != 1 {
		t.Fatalf("expected measured height 1, got %d", height)
	}
	tb.Done()

	if len(dl.draws) != 1 {
		t.Fatalf("expected 1 draw, got %d", len(dl.draws))
	}
	if r := dl.draws[0].Rect; r.Min.X != 2 || r.Min.Y != 1 {
		t.Errorf("expected draw at 2,1 got %d,%d", r.Min.X, r.Min.Y)
	}
	if len(dl.interactions) != 0 {
		t.Errorf("expected no interactions for the dropped line, got %d", len(dl.interactions))
	}
}

func TestTextBuilder_TextInEmptyRectDrawsNothing(t *testing.T) {
	dl := NewDisplayContext()

	dl.TextIn(layout.Rect(0, 0, 0, 0), 0).Write("abc").Done()

	if len(dl.draws) != 0 {
		t.Fatalf("expected no draws, got %d", len(dl.draws))
	}
}
