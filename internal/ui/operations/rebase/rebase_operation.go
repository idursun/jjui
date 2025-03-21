package rebase

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Source int

const (
	SourceRevision Source = iota
	SourceBranch
	SourceDescendants
)

type Target int

const (
	TargetDestination Target = iota
	TargetAfter
	TargetBefore
)

var (
	sourceToFlags = map[Source]string{
		SourceBranch:      "--branch",
		SourceRevision:    "--revisions",
		SourceDescendants: "--source",
	}
	targetToFlags = map[Target]string{
		TargetAfter:       "--insert-after",
		TargetBefore:      "--insert-before",
		TargetDestination: "--destination",
	}
)

type Operation struct {
	context context.AppContext
	From    string
	To      *jj.Commit
	Source  Source
	Target  Target
	keyMap  config.KeyMappings[key.Binding]
}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keyMap.Rebase.Revision):
		r.Source = SourceRevision
	case key.Matches(msg, r.keyMap.Rebase.Branch):
		r.Source = SourceBranch
	case key.Matches(msg, r.keyMap.Rebase.Source):
		r.Source = SourceDescendants
	case key.Matches(msg, r.keyMap.Rebase.Onto):
		r.Target = TargetDestination
	case key.Matches(msg, r.keyMap.Rebase.After):
		r.Target = TargetAfter
	case key.Matches(msg, r.keyMap.Rebase.Before):
		r.Target = TargetBefore
	case key.Matches(msg, r.keyMap.Apply):
		source := sourceToFlags[r.Source]
		target := targetToFlags[r.Target]
		return r.context.RunCommand(jj.Rebase(r.From, r.To.ChangeIdShort, source, target), common.RefreshAndSelect(r.From), common.Close)
	case key.Matches(msg, r.keyMap.Cancel):
		return common.Close
	}
	return nil
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) {
	r.To = commit
}

func (r *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		r.keyMap.Rebase.Revision,
		r.keyMap.Rebase.Branch,
		r.keyMap.Rebase.Source,
		r.keyMap.Rebase.Before,
		r.keyMap.Rebase.After,
		r.keyMap.Rebase.Onto,
	}
}

func (r *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{r.ShortHelp()}
}

func (r *Operation) RenderPosition() operations.RenderPosition {
	if r.Target == TargetAfter {
		return operations.RenderPositionBefore
	}
	if r.Target == TargetDestination {
		return operations.RenderPositionBefore
	}
	return operations.RenderPositionAfter
}

func (r *Operation) Render() string {
	var ret string
	if r.Target == TargetDestination {
		ret = "onto"
	}
	if r.Target == TargetAfter {
		ret = "after"
	}
	if r.Target == TargetBefore {
		ret = "before"
	}
	var source string
	if r.Source == SourceBranch {
		source = "branch of "
	}
	if r.Source == SourceDescendants {
		source = "itself and descendants of "
	}
	if r.Source == SourceRevision {
		source = "only "
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		common.DropStyle.Render("<< "+ret+" >>"),
		" ",
		common.DefaultPalette.Dimmed.Render("rebase"),
		" ",
		common.DefaultPalette.Dimmed.Render(source),
		common.DefaultPalette.ChangeId.Render(r.From),
		" ",
		common.DefaultPalette.Dimmed.Render(ret),
		" ",
		common.DefaultPalette.ChangeId.Render(r.To.ChangeIdShort),
	)
}

func (r *Operation) Name() string {
	return "rebase"
}

func NewOperation(context context.AppContext, from string, source Source, target Target) *Operation {
	return &Operation{
		context: context,
		keyMap:  context.KeyMap(),
		From:    from,
		Source:  source,
		Target:  target,
	}
}
