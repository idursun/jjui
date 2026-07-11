package ui

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"strconv"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

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

func requestTerminalPaletteIfNeeded(backgroundBlend float64) tea.Cmd {
	if backgroundBlend > 0 {
		return requestTerminalPalette()
	}
	return nil
}

func requestTerminalAppearance(backgroundBlend float64) tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		requestTerminalPaletteIfNeeded(backgroundBlend),
	)
}

func parseTerminalPaletteReply(event uv.UnknownOscEvent) (terminalPaletteReply, bool) {
	raw := string(event)
	var reply terminalPaletteReply
	var parsed bool
	parser := ansi.GetParser()
	defer ansi.PutParser(parser)
	parser.SetHandler(ansi.Handler{HandleOsc: func(command int, data []byte) {
		if command != 4 {
			return
		}
		_, payload, ok := bytes.Cut(data, []byte(";"))
		if !ok {
			return
		}
		parts := bytes.SplitN(payload, []byte(";"), 2)
		if len(parts) != 2 {
			return
		}
		slot, err := strconv.Atoi(string(parts[0]))
		if err != nil || slot < 0 || slot >= terminalPaletteSlots {
			return
		}
		hex := colorToHex(ansi.XParseColor(string(parts[1])))
		if hex == "" {
			return
		}
		reply = terminalPaletteReply{Slot: slot, Hex: hex, Raw: raw}
		parsed = true
	}})
	for i := range len(raw) {
		parser.Advance(raw[i])
	}
	return reply, parsed
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
