package context

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
)

type CustomRunCommand struct {
	CustomCommandBase
	Args []string          `toml:"args"`
	Show config.ShowOption `toml:"show"`
}

func (c CustomRunCommand) IsApplicableTo(item SelectedItem) bool {
	hasChangeIdPlaceholder := slices.ContainsFunc(c.Args, func(s string) bool { return strings.Contains(s, jj.ChangeIdPlaceholder) })
	hasCommitIdPlaceholder := slices.ContainsFunc(c.Args, func(s string) bool { return strings.Contains(s, jj.CommitIdPlaceholder) })
	hasFilePlaceholder := slices.ContainsFunc(c.Args, func(s string) bool { return strings.Contains(s, jj.FilePlaceholder) })
	hasOperationIdPlaceholder := slices.ContainsFunc(c.Args, func(s string) bool { return strings.Contains(s, jj.OperationIdPlaceholder) })
	if !hasChangeIdPlaceholder && !hasFilePlaceholder && !hasOperationIdPlaceholder && !hasCommitIdPlaceholder {
		// If no placeholders are used, the command is applicable to any item
		return true
	}

	switch item.(type) {
	case SelectedRevision:
		return hasChangeIdPlaceholder || hasCommitIdPlaceholder
	case SelectedFile:
		return hasFilePlaceholder
	case SelectedOperation:
		return hasOperationIdPlaceholder
	default:
		return false
	}
}

func (c CustomRunCommand) Description(ctx *MainContext) string {
	args := jj.TemplatedArgs(c.Args, ctx.CreateReplacements())
	return fmt.Sprintf("jj %s", strings.Join(args, " "))
}

func (c CustomRunCommand) Prepare(ctx *MainContext) tea.Cmd {
	replacements := ctx.CreateReplacements()
	switch {
	case c.Show == config.ShowOptionDiff:
		return func() tea.Msg {
			output, _ := ctx.RunCommandImmediate(jj.TemplatedArgs(c.Args, replacements))
			return common.ShowDiffMsg(output)
		}
	case c.Show == config.ShowOptionInteractive:
		return ctx.RunInteractiveCommand(jj.TemplatedArgs(c.Args, replacements), common.Refresh)
	default:
		return ctx.RunCommand(jj.TemplatedArgs(c.Args, replacements), common.Refresh)
	}
}
