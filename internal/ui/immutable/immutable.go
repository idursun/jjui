// Package immutable shows a confirmation dialog offering to retry a failed
// jj command with --ignore-immutable when the failure was caused by jj
// refusing to rewrite an immutable commit.
package immutable

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ common.ImmediateModel = (*Model)(nil)

// CloseMsg dismisses the dialog. It's a dedicated message rather than
// common.CloseViewMsg because closeTopScope (ui.go) always closes an open
// diff view before an m.stacked dialog; since this dialog can appear
// asynchronously while a diff is open, reusing CloseViewMsg would close the
// diff and leave this dialog stranded on screen.
type CloseMsg struct{}

func Close() tea.Msg { return CloseMsg{} }

type Model struct {
	confirmation *confirmation.Model
}

func (m *Model) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeConfirmImmutable,
			Leak:    common.LeakNone,
			Handler: m,
		},
	}
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Apply, intents.Cancel, intents.OptionSelect:
		return m.confirmation.Update(intent), true
	}
	return nil, false
}

func (m *Model) Init() tea.Cmd {
	return m.confirmation.Init()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	return m.confirmation.Update(msg)
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	m.confirmation.Styles.Border = common.DefaultPalette.GetBorder("confirm_immutable", "", "border", false, lipgloss.NormalBorder()).Padding(1)
	v := m.confirmation.View()
	w, h := lipgloss.Size(v)
	pw, ph := box.R.Dx(), box.R.Dy()
	sx := box.R.Min.X + max((pw-w)/2, 0)
	sy := box.R.Min.Y + max((ph-h)/2, 0)
	frame := layout.Rect(sx, sy, w, h)
	dl.AddBackdrop(box.R, render.ZDialogs-1)
	m.confirmation.ViewRect(dl, layout.Box{R: frame})
}

// immutableLine returns the line of err's message that mentions the
// immutable commit, ignoring any warnings jj printed to stderr before it and
// the `Hint:` lines that follow it, none of which fit a single-line
// confirmation option list.
func immutableLine(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "is immutable") {
			return strings.TrimSpace(line)
		}
	}
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	return strings.TrimSpace(text)
}

// New builds a confirmation dialog for err, a jj "commit is immutable" error.
// Selecting "Yes" runs retry (which reissues the failed command with
// --ignore-immutable) and closes the dialog; "No" just closes it.
//
// The dialog can appear without the user having asked for it (it's triggered
// by any background command failing), so unlike undo/redo it defaults to the
// non-destructive option and only ever runs retry once, even if Apply is
// dispatched again (e.g. a held key) before the first run closes the dialog.
func New(err error, retry tea.Cmd) *Model {
	var fired bool
	runRetryOnce := func() tea.Msg {
		if fired || retry == nil {
			return nil
		}
		fired = true
		return retry()
	}

	model := confirmation.New(
		[]string{immutableLine(err), "Retry with --ignore-immutable?"},
		confirmation.WithStyleScope("confirm_immutable"),
		confirmation.WithZIndex(render.ZDialogs),
		confirmation.WithOption("Yes", tea.Sequence(runRetryOnce, Close), key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes"))),
		confirmation.WithOption("No", Close, key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "no"))),
		confirmation.WithSelected(1),
	)
	return &Model{confirmation: model}
}
