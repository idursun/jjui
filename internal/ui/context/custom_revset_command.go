package context

import (
	tea "github.com/charmbracelet/bubbletea"
)

type CustomRevsetCommand struct {
	CustomCommandBase
	Revset string `toml:"revset"`
}

func (c CustomRevsetCommand) Description(ctx *MainContext) string {
	//if item, ok := ctx.SelectedItem.(SelectedRevision); ok {
	//	rendered := strings.ReplaceAll(c.Revset, jj.ChangeIdPlaceholder, item.ChangeId)
	//	rendered = strings.ReplaceAll(rendered, jj.CommitIdPlaceholder, item.CommitId)
	//	rendered = strings.ReplaceAll(rendered, jj.RevsetPlaceholder, ctx.Revset.CurrentRevset)
	//	return fmt.Sprintf("change revset to %s", rendered)
	//}
	return ""
}

func (c CustomRevsetCommand) IsApplicableTo(item SelectedItem) bool {
	_, ok := item.(SelectedRevision)
	return ok
}

func (c CustomRevsetCommand) Prepare(ctx *MainContext) tea.Cmd {
	//if item, ok := ctx.SelectedItem.(SelectedRevision); ok {
	//	rendered := strings.ReplaceAll(c.Revset, jj.ChangeIdPlaceholder, item.ChangeId)
	//	rendered = strings.ReplaceAll(rendered, jj.CommitIdPlaceholder, item.CommitId)
	//	rendered = strings.ReplaceAll(rendered, jj.RevsetPlaceholder, ctx.Revset.CurrentRevset)
	//	return common.UpdateRevSet(rendered)
	//}
	return nil
}
