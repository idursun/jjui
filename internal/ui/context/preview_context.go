package context

import (
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context/models"
)

const DebounceTime = 50 * time.Millisecond

type PreviewContext struct {
	CommandRunner
	UI
	revsetContext *RevsetContext
	Content       string
	LineCount     int
	tag           atomic.Int64
}

func NewPreviewContext(runner CommandRunner, ui UI, revsetCtx *RevsetContext) *PreviewContext {
	return &PreviewContext{
		CommandRunner: runner,
		UI:            ui,
		revsetContext: revsetCtx,
		Content:       "",
		tag:           atomic.Int64{},
	}
}

func (p *PreviewContext) LoadRevision(revision *models.RevisionItem) {
	p.debounceCall(func() {
		replacements := map[string]string{
			jj.RevsetPlaceholder:   p.revsetContext.CurrentRevset,
			jj.ChangeIdPlaceholder: revision.Commit.ChangeId,
			jj.CommitIdPlaceholder: revision.Commit.CommitId,
		}
		output, _ := p.RunCommandImmediate(jj.TemplatedArgs(config.Current.Preview.RevisionCommand, replacements))
		p.Content = string(output)
		p.LineCount = lipgloss.Height(p.Content)
		p.UI.Update()
	})
}

func (p *PreviewContext) LoadRevisionFile(file *models.RevisionFile) {

}

func (p *PreviewContext) LoadEvolog(evolog *models.EvologItem) {

}

func (p *PreviewContext) debounceCall(f func()) {
	current := p.tag.Add(1)
	go func() {
		time.Sleep(DebounceTime)
		if current != p.tag.Load() {
			return
		}
		f()
	}()
}
