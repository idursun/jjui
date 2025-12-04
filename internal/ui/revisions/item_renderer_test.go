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
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/stretchr/testify/assert"
)

type cursorBuffer struct {
	*bytes.Buffer
}

func (c cursorBuffer) LocalPos() (line, col int) {
	return 0, 0
}

func (c cursorBuffer) ViewportPos() (line, col int) {
	return 0, 0
}

type trackingWriter struct {
	*bytes.Buffer
	localLine      int
	localCol       int
	viewportStart  int
	viewportHeight int
}

func newTrackingWriter(viewportStart, viewportHeight int) *trackingWriter {
	return &trackingWriter{
		Buffer:         &bytes.Buffer{},
		viewportStart:  viewportStart,
		viewportHeight: viewportHeight,
	}
}

func (t *trackingWriter) LocalPos() (line, col int) {
	return t.localLine, t.localCol
}

func (t *trackingWriter) ViewportPos() (line, col int) {
	screenLine := t.localLine - t.viewportStart
	if screenLine < 0 || screenLine >= t.viewportHeight {
		return -1, -1
	}
	return screenLine, t.localCol
}

func (t *trackingWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if err := t.Buffer.WriteByte(b); err != nil {
			return n, err
		}
		n++
		if b == '\n' {
			t.localLine++
			t.localCol = 0
		} else {
			t.localCol++
		}
	}
	return n, nil
}

type mockOperation struct {
	*common.ViewNode
	*common.MouseAware
	renderBefore          string
	renderAfter           string
	renderOverDescription string
}

func (m *mockOperation) GetViewNode() *common.ViewNode {
	return m.ViewNode
}

func (m *mockOperation) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	switch renderPosition {
	case operations.RenderPositionBefore:
		return m.renderBefore
	case operations.RenderPositionAfter:
		return m.renderAfter
	case operations.RenderOverDescription:
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
		op: &mockOperation{MouseAware: common.NewMouseAware(), renderOverDescription: descriptionOverlay},
	}

	var buf bytes.Buffer
	cb := cursorBuffer{Buffer: &buf}
	renderer.Render(cb, 80)
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
	cb := cursorBuffer{Buffer: &buf}
	renderer.Render(cb, 80)
	output := buf.String()

	// Lines after elided should not appear
	assert.NotContains(t, output, "Should not appear", "Lines after elided marker should not be rendered")
}
