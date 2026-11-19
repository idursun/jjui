package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestPanelResizeShortcutsWorkAcrossSplitContents(t *testing.T) {
	previous := *config.Current
	configCopy := previous
	configCopy.Preview.Position = "right"
	configCopy.Preview.WidthPercentage = 50
	configCopy.Bindings = append([]config.BindingConfig(nil), previous.Bindings...)
	configCopy.Bindings = append(configCopy.Bindings,
		config.BindingConfig{Action: "ui.preview_expand", Scope: "ui.preview", Key: config.StringList{"ctrl+left"}},
		config.BindingConfig{Action: "ui.preview_shrink", Scope: "ui.preview", Key: config.StringList{"ctrl+right"}},
	)
	oldConfig := config.Current
	config.Current = &configCopy
	t.Cleanup(func() { config.Current = oldConfig })

	for _, content := range []string{previewContentID, bookmarkContentID} {
		for _, focused := range []bool{false, true} {
			if content == previewContentID && focused {
				continue
			}
			name := content + "/revisions-focused"
			if focused {
				name = content + "/panel-focused"
			}
			t.Run(name, func(t *testing.T) {
				model := NewUI(test.NewTestContext(test.NewTestCommandRunner(t)))
				model.splitContainer.ShowContent(content)
				if focused {
					model.splitContainer.FocusSplitContent()
				}
				initial := splitSeparatorX(t, model)
				model.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl})
				grown := splitSeparatorX(t, model)
				assert.Less(t, grown, initial, "Ctrl+Left should grow the visible panel")
				model.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl})
				assert.Equal(t, initial, splitSeparatorX(t, model), "Ctrl+Right should restore the original size")
				assert.Equal(t, content, model.splitContainer.ActiveID())
				assert.Equal(t, focused, model.splitContainer.ContentFocused())
			})
		}
	}
}
