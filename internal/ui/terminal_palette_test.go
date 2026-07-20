package ui

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTerminalPaletteReply_BELTerminatedRGB(t *testing.T) {
	reply, ok := parseTerminalPaletteReply(uv.UnknownOscEvent("\x1b]4;8;rgb:8080/8080/8080\x07"))
	require.True(t, ok)

	assert.Equal(t, 8, reply.Slot)
	assert.Equal(t, "#808080", reply.Hex)
}

func TestParseTerminalPaletteReply_STTerminatedHex(t *testing.T) {
	reply, ok := parseTerminalPaletteReply(uv.UnknownOscEvent("\x1b]4;12;#123456\x1b\\"))
	require.True(t, ok)

	assert.Equal(t, 12, reply.Slot)
	assert.Equal(t, "#123456", reply.Hex)
}

func TestParseTerminalPaletteReply_C1TerminatedRGB(t *testing.T) {
	reply, ok := parseTerminalPaletteReply(uv.UnknownOscEvent("\x9d4;3;rgb:1212/3434/5656\x9c"))
	require.True(t, ok)

	assert.Equal(t, 3, reply.Slot)
	assert.Equal(t, "#123456", reply.Hex)
}

func TestParseTerminalPaletteReply_IgnoresNonPaletteOSC(t *testing.T) {
	_, ok := parseTerminalPaletteReply(uv.UnknownOscEvent("\x1b]11;rgb:1a1a/1b1b/2c2c\x07"))
	assert.False(t, ok)
}

func TestParseTerminalPaletteReply_IgnoresOutOfRangeSlot(t *testing.T) {
	_, ok := parseTerminalPaletteReply(uv.UnknownOscEvent("\x1b]4;16;rgb:ffff/ffff/ffff\x07"))
	assert.False(t, ok)
}
