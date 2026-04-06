package preview

import (
	"log"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/routing"
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	view                viewport.Model
	previewVisible      bool
	previewAutoPosition bool
	previewAtBottom     bool
	content             string
	context             *context.MainContext
}

const (
	debounceId       = "preview-refresh"
	debounceDuration = 50 * time.Millisecond
)

type previewMsg struct {
	msg tea.Msg
}

type updatePreviewContentMsg struct {
	Content string
}

type ScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (s ScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	s.Delta = delta
	s.Horizontal = horizontal
	return s
}

func (m *Model) Scopes() []routing.Scope {
	if !m.Visible() {
		return nil
	}
	return []routing.Scope{
		{
			Name:      actions.ScopeUiPreview,
			AllowLeak: true,
			Handler:   m,
		},
	}
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch msg := intent.(type) {
	case intents.PreviewScroll:
		switch msg.Kind {
		case intents.PreviewScrollUp:
			return m.Scroll(-1), true
		case intents.PreviewScrollDown:
			return m.Scroll(1), true
		case intents.PreviewPageUp:
			return m.PageUp(), true
		case intents.PreviewPageDown:
			return m.PageDown(), true
		case intents.PreviewHalfPageUp:
			return m.HalfPageUp(), true
		case intents.PreviewHalfPageDown:
			return m.HalfPageDown(), true
		}
		return nil, true
	}
	return nil, false
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Visible() bool {
	return m.previewVisible
}

func (m *Model) SetVisible(visible bool) {
	m.previewVisible = visible
	if m.previewVisible {
		m.reset()
	}
}

func (m *Model) ToggleVisible() {
	m.previewVisible = !m.previewVisible
	if m.previewVisible {
		m.reset()
	}
}

func (m *Model) SetPosition(autoPos bool, atBottom bool) {
	m.previewAutoPosition = autoPos
	m.previewAtBottom = atBottom
}

func (m *Model) AutoPosition() bool {
	return m.previewAutoPosition
}

func (m *Model) AtBottom() bool {
	return m.previewAtBottom
}

func (m *Model) YOffset() int {
	return m.view.YOffset()
}

func (m *Model) Scroll(delta int) tea.Cmd {
	if delta > 0 {
		m.view.ScrollDown(delta)
	} else if delta < 0 {
		m.view.ScrollUp(-delta)
	}
	return nil
}

func (m *Model) ScrollHorizontal(delta int) tea.Cmd {
	if delta > 0 {
		m.view.ScrollRight(delta)
	} else if delta < 0 {
		m.view.ScrollLeft(-delta)
	}

	return nil
}

func (m *Model) HalfPageDown() tea.Cmd {
	m.view.HalfPageDown()
	return nil
}

func (m *Model) HalfPageUp() tea.Cmd {
	m.view.HalfPageUp()
	return nil
}

func (m *Model) PageDown() tea.Cmd {
	m.view.PageDown()
	return nil
}

func (m *Model) PageUp() tea.Cmd {
	m.view.PageUp()
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(previewMsg); ok {
		msg = k.msg
	}
	switch msg := msg.(type) {
	case ScrollMsg:
		if msg.Horizontal {
			m.ScrollHorizontal(msg.Delta)
		} else {
			m.Scroll(msg.Delta)
		}
	case intents.PreviewShow:
		m.SetContent(msg.Content)
		return nil
	case common.SelectionChangedMsg:
		if msg.Item != nil {
			return m.refreshPreviewForItem(msg.Item)
		}
		return m.refreshPreview()
	case common.RefreshMsg:
		return m.refreshPreview()
	case updatePreviewContentMsg:
		m.SetContent(msg.Content)
		return nil
	}
	return nil
}

func (m *Model) SetContent(content string) {
	content = strings.ReplaceAll(content, "\r", "")
	if strings.ContainsRune(content, '\t') {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = render.ExpandTabs(line)
		}
		content = strings.Join(lines, "\n")
	}
	m.reset()
	m.content = content
	m.view.SetContent(content)
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	m.view.SetWidth(box.R.Dx())
	m.view.SetHeight(box.R.Dy())
	dl.AddDraw(box.R, m.view.View(), render.ZPreview)

	scrollRect := layout.Rect(box.R.Min.X, box.R.Min.Y, box.R.Dx(), box.R.Dy())
	dl.AddInteraction(scrollRect, ScrollMsg{}, render.InteractionScroll, render.ZPreview)
}

func (m *Model) reset() {
	m.view.SetYOffset(0)
	m.view.SetXOffset(0)
}

func (m *Model) refreshPreview() tea.Cmd {
	return m.refreshPreviewForItem(m.context.SelectedItem)
}

func (m *Model) refreshPreviewForItem(item common.SelectedItem) tea.Cmd {
	return common.Debounce(debounceId, debounceDuration, func() tea.Msg {
		var args []string
		previewWidth := strconv.Itoa(m.view.Width())
		switch sel := item.(type) {
		case common.SelectedFile:
			args = jj.TemplatedArgs(config.Current.Preview.FileCommand, map[string]string{
				jj.RevsetPlaceholder:       m.context.CurrentRevset,
				jj.ChangeIdPlaceholder:     sel.ChangeId,
				jj.CommitIdPlaceholder:     sel.CommitId,
				jj.FilePlaceholder:         sel.File,
				jj.PreviewWidthPlaceholder: previewWidth,
			})
		case common.SelectedRevision:
			args = jj.TemplatedArgs(config.Current.Preview.RevisionCommand, map[string]string{
				jj.RevsetPlaceholder:       m.context.CurrentRevset,
				jj.ChangeIdPlaceholder:     sel.ChangeId,
				jj.CommitIdPlaceholder:     sel.CommitId,
				jj.PreviewWidthPlaceholder: previewWidth,
			})
		case common.SelectedCommit:
			args = jj.TemplatedArgs(config.Current.Preview.EvologCommand, map[string]string{
				jj.RevsetPlaceholder:       m.context.CurrentRevset,
				jj.CommitIdPlaceholder:     sel.CommitId,
				jj.PreviewWidthPlaceholder: previewWidth,
			})
		case common.SelectedOperation:
			args = jj.TemplatedArgs(config.Current.Preview.OplogCommand, map[string]string{
				jj.RevsetPlaceholder:       m.context.CurrentRevset,
				jj.OperationIdPlaceholder:  sel.OperationId,
				jj.PreviewWidthPlaceholder: previewWidth,
			})
		}

		env := []string{
			// The preview subprocess does not run in a pane-sized PTY, so let
			// width-sensitive tools like `jj diff` see the preview size via the
			// conventional terminal size environment variables.
			"COLUMNS=" + strconv.Itoa(m.view.Width()),
			"LINES=" + strconv.Itoa(m.view.Height()),
		}
		output, _ := m.context.RunCommandImmediateWithEnv(args, env)
		return updatePreviewContentMsg{
			Content: string(output),
		}
	})
}

func New(context *context.MainContext) *Model {
	previewAutoPosition := false
	previewAtBottom := false
	previewPositionCfg, err := config.GetPreviewPosition(config.Current)
	if err != nil {
		log.Fatal(err)
	}

	if previewPositionCfg == config.PreviewPositionAuto {
		previewAutoPosition = true
	} else if previewPositionCfg == config.PreviewPositionBottom {
		previewAtBottom = true
	}

	return &Model{
		context:             context,
		previewAutoPosition: previewAutoPosition,
		previewAtBottom:     previewAtBottom,
		previewVisible:      config.Current.Preview.ShowAtStart,
	}
}
