package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/theme"
)

// colorSchemePollInterval is how often to poll the terminal for its
// current light/dark mode.
var colorSchemePollInterval = time.Second

type colorSchemePollTickMsg struct{}

func (m *Model) handleAppearanceMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case common.ReloadThemeMsg:
		return m.reloadActiveTheme(), true
	case tea.ModeReportMsg:
		if msg.Mode != ansi.ModeLightDark {
			return nil, false
		}
		// reply from initial ansi.RequestModeLightDark check
		if !msg.Value.IsNotRecognized() {
			m.mode2031Supported = true
		}
		return nil, true
	case uv.UnknownOscEvent:
		if reply, ok := parseTerminalPaletteReply(msg); ok {
			if m.context.TerminalPalette == nil {
				m.context.TerminalPalette = make(map[int]string)
			}
			previous := m.context.TerminalPalette[reply.Slot]
			m.context.TerminalPalette[reply.Slot] = reply.Hex
			log.Printf("terminal palette reply: slot=%d color=%s raw=%q", reply.Slot, reply.Hex, reply.Raw)
			if m.context.ThemeBackgroundBlend > 0 && previous != reply.Hex {
				return m.reloadActiveTheme(), true
			}
			return nil, true
		}
		return nil, true
	case tea.ResumeMsg:
		// common.Suspend disables mode 2031 on the way out, reenable it on resume.
		// Also re-query the background color in case the theme changed while suspended.
		return tea.Batch(
			tea.Raw(ansi.SetModeLightDark),
			requestTerminalAppearance(m.context.ThemeBackgroundBlend),
		), true
	case uv.DarkColorSchemeEvent:
		return m.applyColorScheme(true), true
	case uv.LightColorSchemeEvent:
		return m.applyColorScheme(false), true
	case tea.BackgroundColorMsg:
		if msg.Color == nil {
			return nil, true
		}
		return m.applyTerminalBackground(msg.String(), msg.IsDark()), true
	case colorSchemePollTickMsg:
		if m.mode2031Supported {
			return nil, true
		}
		return tea.Batch(tea.RequestBackgroundColor, scheduleColorSchemePoll()), true
	}
	return nil, false
}

func initTerminalAppearance(backgroundBlend float64) tea.Cmd {
	return tea.Batch(
		// Enable mode 2031 push notifications and probe whether the terminal
		// supports it. If supported, the mode request returns a tea.ModeReportMsg
		// with ansi.ModeLightDark, and uv.Light/DarkColorSchemeEvent are delivered
		// on OS theme change.
		tea.Raw(ansi.SetModeLightDark),
		tea.Raw(ansi.RequestModeLightDark),
		requestTerminalAppearance(backgroundBlend),
		// Start OSC 11 polling as a baseline for light/dark mode detection
		scheduleColorSchemePoll(),
	)
}

// scheduleColorSchemePoll returns a Cmd that fires a colorSchemePollTickMsg
// after colorSchemePollInterval.
func scheduleColorSchemePoll() tea.Cmd {
	return tea.Tick(colorSchemePollInterval, func(time.Time) tea.Msg {
		return colorSchemePollTickMsg{}
	})
}

func (m *Model) changeTheme(intent intents.ChangeTheme) tea.Cmd {
	name := strings.TrimSpace(intent.Name)
	if name == "" {
		err := fmt.Errorf("theme name is required")
		return intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err})
	}

	oldDark := config.Current.UI.Theme.Dark
	oldLight := config.Current.UI.Theme.Light

	if m.context.TerminalHasDarkBackground {
		config.Current.UI.Theme.Dark = name
	} else {
		config.Current.UI.Theme.Light = name
	}

	resolved, err := config.ResolveTheme(m.context.TerminalHasDarkBackground, m.context.JJConfig.GetApplicableColors())
	if err != nil {
		config.Current.UI.Theme.Dark = oldDark
		config.Current.UI.Theme.Light = oldLight
		return intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err})
	}

	cmd := m.applyResolvedTheme(resolved)
	log.Printf("theme changed to %q", name)
	return cmd
}

// applyColorScheme reloads the palette when the terminal's color scheme
// changes (that is, the OS switched between light and dark mode).
func (m *Model) applyColorScheme(isDark bool) tea.Cmd {
	if isDark == m.context.TerminalHasDarkBackground {
		return nil
	}
	scheme := "light"
	if isDark {
		scheme = "dark"
	}
	log.Printf("color scheme changed to %s", scheme)
	m.context.TerminalHasDarkBackground = isDark
	cmd := m.reloadActiveTheme()
	return tea.Batch(cmd, requestTerminalAppearance(m.context.ThemeBackgroundBlend))
}

func (m *Model) applyTerminalBackground(background string, isDark bool) tea.Cmd {
	hadBackground := m.context.TerminalBackground != ""
	backgroundChanged := background != "" && background != m.context.TerminalBackground
	if background != "" {
		m.context.TerminalBackground = background
	}

	if isDark != m.context.TerminalHasDarkBackground {
		return m.applyColorScheme(isDark)
	}
	if backgroundChanged && m.context.ThemeBackgroundBlend > 0 {
		if hadBackground {
			return tea.Batch(
				m.reloadActiveTheme(),
				requestTerminalPaletteIfNeeded(m.context.ThemeBackgroundBlend),
			)
		}
		return m.reloadActiveTheme()
	}
	return nil
}

func (m *Model) reloadActiveTheme() tea.Cmd {
	resolvedTheme, err := config.ResolveTheme(m.context.TerminalHasDarkBackground, m.context.JJConfig.GetApplicableColors())
	if err != nil {
		log.Printf("failed to resolve theme: %v", err)
		return nil
	}
	return m.applyResolvedTheme(resolvedTheme)
}

func (m *Model) applyResolvedTheme(resolvedTheme config.ResolvedTheme) tea.Cmd {
	blendEnabled := m.context.ThemeBackgroundBlend == 0 && resolvedTheme.BackgroundBlend > 0
	m.context.ThemeBackgroundBlend = resolvedTheme.BackgroundBlend
	theme.DefaultPalette.Update(resolvedTheme.Colors)
	theme.DefaultPalette.ConfigureBackgroundBlend(
		resolvedTheme.BackgroundBlend,
		m.context.TerminalBackground,
		m.context.TerminalPalette,
	)
	cmd := func() tea.Msg { return common.ThemeChangedMsg{} }
	if blendEnabled {
		return tea.Batch(cmd, requestTerminalPalette())
	}
	return cmd
}
