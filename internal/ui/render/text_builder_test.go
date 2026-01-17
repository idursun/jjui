package render

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
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

func TestTextBuilder_WindowedInteractions(t *testing.T) {
	dl := NewDisplayContext()

	// Create a window and add clickable text
	windowRect := cellbuf.Rect(10, 5, 20, 1)
	windowedDl := dl.Window(windowRect, 10)

	windowedDl.Text(10, 5, 0).
		Write("Label: ").
		Clickable("Click1", lipgloss.Style{}, testClickMsg{ID: 1}).
		Write(" ").
		Clickable("Click2", lipgloss.Style{}, testClickMsg{ID: 2}).
		Done()

	// Simulate mouse click on "Click1" (should be at x=17 after "Label: " which is 7 chars)
	mouseMsg := tea.MouseMsg{
		X:      17,
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, handled := dl.ProcessMouseEvent(mouseMsg)
	if !handled {
		t.Fatal("expected mouse event to be handled")
	}

	if result == nil {
		t.Fatal("expected result message, got nil")
	}

	msg, ok := result.(testClickMsg)
	if !ok {
		t.Fatalf("expected testClickMsg, got %T", result)
	}

	if msg.ID != 1 {
		t.Errorf("expected ID 1, got %d", msg.ID)
	}
}

func TestTextBuilder_WindowPriority(t *testing.T) {
	dl := NewDisplayContext()

	// Create lower-priority window (z=5)
	lowerWindow := dl.Window(cellbuf.Rect(0, 0, 50, 10), 5)
	lowerWindow.Text(10, 5, 0).
		Clickable("Lower", lipgloss.Style{}, testClickMsg{ID: 100}).
		Done()

	// Create higher-priority window (z=10) overlapping the same area
	higherWindow := dl.Window(cellbuf.Rect(10, 5, 10, 1), 10)
	higherWindow.Text(10, 5, 0).
		Clickable("Higher", lipgloss.Style{}, testClickMsg{ID: 200}).
		Done()

	// Click at position that's inside both windows
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, handled := dl.ProcessMouseEvent(mouseMsg)
	if !handled {
		t.Fatal("expected mouse event to be handled")
	}

	msg, ok := result.(testClickMsg)
	if !ok {
		t.Fatalf("expected testClickMsg, got %T", result)
	}

	// Should get the higher-priority window's message
	if msg.ID != 200 {
		t.Errorf("expected ID 200 (higher priority), got %d", msg.ID)
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
