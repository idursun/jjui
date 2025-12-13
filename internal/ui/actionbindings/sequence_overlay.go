package actionbindings

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/bindings"
	"github.com/idursun/jjui/internal/ui/actiondispatch"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type SequenceEntry struct {
	Name      string
	Remaining []string
}

type SequenceCandidate struct {
	Binding bindings.KeyBinding
	Seq     []key.Binding
	Index   int
}

type SequenceTimeoutMsg struct {
	Started time.Time
}

type SequenceResult struct {
	Cmd     tea.Cmd
	Handled bool
	Active  bool
}

type SequenceOverlay struct {
	*common.ViewNode
	ctx           *context.MainContext
	state         func() map[string]any
	prefix        string
	items         []SequenceEntry
	shortcutStyle lipgloss.Style
	matchedStyle  lipgloss.Style
	textStyle     lipgloss.Style
	candidates    []SequenceCandidate
	started       time.Time
	typed         []string
}

const sequenceTimeout = 4 * time.Second

func NewSequenceOverlay(ctx *context.MainContext, state func() map[string]any) *SequenceOverlay {
	return &SequenceOverlay{
		ViewNode:      common.NewViewNode(0, 0),
		ctx:           ctx,
		state:         state,
		shortcutStyle: common.DefaultPalette.Get("shortcut"),
		matchedStyle:  common.DefaultPalette.Get("matched"),
		textStyle:     common.DefaultPalette.Get("text"),
	}
}

func (s *SequenceOverlay) Init() tea.Cmd {
	return nil
}

func (s *SequenceOverlay) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(SequenceTimeoutMsg); ok {
		res := s.handleTimeout(msg)
		return res.Cmd
	}
	return nil
}

func BindingKeyString(b key.Binding) string {
	if len(b.Keys()) > 0 {
		return b.Keys()[0]
	}
	if h := b.Help(); h.Key != "" {
		return h.Key
	}
	return ""
}

func (s *SequenceOverlay) Set(prefix []string, entries []SequenceEntry) {
	s.prefix = strings.Join(prefix, " ")
	s.items = entries
}

func (s *SequenceOverlay) Active() bool {
	return len(s.candidates) > 0
}

func ToKeyBindings(keys []string) []key.Binding {
	bindings := make([]key.Binding, 0, len(keys))
	for _, k := range keys {
		bindings = append(bindings, key.NewBinding(
			key.WithKeys(k),
			key.WithHelp(k, k),
		))
	}
	return bindings
}

func (s *SequenceOverlay) SetFromCandidates(typed []string, candidates []SequenceCandidate) {
	entries := make([]SequenceEntry, 0, len(candidates))
	for _, cand := range candidates {
		var remaining []string
		for _, b := range cand.Seq[cand.Index:] {
			remaining = append(remaining, BindingKeyString(b))
		}
		entries = append(entries, SequenceEntry{
			Name:      cand.Binding.Action,
			Remaining: remaining,
		})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	s.Set(typed, entries)
}

func (s *SequenceOverlay) HandleKey(msg tea.KeyMsg) SequenceResult {
	now := time.Now()
	s.expire(now)

	if len(s.candidates) > 0 {
		next, res := s.advance(msg)
		if res.Cmd != nil || !res.Active {
			return res
		}
		if len(next) > 0 {
			s.candidates = next
			s.started = now
			s.typed = append(s.typed, BindingKeyString(next[0].Seq[next[0].Index-1]))
			s.SetFromCandidates(s.typed, s.candidates)
			return SequenceResult{
				Cmd:     s.scheduleTimeout(now),
				Handled: true,
				Active:  true,
			}
		}
		s.reset()
		return SequenceResult{Handled: false, Active: false}
	}

	return s.maybeStart(msg, now)
}

func (s *SequenceOverlay) HandleTimeout(msg SequenceTimeoutMsg) SequenceResult {
	return s.handleTimeout(msg)
}

func (s *SequenceOverlay) View() string {
	var view strings.Builder
	for i, it := range s.items {
		view.WriteString(s.matchedStyle.Render(s.prefix))
		if len(it.Remaining) == 0 {
			continue
		}
		for _, r := range it.Remaining {
			view.WriteString(" → ")
			view.WriteString(s.shortcutStyle.Render(r))
		}
		view.WriteString(" ")
		view.WriteString(it.Name)
		if i < len(s.items)-1 {
			view.WriteString("\n")
		}
	}
	w := s.Parent.Frame.Dx()

	content := view.String()
	style := lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1).
		Border(lipgloss.RoundedBorder()).
		Width(w - 2)
	content = style.Render(content)

	h := lipgloss.Height(content)
	sy := s.Parent.Frame.Dy() - h - 1

	s.SetFrame(cellbuf.Rect(0, sy, w, h))
	return content
}

func (s *SequenceOverlay) advance(msg tea.KeyMsg) ([]SequenceCandidate, SequenceResult) {
	matched := false
	var next []SequenceCandidate
	state := s.currentState()
	for _, cand := range s.candidates {
		if cand.Binding.Disabled {
			continue
		}
		if cand.Binding.When != "" && !cand.Binding.Condition.Eval(state) {
			continue
		}
		if key.Matches(msg, cand.Seq[cand.Index]) {
			matched = true
			if cand.Index+1 == len(cand.Seq) {
				cmd := s.ctx.ActionCmd(cand.Binding.Action)
				if cmd == nil {
					cmd = actiondispatch.Cmd(cand.Binding.Action, s.ctx)
				}
				s.reset()
				if cmd == nil {
					return nil, SequenceResult{Handled: true, Active: false}
				}
				return nil, SequenceResult{Cmd: cmd, Handled: true, Active: false}
			}
			cand.Index++
			next = append(next, cand)
		}
	}
	if matched {
		return next, SequenceResult{Handled: true, Active: s.Active()}
	}
	return nil, SequenceResult{Handled: false, Active: s.Active()}
}

func (s *SequenceOverlay) maybeStart(msg tea.KeyMsg, now time.Time) SequenceResult {
	state := s.currentState()
	var starters []SequenceCandidate
	for _, binding := range s.ctx.KeyBindings {
		if len(binding.KeySequence) == 0 || binding.Disabled {
			continue
		}
		if binding.When != "" && !binding.Condition.Eval(state) {
			continue
		}
		seq := ToKeyBindings(binding.KeySequence)
		if len(seq) == 0 {
			continue
		}
		if key.Matches(msg, seq[0]) {
			if len(seq) == 1 {
				cmd := s.ctx.ActionCmd(binding.Action)
				if cmd == nil {
					cmd = actiondispatch.Cmd(binding.Action, s.ctx)
				}
				if cmd == nil {
					return SequenceResult{Handled: true, Active: false}
				}
				return SequenceResult{Cmd: cmd, Handled: true, Active: false}
			}
			starters = append(starters, SequenceCandidate{
				Binding: binding,
				Seq:     seq,
				Index:   1,
			})
		}
	}

	if len(starters) == 0 {
		return SequenceResult{Handled: false, Active: false}
	}

	s.candidates = starters
	s.started = now
	s.typed = []string{BindingKeyString(starters[0].Seq[0])}
	s.SetFromCandidates(s.typed, s.candidates)

	return SequenceResult{
		Cmd:     s.scheduleTimeout(now),
		Handled: true,
		Active:  true,
	}
}

func (s *SequenceOverlay) scheduleTimeout(start time.Time) tea.Cmd {
	if start.IsZero() {
		return nil
	}
	return tea.Tick(sequenceTimeout, func(time.Time) tea.Msg {
		return SequenceTimeoutMsg{Started: start}
	})
}

func (s *SequenceOverlay) handleTimeout(msg SequenceTimeoutMsg) SequenceResult {
	if s.started.IsZero() || !msg.Started.Equal(s.started) {
		return SequenceResult{Handled: false, Active: s.Active()}
	}
	s.reset()
	return SequenceResult{Handled: true, Active: false}
}

func (s *SequenceOverlay) expire(now time.Time) {
	if len(s.candidates) == 0 || s.started.IsZero() {
		return
	}
	if now.Sub(s.started) > sequenceTimeout {
		s.reset()
	}
}

func (s *SequenceOverlay) currentState() map[string]any {
	if s.state == nil {
		return nil
	}
	return s.state()
}

func (s *SequenceOverlay) reset() {
	s.candidates = nil
	s.started = time.Time{}
	s.typed = nil
	s.Set(nil, nil)
}
