package flash

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
)

const commandMarkWidth = 3

type CardRenderer struct{}

func NewCardRenderer() CardRenderer {
	return CardRenderer{}
}

func (r CardRenderer) RenderMessage(command, text string, commandErr error, maxWidth int) string {
	return r.renderCard(command, text, commandErr, maxWidth, true, true)
}

func (r CardRenderer) RenderHistoryEntry(entry commandHistoryEntry, maxWidth int, selected bool) string {
	return r.renderCardOnSurface(entry.Command, entry.Text, entry.Err, maxWidth, selected)
}

func (r CardRenderer) RenderRunningCommand(command, indicator string, maxWidth int) string {
	if command == "" {
		return ""
	}
	return r.wrapCard(
		r.renderCommandLine(command, nil, true, indicator),
		maxWidth,
	)
}

func (r CardRenderer) renderCard(command, text string, commandErr error, maxWidth int, showBody bool, highlight bool) string {
	successStyle := theme.DefaultPalette.Get("flash", "", "success", false)
	errorStyle := theme.DefaultPalette.Get("flash", "", "error", false)

	statusStyle := successStyle
	if commandErr != nil {
		statusStyle = errorStyle
	}

	var parts []string
	if command != "" {
		parts = append(parts, r.renderCommandLine(command, commandErr, false, ""))
	}
	if showBody {
		bodyText := text
		if commandErr != nil {
			bodyText = commandErr.Error()
		}
		if bodyText != "" {
			bodyText = render.ReplayTerminalOutput(bodyText)
			bodyStyle := lipgloss.NewStyle()
			if highlight {
				bodyStyle = statusStyle
			}
			parts = append(parts, bodyStyle.Render(bodyText))
		}
	}

	borderStyle := lipgloss.NewStyle()
	if highlight {
		borderStyle = statusStyle
	}

	return r.wrapCard(strings.Join(parts, "\n"), maxWidth, borderStyle)
}

func (r CardRenderer) renderCardOnSurface(command, text string, commandErr error, maxWidth int, showBody bool) string {
	successStyle := theme.DefaultPalette.Get("flash", "", "success", false)
	errorStyle := theme.DefaultPalette.Get("flash", "", "error", false)

	statusStyle := successStyle
	if commandErr != nil {
		statusStyle = errorStyle
	}

	var parts []string
	if command != "" {
		parts = append(parts, r.renderCommandLineOnSurface(command, commandErr, false, ""))
	}
	if showBody {
		bodyText := text
		if commandErr != nil {
			bodyText = commandErr.Error()
		}
		if bodyText != "" {
			bodyText = render.ReplayTerminalOutput(bodyText)
			parts = append(parts, withoutBackground(statusStyle).Render(bodyText))
		}
	}

	return r.wrapCard(strings.Join(parts, "\n"), maxWidth, statusStyle)
}

func (r CardRenderer) wrapCard(content string, maxWidth int, borderStyle ...lipgloss.Style) string {
	if render.BlockWidth(content) > maxWidth {
		content = lipgloss.NewStyle().Width(maxWidth).Render(content)
	}
	style := theme.DefaultPalette.Get("flash", "", "text", false)
	if len(borderStyle) > 0 {
		// lipgloss reports an unset foreground as NoColor, never nil.
		if _, unset := borderStyle[0].GetForeground().(lipgloss.NoColor); !unset {
			style = borderStyle[0]
		}
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(style.GetForeground()).
		Render(content)
}

func (r CardRenderer) renderCommandLine(command string, commandErr error, running bool, indicator string) string {
	if command == "" {
		return ""
	}
	successStyle := theme.DefaultPalette.Get("flash", "", "success", false)
	errorStyle := theme.DefaultPalette.Get("flash", "", "error", false)
	textStyle := theme.DefaultPalette.Get("flash", "", "text", false)
	matchedStyle := theme.DefaultPalette.Get("flash", "", "matched", false)

	mark := successStyle.Width(commandMarkWidth).Render("✓ ")
	if running {
		mark = textStyle.Width(commandMarkWidth).Render(indicator + " ")
	} else if commandErr != nil {
		mark = errorStyle.Width(commandMarkWidth).Render("✗ ")
	}
	return mark + render.ColorizeCommand(command, textStyle, matchedStyle)
}

func (r CardRenderer) renderCommandLineOnSurface(command string, commandErr error, running bool, indicator string) string {
	if command == "" {
		return ""
	}
	successStyle := withoutBackground(theme.DefaultPalette.Get("flash", "", "success", false))
	errorStyle := withoutBackground(theme.DefaultPalette.Get("flash", "", "error", false))
	textStyle := withoutBackground(theme.DefaultPalette.Get("flash", "", "text", false))
	matchedStyle := withoutBackground(theme.DefaultPalette.Get("flash", "", "matched", false))

	mark := successStyle.Width(commandMarkWidth).Render("✓ ")
	if running {
		mark = textStyle.Width(commandMarkWidth).Render(indicator + " ")
	} else if commandErr != nil {
		mark = errorStyle.Width(commandMarkWidth).Render("✗ ")
	}
	return mark + render.ColorizeCommand(command, textStyle, matchedStyle)
}

func withoutBackground(style lipgloss.Style) lipgloss.Style {
	return style.UnsetBackground()
}
