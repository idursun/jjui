package annotation

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/jj/source"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/render"
)

type revisionDirection int

const (
	revisionParent revisionDirection = iota
	revisionChild
)

type targetPickerPayload struct{}

type commentPickerPayload struct{}

type revisionPickerPayload struct{}

type pickerSource struct {
	items []source.Item
}

func (s pickerSource) Fetch(_ source.Runner) ([]source.Item, error) {
	return slices.Clone(s.items), nil
}

type Model struct {
	context     *appContext.MainContext
	document    reviewDocument
	annotations annotationStore
	loader      annotationLoader

	nextRequestID            uint64
	revisionLoadRequestID    uint64
	revisionTargetsRequestID uint64

	cursor          int
	selectionAnchor int
	scrollY         int
	scrollX         int
	wrap            bool

	editing bool
	editor  textarea.Model

	// editorLayoutWidth is the width last applied during the layout transition.
	// Keeping it separately avoids calling SetWidth for every render at the same
	// size.
	editorLayoutWidth int

	focusedAnnotationID int

	viewportWidth  int
	viewportHeight int
	// sourceVersion scopes renderer and syntax-highlight caches to the current
	// revision, file, presentation, and loaded content.
	sourceVersion uint64

	renderer annotationRenderer
}

var _ common.ImmediateModel = (*Model)(nil)

func New(ctx *appContext.MainContext, revision string) *Model {
	dark := ctx != nil && ctx.TerminalHasDarkBackground
	return &Model{
		context: ctx,
		document: reviewDocument{
			revision: revision,
			loading:  revision != "",
		},
		selectionAnchor: -1,
		loader:          annotationLoader{context: ctx},
		renderer:        newAnnotationRenderer(dark),
	}
}

func (m *Model) Init() tea.Cmd {
	return m.loadRevision(m.changeID())
}

func (m *Model) Scopes() []common.Scope {
	if m.editing {
		return []common.Scope{{
			Name:    actions.ScopeAnnotationEditor,
			Leak:    common.LeakNone,
			Handler: m,
		}}
	}
	return []common.Scope{{
		Name:    actions.ScopeAnnotation,
		Leak:    common.LeakGlobal,
		Handler: m,
	}}
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Cancel:
		if m.editing {
			m.cancelEditor()
			return nil, true
		}
		return common.Close, true
	case intents.AnnotationEditorCancel:
		m.cancelEditor()
		return nil, true
	case intents.AnnotationEditorSave:
		return m.saveEditor(), true
	case intents.AnnotationMove:
		delta := intent.Delta
		switch {
		case intent.Page:
			delta *= max(m.viewportHeight, 1)
		case intent.HalfPage:
			delta *= max(m.viewportHeight/2, 1)
		}
		m.moveCursor(delta, intent.Select)
		return nil, true
	case intents.AnnotationMoveBoundary:
		m.moveBoundary(intent.Last)
		return nil, true
	case intents.AnnotationFileNavigate:
		return m.navigateFile(intent.Delta), true
	case intents.AnnotationOpenTargetPicker:
		return m.openTargetPicker(), true
	case intents.AnnotationOpenCommentPicker:
		return m.openCommentPicker(), true
	case intents.AnnotationNavigateParent:
		return m.navigateRevision(revisionParent), true
	case intents.AnnotationNavigateChild:
		return m.navigateRevision(revisionChild), true
	case intents.AnnotationScrollHorizontal:
		if !m.wrap {
			m.scrollX = max(0, m.scrollX+intent.Delta)
		}
		return nil, true
	case intents.AnnotationTogglePresentation:
		if m.document.presentation == diffPresentation {
			m.document.presentation = filePresentation
			m.clearFocusedAnnotation()
			m.resetPosition()
			return m.loadCurrentFile(), true
		}
		m.document.presentation = diffPresentation
		m.clearFocusedAnnotation()
		m.resetPosition()
		return nil, true
	case intents.AnnotationToggleWrap:
		m.wrap = !m.wrap
		m.scrollX = 0
		m.ensureCursorVisible()
		return nil, true
	case intents.AnnotationAdd:
		annotations := m.annotationAtCursor()
		switch len(annotations) {
		case 1:
			m.focusedAnnotationID = annotations[0].ID
			m.startEditorFor(annotations[0])
		case 0:
			if m.currentCommentable() {
				m.startEditor()
			}
		default:
			return m.openCommentPickerFor(annotations), true
		}
		return nil, true
	case intents.AnnotationDelete:
		annotations := m.annotationAtCursor()
		switch len(annotations) {
		case 1:
			m.removeAnnotation(annotations[0].ID)
			m.focusedAnnotationID = 0
			m.ensureCursorVisible()
		case 0:
			return nil, true
		default:
			return m.openCommentPickerFor(annotations), true
		}
		return nil, true
	case intents.AnnotationClear:
		count := m.annotations.Clear()
		m.focusedAnnotationID = 0
		return intents.Invoke(intents.AddMessage{Text: fmt.Sprintf("Cleared %d annotations", count)}), true
	case intents.AnnotationCopy:
		annotations := m.annotations.All()
		if len(annotations) == 0 {
			return intents.Invoke(intents.AddMessage{Text: "No annotations to copy"}), true
		}
		label := "annotations"
		if len(annotations) == 1 {
			label = "annotation"
		}
		return tea.Batch(
			tea.SetClipboard(formatAnnotationsMarkdown(annotations)),
			intents.Invoke(intents.AddMessage{Text: fmt.Sprintf("Copied %d %s", len(annotations), label)}),
		), true
	}
	return nil, false
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		cmd, _ := m.HandleIntent(msg)
		return cmd
	case revisionLoadedMsg:
		if msg.RequestID == 0 || msg.RequestID != m.revisionLoadRequestID {
			return nil
		}
		initialLoad := m.document.loading && m.document.loadingRevision == ""
		pendingLoad := m.document.loadingRevision != ""
		expectedChangeID := m.changeID()
		if pendingLoad {
			expectedChangeID = m.document.loadingRevision
		}
		if msg.ChangeID != expectedChangeID || (!initialLoad && !pendingLoad) {
			return nil
		}
		m.revisionLoadRequestID = 0
		if pendingLoad {
			m.document.loadingRevision = ""
		}
		m.document.loading = false
		if pendingLoad && m.editing {
			return revisionNavigationBlockedMessage()
		}
		if msg.Err != nil && pendingLoad {
			m.clearFocusedAnnotation()
			return intents.Invoke(intents.AddMessage{Text: msg.Err.Error()})
		}
		if pendingLoad {
			m.document.revision = msg.ChangeID
		}
		m.document.description = msg.Description
		m.document.err = msg.Err
		if msg.Err != nil {
			m.document.files = nil
			m.sourceVersion++
			return nil
		}
		m.document.files = msg.Files
		m.document.file = 0
		m.resetPosition()
		if m.focusedAnnotationID != 0 {
			return m.jumpToFocusedAnnotation()
		}
		if m.document.presentation == filePresentation {
			return m.loadCurrentFile()
		}
		return nil
	case revisionTargetsLoadedMsg:
		if msg.RequestID == 0 || msg.RequestID != m.revisionTargetsRequestID || msg.ChangeID != m.changeID() {
			return nil
		}
		m.revisionTargetsRequestID = 0
		if msg.Err != nil {
			return intents.Invoke(intents.AddMessage{Text: msg.Err.Error()})
		}
		targets := slices.DeleteFunc(msg.Targets, func(target revisionTarget) bool {
			return target.ChangeID == jj.RootChangeId
		})
		switch len(targets) {
		case 0:
			return nil
		case 1:
			m.clearFocusedAnnotation()
			return m.selectRevision(targets[0].ChangeID)
		default:
			return m.openRevisionPicker(targets)
		}
	case fileLoadedMsg:
		if msg.ChangeID != m.changeID() {
			return nil
		}
		index := slices.IndexFunc(m.document.files, func(file fileItem) bool { return file.Path == msg.Path })
		if index < 0 {
			return nil
		}
		m.document.files[index].ContentErr = msg.Err
		m.document.files[index].ContentLoaded = msg.Err == nil
		if msg.Err == nil {
			m.document.files[index].Content = splitFileLines(msg.Content)
		}
		m.sourceVersion++
		if m.focusedAnnotationID != 0 && m.document.presentation == filePresentation {
			annotation, ok := m.annotationByID(m.focusedAnnotationID)
			if ok && annotation.File == msg.Path {
				return m.jumpToFocusedAnnotation()
			}
		}
		return nil
	case target_picker.TargetSelectedMsg:
		switch payload := msg.Payload.(type) {
		case targetPickerPayload:
			return m.selectFile(msg.Target)
		case commentPickerPayload:
			return m.selectCommentPickerTarget(payload, msg.Target)
		case revisionPickerPayload:
			return m.selectRevisionPickerTarget(payload, msg.Target)
		}
		return nil
	case lineClickedMsg:
		if !m.editing && m.commentable(msg.SourceIndex) {
			m.cursor = msg.SourceIndex
			m.selectionAnchor = -1
			if _, ok := m.focusedAnnotationAtCursor(); !ok {
				m.clearFocusedAnnotation()
			}
		}
		return nil
	case scrollMsg:
		if msg.Horizontal {
			if !m.wrap {
				m.scrollX = max(0, m.scrollX+msg.Delta)
			}
			return nil
		}
		m.scrollY += msg.Delta
		lines, _ := m.buildDisplayLines(m.viewportWidth)
		m.clampScroll(len(lines))
		return nil
	case tea.KeyMsg, tea.PasteMsg:
		if !m.editing {
			return nil
		}
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return cmd
	}
	return nil
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}
	headerHeight := min(1, box.R.Dy())
	viewportChanged := m.viewportWidth != box.R.Dx() ||
		m.viewportHeight != max(box.R.Dy()-headerHeight, 0)
	m.viewportWidth = box.R.Dx()
	m.viewportHeight = max(box.R.Dy()-headerHeight, 0)
	dark := m.context != nil && m.context.TerminalHasDarkBackground
	if m.renderer.highlighter == nil || dark != m.renderer.highlighterDark {
		m.renderer = newAnnotationRenderer(dark)
	}
	if m.editing {
		m.resizeEditor(m.viewportWidth)
		if viewportChanged {
			m.ensureCursorVisible()
		}
	}
	result := m.renderer.Render(dl, box, m.viewState(), dark)
	m.scrollY = result.scrollY
}

func (m *Model) buildDisplayLines(width int) ([]displayLine, int) {
	return m.renderer.buildDisplayLines(m.viewState(), width)
}

func (m *Model) viewState() annotationViewState {
	return annotationViewState{
		document:        &m.document,
		annotations:     &m.annotations,
		cursor:          m.cursor,
		selectionAnchor: m.selectionAnchor,
		scrollY:         m.scrollY,
		scrollX:         m.scrollX,
		wrap:            m.wrap,
		editing:         m.editing,
		editor:          &m.editor,
		sourceVersion:   m.sourceVersion,
	}
}

func (m *Model) addAnnotation(annotation Annotation) Annotation {
	return m.annotations.Add(annotation)
}

func (m *Model) annotationByID(id int) (Annotation, bool) {
	return m.annotations.Find(id)
}

func (m *Model) updateAnnotationComment(id int, comment string) bool {
	return m.annotations.UpdateComment(id, comment)
}

func (m *Model) removeAnnotation(id int) bool {
	return m.annotations.Remove(id)
}

func (m *Model) startEditor() {
	m.clearFocusedAnnotation()
	m.startEditorFor(Annotation{})
}

func (m *Model) startEditorFor(annotation Annotation) {
	editor := textarea.New()
	editor.Placeholder = "Add review comment"
	editor.ShowLineNumbers = false
	editor.EndOfBufferCharacter = ' '
	editor.SetVirtualCursor(false)
	editor.SetHeight(4)
	if annotation.ID != 0 {
		editor.SetValue(annotation.Comment)
	}
	editor.Focus()
	m.editor = editor
	m.editing = true
	m.editorLayoutWidth = 0
	m.resizeEditor(m.viewportWidth)
	m.ensureCursorVisible()
}

func (m *Model) resizeEditor(viewportWidth int) {
	width := max(viewportWidth-contentColumn-1, 1)
	if m.editorLayoutWidth == width {
		return
	}
	m.editor.SetWidth(width)
	m.editorLayoutWidth = width
}

func (m *Model) cancelEditor() {
	m.editing = false
	m.selectionAnchor = -1
}

func (m *Model) saveEditor() tea.Cmd {
	if !m.editing {
		return nil
	}
	comment := strings.TrimSpace(m.editor.Value())
	if m.focusedAnnotationID != 0 {
		if comment != "" {
			m.updateAnnotationComment(m.focusedAnnotationID, comment)
		}
	} else if comment != "" {
		m.addAnnotation(m.newAnnotation(comment))
	}
	m.editing = false
	m.selectionAnchor = -1
	m.ensureCursorVisible()
	return nil
}

func (m *Model) newAnnotation(comment string) Annotation {
	start, end := m.selectedRange()
	oldLines, newLines, snippet := m.document.source().AnnotationLocation(start, end)
	annotation := Annotation{
		ChangeID: m.changeID(),
		File:     m.currentFile().Path,
		OldLines: oldLines,
		NewLines: newLines,
		Snippet:  snippet,
		Comment:  comment,
	}
	return annotation
}

func (m *Model) moveCursor(delta int, selecting bool) {
	if delta == 0 || m.sourceLength() == 0 {
		return
	}
	direction := 1
	if delta < 0 {
		direction = -1
		delta = -delta
	}
	origin := m.cursor
	for range delta {
		next := m.nextCommentable(m.cursor, direction)
		if next == m.cursor {
			break
		}
		m.cursor = next
	}
	if selecting && m.cursor != origin {
		if m.selectionAnchor < 0 {
			m.selectionAnchor = origin
		}
	} else if !selecting {
		m.selectionAnchor = -1
	}
	if m.cursor != origin {
		if _, ok := m.focusedAnnotationAtCursor(); !ok {
			m.clearFocusedAnnotation()
		}
	}
	m.ensureCursorVisible()
}

func (m *Model) moveBoundary(last bool) {
	length := m.sourceLength()
	if length == 0 {
		return
	}
	start := -1
	step := 1
	if last {
		start = length
		step = -1
	}
	m.cursor = m.nextCommentable(start, step)
	m.selectionAnchor = -1
	if _, ok := m.focusedAnnotationAtCursor(); !ok {
		m.clearFocusedAnnotation()
	}
	m.ensureCursorVisible()
}

func (m *Model) nextCommentable(from, direction int) int {
	length := m.sourceLength()
	for index := from + direction; index >= 0 && index < length; index += direction {
		if m.commentable(index) {
			return index
		}
	}
	return max(0, min(from, length-1))
}

func (m *Model) selectedRange() (int, int) {
	if m.selectionAnchor < 0 {
		return m.cursor, m.cursor
	}
	return min(m.selectionAnchor, m.cursor), max(m.selectionAnchor, m.cursor)
}

func (m *Model) sourceLength() int {
	return m.document.source().Len()
}

func (m *Model) commentable(index int) bool {
	return m.document.source().Commentable(index)
}

func (m *Model) currentCommentable() bool {
	return m.commentable(m.cursor)
}

func (m *Model) clearFocusedAnnotation() {
	m.focusedAnnotationID = 0
}

func (m *Model) focusedAnnotationAtCursor() (Annotation, bool) {
	if m.focusedAnnotationID == 0 {
		return Annotation{}, false
	}
	annotation, ok := m.annotationByID(m.focusedAnnotationID)
	if !ok {
		m.clearFocusedAnnotation()
		return Annotation{}, false
	}
	if !m.annotationContainsCursor(annotation) {
		return Annotation{}, false
	}
	return annotation, true
}

func (m *Model) annotationAtCursor() []Annotation {
	if annotation, ok := m.focusedAnnotationAtCursor(); ok {
		return []Annotation{annotation}
	}

	var matches []Annotation
	for _, annotation := range m.annotations.ForFile(m.changeID(), m.currentFilePath()) {
		if !m.annotationContainsCursor(annotation) {
			continue
		}
		matches = append(matches, annotation)
	}
	return matches
}

func (m *Model) currentFilePath() string {
	file := m.currentFile()
	if file == nil {
		return ""
	}
	return file.Path
}

func (m *Model) annotationContainsCursor(annotation Annotation) bool {
	file := m.currentFile()
	if file == nil || annotation.ChangeID != m.changeID() || annotation.File != file.Path {
		return false
	}
	return m.document.source().Contains(m.cursor, annotation)
}

func (m *Model) annotationAnchor(annotation Annotation) (int, bool) {
	file := m.currentFile()
	if file == nil || annotation.ChangeID != m.changeID() || annotation.File != file.Path {
		return 0, false
	}
	return m.document.source().Anchor(annotation)
}

func (m *Model) navigateFile(delta int) tea.Cmd {
	if len(m.document.files) == 0 || delta == 0 {
		return nil
	}
	m.document.file = wrapIndex(m.document.file+delta, len(m.document.files))
	m.clearFocusedAnnotation()
	m.resetPosition()
	if m.document.presentation == filePresentation {
		return m.loadCurrentFile()
	}
	return nil
}

func (m *Model) openTargetPicker() tea.Cmd {
	if m.document.loading {
		return intents.Invoke(intents.AddMessage{Text: "File picker is still loading"})
	}
	if len(m.document.files) == 0 {
		return intents.Invoke(intents.AddMessage{Text: "No changed files found"})
	}
	files := make([]jj.FileName, 0, len(m.document.files))
	for _, file := range m.document.files {
		files = append(files, jj.NewFileName(file.Path))
	}
	return common.OpenTargetPickerWithPayload(targetPickerPayload{}, source.FileSource{Files: files})
}

func (m *Model) openCommentPicker() tea.Cmd {
	return m.openCommentPickerFor(m.annotations.All())
}

func (m *Model) openCommentPickerFor(annotations []Annotation) tea.Cmd {
	payload := commentPickerPayload{}
	items := make([]source.Item, 0, len(annotations))
	labels := make(map[string]struct{}, len(annotations))
	for _, annotation := range annotations {
		label := m.commentPickerLabel(annotation)
		if _, exists := labels[label]; exists {
			label = fmt.Sprintf("%s [%d]", label, annotation.ID)
		}
		labels[label] = struct{}{}
		items = append(items, source.Item{
			Name:  label,
			Value: strconv.Itoa(annotation.ID),
			Kind:  source.KindComment,
		})
	}
	if len(items) == 0 {
		return intents.Invoke(intents.AddMessage{Text: "No annotations"})
	}
	return common.OpenTargetPickerWithPayload(payload, pickerSource{items: items})
}

func (m *Model) commentPickerLabel(annotation Annotation) string {
	preview := strings.Join(strings.Fields(annotation.Comment), " ")
	preview = ansi.Truncate(preview, 48, "…")
	lineRange := annotation.OldLines
	if annotation.NewLines.Start != 0 {
		lineRange = annotation.NewLines
	}
	return fmt.Sprintf("%s · %s:%s · %s", shortRevision(annotation.ChangeID), annotation.File, formatLineRange(lineRange), preview)
}

func (m *Model) selectCommentPickerTarget(_ commentPickerPayload, target string) tea.Cmd {
	id, err := strconv.Atoi(target)
	if err != nil {
		return intents.Invoke(intents.AddMessage{Text: "Annotation not found"})
	}
	annotation, ok := m.annotationByID(id)
	if !ok {
		return intents.Invoke(intents.AddMessage{Text: "Annotation not found"})
	}
	if annotation.ChangeID != m.changeID() && m.editing {
		return revisionNavigationBlockedMessage()
	}
	m.document.presentation = diffPresentation
	m.sourceVersion++
	m.focusedAnnotationID = annotation.ID
	if annotation.ChangeID == m.changeID() && !m.document.loading {
		return m.jumpToFocusedAnnotation()
	}
	return m.selectRevision(annotation.ChangeID)
}

func (m *Model) jumpToFocusedAnnotation() tea.Cmd {
	annotation, ok := m.annotationByID(m.focusedAnnotationID)
	if !ok || annotation.ChangeID != m.changeID() {
		m.clearFocusedAnnotation()
		return nil
	}
	file := slices.IndexFunc(m.document.files, func(candidate fileItem) bool {
		return candidate.Path == annotation.File
	})
	if file < 0 {
		m.clearFocusedAnnotation()
		return intents.Invoke(intents.AddMessage{Text: "Annotation file is unavailable"})
	}
	m.document.file = file
	m.sourceVersion++
	index, ok := m.annotationAnchor(annotation)
	if !ok {
		if m.document.presentation == diffPresentation {
			m.document.presentation = filePresentation
			m.resetPosition()
			if m.currentFile().ContentLoaded {
				return m.jumpToFocusedAnnotation()
			}
			if cmd := m.loadCurrentFile(); cmd != nil {
				return cmd
			}
		}
		m.clearFocusedAnnotation()
		if current := m.currentFile(); current != nil && current.ContentErr != nil {
			return intents.Invoke(intents.AddMessage{
				Text: fmt.Sprintf("Annotation file could not be loaded: %v", current.ContentErr),
			})
		}
		return intents.Invoke(intents.AddMessage{Text: "Annotation location is unavailable"})
	}
	m.cursor = index
	m.selectionAnchor = -1
	m.scrollX = 0
	m.ensureCursorVisible()
	return nil
}

func (m *Model) selectFile(path string) tea.Cmd {
	index := slices.IndexFunc(m.document.files, func(file fileItem) bool {
		return file.Path == path
	})
	if index < 0 {
		return nil
	}
	m.document.file = index
	m.clearFocusedAnnotation()
	m.resetPosition()
	if m.document.presentation == filePresentation {
		return m.loadCurrentFile()
	}
	return nil
}

func (m *Model) navigateRevision(direction revisionDirection) tea.Cmd {
	if m.editing {
		return revisionNavigationBlockedMessage()
	}
	if m.document.loadingRevision != "" {
		return nil
	}
	requestID := m.nextRequest()
	cmd := m.loader.LoadRevisionTargets(m.changeID(), direction, requestID)
	if cmd != nil {
		m.revisionTargetsRequestID = requestID
	}
	return cmd
}

func (m *Model) openRevisionPicker(revisions []revisionTarget) tea.Cmd {
	items := make([]source.Item, 0, len(revisions))
	for _, revision := range revisions {
		items = append(items, source.Item{
			Name:  fmt.Sprintf("%s · %s", revision.ChangeID, revision.Description),
			Value: revision.ChangeID,
			Kind:  source.KindRevision,
		})
	}
	return common.OpenTargetPickerWithPayload(
		revisionPickerPayload{},
		pickerSource{items: items},
	)
}

func (m *Model) selectRevisionPickerTarget(_ revisionPickerPayload, target string) tea.Cmd {
	if target == "" {
		return intents.Invoke(intents.AddMessage{Text: "Revision not found"})
	}
	if target != m.changeID() && m.editing {
		return revisionNavigationBlockedMessage()
	}
	m.clearFocusedAnnotation()
	return m.selectRevision(target)
}

func (m *Model) selectRevision(revision string) tea.Cmd {
	if revision != m.changeID() && m.editing {
		return revisionNavigationBlockedMessage()
	}
	if revision == "" || revision == m.changeID() || revision == m.document.loadingRevision {
		return nil
	}
	m.revisionTargetsRequestID = 0
	cmd := m.loadRevision(revision)
	if cmd == nil {
		return nil
	}
	m.document.loadingRevision = revision
	return cmd
}

func wrapIndex(index, length int) int {
	index %= length
	if index < 0 {
		index += length
	}
	return index
}

func (m *Model) resetPosition() {
	m.sourceVersion++
	m.cursor = 0
	m.selectionAnchor = -1
	m.scrollY = 0
	m.scrollX = 0
	m.editing = false
	if length := m.sourceLength(); length > 0 && !m.commentable(0) {
		m.cursor = m.nextCommentable(-1, 1)
	}
}

func (m *Model) ensureCursorVisible() {
	if m.viewportHeight <= 0 {
		return
	}
	lines, editorStart := m.buildDisplayLines(m.viewportWidth)
	position := slices.IndexFunc(lines, func(line displayLine) bool {
		return line.SourceIndex == m.cursor
	})
	if position < 0 {
		return
	}
	if position < m.scrollY {
		m.scrollY = position
	}
	if position >= m.scrollY+m.viewportHeight {
		m.scrollY = position - m.viewportHeight + 1
	}
	if m.editing && editorStart >= 0 {
		editorEnd := editorStart
		for editorEnd+1 < len(lines) && lines[editorEnd+1].Editor {
			editorEnd++
		}
		if editorEnd >= m.scrollY+m.viewportHeight {
			m.scrollY = editorEnd - m.viewportHeight + 1
		}
	}
	m.clampScroll(len(lines))
}

func (m *Model) clampScroll(total int) {
	m.scrollY = max(0, min(m.scrollY, max(0, total-m.viewportHeight)))
}

func (m *Model) loadRevision(changeID string) tea.Cmd {
	m.revisionTargetsRequestID = 0
	requestID := m.nextRequest()
	cmd := m.loader.LoadRevision(changeID, m.document.presentation == filePresentation, requestID)
	if cmd != nil {
		m.revisionLoadRequestID = requestID
	}
	return cmd
}

func (m *Model) nextRequest() uint64 {
	m.nextRequestID++
	return m.nextRequestID
}

func revisionNavigationBlockedMessage() tea.Cmd {
	return intents.Invoke(intents.AddMessage{Text: "Finish editing the comment before navigating revisions"})
}

func (m *Model) loadCurrentFile() tea.Cmd {
	return m.loader.LoadFile(m.changeID(), m.currentFile())
}

func (m *Model) changeID() string {
	return m.document.changeID()
}

func shortRevision(changeID string) string {
	if len(changeID) > 12 {
		return changeID[:12]
	}
	return changeID
}

func (m *Model) currentFile() *fileItem {
	return m.document.currentFile()
}
