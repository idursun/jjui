package context

import (
	"bytes"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/context/models"
	"github.com/idursun/jjui/internal/ui/helpers"
)

type EvologContext struct {
	CommandRunner
	*helpers.List[*models.EvologItem]
}

func NewEvologContext(runner CommandRunner) *EvologContext {
	return &EvologContext{
		CommandRunner: runner,
		List:          helpers.NewList[*models.EvologItem](),
	}
}

func (e *EvologContext) Load(revision *models.RevisionItem) {
	output, _ := e.RunCommandImmediate(jj.Evolog(revision.Commit.GetChangeId()))
	rows := parser.ParseRows(bytes.NewReader(output))
	e.Items = make([]*models.EvologItem, 0)
	for _, row := range rows {
		e.Items = append(e.Items, &models.EvologItem{Row: &row})
	}
	e.SetCursor(0)
}
