package annotation

import (
	"testing"

	"charm.land/bubbles/v2/textarea"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderEditorIsStateIdempotent(t *testing.T) {
	editor := textarea.New()
	editor.ShowLineNumbers = false
	editor.SetVirtualCursor(false)
	editor.SetHeight(4)
	editor.SetWidth(24)
	editor.SetValue("a comment that wraps")
	editor.Focus()
	editor.SetCursorColumn(3)

	renderer := newAnnotationRenderer(false)
	before := editorState(editor)
	first := renderer.renderEditor(&editor, 40)
	second := renderer.renderEditor(&editor, 12)

	assert.Equal(t, first, second)
	assert.Equal(t, before, editorState(editor))
}

func TestMovingCursorReusesHighlightedDisplayLines(t *testing.T) {
	model := mouseTestModel(320)
	model.renderer = newAnnotationRenderer(false)
	display := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 60, 10))
	model.ViewRect(display, box)
	require.Equal(t, 320, model.renderer.highlighter.highlightMisses)

	model.moveCursor(1, false)
	model.ViewRect(render.NewDisplayContext(), box)

	assert.Equal(t, 320, model.renderer.highlighter.highlightMisses)
}

func TestLargeDiffDoesNotThrashHighlightCacheOnLayoutChange(t *testing.T) {
	model := mouseTestModel(320)
	model.renderer = newAnnotationRenderer(false)
	box := layout.NewBox(layout.Rect(0, 0, 40, 10))
	model.ViewRect(render.NewDisplayContext(), box)
	require.Equal(t, 320, model.renderer.highlighter.highlightMisses)
	require.Len(t, model.renderer.highlighter.cache, 320)

	model.wrap = true
	model.ViewRect(render.NewDisplayContext(), box)

	assert.Equal(t, 320, model.renderer.highlighter.highlightMisses)
	assert.Len(t, model.renderer.highlighter.cache, 320)
}

func TestThemeChangeInvalidatesRenderedData(t *testing.T) {
	model := mouseTestModel(1)
	model.document.files[0].Patch.Lines[0].Content = "package main"
	renderer := newAnnotationRenderer(false)
	box := layout.NewBox(layout.Rect(0, 0, 40, 3))

	lightDisplay := render.NewDisplayContext()
	renderer.Render(lightDisplay, box, model.viewState(), false)
	lightHighlighter := renderer.highlighter
	lightRendered := lightDisplay.RenderToString(40, 3)
	require.Equal(t, 1, lightHighlighter.highlightMisses)

	darkDisplay := render.NewDisplayContext()
	renderer.Render(darkDisplay, box, model.viewState(), true)
	darkRendered := darkDisplay.RenderToString(40, 3)

	assert.NotSame(t, lightHighlighter, renderer.highlighter)
	assert.Equal(t, 1, renderer.highlighter.highlightMisses)
	assert.NotEqual(t, lightRendered, darkRendered)
}

func BenchmarkCachedLargeDiffCursorRender(b *testing.B) {
	model := mouseTestModel(1000)
	model.renderer = newAnnotationRenderer(false)
	box := layout.NewBox(layout.Rect(0, 0, 80, 24))
	model.ViewRect(render.NewDisplayContext(), box)
	missesBefore := model.renderer.highlighter.highlightMisses

	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		model.cursor = index % 1000
		model.ensureCursorVisible()
		model.ViewRect(render.NewDisplayContext(), box)
	}
	b.ReportMetric(
		float64(model.renderer.highlighter.highlightMisses-missesBefore)/float64(b.N),
		"highlight-misses/op",
	)
}

type textareaState struct {
	value  string
	width  int
	height int
	line   int
	column int
	cursor any
}

func editorState(editor textarea.Model) textareaState {
	return textareaState{
		value:  editor.Value(),
		width:  editor.Width(),
		height: editor.Height(),
		line:   editor.Line(),
		column: editor.Column(),
		cursor: editor.Cursor(),
	}
}
