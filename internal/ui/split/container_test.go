package split

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeContent struct {
	shows   int
	updates int
}

func (f *fakeContent) Init() tea.Cmd { return nil }

func (f *fakeContent) Update(tea.Msg) tea.Cmd {
	f.updates++
	return nil
}

func (f *fakeContent) ViewRect(*render.DisplayContext, layout.Box) {}

func (f *fakeContent) OnShow() { f.shows++ }

func TestSplitContainerShowToggleAndClose(t *testing.T) {
	sc := NewSplitContainer(NewSplitState(50))
	a := &fakeContent{}
	b := &fakeContent{}
	sc.RegisterContent("a", a)
	sc.RegisterContent("b", b)

	require.True(t, sc.ShowContent("a"))
	assert.Equal(t, 1, a.shows)
	require.False(t, sc.ShowContent("a"))
	assert.Equal(t, 1, a.shows)
	sc.Update(struct{}{})
	assert.Equal(t, 1, a.updates)
	assert.Equal(t, 0, b.updates)

	require.True(t, sc.ToggleContent("a"))
	sc.Update(struct{}{})
	assert.Equal(t, 1, a.updates, "closed content should not receive updates")

	require.True(t, sc.ToggleContent("b"))
	assert.Equal(t, 1, b.shows)
	sc.Update(struct{}{})
	assert.Equal(t, 1, b.updates)

	require.True(t, sc.ToggleContent("a"))
	assert.Equal(t, 2, a.shows)
	sc.Update(struct{}{})
	assert.Equal(t, 2, a.updates)

	require.True(t, sc.Close())
	require.False(t, sc.Close())
}

func TestSplitContainerResizeAndPositionAreContainerState(t *testing.T) {
	state := NewSplitState(50)
	state.SetPlacement(PlacementAuto)
	sc := NewSplitContainer(state)

	require.True(t, sc.Resize(10))
	assert.Equal(t, 60.0, state.Percent)
	require.True(t, sc.Resize(-20))
	assert.Equal(t, 40.0, state.Percent)

	sc.SetAutoPosition(true)
	assert.True(t, state.AtBottom)
	sc.TogglePosition()
	assert.False(t, state.AutoPosition)
	assert.False(t, state.AtBottom)
	sc.SetAutoPosition(true)
	assert.False(t, state.AtBottom, "manual placement should ignore auto-position updates")
}
