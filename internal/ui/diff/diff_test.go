package diff

import (
	"strings"
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

func TestScroll_ChangesVisibleContent(t *testing.T) {
	model := New("1\n2\n3\n4\n5")

	// Initially line 1 should be visible
	before := test.Stripped(test.RenderImmediate(model, 10, 2))
	assert.Contains(t, before, "1")

	// Scroll down 2 — line 1 should no longer be visible
	model.Update(intents.DiffScroll{Kind: intents.DiffScrollDown})
	model.Update(intents.DiffScroll{Kind: intents.DiffScrollDown})
	after := test.Stripped(test.RenderImmediate(model, 10, 2))
	assert.NotContains(t, after, "1")
	assert.Contains(t, after, "3")
}

func TestUpdate_ScrollMsgScrollsContent(t *testing.T) {
	model := New("a\nb\nc\nd\ne")

	// Line "a" should be visible before scroll
	before := test.Stripped(test.RenderImmediate(model, 10, 2))
	assert.Contains(t, before, "a")

	model.Update(ScrollMsg{Delta: 2})
	after := test.Stripped(test.RenderImmediate(model, 10, 2))
	assert.NotContains(t, after, "a")
	assert.Contains(t, after, "c")
}

func TestUpdate_DiffScrollIntentChangesContent(t *testing.T) {
	model := New("1\n2\n3\n4\n5")

	first := test.Stripped(test.RenderImmediate(model, 10, 2))
	assert.Contains(t, first, "1")

	model.Update(intents.DiffScroll{Kind: intents.DiffScrollDown})
	second := test.Stripped(test.RenderImmediate(model, 10, 2))
	assert.NotContains(t, second, "1")
	assert.Contains(t, second, "2")

	model.Update(intents.DiffScroll{Kind: intents.DiffScrollUp})
	third := test.Stripped(test.RenderImmediate(model, 10, 2))
	assert.Contains(t, third, "1")
}

func TestWrap_LongLinesWrapAtViewportWidth(t *testing.T) {
	// 20-character line rendered in a 10-wide viewport should produce 2 visual rows
	model := New("12345678901234567890")
	model.Update(intents.DiffToggleWrap{})

	rendered := test.Stripped(test.RenderImmediate(model, 10, 3))
	lines := strings.Split(rendered, "\n")
	// Both halves should be present
	assert.Equal(t, "1234567890", lines[0])
	assert.Equal(t, "1234567890", lines[1])
}

func TestWrap_ResizeRecomputes(t *testing.T) {
	// Line of 20 chars
	model := New("12345678901234567890")
	model.Update(intents.DiffToggleWrap{})

	// At width 10, line occupies 2 visual rows.
	rendered := test.Stripped(test.RenderImmediate(model, 10, 5))
	assert.Contains(t, rendered, "1234567890\n1234567890")

	// At width 5, line occupies 4 visual rows.
	rendered = test.Stripped(test.RenderImmediate(model, 5, 10))
	assert.Contains(t, rendered, "12345\n67890\n12345\n67890")
}

func TestNoWrap_HorizontalScroll(t *testing.T) {
	// 20-character line rendered in a 10-wide viewport
	model := New("abcdefghijklmnopqrst")

	// Without horizontal scroll, first 10 chars visible
	rendered := test.Stripped(test.RenderImmediate(model, 10, 1))
	assert.Equal(t, "abcdefghij", rendered)

	// Scroll right 5 columns
	for range 5 {
		model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	}
	rendered = test.Stripped(test.RenderImmediate(model, 10, 1))
	assert.Equal(t, "fghijklmno", rendered)
}

func TestWrap_HorizontalScrollIsNoop(t *testing.T) {
	model := New("abcdefghijklmnopqrst")
	model.Update(intents.DiffToggleWrap{})
	before := test.Stripped(test.RenderImmediate(model, 10, 2))

	// Horizontal scroll should not change output in wrap mode.
	model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	after := test.Stripped(test.RenderImmediate(model, 10, 2))
	assert.Equal(t, before, after)
}

func TestSetContent_ResetsScrollOffsets(t *testing.T) {
	model := New("abcdefghijklmnopqrstuvwxyz")

	model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	model.Update(intents.DiffShow{Content: "abcdefghijklmnopqrstuvwxyz"})

	rendered := test.Stripped(test.RenderImmediate(model, 10, 1))
	assert.Equal(t, "abcdefghij", rendered)
}

func TestSetContent_PreservesWrapMode(t *testing.T) {
	model := New("12345678901234567890")
	model.Update(intents.DiffToggleWrap{})
	model.Update(intents.DiffShow{Content: "abcdefghij1234567890"})

	rendered := test.Stripped(test.RenderImmediate(model, 10, 3))
	assert.Contains(t, rendered, "abcdefghij\n1234567890")
}

func TestTabs_RenderIndentedInDefaultView(t *testing.T) {
	model := New("+\tfoo")

	rendered := test.RenderImmediate(model, 12, 1)
	assert.Equal(t, "+    foo", rendered)
}

func TestTabs_AffectHorizontalScrollWidth(t *testing.T) {
	model := New("\tabcdefghij")

	rendered := test.RenderImmediate(model, 8, 1)
	assert.Equal(t, "    abcd", rendered)

	for range 6 {
		model.Update(intents.DiffScrollHorizontal{Kind: intents.DiffScrollRight})
	}
	rendered = test.RenderImmediate(model, 8, 1)
	assert.Equal(t, "cdefghij", rendered)
}

func TestTabs_WrapUsingExpandedWidth(t *testing.T) {
	model := New("\t123456")
	model.Update(intents.DiffToggleWrap{})

	rendered := test.RenderImmediate(model, 5, 2)
	assert.Equal(t, "    1\n23456", rendered)
}
