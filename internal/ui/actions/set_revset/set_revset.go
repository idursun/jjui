package set_revset

import (
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

func Call(ctx *appContext.MainContext, revset string) {
	ctx.CurrentRevset = revset
	go func() {
		ctx.App.Send(common.Refresh())
	}()
}
