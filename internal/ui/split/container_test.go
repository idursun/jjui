package split

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeContent struct {
	scope   keybindings.ScopeName
	shows   int
	hides   int
	updates int
	focused bool
}

func (f *fakeContent) Init() tea.Cmd { return nil }

func (f *fakeContent) Update(tea.Msg) tea.Cmd {
	f.updates++
	return nil
}

func (f *fakeContent) ViewRect(*render.DisplayContext, layout.Box) {}

func (f *fakeContent) OnShow() tea.Cmd {
	f.shows++
	return nil
}

func (f *fakeContent) OnHide() {
	f.hides++
}

func (f *fakeContent) SetFocused(focused bool) {
	f.focused = focused
}

func (f *fakeContent) Scopes() []common.Scope {
	return []common.Scope{{Name: f.scope, Handler: f}}
}

func (f *fakeContent) HandleIntent(intents.Intent) (tea.Cmd, bool) {
	return nil, false
}

func TestSplitContainerShowToggleAndClose(t *testing.T) {
	sc := NewSplitContainer(NewSplitState(50))
	a := &fakeContent{scope: "a"}
	b := &fakeContent{scope: "b"}
	sc.RegisterContent("a", a)
	sc.RegisterContent("b", b)

	_, shown := sc.ShowContent("a")
	require.True(t, shown)
	assert.Equal(t, 1, a.shows)
	_, shown = sc.ShowContent("a")
	assert.False(t, shown)

	sc.Update(struct{}{})
	assert.Equal(t, 1, a.updates)
	assert.Equal(t, 0, b.updates)

	_, shown = sc.ShowContent("b")
	require.True(t, shown)
	assert.Equal(t, 1, a.hides)
	assert.Equal(t, 1, b.shows)

	_, changed := sc.ToggleContent("b")
	require.True(t, changed)
	assert.Equal(t, 1, b.hides)
	assert.Empty(t, sc.ActiveID())

	assert.False(t, sc.Close())
}

func TestSplitContainerShowContentUnknownIDReturnsFalseWithoutChangingActiveContent(t *testing.T) {
	sc := NewSplitContainer(NewSplitState(50))
	a := &fakeContent{scope: "a"}
	sc.RegisterContent("a", a)

	_, shown := sc.ShowContent("a")
	require.True(t, shown)

	_, shown = sc.ShowContent("missing")
	assert.False(t, shown)
	assert.Equal(t, "a", sc.ActiveID())
	assert.Equal(t, 0, a.hides)
	assert.Equal(t, 1, a.shows)
}

func TestSplitContainerTracksOnlyWhetherContentIsFocused(t *testing.T) {
	sc := NewSplitContainer(NewSplitState(50))
	content := &fakeContent{scope: "content"}
	sc.RegisterContent("content", content)
	_, _ = sc.ShowContent("content")

	primary := []common.Scope{{Name: "primary"}}
	assert.False(t, sc.ContentFocused())
	assert.Equal(t, []keybindings.ScopeName{"primary", "content"}, scopeNames(sc.Scopes(primary)))

	sc.FocusSplitContent()
	assert.True(t, sc.ContentFocused())
	assert.True(t, content.focused)
	assert.Equal(t, []keybindings.ScopeName{"content"}, scopeNames(sc.Scopes(primary)))

	sc.ToggleFocus()
	assert.False(t, sc.ContentFocused())
	assert.False(t, content.focused)

	sc.ToggleFocus()
	assert.True(t, sc.ContentFocused())
	assert.True(t, content.focused)
	sc.Close()
	assert.False(t, sc.ContentFocused())
	assert.False(t, content.focused)
}

func TestSplitContainerResizeAndPosition(t *testing.T) {
	state := NewSplitState(50)
	state.SetPlacement(PlacementAuto)
	sc := NewSplitContainer(state)

	sc.Resize(10)
	assert.Equal(t, 60.0, state.Percent)
	sc.Resize(-20)
	assert.Equal(t, 40.0, state.Percent)

	sc.SetAutoPosition(true)
	assert.True(t, state.AtBottom)
	sc.TogglePosition()
	assert.False(t, state.AutoPosition)
	assert.False(t, state.AtBottom)
	sc.SetAutoPosition(true)
	assert.False(t, state.AtBottom)
}

func scopeNames(scopes []common.Scope) []keybindings.ScopeName {
	names := make([]keybindings.ScopeName, len(scopes))
	for i, scope := range scopes {
		names[i] = scope.Name
	}
	return names
}
