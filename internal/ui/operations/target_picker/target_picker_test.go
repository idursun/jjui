package target_picker

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/jj/source"
	jjuitest "github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileItemUsesDisplayLabelAndTypedSelection(t *testing.T) {
	ctx := jjuitest.NewTestContext(jjuitest.NewTestCommandRunner(t))
	ctx.Location = "/work/repo"
	ctx.WorkingDirectory = "/work/repo/internal"
	model := NewModel(ctx, nil, source.FileSource{Files: []jj.FileName{jj.NewFileName("internal/ui/ui.go")}})

	loaded, ok := model.Init()().(itemsLoadedMsg)
	require.True(t, ok)
	jjuitest.SimulateModel(model, func() tea.Msg { return loaded })
	require.Len(t, model.items, 1)
	assert.Equal(t, "ui/ui.go", model.items[0].Name)

	selected, ok := model.accept(false)().(TargetSelectedMsg)
	require.True(t, ok)
	assert.Empty(t, selected.Target)
	assert.Equal(t, "internal/ui/ui.go", selected.File.Path())
}
