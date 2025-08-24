package set_selected_file

import (
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions/update_preview"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

func Call(ctx *appContext.MainContext, revision *jj.Commit, filePath string) {
	ctx.SelectedItem = appContext.SelectedFile{
		ChangeId: revision.ChangeId,
		CommitId: revision.CommitId,
		File:     filePath,
	}
	update_preview.Call(ctx)
}
