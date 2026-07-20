package common

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
)

func boolPtr(v bool) *bool {
	return &v
}

const (
	Black  = "0"
	Red    = "1"
	Green  = "2"
	Yellow = "3"
	Blue   = "4"
	Cyan   = "6"
	White  = "7"
)

func TestPaletteGet_BaseCandidatePrecedence(t *testing.T) {
	tests := []struct {
		name   string
		colors map[string]config.Color
		want   string
	}{
		{
			name: "exact component role overrides all fallbacks",
			colors: map[string]config.Color{
				"status input text": {Fg: Red},
				"status input":      {Fg: Blue},
				"status text":       {Fg: Green},
				"text":              {Fg: White},
			},
			want: Red,
		},
		{
			name: "component base overrides scoped role",
			colors: map[string]config.Color{
				"status input": {Fg: Blue},
				"status text":  {Fg: Green},
			},
			want: Blue,
		},
		{
			name: "scope overrides generic component role",
			colors: map[string]config.Color{
				"status":     {Fg: Yellow},
				"input text": {Fg: Green},
				"text":       {Fg: White},
			},
			want: Yellow,
		},
		{
			name: "generic component role applies",
			colors: map[string]config.Color{
				"input text": {Fg: Green},
				"input":      {Fg: Blue},
				"text":       {Fg: White},
			},
			want: Green,
		},
		{
			name: "generic component applies before generic role",
			colors: map[string]config.Color{
				"input": {Fg: Blue},
				"text":  {Fg: White},
			},
			want: Blue,
		},
		{
			name: "generic role applies",
			colors: map[string]config.Color{
				"text": {Fg: White},
			},
			want: White,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPalette()
			p.Update(tt.colors)
			assert.Equal(t, lipgloss.Color(tt.want), p.Get("status", "input", "text", false).GetForeground())
		})
	}
}

func TestPaletteGet_BaseCandidatesFillOmittedProperties(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"status input": {Bg: Black},
		"status text":  {Fg: Cyan},
		"input text":   {Underline: boolPtr(true)},
		"text":         {Italic: boolPtr(true)},
	})

	got := p.Get("status", "input", "text", false)
	assert.Equal(t, lipgloss.Color(Black), got.GetBackground())
	assert.Equal(t, lipgloss.Color(Cyan), got.GetForeground())
	assert.True(t, got.GetUnderline())
	assert.True(t, got.GetItalic())
}

func TestPaletteGet_SelectedCandidatePrecedence(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "exact component role selected", key: "scope component role:selected", want: Red},
		{name: "component selected", key: "scope component:selected", want: Green},
		{name: "scoped role selected", key: "scope role:selected", want: Yellow},
		{name: "scope selected", key: "scope:selected", want: Blue},
		{name: "generic component role selected", key: "component role:selected", want: Cyan},
		{name: "generic component selected", key: "component:selected", want: White},
		{name: "generic role selected", key: "role:selected", want: "8"},
		{name: "global selected", key: ":selected", want: "9"},
		{name: "unselected base fallback", key: "scope component role", want: "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colors := map[string]config.Color{
				"scope component role": {Fg: "10"},
				":selected":            {Fg: "9"},
			}
			if tt.key == "scope component role" {
				delete(colors, ":selected")
			} else if tt.key != ":selected" {
				colors[tt.key] = config.Color{Fg: tt.want}
				delete(colors, ":selected")
			} else {
				colors[tt.key] = config.Color{Fg: tt.want}
			}
			p := NewPalette()
			p.Update(colors)
			assert.Equal(t, lipgloss.Color(tt.want), p.Get("scope", "component", "role", true).GetForeground())
		})
	}
}

func TestPaletteGet_SelectedCandidatesFillOmittedProperties(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"git menu text:selected": {Italic: boolPtr(true)},
		"git menu:selected":      {Bg: Blue, Underline: boolPtr(true)},
		"git text:selected":      {Fg: Yellow},
		"git:selected":           {Bold: boolPtr(true)},
	})

	got := p.Get("git", "menu", "text", true)
	assert.Equal(t, lipgloss.Color(Yellow), got.GetForeground())
	assert.Equal(t, lipgloss.Color(Blue), got.GetBackground())
	assert.True(t, got.GetBold())
	assert.True(t, got.GetUnderline())
	assert.True(t, got.GetItalic())
}

func TestPaletteGet_ExplicitDefaultBackgroundStopsInheritance(t *testing.T) {
	tests := []struct {
		name       string
		colors     map[string]config.Color
		isSelected bool
	}{
		{
			name: "base",
			colors: map[string]config.Color{
				"git menu": {Bg: "default"},
				"git":      {Bg: Blue},
			},
		},
		{
			name: "selected",
			colors: map[string]config.Color{
				"git:selected": {Bg: "default"},
				":selected":    {Bg: Blue},
			},
			isSelected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPalette()
			p.Update(tt.colors)
			assert.IsType(t, lipgloss.NoColor{}, p.Get("git", "menu", "text", tt.isSelected).GetBackground())
		})
	}
}

func TestPaletteGet_ExplicitFalseOverridesInheritedTrue(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"matched":                   {Underline: boolPtr(true)},
		"revisions details matched": {Underline: boolPtr(false)},
		":selected":                 {Bold: boolPtr(true)},
		"revisions:selected":        {Bold: boolPtr(false)},
	})

	got := p.Get("revisions", "details", "matched", true)
	assert.False(t, got.GetUnderline())
	assert.False(t, got.GetBold())
}

func TestPaletteGet_LegacyAndSuffixThemeSelectorsNormalizeIdentically(t *testing.T) {
	legacy := NewPalette()
	legacy.Update(map[string]config.Color{"revisions details selected text": {Fg: Yellow}})
	suffix := NewPalette()
	suffix.Update(map[string]config.Color{"revisions details text:selected": {Fg: Yellow}})

	assert.Equal(t,
		legacy.Get("revisions", "details", "text", true),
		suffix.Get("revisions", "details", "text", true),
	)
}

func TestPaletteGet_DefaultThemePreservesComponentBehavior(t *testing.T) {
	for _, isDark := range []bool{false, true} {
		t.Run(map[bool]string{false: "light", true: "dark"}[isDark], func(t *testing.T) {
			theme, err := config.LoadEmbeddedTheme("default", isDark)
			assert.NoError(t, err)
			theme.Colors["diff removed"] = config.Color{Fg: Red}
			p := NewPalette()
			p.Update(theme.Colors)

			assert.Equal(t, p.Get("git", "", "title", false), p.Get("git", "remote", "title", false))
			assert.Equal(t, p.Get("bookmarks", "", "title", false), p.Get("bookmarks", "remote", "title", false))
			assert.Equal(t, p.Get("git", "", "matched", false), p.Get("git", "input", "matched", false))
			assert.Equal(t, p.Get("bookmarks", "", "matched", false), p.Get("bookmarks", "input", "matched", false))
			assert.Equal(t, p.Get("git", "", "text", false), p.Get("git", "input", "text", false))
			assert.Equal(t, p.Get("bookmarks", "", "text", false), p.Get("bookmarks", "input", "text", false))
			assert.IsType(t, lipgloss.NoColor{}, p.Get("git", "", "text", true).GetBackground())
			assert.IsType(t, lipgloss.NoColor{}, p.Get("bookmarks", "", "text", true).GetBackground())
			assert.IsType(t, lipgloss.NoColor{}, p.Get("git", "remote", "", true).GetBackground())
			assert.IsType(t, lipgloss.NoColor{}, p.Get("bookmarks", "remote", "", true).GetBackground())
			selectedBackground := map[bool]string{false: "7", true: "8"}[isDark]
			assert.Equal(t, lipgloss.Color(selectedBackground), p.Get("other", "", "text", true).GetBackground())
			selectedDeleted := p.Get("revisions", "details", "deleted", true)
			assert.Equal(t, lipgloss.Color(Red), selectedDeleted.GetForeground())
			assert.Equal(t, p.Get("revisions", "details", "text", true).GetBackground(), selectedDeleted.GetBackground())
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
				"heading": {Fg: Blue, Bold: boolPtr(true), Italic: boolPtr(true)},
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

			got := p.Get("", "", tt.selector, false)

			if tt.name == "diff shortcuts" {
				// Check that all diff shortcuts were properly added
				assert.Equal(t, lipgloss.Color("2"), p.Get("", "", "added", false).GetForeground(), "added style not set correctly")
				assert.Equal(t, lipgloss.Color("4"), p.Get("", "", "renamed", false).GetForeground(), "renamed style not set correctly")
				assert.Equal(t, lipgloss.Color("4"), p.Get("", "", "copied", false).GetForeground(), "copied style not set correctly")
				assert.Equal(t, lipgloss.Color("3"), p.Get("", "", "modified", false).GetForeground(), "modified style not set correctly")
				assert.Equal(t, lipgloss.Color("1"), p.Get("", "", "deleted", false).GetForeground(), "deleted style not set correctly")
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
		want  color.Color
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
			name:  "default color",
			color: "default",
			want:  lipgloss.NoColor{},
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

func TestPaletteUpdate_InheritsWhenAttributeOmitted(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"matched":           {Underline: boolPtr(true)},
		"revisions matched": {Fg: Cyan},
	})

	got := p.Get("revisions", "", "matched", false)
	assert.True(t, got.GetUnderline())
}

func TestPaletteUpdate_ClearsCachedStyles(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"text": {Fg: Red},
	})

	// Populate the cache.
	got := p.Get("", "", "text", false)
	assert.Equal(t, lipgloss.Color("1"), got.GetForeground())

	// A second Update with different colors should invalidate the cache.
	p.Update(map[string]config.Color{
		"text": {Fg: Blue},
	})

	got = p.Get("", "", "text", false)
	assert.Equal(t, lipgloss.Color("4"), got.GetForeground())
}

func TestPaletteUpdate_ClearsStaleKeysFromPreviousTheme(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"text":      {Fg: Red},
		"dark_only": {Fg: Green},
	})

	assert.Equal(t, lipgloss.Color("2"), p.Get("", "", "dark_only", false).GetForeground())

	// Switch to a theme that lacks the "dark only" key.
	p.Update(map[string]config.Color{
		"text": {Fg: Blue},
	})

	got := p.Get("", "", "dark_only", false)
	assert.Equal(t, lipgloss.NewStyle().GetForeground(), got.GetForeground())
}

func TestPaletteUpdate_ExplicitFalseOverridesInheritedAttribute(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"matched":           {Underline: boolPtr(true)},
		"revisions matched": {Underline: boolPtr(false)},
	})

	got := p.Get("revisions", "", "matched", false)
	assert.False(t, got.GetUnderline())
}
