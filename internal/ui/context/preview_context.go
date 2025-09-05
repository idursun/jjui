package context

import (
	"github.com/idursun/jjui/internal/config"
)

type PreviewContext struct {
	CommandRunner
	Current          any
	AtBottom         bool
	WindowPercentage float64
}

func NewPreviewContext(commandRunner CommandRunner) *PreviewContext {
	return &PreviewContext{
		CommandRunner:    commandRunner,
		WindowPercentage: config.Current.Preview.WidthPercentage,
	}
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
