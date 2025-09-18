package exec_prompt

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
	"github.com/idursun/jjui/test"
)

func TestRevSearch(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := context.NewAppContext(commandRunner, "")
	histories := ctx.Histories.GetHistory("exec_sh", true)
	histories.Append("jj log -r n")
	histories.Append("jj commit")
	histories.Append("jj abandon")
	histories.Append("exit")
	model := NewExecPrompt(ctx, common.ExecShell)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	tm := teatest.NewTestModel(t, model)
	tm.Type("jj ab")
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("jj ab"))
	})
}
