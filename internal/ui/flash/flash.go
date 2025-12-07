package flash

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

const expiringMessageTimeout = 4 * time.Second

type Intent interface {
	apply(*Model) tea.Cmd
}

// Cmd wraps a flash intent into a Tea command.
func Cmd(intent Intent) tea.Cmd {
	return func() tea.Msg {
		return intent
	}
}

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
	case Intent:
		return msg.apply(m)
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

func (m *Model) View() []FlashMessageView {
	messages := m.messages
	if len(messages) == 0 {
		return nil
	}

	y := m.Height - 1
	var messageBoxes []FlashMessageView
	for _, message := range messages {
		var content string
		if message.error != nil {
			content = m.errorStyle.Render(message.error.Error())
		} else {
			content = m.successStyle.Render(message.text)
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

// AddMessage adds a flash message with optional error; non-error messages expire.
type AddMessage struct {
	Text      string
	Err       error
	NoTimeout bool
}

func (a AddMessage) apply(m *Model) tea.Cmd {
	id := m.add(a.Text, a.Err)
	if a.Err == nil && !a.NoTimeout && id != 0 {
		return tea.Tick(expiringMessageTimeout, func(t time.Time) tea.Msg {
			return expireMessageMsg{id: id}
		})
	}
	return nil
}

// DismissOldest removes the oldest flash message if present.
type DismissOldest struct{}

func (DismissOldest) apply(m *Model) tea.Cmd {
	if len(m.messages) == 0 {
		return nil
	}
	m.DeleteOldest()
	return nil
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
