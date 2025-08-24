package set_selected_revision

import (
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions/update_preview"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

func Call(ctx *appContext.MainContext, commit *jj.Commit) {
	ctx.SelectedItem = appContext.SelectedRevision{
		ChangeId: commit.ChangeId,
		CommitId: commit.CommitId,
	}
	update_preview.Call(ctx)
}
