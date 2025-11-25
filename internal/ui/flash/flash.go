package flash

import (
	"strings"
	"time"

	"github.com/idursun/jjui/internal/screen"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
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

var _ common.Model = (*Model)(nil)

type Model struct {
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

func (m *Model) View() string {
	messages := m.messages
	if len(messages) == 0 {
		return ""
	}

	var messageBoxes []string
	for _, message := range messages {
		style := m.successStyle
		if message.error != nil {
			style = m.errorStyle
			messageBoxes = append(messageBoxes, style.Render(message.error.Error()))
		} else {
			messageBoxes = append(messageBoxes, style.Render(message.text))
		}
	}
	maxWidth, maxHeight := 0, 0
	var combined []string
	for _, box := range messageBoxes {
		width, height := lipgloss.Size(box)
		if width > maxWidth {
			maxWidth = width
		}
		if height > maxHeight {
			maxHeight = height
		}
	}
	for _, box := range messageBoxes {
		combined = append(combined,
			lipgloss.PlaceHorizontal(maxWidth,
				lipgloss.Right, box,
				lipgloss.WithWhitespaceForeground(screen.TransparentFg),
				lipgloss.WithWhitespaceBackground(screen.TransparentBg),
			),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Right, combined...)
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
		context:      context,
		messages:     make([]flashMessage, 0),
		successStyle: successStyle,
		errorStyle:   errorStyle,
	}
}
