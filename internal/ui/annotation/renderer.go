package annotation

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	contentColumn                       = 12
	diffLineBackgroundBlendRatio        = 0.9
	diffChangedWordBackgroundBlendRatio = 0.7
)

type displayLine struct {
	Prefix         string
	Gutter         string
	GutterX        int
	Content        string
	ContentX       int
	SourceIndex    int
	Editor         bool
	BackgroundRole string
}

func (l displayLine) paint(dl *render.DisplayContext, rect layout.Rectangle) {
	drawRowSegment(dl, rect, 0, l.Prefix)
	drawRowSegment(dl, rect, l.GutterX, l.Gutter)
	drawRowSegment(dl, rect, l.ContentX, l.Content)
}

type lineClickedMsg struct {
	SourceIndex int
}

type scrollMsg struct {
	Delta      int
	Horizontal bool
}

func (m scrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	m.Delta = delta
	m.Horizontal = horizontal
	return m
}

type annotationRenderer struct {
	highlighter              *sourceHighlighter
	highlighterDark          bool
	highlighterSourceVersion uint64
	highlighterSourceSet     bool
	displayCache             displayLineCache
}

type displayLineCache struct {
	valid       bool
	key         displayLineCacheKey
	lines       []displayLine
	editorStart int
}

type displayLineCacheKey struct {
	document          *reviewDocument
	file              *fileItem
	annotations       *annotationStore
	sourceVersion     uint64
	annotationVersion uint64
	presentation      presentation
	loading           bool
	width             int
	scrollX           int
	wrap              bool
}

func newAnnotationRenderer(dark bool) annotationRenderer {
	return annotationRenderer{
		highlighter:     newSourceHighlighter(dark),
		highlighterDark: dark,
	}
}

type annotationViewState struct {
	document        *reviewDocument
	annotations     *annotationStore
	cursor          int
	selectionAnchor int
	scrollY         int
	scrollX         int
	wrap            bool
	editing         bool
	editor          *textarea.Model
	sourceVersion   uint64
}

func (s annotationViewState) selectedRange() (int, int) {
	if s.selectionAnchor < 0 {
		return s.cursor, s.cursor
	}
	return min(s.selectionAnchor, s.cursor), max(s.selectionAnchor, s.cursor)
}

type annotationRenderResult struct {
	viewportWidth  int
	viewportHeight int
	scrollY        int
}

func (r *annotationRenderer) Render(
	dl *render.DisplayContext,
	box layout.Box,
	state annotationViewState,
	dark bool,
) annotationRenderResult {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return annotationRenderResult{}
	}
	if r.highlighter == nil || dark != r.highlighterDark {
		r.highlighter = newSourceHighlighter(dark)
		r.highlighterDark = dark
		r.highlighterSourceSet = false
		r.displayCache.valid = false
	}

	surface := common.DefaultPalette.Get("annotation", "", "", false)
	changeIDStyle := common.DefaultPalette.Get("annotation", "", "change_id", false)
	textStyle := common.DefaultPalette.Get("annotation", "", "text", false)
	dimmedStyle := common.DefaultPalette.Get("annotation", "", "dimmed", false)
	dl.AddFill(box.R, ' ', surface, 0)

	headerHeight := min(1, box.R.Dy())
	rows := box.V(layout.Fixed(headerHeight), layout.Fill(1))
	headerBox := rows[0]
	bodyBox := rows[1]
	result := annotationRenderResult{
		viewportWidth:  bodyBox.R.Dx(),
		viewportHeight: bodyBox.R.Dy(),
	}

	r.renderHeader(dl, headerBox, state, changeIDStyle, textStyle, dimmedStyle)

	lines, editorStart := r.buildDisplayLines(state, bodyBox.R.Dx())
	result.scrollY = clampScroll(state.scrollY, len(lines), bodyBox.R.Dy())

	selectedStyle := common.DefaultPalette.Get("annotation", "", "", true)
	selectedStart, selectedEnd := state.selectedRange()
	cursorY := -1
	cursorX := contentColumn
	source := state.document.source()
	for y := 0; y < bodyBox.R.Dy(); y++ {
		index := result.scrollY + y
		if index >= len(lines) {
			break
		}
		line := lines[index]
		rowRect := layout.Rect(bodyBox.R.Min.X, bodyBox.R.Min.Y+y, bodyBox.R.Dx(), 1)
		line.paint(dl, rowRect)
		if line.BackgroundRole != "" {
			background := diffBackgroundStyle(
				line.BackgroundRole,
				diffLineBackgroundBlendRatio,
			)
			backgroundX := min(max(line.ContentX, 0), bodyBox.R.Dx())
			dl.AddHighlight(
				layout.Rect(rowRect.Min.X+backgroundX, rowRect.Min.Y, rowRect.Dx()-backgroundX, 1),
				background,
				1,
			)
		}
		if line.SourceIndex >= selectedStart &&
			line.SourceIndex <= selectedEnd &&
			source.Commentable(line.SourceIndex) {
			dl.AddPaint(rowRect, selectedStyle, 2)
		}
		if !state.editing && line.SourceIndex >= 0 && source.Commentable(line.SourceIndex) {
			dl.AddInteraction(
				rowRect,
				lineClickedMsg{SourceIndex: line.SourceIndex},
				render.InteractionClick,
				0,
			)
		}
		if line.SourceIndex == state.cursor && cursorY < 0 {
			cursorY = y
			cursorX = line.ContentX
		}
	}
	dl.AddInteraction(bodyBox.R, scrollMsg{}, render.InteractionScroll, 0)

	if state.editing &&
		editorStart >= result.scrollY &&
		editorStart < result.scrollY+bodyBox.R.Dy() {
		dl.SetCursorInRect(state.editor.Cursor(), bodyBox.R, contentColumn, editorStart-result.scrollY)
	} else if cursorY >= 0 {
		dl.SetCursorAt(tea.NewCursor(cursorX, 0), bodyBox.R.Min.X, bodyBox.R.Min.Y+cursorY)
	}
	return result
}

func (r *annotationRenderer) renderHeader(
	dl *render.DisplayContext,
	box layout.Box,
	state annotationViewState,
	changeIDStyle, textStyle, dimmedStyle lipgloss.Style,
) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}
	fileNumber := 0
	path := "(no files)"
	if file := state.document.currentFile(); file != nil {
		fileNumber = state.document.file + 1
		path = file.Path
	}
	if state.document.presentation == filePresentation {
		path += " [full]"
	}
	description := "loading description…"
	if state.document.description != "" {
		description = state.document.description
	}

	changeID := state.document.changeID()
	dl.Text(box.R.Min.X, box.R.Min.Y, 0).
		Styled(path, dimmedStyle).
		Styled(" · ", dimmedStyle).
		Styled(fmt.Sprintf("file %d/%d", fileNumber, len(state.document.files)), dimmedStyle).
		Styled(" · ", dimmedStyle).
		Styled(fmt.Sprintf("%d annotations", state.annotations.countRevision(changeID)), dimmedStyle).
		Styled(" │ ", dimmedStyle).
		Styled(shortRevision(changeID), changeIDStyle).
		Styled(" · ", dimmedStyle).
		Styled(description, textStyle).
		Done()
}

func (r *annotationRenderer) buildDisplayLines(
	state annotationViewState,
	width int,
) ([]displayLine, int) {
	r.ensureSource(state.sourceVersion)
	if state.editing {
		// Textarea contents and wrapping can change on every key press. Keeping the
		// short-lived editor path uncached avoids mirroring its internal state here.
		return r.buildDisplayLinesUncached(state, width)
	}
	key := displayLineCacheKey{
		document:          state.document,
		file:              state.document.currentFile(),
		annotations:       state.annotations,
		sourceVersion:     state.sourceVersion,
		annotationVersion: state.annotations.version,
		presentation:      state.document.presentation,
		loading:           state.document.loading,
		width:             width,
		scrollX:           state.scrollX,
		wrap:              state.wrap,
	}
	if r.displayCache.valid && r.displayCache.key == key {
		return r.displayCache.lines, r.displayCache.editorStart
	}
	lines, editorStart := r.buildDisplayLinesUncached(state, width)
	r.displayCache = displayLineCache{
		valid:       true,
		key:         key,
		lines:       lines,
		editorStart: editorStart,
	}
	return lines, editorStart
}

func (r *annotationRenderer) ensureSource(version uint64) {
	if r.highlighterSourceSet && r.highlighterSourceVersion == version {
		return
	}
	r.highlighter.resetSource()
	r.highlighterSourceVersion = version
	r.highlighterSourceSet = true
	r.displayCache.valid = false
}

func (r *annotationRenderer) buildDisplayLinesUncached(
	state annotationViewState,
	width int,
) ([]displayLine, int) {
	if state.document.loading {
		return []displayLine{{Content: "Loading annotation view...", SourceIndex: -1}}, -1
	}
	if state.document.err != nil {
		style := common.DefaultPalette.Get("annotation", "", "error", false)
		return []displayLine{{Content: style.Render(state.document.err.Error()), SourceIndex: -1}}, -1
	}
	file := state.document.currentFile()
	if file == nil {
		return []displayLine{{Content: "(no files)", SourceIndex: -1}}, -1
	}
	if state.document.presentation == filePresentation {
		if file.ContentErr != nil {
			style := common.DefaultPalette.Get("annotation", "", "error", false)
			return []displayLine{{Content: style.Render(file.ContentErr.Error()), SourceIndex: -1}}, -1
		}
		if !file.ContentLoaded {
			return []displayLine{{Content: "Loading complete file...", SourceIndex: -1}}, -1
		}
		return r.buildFileLines(state, *file, width)
	}
	return r.buildDiffLines(state, *file, width)
}

func (r *annotationRenderer) buildDiffLines(
	state annotationViewState,
	file fileItem,
	width int,
) ([]displayLine, int) {
	if file.Patch == nil || len(file.Patch.Lines) == 0 {
		text := common.DefaultPalette.Get("annotation", "", "dimmed", false).
			Render("(unchanged file; press v to view the complete file)")
		return []displayLine{{Content: text, SourceIndex: -1}}, -1
	}

	var lines []displayLine
	editorStart := -1
	_, selectedEnd := state.selectedRange()
	annotations := state.annotations.ForFile(state.document.changeID(), file.Path)
	for index, line := range file.Patch.Lines {
		backgroundRole := ""
		switch line.Kind {
		case lineAdded:
			backgroundRole = "added"
		case lineRemoved:
			backgroundRole = "deleted"
		}
		for _, rendered := range r.renderPatchLine(state, file.Path, line, width) {
			rendered.SourceIndex = index
			rendered.BackgroundRole = backgroundRole
			lines = append(lines, rendered)
		}
		for _, annotation := range annotationsAfterPatchLine(annotations, line) {
			lines = append(lines, r.renderAnnotation(annotation, width)...)
		}
		if state.editing && index == selectedEnd {
			editorStart = len(lines)
			lines = append(lines, r.renderEditor(state.editor, width)...)
		}
	}
	return lines, editorStart
}

func (r *annotationRenderer) buildFileLines(
	state annotationViewState,
	file fileItem,
	width int,
) ([]displayLine, int) {
	var lines []displayLine
	editorStart := -1
	_, selectedEnd := state.selectedRange()
	annotations := state.annotations.ForFile(state.document.changeID(), file.Path)
	numberWidth := max(4, len(strconv.Itoa(len(file.Content))))
	prefixWidth := numberWidth + 3
	source := state.document.source()
	for index, content := range file.Content {
		highlighted := render.ExpandTabs(r.highlighter.render(file.Path, content))
		for row, content := range r.contentSegments(state, highlighted, width, prefixWidth) {
			leading := strings.Repeat(" ", numberWidth+1)
			if row == 0 {
				leading = fmt.Sprintf("%*d ", numberWidth, index+1)
			}
			lines = append(lines, displayLine{
				Prefix:      leading,
				Gutter:      "│",
				GutterX:     prefixWidth - 2,
				Content:     content,
				ContentX:    prefixWidth,
				SourceIndex: index,
			})
		}
		for _, annotation := range annotations {
			if source.annotationLines(annotation).End == index+1 {
				lines = append(lines, r.renderAnnotation(annotation, width)...)
			}
		}
		if state.editing && index == selectedEnd {
			editorStart = len(lines)
			lines = append(lines, r.renderEditor(state.editor, width)...)
		}
	}
	return lines, editorStart
}

func (r *annotationRenderer) renderPatchLine(
	state annotationViewState,
	path string,
	line patchLine,
	width int,
) []displayLine {
	lineNumber := common.DefaultPalette.Get("annotation", "", "dimmed", false)
	hunkStyle := common.DefaultPalette.Get("annotation", "", "title", false)
	metadataStyle := common.DefaultPalette.Get("annotation", "", "dimmed", false)
	addedStyle := common.DefaultPalette.Get("", "", "added", false)
	deletedStyle := common.DefaultPalette.Get("", "", "deleted", false)
	addedWord := diffBackgroundStyle(
		"added",
		diffChangedWordBackgroundBlendRatio,
	)
	deletedWord := diffBackgroundStyle(
		"deleted",
		diffChangedWordBackgroundBlendRatio,
	)

	oldNumber := ""
	newNumber := ""
	if line.OldLine > 0 {
		oldNumber = strconv.Itoa(line.OldLine)
	}
	if line.NewLine > 0 {
		newNumber = strconv.Itoa(line.NewLine)
	}
	gutterStyle := lineNumber
	switch line.Kind {
	case lineAdded:
		gutterStyle = lineNumber.Foreground(addedStyle.GetForeground())
	case lineRemoved:
		gutterStyle = lineNumber.Foreground(deletedStyle.GetForeground())
	}

	content := line.Content
	switch line.Kind {
	case lineHunk:
		content = hunkStyle.Render(content)
	case lineMetadata:
		content = metadataStyle.Render(content)
	case lineAdded:
		if line.PairContent != "" {
			content = r.highlighter.renderChanged(path, content, line.PairContent, addedWord)
		} else {
			content = r.highlighter.render(path, content)
		}
	case lineRemoved:
		if line.PairContent != "" {
			content = r.highlighter.renderChanged(path, content, line.PairContent, deletedWord)
		} else {
			content = r.highlighter.render(path, content)
		}
	default:
		content = r.highlighter.render(path, content)
	}
	content = render.ExpandTabs(content)

	segments := r.contentSegments(state, content, width, contentColumn)
	lines := make([]displayLine, 0, len(segments))
	for row, segment := range segments {
		numberPrefix := strings.Repeat(" ", contentColumn-2)
		if row == 0 {
			numberPrefix = fmt.Sprintf("%4s %4s ", oldNumber, newNumber)
		}
		lines = append(lines, displayLine{
			Prefix:      lineNumber.Render(numberPrefix),
			Gutter:      gutterStyle.Render("│"),
			GutterX:     contentColumn - 2,
			Content:     segment,
			ContentX:    contentColumn,
			SourceIndex: -1,
		})
	}
	return lines
}

func diffBackgroundStyle(role string, blendRatio float64) lipgloss.Style {
	foreground := common.DefaultPalette.Get("", "", role, false).GetForeground()
	background := lipgloss.NewStyle().Background(foreground)
	return common.DefaultPalette.BlendBackgroundCustom(
		background,
		"annotation",
		"",
		blendRatio,
	)
}

func (r *annotationRenderer) contentSegments(
	state annotationViewState,
	content string,
	width, prefixWidth int,
) []string {
	available := max(width-prefixWidth, 1)
	contentWidth := ansi.StringWidth(content)
	if !state.wrap {
		return []string{ansi.Cut(content, state.scrollX, state.scrollX+available)}
	}
	if contentWidth <= available {
		return []string{content}
	}
	var result []string
	for start := 0; start < contentWidth; start += available {
		result = append(result, ansi.Cut(content, start, min(start+available, contentWidth)))
	}
	return result
}

func drawRowSegment(
	dl *render.DisplayContext,
	rect layout.Rectangle,
	offset int,
	content string,
) {
	if content == "" || rect.Dx() <= 0 || offset < 0 || offset >= rect.Dx() {
		return
	}
	dl.AddDraw(
		layout.Rect(rect.Min.X+offset, rect.Min.Y, rect.Dx()-offset, 1),
		content,
		0,
		render.PreserveBackground(),
	)
}

func (r *annotationRenderer) renderAnnotation(
	annotation Annotation,
	width int,
) []displayLine {
	style := common.DefaultPalette.Get("annotation", "", "text", false)
	commentLines := strings.Split(annotation.Comment, "\n")
	result := make([]displayLine, 0, len(commentLines))
	for index, line := range commentLines {
		label := "comment: "
		if index > 0 {
			label = "         "
		}
		text := style.Render(label + line)
		available := max(width-contentColumn, 1)
		textWidth := ansi.StringWidth(text)
		if textWidth == 0 {
			result = append(result, displayLine{ContentX: contentColumn, SourceIndex: -1})
			continue
		}
		for start := 0; start < textWidth; start += available {
			result = append(result, displayLine{
				Content:     ansi.Cut(text, start, min(start+available, textWidth)),
				ContentX:    contentColumn,
				SourceIndex: -1,
			})
		}
	}
	return result
}

func (r *annotationRenderer) renderEditor(
	editor *textarea.Model,
	_ int,
) []displayLine {
	view := strings.Split(strings.TrimRight(editor.View(), "\n"), "\n")
	result := make([]displayLine, 0, len(view))
	for _, line := range view {
		result = append(result, displayLine{
			Content:     line,
			ContentX:    contentColumn,
			SourceIndex: -1,
			Editor:      true,
		})
	}
	return result
}

func annotationsAfterPatchLine(annotations []Annotation, line patchLine) []Annotation {
	var result []Annotation
	for _, annotation := range annotations {
		switch {
		case annotation.NewLines.End > 0 && annotation.NewLines.End == line.NewLine:
			result = append(result, annotation)
		case annotation.NewLines.End == 0 && annotation.OldLines.End == line.OldLine:
			result = append(result, annotation)
		}
	}
	return result
}

func clampScroll(scrollY, total, viewportHeight int) int {
	return max(0, min(scrollY, max(0, total-viewportHeight)))
}
