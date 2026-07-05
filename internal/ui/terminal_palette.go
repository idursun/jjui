package ui

import (
	"fmt"
	"image/color"
	"log"
	"strconv"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/config"

	tea "charm.land/bubbletea/v2"
)

const terminalPaletteSlots = 16

type terminalPaletteReply struct {
	Slot int
	Hex  string
	Raw  string
}

func requestTerminalPalette() tea.Cmd {
	cmds := make([]tea.Cmd, 0, terminalPaletteSlots)
	for slot := range terminalPaletteSlots {
		query := fmt.Sprintf("\x1b]4;%d;?\x07", slot)
		log.Printf("terminal palette request: slot=%d raw=%q", slot, query)
		cmds = append(cmds, tea.Raw(query))
	}
	return tea.Batch(cmds...)
}

func requestTerminalPaletteIfNeeded() tea.Cmd {
	if config.Current.UI.BackgroundBlend > 0 {
		return requestTerminalPalette()
	}
	return nil
}

func requestTerminalAppearance() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		requestTerminalPaletteIfNeeded(),
	)
}

func parseTerminalPaletteReply(event uv.UnknownOscEvent) (terminalPaletteReply, bool) {
	raw := string(event)
	payload, ok := oscPayload(raw)
	if !ok || !strings.HasPrefix(payload, "4;") {
		return terminalPaletteReply{}, false
	}

	parts := strings.SplitN(payload, ";", 3)
	if len(parts) != 3 || parts[0] != "4" {
		return terminalPaletteReply{}, false
	}

	slot, err := strconv.Atoi(parts[1])
	if err != nil || slot < 0 || slot >= terminalPaletteSlots {
		return terminalPaletteReply{}, false
	}

	hex := colorToHex(ansi.XParseColor(parts[2]))
	if hex == "" {
		return terminalPaletteReply{}, false
	}

	return terminalPaletteReply{Slot: slot, Hex: hex, Raw: raw}, true
}

func oscPayload(raw string) (string, bool) {
	switch {
	case strings.HasPrefix(raw, "\x1b]"):
		raw = strings.TrimPrefix(raw, "\x1b]")
	case strings.HasPrefix(raw, "\x9d"):
		raw = strings.TrimPrefix(raw, "\x9d")
	default:
		return "", false
	}

	switch {
	case strings.HasSuffix(raw, "\x07"):
		raw = strings.TrimSuffix(raw, "\x07")
	case strings.HasSuffix(raw, "\x1b\\"):
		raw = strings.TrimSuffix(raw, "\x1b\\")
	case strings.HasSuffix(raw, "\x9c"):
		raw = strings.TrimSuffix(raw, "\x9c")
	default:
		return "", false
	}

	return raw, true
}

func colorToHex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
