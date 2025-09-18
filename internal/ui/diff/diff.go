package diff

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ view.IViewModel = (*Model)(nil)

type updateDiffContentMsg string
type updateDiffCommandMsg jj.DiffCommandArgs

var ignoreAllWhiteSpaceKey = key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "ignore all white space"))
var colorWordsKey = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "color words"))
var summaryKey = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "summary"))
var compareToMainKey = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "compare to main"))

type Model struct {
	*view.ViewNode
	view        viewport.Model
	keymap      config.KeyMappings[key.Binding]
	commandArgs *jj.DiffCommandArgs
	context     *context.MainContext
}

func (m *Model) GetId() view.ViewId {
	return "diff"
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	m.view.Width = v.Width
	m.view.Height = v.Height
	v.ViewOpts.Sizeable.SetWidth(m.view.Width)
	v.ViewOpts.Sizeable.SetHeight(m.view.Height)
	v.ViewOpts.Id = m.GetId()
}

func (m *Model) ShortHelp() []key.Binding {
	vkm := m.view.KeyMap
	if m.commandArgs != nil {
		switch m.commandArgs.Formatting.WhiteSpace {
		case jj.WhitespaceIgnoreChange:
			ignoreAllWhiteSpaceKey.SetHelp("w", "ignore all white space: changes")
		case jj.WhitespaceIgnoreAll:
			ignoreAllWhiteSpaceKey.SetHelp("w", "ignore all white space: all")
		case jj.WhitespaceDefault:
			ignoreAllWhiteSpaceKey.SetHelp("w", "ignore all white space: OFF")
		}

		switch m.commandArgs.Formatting.Display {
		case jj.DiffDisplayNone:
			colorWordsKey.SetHelp("c", "no color")
		case jj.DiffDisplayColorWords:
			colorWordsKey.SetHelp("c", "color words")
		case jj.DiffDisplayGit:
			colorWordsKey.SetHelp("c", "git")
		}
	}
	return []key.Binding{
		m.keymap.Cancel,
		ignoreAllWhiteSpaceKey,
		colorWordsKey,
		summaryKey,
		vkm.Up,
		vkm.Down,
		vkm.HalfPageDown,
		vkm.HalfPageUp,
		vkm.PageDown,
		vkm.PageUp,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	if m.commandArgs != nil {
		return m.executeCommand()
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateDiffContentMsg:
		content := strings.ReplaceAll(string(msg), "\r", "")
		if content == "" {
			content = "(empty)"
		}
		m.view.SetContent(content)
		return m, nil

	case updateDiffCommandMsg:
		m.commandArgs = (*jj.DiffCommandArgs)(&msg)
		m.view.SetContent("(loading)")
		return m, m.executeCommand()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ignoreAllWhiteSpaceKey):
			if m.commandArgs != nil {
				switch m.commandArgs.Formatting.WhiteSpace {
				case jj.WhitespaceDefault:
					m.commandArgs.Formatting.WhiteSpace = jj.WhitespaceIgnoreAll
				case jj.WhitespaceIgnoreAll:
					m.commandArgs.Formatting.WhiteSpace = jj.WhitespaceIgnoreChange
				case jj.WhitespaceIgnoreChange:
					m.commandArgs.Formatting.WhiteSpace = jj.WhitespaceDefault
				}
				return m, m.executeCommand()
			}
		case key.Matches(msg, colorWordsKey):
			if m.commandArgs != nil {
				switch m.commandArgs.Formatting.Display {
				case jj.DiffDisplayNone:
					m.commandArgs.Formatting.Display = jj.DiffDisplayGit
				case jj.DiffDisplayGit:
					m.commandArgs.Formatting.Display = jj.DiffDisplayColorWords
				case jj.DiffDisplayColorWords:
					m.commandArgs.Formatting.Display = jj.DiffDisplayNone
				}
				return m, m.executeCommand()
			}
		case key.Matches(msg, summaryKey):
			if m.commandArgs != nil {
				switch {
				case m.commandArgs.Formatting.Output == jj.OutputModeNone:
					m.commandArgs.Formatting.Output = jj.OutputModeSummary
				case m.commandArgs.Formatting.Output == jj.OutputModeSummary:
					m.commandArgs.Formatting.Output = jj.OutputModeStat
				case m.commandArgs.Formatting.Output == jj.OutputModeStat:
					m.commandArgs.Formatting.Output = jj.OutputModeTypes
				case m.commandArgs.Formatting.Output == jj.OutputModeTypes:
					m.commandArgs.Formatting.Output = jj.OutputModeNameOnly
				case m.commandArgs.Formatting.Output == jj.OutputModeNameOnly:
					m.commandArgs.Formatting.Output = jj.OutputModeNone
				}
				return m, m.executeCommand()
			}
		case key.Matches(msg, compareToMainKey):
			if m.commandArgs != nil {
				m.commandArgs.Source = jj.NewDiffRangeArgs(
					jj.NewSingleSourceFromRevision(m.context.Revisions.Current()),
					jj.NewRevsetSource("main"),
				)
				return m, m.executeCommand()
			}
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
		}
	}
	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	m.view.Height = m.Sizeable.Height
	m.view.Width = m.Sizeable.Width
	return m.view.View()
}

func (m *Model) executeCommand() tea.Cmd {
	return func() tea.Msg {
		output, _ := m.context.RunCommandImmediate(jj.Args(m.commandArgs))
		return updateDiffContentMsg(output)
	}
}

func UpdateDiffCommand(args jj.DiffCommandArgs) tea.Cmd {
	return func() tea.Msg {
		return updateDiffCommandMsg(args)
	}
}

func New(ctx *context.MainContext) view.IViewModel {
	v := viewport.New(0, 0)
	v.SetContent("(empty)")
	m := &Model{
		context: ctx,
		view:    v,
		keymap:  config.Current.GetKeyMap(),
	}
	return m
}
