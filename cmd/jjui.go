package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"jjui/internal/jj"
	"os"
	"strings"
)

type mode int

const (
	normalMode mode = iota
	moveMode
)

type model struct {
	items              []jj.Commit
	mode               mode
	draggedCommitIndex int
	cursor             int
	width              int
}

func fetchLog(location string) tea.Cmd {
	return func() tea.Msg {
		lines := jj.GetCommits(location)
		return logCommand(lines)
	}
}

func rebaseCommand(from, to string) tea.Cmd {
	if err := jj.RebaseCommand(from, to); err != nil {
		fmt.Printf("error: %v\n", err)
	}
	return fetchLog(os.Getenv("PWD"))
}

type logCommand []jj.Commit

func (m model) Init() tea.Cmd {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil
	}
	return fetchLog(dir)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "esc":
			m.draggedCommitIndex = -1
			m.mode = normalMode
		case " ":
			if m.mode == normalMode {
				m.mode = moveMode
				m.draggedCommitIndex = m.cursor
			} else {
				m.mode = normalMode
				m.draggedCommitIndex = -1
			}
		case "enter":
			if m.mode == moveMode {
				m.mode = normalMode
				fromRevision := m.items[m.draggedCommitIndex].ChangeIdShort
				toRevision := m.items[m.cursor].ChangeIdShort
				m.draggedCommitIndex = -1
				return m, rebaseCommand(fromRevision, toRevision)
			}
		default:
			return m, nil
		}
	case logCommand:
		commits := []jj.Commit(msg)
		m.items = commits
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

var highlightColor = lipgloss.Color("#39a8f7")
var commitShortStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#DC8CCA"))

var commitShortStyleHighlighted = lipgloss.NewStyle().
	Background(highlightColor).
	Inherit(commitShortStyle)

var commitIdRestStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#696969"))

var commitIdRestHighlightedStyle = lipgloss.NewStyle().
	Background(highlightColor).
	Inherit(commitIdRestStyle)

var normal = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#e6e6e6"))

var normalHighlighted = lipgloss.NewStyle().
	Background(highlightColor).
	Inherit(normal)

func (m model) View() string {
	items := strings.Builder{}
	for i := 0; i < len(m.items); i++ {
		commit := &m.items[i]
		switch m.mode {
		case moveMode:
			if i == m.cursor {
				draggedCommit := &m.items[m.draggedCommitIndex]
				items.WriteString(m.viewCommit(draggedCommit, i == m.cursor, commit.Level()))
			}
			if i != m.draggedCommitIndex {
				items.WriteString(m.viewCommit(commit, false, commit.Level()))
			}
		case normalMode:
			items.WriteString(m.viewCommit(commit, i == m.cursor, commit.Level()))
		}
	}
	bottom := fmt.Sprintf("use j,k keys to move up and down: %v\n", m.cursor)
	if m.mode == moveMode {
		bottom += "jj rebase -r " + m.items[m.draggedCommitIndex].ChangeIdShort + " -d " + m.items[m.cursor].ChangeIdShort + "\n"
	}
	items.WriteString(bottom)
	return items.String()
}

func (m model) viewCommit(commit *jj.Commit, highlighted bool, level int) string {
	changeIdRemaining := strings.TrimPrefix(commit.ChangeId, commit.ChangeIdShort)
	builder := strings.Builder{}
	for j := 0; j < level; j++ {
		builder.WriteString(normal.Render(" │ "))
	}

	if commit.IsWorkingCopy {
		builder.WriteString(normal.Render(" @ "))
	} else {
		builder.WriteString(normal.Render(" o "))
	}

	if highlighted {
		builder.WriteString(commitShortStyle.Background(highlightColor).Render(commit.ChangeIdShort))
		builder.WriteString(commitIdRestStyle.Background(highlightColor).Render(changeIdRemaining + " "))
		builder.WriteString(authorStyle.Background(highlightColor).Render(commit.Author) + "\n")
		builder.WriteString(strings.Repeat(" │ ", level+1))
		if commit.Description == "" {
			builder.WriteString(normal.Background(highlightColor).Bold(true).Foreground(lipgloss.Color("#50fa7b")).Width(m.width).Render("(no description)"))
		} else {
			builder.WriteString(normal.Background(highlightColor).Width(m.width).Render(commit.Description))
		}
	} else {
		builder.WriteString(commitShortStyle.Render(commit.ChangeIdShort))
		builder.WriteString(commitIdRestStyle.Render(changeIdRemaining + " "))
		builder.WriteString(authorStyle.Render(commit.Author) + "\n")
		builder.WriteString(strings.Repeat(" │ ", level+1))
		if commit.Description == "" {
			builder.WriteString(normal.Bold(true).Foreground(lipgloss.Color("#50fa7b")).Render("(no description)"))
		} else {
			builder.WriteString(normal.Render(commit.Description))
		}
	}
	builder.WriteString("\n")
	return builder.String()
}

func initialModel() model {
	return model{
		items:              []jj.Commit{},
		draggedCommitIndex: -1,
		mode:               normalMode,
		cursor:             0,
		width:              20,
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}