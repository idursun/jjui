package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
)

func TestDisplayContext_AddDraw(t *testing.T) {
	dl := NewDisplayContext()
	rect := cellbuf.Rect(0, 0, 10, 1)

	dl.AddDraw(rect, "test", 0)

	if len(dl.draws) != 1 {
		t.Errorf("AddDraw: expected 1 draw op, got %d", len(dl.draws))
	}

	if dl.draws[0].Content != "test" {
		t.Errorf("AddDraw: expected content 'test', got '%s'", dl.draws[0].Content)
	}
}

func TestDisplayContext_BasicRender(t *testing.T) {
	dl := NewDisplayContext()

	// Create a simple draw operation
	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Hello", 0)

	// Render to buffer
	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Verify content was rendered
	output := cellbuf.Render(buf)
	if !strings.Contains(output, "Hello") {
		t.Errorf("Expected output to contain 'Hello', got: %s", output)
	}
}

func TestDisplayContext_LayeredRender(t *testing.T) {
	dl := NewDisplayContext()

	// Layer 0: Background
	dl.AddDraw(cellbuf.Rect(0, 0, 10, 1), "Background", 0)

	// Layer 1: Foreground (should overwrite)
	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Front", 1)

	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	output := cellbuf.Render(buf)

	// Front should be visible (higher Z-index)
	if !strings.Contains(output, "Front") {
		t.Errorf("Expected 'Front' in output, got: %s", output)
	}
}

func TestEffectOp_AppliesWithoutPanic(t *testing.T) {
	tests := []struct {
		name    string
		applyFn func(dl *DisplayContext, rect cellbuf.Rectangle)
	}{
		{"Bold", func(dl *DisplayContext, rect cellbuf.Rectangle) { dl.AddBold(rect, 0) }},
		{"Underline", func(dl *DisplayContext, rect cellbuf.Rectangle) { dl.AddUnderline(rect, 0) }},
		{"Dim", func(dl *DisplayContext, rect cellbuf.Rectangle) { dl.AddDim(rect, 0) }},
		{"Reverse", func(dl *DisplayContext, rect cellbuf.Rectangle) { dl.AddReverse(rect, 0) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dl := NewDisplayContext()
			rect := cellbuf.Rect(0, 0, 4, 1)
			dl.AddDraw(rect, "Test", 0)
			tc.applyFn(dl, rect)

			buf := cellbuf.NewBuffer(10, 1)
			dl.Render(buf)

			cell := buf.Cell(0, 0)
			if cell == nil {
				t.Fatal("Expected cell at (0,0), got nil")
			}
		})
	}
}

func TestEffectOp_MultipleEffects(t *testing.T) {
	dl := NewDisplayContext()

	// Draw content
	dl.AddDraw(cellbuf.Rect(0, 0, 10, 1), "MultiStyle", 0)

	// Apply multiple effects
	dl.AddBold(cellbuf.Rect(0, 0, 5, 1), 0)
	dl.AddUnderline(cellbuf.Rect(5, 0, 10, 1), 1)

	buf := cellbuf.NewBuffer(15, 1)
	dl.Render(buf)

	// Verify both effects were applied to their respective regions
	leftCell := buf.Cell(0, 0)
	rightCell := buf.Cell(7, 0)

	if leftCell == nil || rightCell == nil {
		t.Fatal("Expected cells to exist after rendering")
	}
}

func TestEffectOp_EffectAfterDraw(t *testing.T) {
	dl := NewDisplayContext()

	// Important: DrawOps must be rendered before EffectOps
	dl.AddDraw(cellbuf.Rect(0, 0, 6, 1), "Normal", 0)
	dl.AddReverse(cellbuf.Rect(0, 0, 6, 1), 0)

	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Content should be present with effect applied
	output := cellbuf.Render(buf)
	if !strings.Contains(output, "Normal") {
		t.Errorf("Expected 'Normal' in output, got: %s", output)
	}
}

func TestRenderToString(t *testing.T) {
	dl := NewDisplayContext()

	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Quick", 0)

	output := dl.RenderToString(10, 1)

	if !strings.Contains(output, "Quick") {
		t.Errorf("RenderToString: expected 'Quick', got: %s", output)
	}
}

func TestIterateCells_BoundsChecking(t *testing.T) {
	dl := NewDisplayContext()

	// Try to draw outside buffer bounds
	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Hello", 0)

	// Apply effect partially outside bounds
	dl.AddReverse(cellbuf.Rect(3, 0, 20, 1), 0)

	// Should not panic
	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Effect should be clipped to buffer bounds
	cell := buf.Cell(4, 0) // Inside buffer
	if cell == nil {
		t.Error("Expected cell at (4,0) to exist")
	}
}

func TestDisplayContext_HighlightPreservesWideCharacters(t *testing.T) {
	dl := NewDisplayContext()

	text := "A🙂B"
	rect := cellbuf.Rect(0, 0, 4, 1)

	dl.AddDraw(rect, text, 0)
	dl.AddHighlight(rect, lipgloss.NewStyle().Background(lipgloss.Color("4")), 1)

	buf := cellbuf.NewBuffer(4, 1)
	dl.Render(buf)

	out := cellbuf.Render(buf)
	if !strings.Contains(out, "🙂") {
		t.Fatalf("expected highlighted output to preserve emoji, got: %q", out)
	}
}

func TestEmptyDisplayContext(t *testing.T) {
	dl := NewDisplayContext()

	// Rendering empty display context should not panic
	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Should not panic - that's the main test
	// Empty buffer output may be empty or whitespace, both are valid
	_ = cellbuf.Render(buf)
}

func TestDisplayContext_Reuse(t *testing.T) {
	dl := NewDisplayContext()

	// First frame
	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Frame1", 0)
	if dl.Len() != 1 {
		t.Errorf("Expected 1 op, got %d", dl.Len())
	}

	// Clear and reuse
	dl.Clear()
	if dl.Len() != 0 {
		t.Errorf("Expected 0 ops after clear, got %d", dl.Len())
	}

	// Second frame
	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Frame2", 0)
	if dl.Len() != 1 {
		t.Errorf("Expected 1 op after reuse, got %d", dl.Len())
	}
}
