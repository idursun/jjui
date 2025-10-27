package common

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
)

const (
	Black  = "0"
	Red    = "1"
	Green  = "2"
	Yellow = "3"
	Blue   = "4"
	Cyan   = "6"
	White  = "7"
)

func TestPalette_Get(t *testing.T) {
	type args struct {
		selector string
		styles   map[string]lipgloss.Style
	}

	tests := []struct {
		name string
		args args
		want lipgloss.Style
	}{
		{
			name: "exact match for single label",
			args: args{
				selector: "text",
				styles: map[string]lipgloss.Style{
					"text": lipgloss.NewStyle().Foreground(lipgloss.Color(White)),
				},
			},
			want: lipgloss.NewStyle().Foreground(lipgloss.Color(White)),
		},
		{
			name: "combined labels",
			args: args{
				selector: "revisions selected",
				styles: map[string]lipgloss.Style{
					"revisions": lipgloss.NewStyle().Italic(true),
					"selected":  lipgloss.NewStyle().Background(lipgloss.Color(Black)).Bold(true),
				},
			},
			want: lipgloss.NewStyle().Background(lipgloss.Color(Black)).Bold(true).Italic(true),
		},
		{
			name: "non-existent label",
			args: args{selector: "nonexistent", styles: nil},
			want: lipgloss.NewStyle(),
		},
		{
			name: "mixed existing and non-existent labels",
			args: args{
				selector: "text nonexistent",
				styles: map[string]lipgloss.Style{
					"text": lipgloss.NewStyle().Foreground(lipgloss.Color(White)),
				},
			},
			want: lipgloss.NewStyle().Foreground(lipgloss.Color(White)),
		},
		{
			name: "empty selector",
			args: args{selector: "", styles: nil},
			want: lipgloss.NewStyle(),
		},
		{
			name: "exact match for compound label",
			args: args{
				selector: "revisions text",
				styles: map[string]lipgloss.Style{
					"revisions text": lipgloss.NewStyle().Foreground(lipgloss.Color(Cyan)).Background(lipgloss.Color(Green)).Italic(true),
				},
			},
			want: lipgloss.NewStyle().Foreground(lipgloss.Color(Cyan)).Background(lipgloss.Color(Green)).Italic(true),
		},
		{
			name: "attribute inheritance",
			args: args{
				selector: "revisions matched",
				styles: map[string]lipgloss.Style{
					"matched":           lipgloss.NewStyle().Underline(true),
					"revisions matched": lipgloss.NewStyle().Underline(false),
				},
			},
			want: lipgloss.NewStyle().Underline(false),
		},
		{
			name: "attribute inheritance2",
			args: args{
				selector: "revisions matched",
				styles: map[string]lipgloss.Style{
					"matched":           lipgloss.NewStyle().Underline(false),
					"revisions matched": lipgloss.NewStyle().Underline(true),
				},
			},
			want: lipgloss.NewStyle().Underline(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up a palette with test styles using the add method
			p := NewPalette()

			for key, style := range tt.args.styles {
				p.add(key, style)
			}

			got := p.Get(tt.args.selector)

			// Compare foreground colours
			assert.Equal(t, tt.want.GetForeground(), got.GetForeground(), "foreground color mismatch")

			// Compare background colours
			assert.Equal(t, tt.want.GetBackground(), got.GetBackground(), "background color mismatch")

			// Compare style attributes
			assert.Equal(t, tt.want.GetBold(), got.GetBold(), "bold attribute mismatch")
			assert.Equal(t, tt.want.GetItalic(), got.GetItalic(), "italic attribute mismatch")
		})
	}
}

func TestPalette_Update(t *testing.T) {
	tests := []struct {
		name     string
		styleMap map[string]config.Color
		selector string
		want     lipgloss.Style
	}{
		{
			name: "basic color update",
			styleMap: map[string]config.Color{
				"text": {Fg: Red},
			},
			selector: "text",
			want:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		},
		{
			name: "update with multiple attributes",
			styleMap: map[string]config.Color{
				"heading": {Fg: Blue, Bold: true, Italic: true},
			},
			selector: "heading",
			want:     lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true).Italic(true),
		},
		{
			name: "update with background color",
			styleMap: map[string]config.Color{
				"highlight": {Fg: Black, Bg: Yellow},
			},
			selector: "highlight",
			want:     lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3")),
		},
		{
			name: "diff shortcuts",
			styleMap: map[string]config.Color{
				"diff added":    {Fg: Green},
				"diff renamed":  {Fg: Blue},
				"diff copied":   {Fg: Blue},
				"diff modified": {Fg: Yellow},
				"diff removed":  {Fg: Red},
			},
			selector: "added",
			want:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPalette()
			p.Update(tt.styleMap)

			got := p.Get(tt.selector)

			if tt.name == "diff shortcuts" {
				// Check that all diff shortcuts were properly added
				assert.Equal(t, lipgloss.Color("2"), p.Get("added").GetForeground(), "added style not set correctly")
				assert.Equal(t, lipgloss.Color("4"), p.Get("renamed").GetForeground(), "renamed style not set correctly")
				assert.Equal(t, lipgloss.Color("4"), p.Get("copied").GetForeground(), "copied style not set correctly")
				assert.Equal(t, lipgloss.Color("3"), p.Get("modified").GetForeground(), "modified style not set correctly")
				assert.Equal(t, lipgloss.Color("1"), p.Get("deleted").GetForeground(), "deleted style not set correctly")
			} else {
				assert.Equal(t, tt.want.GetForeground(), got.GetForeground(), "foreground color mismatch")
				assert.Equal(t, tt.want.GetBackground(), got.GetBackground(), "background color mismatch")
				assert.Equal(t, tt.want.GetBold(), got.GetBold(), "bold attribute mismatch")
				assert.Equal(t, tt.want.GetItalic(), got.GetItalic(), "italic attribute mismatch")
			}
		})
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  lipgloss.Color
	}{
		{
			name:  "hex color",
			color: "#ff0000",
			want:  lipgloss.Color("#ff0000"),
		},
		{
			name:  "ansi256 color by number",
			color: "123",
			want:  lipgloss.Color("123"),
		},
		{
			name:  "named color - red",
			color: "red",
			want:  lipgloss.Color("1"),
		},
		{
			name:  "named color - bright blue",
			color: "bright blue",
			want:  lipgloss.Color("12"),
		},
		{
			name:  "ansi-color prefix",
			color: "ansi-color-42",
			want:  lipgloss.Color("42"),
		},
		{
			name:  "invalid color",
			color: "not-a-color",
			want:  lipgloss.Color(""),
		},
		{
			name:  "out of range ansi256",
			color: "300",
			want:  lipgloss.Color(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseColor(tt.color)
			assert.Equal(t, tt.want, got)
		})
	}
}
