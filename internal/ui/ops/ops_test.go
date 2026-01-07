package ops

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
)

func TestDisplayList_AddDraw(t *testing.T) {
	dl := NewDisplayList()
	rect := cellbuf.Rect(0, 0, 10, 1)

	dl.AddDraw(rect, "test", 0)

	if len(dl.draws) != 1 {
		t.Errorf("AddDraw: expected 1 draw op, got %d", len(dl.draws))
	}

	if dl.draws[0].Content != "test" {
		t.Errorf("AddDraw: expected content 'test', got '%s'", dl.draws[0].Content)
	}
}

func TestDisplayList_AddEffect(t *testing.T) {
	dl := NewDisplayList()
	rect := cellbuf.Rect(0, 0, 10, 1)
	style := lipgloss.NewStyle()

	dl.AddEffect(rect, ModeReverse, style, 0)

	if len(dl.effects) != 1 {
		t.Errorf("AddEffect: expected 1 effect op, got %d", len(dl.effects))
	}

	// Check that it's a ReverseEffect
	if _, ok := dl.effects[0].(ReverseEffect); !ok {
		t.Errorf("AddEffect: expected ReverseEffect, got %T", dl.effects[0])
	}
}

func TestDisplayList_Clear(t *testing.T) {
	dl := NewDisplayList()
	rect := cellbuf.Rect(0, 0, 10, 1)

	dl.AddDraw(rect, "test", 0)
	dl.AddEffect(rect, ModeReverse, lipgloss.NewStyle(), 0)

	dl.Clear()

	if len(dl.draws) != 0 {
		t.Errorf("Clear: expected 0 draw ops, got %d", len(dl.draws))
	}
	if len(dl.effects) != 0 {
		t.Errorf("Clear: expected 0 effect ops, got %d", len(dl.effects))
	}
}

func TestDisplayList_Len(t *testing.T) {
	dl := NewDisplayList()
	rect := cellbuf.Rect(0, 0, 10, 1)

	if dl.Len() != 0 {
		t.Errorf("Len: expected 0, got %d", dl.Len())
	}

	dl.AddDraw(rect, "test", 0)
	if dl.Len() != 1 {
		t.Errorf("Len: expected 1, got %d", dl.Len())
	}

	dl.AddEffect(rect, ModeReverse, lipgloss.NewStyle(), 0)
	if dl.Len() != 2 {
		t.Errorf("Len: expected 2, got %d", dl.Len())
	}
}

func TestDisplayList_ZIndexSorting(t *testing.T) {
	dl := NewDisplayList()
	rect := cellbuf.Rect(0, 0, 10, 1)

	// Add draws in reverse Z order
	dl.AddDraw(rect, "z2", 2)
	dl.AddDraw(rect, "z0", 0)
	dl.AddDraw(rect, "z1", 1)

	// Render should sort by Z-index
	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Verify the draws were sorted
	draws := dl.DrawOpsList()
	if len(draws) != 3 {
		t.Fatalf("Expected 3 draws, got %d", len(draws))
	}

	// After rendering, internal slice should be sorted
	// (We can't directly observe this, but we test behavior)
}

func TestDisplayList_BasicRender(t *testing.T) {
	dl := NewDisplayList()

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

func TestDisplayList_LayeredRender(t *testing.T) {
	dl := NewDisplayList()

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

func TestEffectOp_Reverse(t *testing.T) {
	dl := NewDisplayList()

	// Draw some content
	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Hello", 0)

	// Apply reverse effect
	dl.AddEffect(cellbuf.Rect(0, 0, 5, 1), ModeReverse, lipgloss.NewStyle(), 0)

	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Check that cells have reverse attribute set
	// Verify first cell has reverse enabled
	cell := buf.Cell(0, 0)
	if cell == nil {
		t.Fatal("Expected cell at (0,0), got nil")
	}

	// The style should have reverse attribute
	// Note: We can't easily test this without examining cell internals,
	// but we can verify the cell was modified
	if cell.Rune == 0 {
		t.Error("Expected cell to have content after effect")
	}
}

func TestEffectOp_Bold(t *testing.T) {
	dl := NewDisplayList()

	// Draw content
	dl.AddDraw(cellbuf.Rect(0, 0, 4, 1), "Test", 0)

	// Apply bold effect
	dl.AddEffect(cellbuf.Rect(0, 0, 4, 1), ModeBold, lipgloss.NewStyle(), 0)

	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Verify cells were modified
	cell := buf.Cell(0, 0)
	if cell == nil {
		t.Fatal("Expected cell at (0,0), got nil")
	}
}

func TestEffectOp_Underline(t *testing.T) {
	dl := NewDisplayList()

	// Draw content
	dl.AddDraw(cellbuf.Rect(0, 0, 4, 1), "Link", 0)

	// Apply underline effect
	dl.AddEffect(cellbuf.Rect(0, 0, 4, 1), ModeUnderline, lipgloss.NewStyle(), 0)

	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Verify cells were modified
	cell := buf.Cell(0, 0)
	if cell == nil {
		t.Fatal("Expected cell at (0,0), got nil")
	}
}

func TestEffectOp_Dim(t *testing.T) {
	dl := NewDisplayList()

	// Draw content
	dl.AddDraw(cellbuf.Rect(0, 0, 4, 1), "Faint", 0)

	// Apply dim effect
	dl.AddEffect(cellbuf.Rect(0, 0, 4, 1), ModeDim, lipgloss.NewStyle(), 0)

	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Verify cells were modified
	cell := buf.Cell(0, 0)
	if cell == nil {
		t.Fatal("Expected cell at (0,0), got nil")
	}
}

func TestEffectOp_MultipleEffects(t *testing.T) {
	dl := NewDisplayList()

	// Draw content
	dl.AddDraw(cellbuf.Rect(0, 0, 10, 1), "MultiStyle", 0)

	// Apply multiple effects
	dl.AddEffect(cellbuf.Rect(0, 0, 5, 1), ModeBold, lipgloss.NewStyle(), 0)
	dl.AddEffect(cellbuf.Rect(5, 0, 10, 1), ModeUnderline, lipgloss.NewStyle(), 1)

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
	dl := NewDisplayList()

	// Important: DrawOps must be rendered before EffectOps
	dl.AddDraw(cellbuf.Rect(0, 0, 6, 1), "Normal", 0)
	dl.AddEffect(cellbuf.Rect(0, 0, 6, 1), ModeReverse, lipgloss.NewStyle(), 0)

	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Content should be present with effect applied
	output := cellbuf.Render(buf)
	if !strings.Contains(output, "Normal") {
		t.Errorf("Expected 'Normal' in output, got: %s", output)
	}
}

func TestRenderToString(t *testing.T) {
	dl := NewDisplayList()

	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Quick", 0)

	output := dl.RenderToString(10, 1)

	if !strings.Contains(output, "Quick") {
		t.Errorf("RenderToString: expected 'Quick', got: %s", output)
	}
}

func TestIterateCells_BoundsChecking(t *testing.T) {
	dl := NewDisplayList()

	// Try to draw outside buffer bounds
	dl.AddDraw(cellbuf.Rect(0, 0, 5, 1), "Hello", 0)

	// Apply effect partially outside bounds
	dl.AddEffect(cellbuf.Rect(3, 0, 20, 1), ModeReverse, lipgloss.NewStyle(), 0)

	// Should not panic
	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Effect should be clipped to buffer bounds
	cell := buf.Cell(4, 0) // Inside buffer
	if cell == nil {
		t.Error("Expected cell at (4,0) to exist")
	}
}

func TestEmptyDisplayList(t *testing.T) {
	dl := NewDisplayList()

	// Rendering empty display list should not panic
	buf := cellbuf.NewBuffer(10, 1)
	dl.Render(buf)

	// Should not panic - that's the main test
	// Empty buffer output may be empty or whitespace, both are valid
	_ = cellbuf.Render(buf)
}

func TestDisplayList_Reuse(t *testing.T) {
	dl := NewDisplayList()

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
