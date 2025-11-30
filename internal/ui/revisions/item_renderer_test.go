package revisions

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/stretchr/testify/assert"
)

type mockOperation struct {
	renderOverDescription string
}

func (m *mockOperation) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	if renderPosition == operations.RenderOverDescription {
		return m.renderOverDescription
	}
	return ""
}

func (m *mockOperation) Name() string {
	return "mock"
}

func (m *mockOperation) Init() tea.Cmd {
	return nil
}

func (m *mockOperation) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (m *mockOperation) View() string {
	return ""
}

// Helper function to create a basic GraphRowLine
func createGraphRowLine(text string, flags parser.RowLineFlags) *parser.GraphRowLine {
	segment := &screen.Segment{
		Text:  text,
		Style: lipgloss.NewStyle(),
	}
	line := &parser.GraphRowLine{
		Segments: []*screen.Segment{segment},
		Gutter: parser.GraphGutter{
			Segments: []*screen.Segment{
				{Text: "│ ", Style: lipgloss.NewStyle()},
			},
		},
		Flags: flags,
	}
	return line
}

// TestRenderMainLines_MultipleDescriptionLines tests a row with multiple description lines
func TestRenderMainLines_MultipleDescriptionLines(t *testing.T) {
	descriptionOverlay := "Overlay description"

	row := parser.Row{
		Commit: &jj.Commit{
			ChangeId: "test123",
			CommitId: "abc456",
		},
		Lines: []*parser.GraphRowLine{
			createGraphRowLine("test123 abc456", parser.Revision|parser.Highlightable),
			createGraphRowLine("Description line 1", parser.Highlightable),
			createGraphRowLine("Description line 2", parser.Highlightable),
			createGraphRowLine("Description line 3", parser.Highlightable),
		},
	}

	renderer := itemRenderer{
		row:           row,
		isHighlighted: true,
		selectedStyle: lipgloss.NewStyle().Background(lipgloss.Color("blue")),
		textStyle:     lipgloss.NewStyle(),
		dimmedStyle:   lipgloss.NewStyle(),
		isGutterInLane: func(lineIndex, segmentIndex int) bool {
			return true
		},
		updateGutterText: func(lineIndex, segmentIndex int, text string) string {
			return text
		},
		op: &mockOperation{renderOverDescription: descriptionOverlay},
	}

	var buf bytes.Buffer
	renderer.Render(&buf, 80)
	output := buf.String()

	// The overlay should appear exactly once
	overlayCount := strings.Count(output, descriptionOverlay)
	assert.Equal(t, 1, overlayCount, "Description overlay should appear exactly once")

	// None of the original description lines should appear
	assert.NotContains(t, output, "Description line 1")
	assert.NotContains(t, output, "Description line 2")
	assert.NotContains(t, output, "Description line 3")

	// The revision line should still be rendered
	assert.Contains(t, output, "test123 abc456")
}

// TestRenderMainLines_WithElidedLine tests that elided lines stop processing
func TestRenderMainLines_WithElidedLine(t *testing.T) {
	row := parser.Row{
		Commit: &jj.Commit{
			ChangeId: "test123",
			CommitId: "abc456",
		},
		Lines: []*parser.GraphRowLine{
			createGraphRowLine("test123 abc456", parser.Revision|parser.Highlightable),
			createGraphRowLine("...", parser.Elided),
			createGraphRowLine("Should not appear", parser.Highlightable), // After elided
		},
	}

	renderer := itemRenderer{
		row:           row,
		isHighlighted: true,
		selectedStyle: lipgloss.NewStyle().Background(lipgloss.Color("blue")),
		textStyle:     lipgloss.NewStyle(),
		dimmedStyle:   lipgloss.NewStyle(),
		isGutterInLane: func(lineIndex, segmentIndex int) bool {
			return true
		},
		updateGutterText: func(lineIndex, segmentIndex int, text string) string {
			return text
		},
		op: &mockOperation{},
	}

	var buf bytes.Buffer
	renderer.Render(&buf, 80)
	output := buf.String()

	// Lines after elided should not appear
	assert.NotContains(t, output, "Should not appear", "Lines after elided marker should not be rendered")
}
