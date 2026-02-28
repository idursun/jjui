package render

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	draws := dl.DrawList()
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

	draws := dl.DrawList()
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

	draws := dl.DrawList()
	if len(draws) != 1 {
		t.Fatalf("expected 1 draw, got %d", len(draws))
	}

	interactions := dl.InteractionsList()
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

	draws := dl.DrawList()
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
	interactions := dl.InteractionsList()
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

	draws := dl.DrawList()
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
	draws := dl.DrawList()
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

	draws := dl.DrawList()
	if len(draws) != 3 {
		t.Fatalf("expected 3 draws, got %d", len(draws))
	}

	if draws[2].Rect.Min.X != 3 {
		t.Errorf("expected last segment at x=3, got x=%d", draws[2].Rect.Min.X)
	}
}
