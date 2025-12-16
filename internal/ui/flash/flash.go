package flash

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
)

const expiringMessageTimeout = 4 * time.Second

type expireMessageMsg struct {
	id uint64
}

type flashMessage struct {
	text    string
	error   error
	timeout int
	id      uint64
}

type FlashMessageView struct {
	// Content might contain ANSI colour codes
	Content string
	Rect    cellbuf.Rectangle
}

type Model struct {
	*common.ViewNode
	context      *context.MainContext
	messages     []flashMessage
	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
	currentId    uint64
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return m.handleIntent(msg)
	case expireMessageMsg:
		for i, message := range m.messages {
			if message.id == msg.id {
				m.messages = append(m.messages[:i], m.messages[i+1:]...)
				break
			}
		}
		return nil
	case common.CommandCompletedMsg:
		id := m.add(msg.Output, msg.Err)
		if msg.Err == nil {
			return tea.Tick(expiringMessageTimeout, func(t time.Time) tea.Msg {
				return expireMessageMsg{id: id}
			})
		}
		return nil
	case common.UpdateRevisionsFailedMsg:
		m.add(msg.Output, msg.Err)
	}
	return nil
}

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.AddMessage:
		id := m.add(intent.Text, intent.Err)
		if intent.Err == nil && !intent.NoTimeout && id != 0 {
			return tea.Tick(expiringMessageTimeout, func(t time.Time) tea.Msg {
				return expireMessageMsg{id: id}
			})
		}
		return nil
	case intents.DismissOldest:
		if len(m.messages) == 0 {
			return nil
		}
		m.DeleteOldest()
		return nil
	}
	return nil
}

func (m *Model) View() []FlashMessageView {
	messages := m.messages
	if len(messages) == 0 {
		return nil
	}

	y := m.Height - 1
	var messageBoxes []FlashMessageView
	// reserve padding and calculate max width for messages
	maxWidth := m.Width - 4

	for _, message := range messages {
		var text string
		var style lipgloss.Style
		if message.error != nil {
			text = message.error.Error()
			style = m.errorStyle
		} else {
			text = message.text
			style = m.successStyle
		}

		// first render without width to check natural size
		naturalContent := style.Render(text)
		naturalWidth, _ := lipgloss.Size(naturalContent)

		var content string
		if naturalWidth <= maxWidth {
			content = naturalContent
		} else {
			// width doesn't fit within maxWidth, set Width for line wrap
			content = style.Width(maxWidth).Render(text)
		}

		w, h := lipgloss.Size(content)
		y -= h
		messageBoxes = append(messageBoxes, FlashMessageView{
			Content: content,
			Rect:    cellbuf.Rect(m.Width-w, y, w, h),
		})
	}
	return messageBoxes
}

func (m *Model) add(text string, error error) uint64 {
	text = strings.TrimSpace(text)
	if text == "" && error == nil {
		return 0
	}

	msg := flashMessage{
		id:    m.nextId(),
		text:  text,
		error: error,
	}

	m.messages = append(m.messages, msg)
	return msg.id
}

func (m *Model) Any() bool {
	return len(m.messages) > 0
}

func (m *Model) DeleteOldest() {
	m.messages = m.messages[1:]
}

func (m *Model) nextId() uint64 {
	m.currentId = m.currentId + 1
	return m.currentId
}

func New(context *context.MainContext) *Model {
	fg := lipgloss.NewStyle().GetForeground()
	successStyle := common.DefaultPalette.GetBorder("success", lipgloss.NormalBorder()).Foreground(fg).PaddingLeft(1).PaddingRight(1)
	errorStyle := common.DefaultPalette.GetBorder("error", lipgloss.NormalBorder()).Foreground(fg).PaddingLeft(1).PaddingRight(1)
	return &Model{
		ViewNode:     common.NewViewNode(0, 0),
		context:      context,
		messages:     make([]flashMessage, 0),
		successStyle: successStyle,
		errorStyle:   errorStyle,
	}
}
