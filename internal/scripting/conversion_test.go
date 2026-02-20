package scripting

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func boolRef(v bool) *bool {
	return &v
}

func TestToLuaTableIncludesNestedStructFields(t *testing.T) {
	L := lua.NewState()
	t.Cleanup(L.Close)

	cfg := *config.Current
	tbl := toLuaTable(L, &cfg)

	uiVal := tbl.RawGetString("ui")
	uiTbl, ok := uiVal.(*lua.LTable)
	require.True(t, ok, "config.ui should be a table")

	themeVal := uiTbl.RawGetString("theme")
	themeTbl, ok := themeVal.(*lua.LTable)
	require.True(t, ok, "config.ui.theme should be a table")

	darkVal := themeTbl.RawGetString("dark")
	_, ok = darkVal.(lua.LString)
	require.True(t, ok, "config.ui.theme.dark should be a string")

	lightVal := themeTbl.RawGetString("light")
	_, ok = lightVal.(lua.LString)
	require.True(t, ok, "config.ui.theme.light should be a string")
}

func TestToLuaTable_UsesOnlyTomlTaggedFields(t *testing.T) {
	L := lua.NewState()
	t.Cleanup(L.Close)

	type sample struct {
		Tagged   int `toml:"tagged"`
		Untagged int
	}

	tbl := toLuaTable(L, sample{Tagged: 1, Untagged: 2})
	assert.Equal(t, lua.LNumber(1), tbl.RawGetString("tagged"))
	assert.Equal(t, lua.LNil, tbl.RawGetString("Untagged"))
}

func TestFromLuaTable_ColorBoolPointers(t *testing.T) {
	tests := []struct {
		name             string
		selectedBuilder  func(*lua.LState) *lua.LTable
		expectedNil      bool
		expectedValue    bool
		expectedBg       string
		initialUnderline *bool
	}{
		{
			name:             "omitted keeps nil",
			selectedBuilder:  func(L *lua.LState) *lua.LTable { return tableWithBg(L, "0") },
			expectedNil:      true,
			expectedBg:       "0",
			initialUnderline: nil,
		},
		{
			name:             "explicit false is preserved",
			selectedBuilder:  func(L *lua.LState) *lua.LTable { return tableWithBgAndUnderline(L, "0", false) },
			expectedNil:      false,
			expectedValue:    false,
			expectedBg:       "0",
			initialUnderline: nil,
		},
		{
			name:             "explicit true is preserved",
			selectedBuilder:  func(L *lua.LState) *lua.LTable { return tableWithBgAndUnderline(L, "0", true) },
			expectedNil:      false,
			expectedValue:    true,
			expectedBg:       "0",
			initialUnderline: nil,
		},
		{
			name:             "omitted keeps existing value",
			selectedBuilder:  func(L *lua.LState) *lua.LTable { return tableWithBg(L, "0") },
			expectedNil:      false,
			expectedValue:    true,
			expectedBg:       "0",
			initialUnderline: boolRef(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			L := lua.NewState()
			t.Cleanup(L.Close)

			cfg := config.Config{
				UI: config.UIConfig{
					Colors: map[string]config.Color{
						"selected": {
							Fg:        "white",
							Underline: tt.initialUnderline,
						},
					},
				},
			}

			root := L.NewTable()
			ui := L.NewTable()
			colors := L.NewTable()
			colors.RawSetString("selected", tt.selectedBuilder(L))
			ui.RawSetString("colors", colors)
			root.RawSetString("ui", ui)

			err := fromLuaTable(root, &cfg)
			require.NoError(t, err)

			selected := cfg.UI.Colors["selected"]
			assert.Equal(t, tt.expectedBg, selected.Bg)
			if tt.expectedNil {
				assert.Nil(t, selected.Underline)
				return
			}
			if assert.NotNil(t, selected.Underline) {
				assert.Equal(t, tt.expectedValue, *selected.Underline)
			}
		})
	}
}

func tableWithBg(L *lua.LState, bg string) *lua.LTable {
	tbl := L.NewTable()
	tbl.RawSetString("bg", lua.LString(bg))
	return tbl
}

func tableWithBgAndUnderline(L *lua.LState, bg string, underline bool) *lua.LTable {
	tbl := L.NewTable()
	tbl.RawSetString("bg", lua.LString(bg))
	tbl.RawSetString("underline", lua.LBool(underline))
	return tbl
}
