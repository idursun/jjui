package flash

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/render"
)

const commandMarkWidth = 3

type CardRenderer struct {
	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
	textStyle    lipgloss.Style
	matchedStyle lipgloss.Style
}

func NewCardRenderer() CardRenderer {
	return CardRenderer{
		successStyle: common.DefaultPalette.Get("flash success"),
		errorStyle:   common.DefaultPalette.Get("flash error"),
		textStyle:    common.DefaultPalette.Get("flash text"),
		matchedStyle: common.DefaultPalette.Get("flash matched"),
	}
}

func (r CardRenderer) RenderMessage(command, text string, commandErr error, maxWidth int) string {
	return r.renderCard(command, text, commandErr, maxWidth, true, true)
}

func (r CardRenderer) RenderHistoryEntry(entry commandHistoryEntry, maxWidth int, selected bool) string {
	return r.renderCard(entry.Command, entry.Text, entry.Err, maxWidth, selected, selected)
}

func (r CardRenderer) RenderRunningCommand(command, indicator string, maxWidth int) string {
	if command == "" {
		return ""
	}
	return r.wrapCard(
		r.renderCommandLine(command, nil, true, indicator),
		maxWidth,
		r.textStyle,
	)
}

func (r CardRenderer) renderCard(command, text string, commandErr error, maxWidth int, showBody bool, highlight bool) string {
	statusStyle := r.successStyle
	if commandErr != nil {
		statusStyle = r.errorStyle
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

func (r CardRenderer) wrapCard(content string, maxWidth int, borderStyle lipgloss.Style) string {
	if render.BlockWidth(content) > maxWidth {
		content = lipgloss.NewStyle().Width(maxWidth).Render(content)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(borderStyle.GetForeground()).
		Render(content)
}

func (r CardRenderer) renderCommandLine(command string, commandErr error, running bool, indicator string) string {
	if command == "" {
		return ""
	}
	mark := r.successStyle.Width(commandMarkWidth).Render("✓ ")
	if running {
		mark = r.textStyle.Width(commandMarkWidth).Render(indicator + " ")
	} else if commandErr != nil {
		mark = r.errorStyle.Width(commandMarkWidth).Render("✗ ")
	}
	return mark + ColorizeCommand(command, r.textStyle, r.matchedStyle)
}
