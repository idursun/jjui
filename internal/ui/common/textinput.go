package common

import "charm.land/bubbles/v2/textinput"

// Create a textinput.Model extending the KeyMap so that shift+backspace and
// shift+delete behave the same as their unshifted counterparts. Some terminals
// (e.g. wezterm, ghostty on Linux) report these as distinct key events via the
// Kitty keyboard protocol, but the bubbles default KeyMap does not include
// them.
func TextInputNew() textinput.Model {
	ti := textinput.New()

	km := &ti.KeyMap
	km.DeleteCharacterBackward.SetKeys(
		append(km.DeleteCharacterBackward.Keys(), "shift+backspace")...,
	)
	km.DeleteCharacterForward.SetKeys(
		append(km.DeleteCharacterForward.Keys(), "shift+delete")...,
	)
	km.DeleteWordBackward.SetKeys(
		append(km.DeleteWordBackward.Keys(), "alt+shift+backspace")...,
	)

	return ti;
}
