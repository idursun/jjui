package customcommands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type CustomCommand struct {
	Name           string
	Key            key.Binding
	Args           []string
	Show           config.ShowOption
	hasChangeId    bool
	hasFile        bool
	hasOperationId bool
}

type InvokableCustomCommand struct {
	args []string
	desc string
	Cmd  tea.Cmd
}

func newCustomCommand(name string, definition config.CustomCommandDefinition) CustomCommand {
	var hasChangeId, hasFile, hasOperationId bool
	for _, arg := range definition.Args {
		if strings.Contains(arg, config.ChangeIdPlaceholder) {
			hasChangeId = true
		}
		if strings.Contains(arg, config.FilePlaceholder) {
			hasFile = true
		}
		if strings.Contains(arg, config.OperationIdPlaceholder) {
			hasOperationId = true
		}
	}

	binding := key.NewBinding(key.WithKeys(definition.Key...), key.WithHelp(config.JoinKeys(definition.Key), name))
	return CustomCommand{
		Name:           name,
		Key:            binding,
		Args:           definition.Args,
		Show:           definition.Show,
		hasChangeId:    hasChangeId,
		hasFile:        hasFile,
		hasOperationId: hasOperationId,
	}
}

func (cc CustomCommand) Prepare(ctx *context.MainContext) InvokableCustomCommand {
	replacements := make(map[string]string)

	switch selectedItem := ctx.SelectedItem.(type) {
	case context.SelectedRevision:
		replacements[config.ChangeIdPlaceholder] = selectedItem.ChangeId
	case context.SelectedFile:
		replacements[config.ChangeIdPlaceholder] = selectedItem.ChangeId
		replacements[config.FilePlaceholder] = selectedItem.File
	case context.SelectedOperation:
		replacements[config.OperationIdPlaceholder] = selectedItem.OperationId
	}

	return InvokableCustomCommand{
		args: jj.TemplatedArgs(cc.Args, replacements),
		show: cc.Show,
	}
}

func (cc InvokableCustomCommand) Invoke(ctx context.CommandRunner) tea.Cmd {
	switch cc.show {
	case "":
		return ctx.RunCommand(jj.Args(cc.args...), common.Refresh)
	case config.ShowOptionDiff:
		return InvokableCustomCommand{
			desc: fmt.Sprintf("jj %s", strings.Join(args, " ")),
			Cmd: func() tea.Msg {
				output, _ := ctx.RunCommandImmediate(jj.Args(args...))
				return common.ShowDiffMsg(output)
			},
		}
	case config.ShowOptionInteractive:
		return InvokableCustomCommand{
			desc: fmt.Sprintf("jj %s", strings.Join(args, " ")),
			Cmd:  ctx.RunInteractiveCommand(jj.Args(args...), common.Refresh),
		}
	default:
		return InvokableCustomCommand{
			desc: fmt.Sprintf("jj %s", strings.Join(args, " ")),
			Cmd:  ctx.RunCommand(jj.Args(args...), common.Refresh),
		}
	}
}

func (cc CustomCommand) applicableTo(selectedItem context.SelectedItem) bool {
	if !cc.hasOperationId && !cc.hasFile && !cc.hasChangeId {
		return true
	}
	switch selectedItem.(type) {
	case context.SelectedRevision:
		return cc.hasChangeId
	case context.SelectedFile:
		return cc.hasFile
	case context.SelectedOperation:
		return cc.hasOperationId
	default:
		return false
	}
}
