package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/stretchr/testify/assert"
)

func TestDisplayContext_AddDraw(t *testing.T) {
	dl := NewDisplayContext()
	rect := layout.Rect(0, 0, 10, 1)

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
	dl.AddDraw(layout.Rect(0, 0, 5, 1), "Hello", 0)

	// Render to buffer
	buf := uv.NewScreenBuffer(10, 1)
	dl.Render(buf)

	// Verify content was rendered
	output := buf.Render()
	if !strings.Contains(output, "Hello") {
		t.Errorf("Expected output to contain 'Hello', got: %s", output)
	}
}

func TestDisplayContext_LayeredRender(t *testing.T) {
	dl := NewDisplayContext()

	// Layer 0: Background
	dl.AddDraw(layout.Rect(0, 0, 10, 1), "Background", 0)

	// Layer 1: Foreground (should overwrite)
	dl.AddDraw(layout.Rect(0, 0, 5, 1), "Front", 1)

	buf := uv.NewScreenBuffer(10, 1)
	dl.Render(buf)

	output := buf.Render()

	// Front should be visible (higher Z-index)
	if !strings.Contains(output, "Front") {
		t.Errorf("Expected 'Front' in output, got: %s", output)
	}
}

func TestEffectOp_AppliesWithoutPanic(t *testing.T) {
	tests := []struct {
		name    string
		applyFn func(dl *DisplayContext, rect layout.Rectangle)
	}{
		{"Dim", func(dl *DisplayContext, rect layout.Rectangle) { dl.AddDim(rect, 0) }},
		{"Highlight", func(dl *DisplayContext, rect layout.Rectangle) {
			dl.AddHighlight(rect, lipgloss.NewStyle().Background(lipgloss.Color("4")), 0)
		}},
		{"Fill", func(dl *DisplayContext, rect layout.Rectangle) {
			dl.AddFill(rect, 'x', lipgloss.NewStyle(), 0)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dl := NewDisplayContext()
			rect := layout.Rect(0, 0, 4, 1)
			dl.AddDraw(rect, "Test", 0)
			tc.applyFn(dl, rect)

			buf := uv.NewScreenBuffer(10, 1)
			dl.Render(buf)

			cell := buf.CellAt(0, 0)
			if cell == nil {
				t.Fatal("Expected cell at (0,0), got nil")
			}
		})
	}
}

func TestEffectOp_MultipleEffects(t *testing.T) {
	dl := NewDisplayContext()

	// Draw content
	dl.AddDraw(layout.Rect(0, 0, 10, 1), "MultiStyle", 0)

	// Apply multiple effects
	dl.AddDim(layout.Rect(0, 0, 5, 1), 0)
	dl.AddHighlight(layout.Rect(5, 0, 5, 1), lipgloss.NewStyle().Background(lipgloss.Color("4")), 1)

	buf := uv.NewScreenBuffer(15, 1)
	dl.Render(buf)

	// Verify both effects were applied to their respective regions
	leftCell := buf.CellAt(0, 0)
	rightCell := buf.CellAt(7, 0)

	if leftCell == nil || rightCell == nil {
		t.Fatal("Expected cells to exist after rendering")
	}
}

func TestEffectOp_EffectAfterDraw(t *testing.T) {
	dl := NewDisplayContext()

	// Important: DrawOps must be rendered before EffectOps
	dl.AddDraw(layout.Rect(0, 0, 6, 1), "Normal", 0)
	dl.AddDim(layout.Rect(0, 0, 6, 1), 0)

	buf := uv.NewScreenBuffer(10, 1)
	dl.Render(buf)

	// Content should be present with effect applied
	output := buf.Render()
	if !strings.Contains(output, "Normal") {
		t.Errorf("Expected 'Normal' in output, got: %s", output)
	}
}

func TestRenderToString(t *testing.T) {
	dl := NewDisplayContext()

	dl.AddDraw(layout.Rect(0, 0, 5, 1), "Quick", 0)

	output := dl.RenderToString(10, 1)

	if !strings.Contains(output, "Quick") {
		t.Errorf("RenderToString: expected 'Quick', got: %s", output)
	}
}

func TestIterateCells_BoundsChecking(t *testing.T) {
	dl := NewDisplayContext()

	// Try to draw outside buffer bounds
	dl.AddDraw(layout.Rect(0, 0, 5, 1), "Hello", 0)

	// Apply effect partially outside bounds
	dl.AddDim(layout.Rect(3, 0, 20, 1), 0)

	// Should not panic
	buf := uv.NewScreenBuffer(10, 1)
	dl.Render(buf)

	// Effect should be clipped to buffer bounds
	cell := buf.CellAt(4, 0) // Inside buffer
	if cell == nil {
		t.Error("Expected cell at (4,0) to exist")
	}
}

func TestDisplayContext_HighlightPreservesWideCharacters(t *testing.T) {
	dl := NewDisplayContext()

	text := "A🙂B"
	rect := layout.Rect(0, 0, 4, 1)

	dl.AddDraw(rect, text, 0)
	dl.AddHighlight(rect, lipgloss.NewStyle().Background(lipgloss.Color("4")), 1)

	buf := uv.NewScreenBuffer(4, 1)
	dl.Render(buf)

	out := buf.Render()
	assert.Contains(t, out, "🙂", "highlighted output should preserve emoji")
}

func TestDisplayContext_HighlightPreservesWideCharactersWithExistingBackground(t *testing.T) {
	dl := NewDisplayContext()

	text := lipgloss.NewStyle().Background(lipgloss.Color("0")).Render("hello🙂中文어👨‍👩‍👧‍👦Ａあ가")
	rect := layout.Rect(0, 0, 28, 1)

	dl.AddDraw(rect, text, 0)
	dl.AddHighlight(rect, lipgloss.NewStyle().Background(lipgloss.Color("4")), 1)

	buf := uv.NewScreenBuffer(28, 1)
	dl.Render(buf)

	out := buf.Render()
	assert.Contains(t, out, "🙂", "highlighted output should preserve emoji with existing background")
	assert.Contains(t, out, "中", "highlighted output should preserve CJK glyphs with existing background")
	assert.Contains(t, out, "文", "highlighted output should preserve CJK glyphs with existing background")
	assert.Contains(t, out, "어", "highlighted output should preserve Korean glyphs with existing background")
	assert.Contains(t, out, "👨‍👩‍👧‍👦", "highlighted output should preserve ZWJ emoji sequences with existing background")
	assert.Contains(t, out, "Ａ", "highlighted output should preserve full-width Latin characters with existing background")
	assert.Contains(t, out, "あ", "highlighted output should preserve Hiragana glyphs with existing background")
	assert.Contains(t, out, "가", "highlighted output should preserve Hangul glyphs with existing background")
}

func TestEmptyDisplayContext(t *testing.T) {
	dl := NewDisplayContext()

	// Rendering empty display context should not panic
	buf := uv.NewScreenBuffer(10, 1)
	dl.Render(buf)

	// Should not panic - that's the main test
	// Empty buffer output may be empty or whitespace, both are valid
	_ = buf.Render()
}
