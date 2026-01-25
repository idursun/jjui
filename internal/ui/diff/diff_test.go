package diff

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestNew_TrimsCarriageReturnsAndHandlesEmpty(t *testing.T) {
	model := New(nil, "", nil, "line1\r\nline2\r\n")
	assert.Equal(t, "line1\nline2", test.Stripped(test.RenderImmediate(model, 20, 5)))

	emptyModel := New(nil, "", nil, "")
	assert.Equal(t, "(empty)", test.Stripped(test.RenderImmediate(emptyModel, 10, 3)))
}

func TestScroll_AdjustsViewportOffset(t *testing.T) {
	content := "1\n2\n3\n4\n5\n"
	model := New(nil, "", nil, content)

	model.Scroll(2)
	assert.Equal(t, 2, model.startLine)

	model.Scroll(-1)
	assert.Equal(t, 1, model.startLine)
}

func TestUpdate_CancelReturnsClose(t *testing.T) {
	model := New(nil, "", nil, "content")
	model.keymap.Cancel = key.NewBinding(key.WithKeys("q"))

	var msgs []tea.Msg
	test.SimulateModel(model, test.Type("q"), func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	assert.Contains(t, msgs, common.CloseViewMsg{})
}
