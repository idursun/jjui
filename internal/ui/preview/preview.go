package preview

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	scrollAmount = 3
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	view                viewport.Model
	previewVisible      bool
	previewAutoPosition bool
	previewAtBottom     bool
	content             string
	contentLineCount    int
	contentWidth        int
	context             *context.MainContext
	keyMap              config.KeyMappings[key.Binding]
}

const (
	debounceId       = "preview-refresh"
	debounceDuration = 50 * time.Millisecond
)

type previewMsg struct {
	msg tea.Msg
}

// Allow a message to be targetted to this component.
func PreviewCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return previewMsg{msg: msg}
	}
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
	return m.view.YOffset
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
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Preview.ScrollDown):
			m.Scroll(1)
		case key.Matches(msg, m.keyMap.Preview.ScrollUp):
			m.Scroll(-1)
		case key.Matches(msg, m.keyMap.Preview.HalfPageDown):
			m.view.HalfPageDown()
		case key.Matches(msg, m.keyMap.Preview.HalfPageUp):
			m.view.HalfPageUp()
		}
	}
	return nil
}

func (m *Model) SetContent(content string) {
	m.reset()
	m.content = strings.ReplaceAll(content, "\r", "")
	m.view.SetContent(content)
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	m.view.Width = box.R.Dx()
	m.view.Height = box.R.Dy()
	dl.AddDraw(box.R, m.view.View(), 0)

	scrollRect := cellbuf.Rect(box.R.Min.X, box.R.Min.Y, box.R.Dx(), box.R.Dy())
	dl.AddInteraction(scrollRect, ScrollMsg{}, render.InteractionScroll, 0)
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
		previewWidth := strconv.Itoa(m.view.Width)
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

		output, _ := m.context.RunCommandImmediate(args)
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
		keyMap:              config.Current.GetKeyMap(),
		previewAutoPosition: previewAutoPosition,
		previewAtBottom:     previewAtBottom,
		previewVisible:      config.Current.Preview.ShowAtStart,
	}
}
