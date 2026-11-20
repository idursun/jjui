package tag

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestSetTagOperation_Apply(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.TagSet("revision", "v1.0-rc"))
	defer commandRunner.Verify()

	op := NewSetTagOperation(test.NewTestContext(commandRunner), "revision", "")
	test.SimulateModel(op, op.Init())
	test.SimulateModel(op, test.Type("v1.0 rc"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func TestSetTagOperation_EmptyApplyRunsNothing(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	op := NewSetTagOperation(test.NewTestContext(commandRunner), "revision", "")
	test.SimulateModel(op, op.Init())
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func TestSetTagOperation_ShowsWholePlaceholder(t *testing.T) {
	op := NewSetTagOperation(test.NewTestContext(test.NewTestCommandRunner(t)), "revision", "")
	assert.Contains(t, ansi.Strip(op.viewContent()), "tag name")
}
