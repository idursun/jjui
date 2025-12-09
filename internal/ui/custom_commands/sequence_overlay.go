package customcommands

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type SequenceEntry struct {
	Name      string
	Remaining []string
}

type SequenceCandidate struct {
	Command context.CustomCommand
	Seq     []key.Binding
	Index   int
}

type SequenceTimeoutMsg struct {
	Started time.Time
}

// SequenceResult captures the outcome of a key/timeout handled by the overlay.
type SequenceResult struct {
	Cmd     tea.Cmd
	Handled bool
	Active  bool
}

// SequenceOverlay renders a lightweight hint panel for in-flight key sequences.
type SequenceOverlay struct {
	*common.ViewNode
	ctx           *context.MainContext
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

func NewSequenceOverlay(ctx *context.MainContext) *SequenceOverlay {
	return &SequenceOverlay{
		ViewNode:      common.NewViewNode(0, 0),
		ctx:           ctx,
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

func (s *SequenceOverlay) SetFromCandidates(typed []string, candidates []SequenceCandidate) {
	entries := make([]SequenceEntry, 0, len(candidates))
	for _, cand := range candidates {
		var remaining []string
		for _, b := range cand.Seq[cand.Index:] {
			remaining = append(remaining, BindingKeyString(b))
		}
		label := cand.Command.Description(s.ctx)
		if lc, ok := cand.Command.(context.LabeledCommand); ok {
			label = lc.Label()
		}
		entries = append(entries, SequenceEntry{
			Name:      label,
			Remaining: remaining,
		})
	}

	// Ensure deterministic order for display
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
		// No continuation matched; reset and let other handlers process.
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
	for _, cand := range s.candidates {
		if !cand.Command.IsApplicableTo(s.ctx.SelectedItem) {
			continue
		}
		if key.Matches(msg, cand.Seq[cand.Index]) {
			matched = true
			if cand.Index+1 == len(cand.Seq) {
				cmd := cand.Command.Prepare(s.ctx)
				s.reset()
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
	var starters []SequenceCandidate
	for _, command := range SortedCustomCommands(s.ctx) {
		seq := command.Sequence()
		if len(seq) == 0 || !command.IsApplicableTo(s.ctx.SelectedItem) {
			continue
		}
		if key.Matches(msg, seq[0]) {
			if len(seq) == 1 {
				return SequenceResult{Cmd: command.Prepare(s.ctx), Handled: true, Active: false}
			}
			starters = append(starters, SequenceCandidate{
				Command: command,
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

func (s *SequenceOverlay) reset() {
	s.candidates = nil
	s.started = time.Time{}
	s.typed = nil
	s.Set(nil, nil)
}
