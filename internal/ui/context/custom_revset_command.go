package context

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
)

type CustomRevsetCommand struct {
	CustomCommandBase
	Revset string `toml:"revset"`
}

func (c CustomRevsetCommand) Description(ctx *MainContext) string {
	replacements := ctx.CreateReplacements()
	rendered := c.Revset
	for k, v := range replacements {
		rendered = strings.ReplaceAll(rendered, k, v)
	}
	return rendered
}

func (c CustomRevsetCommand) IsApplicableTo(ctx *MainContext) bool {
	// FIXME: This should return true only if the active element is a revision
	return true
}

func (c CustomRevsetCommand) Prepare(ctx *MainContext) tea.Cmd {
	replacements := ctx.CreateReplacements()
	rendered := c.Revset
	for k, v := range replacements {
		rendered = strings.ReplaceAll(rendered, k, v)
	}
	return common.UpdateRevSet(rendered)
}
