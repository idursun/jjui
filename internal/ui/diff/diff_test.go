package diff

import (
	"testing"

	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestNew_TrimsCarriageReturnsAndHandlesEmpty(t *testing.T) {
	model := New("line1\r\nline2\r\n")
	assert.Equal(t, "line1\nline2", test.Stripped(test.RenderImmediate(model, 20, 5)))

	emptyModel := New("")
	assert.Equal(t, "(empty)", test.Stripped(test.RenderImmediate(emptyModel, 10, 3)))
}

func TestScroll_AdjustsViewportOffset(t *testing.T) {
	content := "1\n2\n3\n4\n5\n"
	model := New(content)

	model.Scroll(2)
	assert.Equal(t, 2, model.view.YOffset)

	model.Scroll(-1)
	assert.Equal(t, 1, model.view.YOffset)
}

func TestUpdate_ScrollMsgStillScrolls(t *testing.T) {
	model := New("1\n2\n3\n4\n5\n")
	cmd := model.Update(ScrollMsg{Delta: 1})
	assert.Nil(t, cmd)
	assert.Equal(t, 1, model.view.YOffset)
}

func TestUpdate_DiffScrollIntent(t *testing.T) {
	model := New("1\n2\n3\n4\n5\n")
	model.view.Height = 3

	cmd := model.Update(intents.DiffScroll{Kind: intents.DiffScrollDown})
	assert.Nil(t, cmd)
	assert.Equal(t, 1, model.view.YOffset)

	cmd = model.Update(intents.DiffScroll{Kind: intents.DiffScrollUp})
	assert.Nil(t, cmd)
	assert.Equal(t, 0, model.view.YOffset)
}
