package customcommands

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"strings"
)

type CustomCommand struct {
	name           string
	key            key.Binding
	args           []string
	show           config.ShowOption
	hasChangeId    bool
	hasFile        bool
	hasOperationId bool
}

type InvokableCustomCommand struct {
	args []string
	show config.ShowOption
}

const (
	ChangeIdPlaceholder    = "$change_id"
	FilePlaceholder        = "$file"
	OperationIdPlaceholder = "$operation_id"
)

func NewCustomCommand(name string, definition config.CustomCommandDefinition) CustomCommand {
	var hasChangeId, hasFile, hasOperationId bool
	for _, arg := range definition.Args {
		if strings.Contains(arg, ChangeIdPlaceholder) {
			hasChangeId = true
		}
		if strings.Contains(arg, FilePlaceholder) {
			hasFile = true
		}
		if strings.Contains(arg, OperationIdPlaceholder) {
			hasOperationId = true
		}
	}
	return CustomCommand{
		name:           name,
		key:            key.NewBinding(key.WithKeys(definition.Key...)),
		args:           definition.Args,
		show:           definition.Show,
		hasChangeId:    hasChangeId,
		hasFile:        hasFile,
		hasOperationId: hasOperationId,
	}
}

func (cc CustomCommand) Prepare(ctx context.AppContext) InvokableCustomCommand {
	replacements := map[string]string{
		//"$revset": m.revsetModel.Value,
	}
	switch selectedItem := ctx.SelectedItem().(type) {
	case context.SelectedRevision:
		replacements[ChangeIdPlaceholder] = selectedItem.ChangeId
	case context.SelectedFile:
		replacements[ChangeIdPlaceholder] = selectedItem.ChangeId
		replacements[FilePlaceholder] = selectedItem.File
	case context.SelectedOperation:
		replacements[OperationIdPlaceholder] = selectedItem.OperationId
	}
	var args []string
	for _, arg := range cc.args {
		for k, v := range replacements {
			arg = strings.ReplaceAll(arg, k, v)
		}
		args = append(args, arg)
	}

	return InvokableCustomCommand{
		args: args,
		show: cc.show,
	}
}

func (cc InvokableCustomCommand) Invoke(ctx context.AppContext) tea.Cmd {
	switch cc.show {
	case "":
		return ctx.RunCommand(jj.Args(cc.args...), common.Refresh)
	case config.ShowOptionDiff:
		output, _ := ctx.RunCommandImmediate(jj.Args(cc.args...))
		return func() tea.Msg {
			return common.ShowDiffMsg(output)
		}
	case config.ShowOptionInteractive:
		return ctx.RunInteractiveCommand(jj.Args(cc.args...), common.Refresh)
	}
	return nil
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
