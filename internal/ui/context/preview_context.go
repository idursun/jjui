package context

import (
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/view"
)

type PreviewContext struct {
	*view.BaseView
	CommandRunner
	Current          any
	AtBottom         bool
	WindowPercentage float64
}

func NewPreviewContext(commandRunner CommandRunner) *PreviewContext {
	return &PreviewContext{
		BaseView: &view.BaseView{
			Visible: config.Current.Preview.ShowAtStart,
			Focused: false,
		},
		CommandRunner:    commandRunner,
		AtBottom:         config.Current.Preview.ShowAtBottom,
		WindowPercentage: config.Current.Preview.WidthPercentage,
	}
}

func (p *PreviewContext) Focus() {
	p.Focused = true
}

func (p *PreviewContext) TogglePosition() {
	p.AtBottom = !p.AtBottom
}

func (p *PreviewContext) SetVisible(visible bool) {
	p.Visible = visible
}

func (p *PreviewContext) ToggleVisible() {
	p.Visible = !p.Visible
}

func (p *PreviewContext) Expand() {
	p.WindowPercentage += config.Current.Preview.WidthIncrementPercentage
	if p.WindowPercentage > 95 {
		p.WindowPercentage = 95
	}
}

func (p *PreviewContext) Shrink() {
	p.WindowPercentage -= config.Current.Preview.WidthIncrementPercentage
	if p.WindowPercentage < 10 {
		p.WindowPercentage = 10
	}
}
