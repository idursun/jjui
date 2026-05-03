package absorb

import (
	"log"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var (
	_ operations.Operation = (*Operation)(nil)
	_ common.Focusable     = (*Operation)(nil)
	_ common.ScopeProvider = (*Operation)(nil)
)

type Operation struct {
	context  *context.MainContext
	source   *jj.Commit
	current  *jj.Commit
	defaults map[string]bool
	targets  map[string]bool
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeAbsorb,
			Leak:    common.LeakAll,
			Handler: o,
		},
	}
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	if intent, ok := msg.(intents.Intent); ok {
		cmd, _ := o.HandleIntent(intent)
		return cmd
	}
	return nil
}

func (o *Operation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (o *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	o.current = commit
	return nil
}

func (o *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump(), true
	case intents.AbsorbToggleSelect:
		if o.current == nil {
			return nil, true
		}
		changeId := o.current.GetChangeId()
		if changeId == "" || changeId == o.source.GetChangeId() {
			return nil, true
		}
		if o.targets[changeId] {
			delete(o.targets, changeId)
		} else {
			o.targets[changeId] = true
		}
		return nil, true
	case intents.Apply:
		var into []string
		if !equalSets(o.targets, o.defaults) {
			if len(o.targets) == 0 {
				return common.Close, true
			}
			into = make([]string, 0, len(o.targets))
			for changeId := range o.targets {
				into = append(into, changeId)
			}
			slices.Sort(into)
		}
		return o.context.RunCommand(
			jj.Absorb(o.source.GetChangeId(), into),
			common.RefreshAndSelect(o.source.GetChangeId()),
			common.CloseApplied,
		), true
	case intents.Cancel:
		return common.Close, true
	}
	return nil, false
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}
	sourceMarkerStyle := common.DefaultPalette.Get("absorb source_marker")
	targetMarkerStyle := common.DefaultPalette.Get("absorb target_marker")
	dimmedStyle := common.DefaultPalette.Get("absorb dimmed")

	changeId := commit.GetChangeId()
	if changeId == o.source.GetChangeId() {
		return sourceMarkerStyle.Render("<< absorb >>")
	}
	if o.targets[changeId] {
		return targetMarkerStyle.Render("<< into >>")
	}
	if o.defaults[changeId] {
		return dimmedStyle.Render("<< default >>")
	}
	return ""
}

func (o *Operation) Name() string {
	return "absorb"
}

func NewOperation(ctx *context.MainContext, source *jj.Commit) *Operation {
	defaultIds := loadDefaultTargets(ctx, source)
	defaults := make(map[string]bool, len(defaultIds))
	targets := make(map[string]bool, len(defaultIds))
	for _, changeId := range defaultIds {
		defaults[changeId] = true
		targets[changeId] = true
	}
	return &Operation{
		context:  ctx,
		source:   source,
		defaults: defaults,
		targets:  targets,
	}
}

func equalSets(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func loadDefaultTargets(ctx *context.MainContext, source *jj.Commit) []string {
	output, err := ctx.RunCommandImmediate(jj.AbsorbDefaultTargets(source.GetChangeId()))
	if err != nil {
		log.Println("Failed to load default absorb targets for", source.GetChangeId(), err)
		return nil
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil
	}
	ids := strings.Split(trimmed, "\n")
	out := make([]string, 0, len(ids))
	sourceId := source.GetChangeId()
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || id == sourceId {
			continue
		}
		out = append(out, id)
	}
	return out
}
