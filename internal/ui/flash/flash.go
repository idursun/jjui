package flash

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.ImmediateModel = (*Model)(nil)

type expireMessageMsg struct {
	id uint64
}

type flashMessage struct {
	text    string
	command string
	error   error
	id      uint64
}

type Model struct {
	messages        []flashMessage
	messageHistory  []flashMessage // completed commands only
	pendingCommands map[int]string
	pendingResults  map[int]pendingResult
	spinner         spinner.Model
	renderer        CardRenderer
	currentId       uint64
}

const HistoryLimit = 50

type pendingResult struct {
	Output string
	Err    error
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return m.handleIntent(msg)
	case expireMessageMsg:
		m.removeLiveMessageByID(msg.id)
		return nil
	case common.CommandRunningMsg:
		m.pendingCommands[msg.ID] = msg.Command
		if result, ok := m.pendingResults[msg.ID]; ok {
			delete(m.pendingCommands, msg.ID)
			delete(m.pendingResults, msg.ID)
			return m.completeCommand(msg.Command, result.Output, result.Err)
		}
		return m.spinner.Tick
	case common.CommandCompletedMsg:
		if msg.ID == 0 {
			return m.completeCommand("", msg.Output, msg.Err)
		}
		cmd := m.pendingCommands[msg.ID]
		if cmd == "" {
			if m.pendingResults == nil {
				m.pendingResults = make(map[int]pendingResult)
			}
			m.pendingResults[msg.ID] = pendingResult{
				Output: msg.Output,
				Err:    msg.Err,
			}
			return nil
		}
		delete(m.pendingCommands, msg.ID)
		return m.completeCommand(cmd, msg.Output, msg.Err)
	case common.UpdateRevisionsFailedMsg:
		m.add(msg.Output, msg.Err)
	default:
		if len(m.pendingCommands) > 0 {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.AddMessage:
		id := m.add(intent.Text, intent.Err)
		if intent.Err == nil && !intent.Sticky && id != 0 {
			expiringMessageTimeout := config.GetExpiringFlashMessageTimeout(config.Current)
			if expiringMessageTimeout > time.Duration(0) {
				return tea.Tick(expiringMessageTimeout, func(t time.Time) tea.Msg {
					return expireMessageMsg{id: id}
				})
			}
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

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	area := box.R
	y := area.Max.Y - 1
	y = m.renderMessages(dl, area, m.messages, y)
	m.renderPendingCommands(dl, area, y)
}

func (m *Model) renderMessages(dl *render.DisplayContext, area layout.Rectangle, messages []flashMessage, y int) int {
	maxWidth := area.Dx() - 4
	for _, message := range messages {
		content := m.renderer.RenderMessage(message.command, message.text, message.error, maxWidth)
		w, h := lipgloss.Size(content)
		y -= h

		rect := layout.Rect(area.Max.X-w, y, w, h)
		dl.AddDraw(rect, content, render.ZOverlay)
	}
	return y
}

func (m *Model) renderPendingCommands(dl *render.DisplayContext, area layout.Rectangle, y int) int {
	maxWidth := area.Dx() - 4
	for _, cmd := range m.pendingCommands {
		content := m.renderer.RenderRunningCommand(cmd, m.spinner.View(), maxWidth)
		w, h := lipgloss.Size(content)
		y -= h
		rect := layout.Rect(area.Max.X-w, y, w, h)
		dl.AddDraw(rect, content, render.ZOverlay)
	}
	return y
}

func (m *Model) removeLiveMessageByID(id uint64) bool {
	for i, message := range m.messages {
		if message.id != id {
			continue
		}
		m.messages = append(m.messages[:i], m.messages[i+1:]...)
		return true
	}
	return false
}

func (m *Model) completeCommand(command string, output string, commandErr error) tea.Cmd {
	id := m.AddWithCommand(output, command, commandErr)
	if id != 0 && commandErr == nil {
		expiringMessageTimeout := config.GetExpiringFlashMessageTimeout(config.Current)
		if expiringMessageTimeout > time.Duration(0) {
			return tea.Tick(expiringMessageTimeout, func(t time.Time) tea.Msg {
				return expireMessageMsg{id: id}
			})
		}
	}
	return nil
}

// ColorizeCommand tokenizes cmd and applies textStyle to plain tokens and
// matchedStyle to flag tokens (those starting with "-").
func ColorizeCommand(cmd string, textStyle, matchedStyle lipgloss.Style) string {
	tokens := strings.Split(strings.ReplaceAll(cmd, "\n", "⏎"), " ")
	var b strings.Builder
	for i, token := range tokens {
		if i > 0 {
			b.WriteByte(' ')
		}
		if strings.HasPrefix(token, "-") {
			b.WriteString(matchedStyle.Render(token))
		} else {
			b.WriteString(textStyle.Render(token))
		}
	}
	return b.String()
}

func (m *Model) add(text string, error error) uint64 {
	return m.AddWithCommand(text, "", error)
}

func (m *Model) AddWithCommand(text string, command string, error error) uint64 {
	text = strings.TrimSpace(text)
	if text == "" && error == nil && command == "" {
		return 0
	}

	msg := flashMessage{
		id:      m.nextId(),
		text:    text,
		command: command,
		error:   error,
	}

	m.messages = append(m.messages, msg)
	if msg.command != "" {
		m.messageHistory = append(m.messageHistory, msg)
		if len(m.messageHistory) > HistoryLimit {
			m.messageHistory = append([]flashMessage(nil), m.messageHistory[len(m.messageHistory)-HistoryLimit:]...)
		}
	}
	return msg.id
}

func (m *Model) Any() bool {
	return len(m.messages) > 0
}

func (m *Model) LiveMessagesCount() int {
	return len(m.messages)
}

func (m *Model) DeleteOldest() {
	if len(m.messages) == 0 {
		return
	}
	m.messages = m.messages[1:]
}

type CommandHistoryEntry struct {
	ID      uint64
	Command string
	Text    string
	Err     error
}

type CommandHistorySource interface {
	CommandHistorySnapshot() []CommandHistoryEntry
	DeleteCommandHistoryByID(id uint64)
}

func (m *Model) CommandHistorySnapshot() []CommandHistoryEntry {
	out := make([]CommandHistoryEntry, 0, len(m.messageHistory))
	for _, item := range m.messageHistory {
		out = append(out, CommandHistoryEntry{
			ID:      item.id,
			Command: item.command,
			Text:    item.text,
			Err:     item.error,
		})
	}
	return out
}

func (m *Model) DeleteCommandHistoryByID(id uint64) {
	for i, item := range m.messageHistory {
		if item.id != id {
			continue
		}
		m.messageHistory = append(m.messageHistory[:i], m.messageHistory[i+1:]...)
		break
	}
	m.removeLiveMessageByID(id)
}

func (m *Model) nextId() uint64 {
	m.currentId = m.currentId + 1
	return m.currentId
}

func New() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return &Model{
		messages:        make([]flashMessage, 0),
		messageHistory:  make([]flashMessage, 0),
		pendingCommands: make(map[int]string),
		pendingResults:  make(map[int]pendingResult),
		renderer:        NewCardRenderer(),
		spinner:         s,
	}
}
