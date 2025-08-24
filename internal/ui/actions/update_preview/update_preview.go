package update_preview

import (
	"sync/atomic"
	"time"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

var debounceTag atomic.Uint64

const debounceTime = 100 * time.Millisecond

func Call(ctx *appContext.MainContext) {
	currentTag := debounceTag.Add(1)

	go func() {
		time.Sleep(debounceTime)

		if currentTag != debounceTag.Load() {
			return
		}

		switch msg := ctx.SelectedItem.(type) {
		case appContext.SelectedFile:
			replacements := map[string]string{
				jj.RevsetPlaceholder:   ctx.CurrentRevset,
				jj.ChangeIdPlaceholder: msg.ChangeId,
				jj.CommitIdPlaceholder: msg.CommitId,
				jj.FilePlaceholder:     msg.File,
			}

			if currentTag != debounceTag.Load() {
				return
			}
			output, _ := ctx.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.FileCommand, replacements))
			ctx.App.Send(common.UpdatePreviewContent(string(output)))

		case appContext.SelectedRevision:
			replacements := map[string]string{
				jj.RevsetPlaceholder:   ctx.CurrentRevset,
				jj.ChangeIdPlaceholder: msg.ChangeId,
				jj.CommitIdPlaceholder: msg.CommitId,
			}

			if currentTag != debounceTag.Load() {
				return
			}

			output, _ := ctx.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.RevisionCommand, replacements))
			ctx.App.Send(common.UpdatePreviewContent(string(output)))

		case appContext.SelectedOperation:
			replacements := map[string]string{
				jj.RevsetPlaceholder:      ctx.CurrentRevset,
				jj.OperationIdPlaceholder: msg.OperationId,
			}

			if currentTag != debounceTag.Load() {
				return
			}
			output, _ := ctx.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.OplogCommand, replacements))
			ctx.App.Send(common.UpdatePreviewContent(string(output)))
		}
	}()
}
