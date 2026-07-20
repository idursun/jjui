package common

import (
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

func TestPaletteUpdate_AddsDiffRoleAliases(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"diff added":    {Fg: Green},
		"diff renamed":  {Fg: Blue},
		"diff copied":   {Fg: Blue},
		"diff modified": {Fg: Yellow},
		"diff removed":  {Fg: Red},
	})

	for role, want := range map[string]string{
		"added": Green, "renamed": Blue, "copied": Blue, "modified": Yellow, "deleted": Red,
	} {
		assert.Equal(t, lipgloss.Color(want), p.Get("", "", role, false).GetForeground())
	}
}

func TestPaletteUpdate_ReplacesCachedTheme(t *testing.T) {
	p := NewPalette()
	p.Update(map[string]config.Color{
		"text":      {Fg: Red},
		"dark_only": {Fg: Green},
	})

	assert.Equal(t, lipgloss.Color(Red), p.Get("", "", "text", false).GetForeground())
	assert.Equal(t, lipgloss.Color(Green), p.Get("", "", "dark_only", false).GetForeground())

	p.Update(map[string]config.Color{
		"text": {Fg: Blue},
	})

	assert.Equal(t, lipgloss.Color(Blue), p.Get("", "", "text", false).GetForeground())
	assert.Equal(t, lipgloss.NewStyle().GetForeground(), p.Get("", "", "dark_only", false).GetForeground())
}
