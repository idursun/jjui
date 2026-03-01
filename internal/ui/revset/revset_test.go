package revset

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestModel_Init(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	test.SimulateModel(model, model.Init())
}

func TestModel_Update_IntentDoesNotAlterCurrentRevsetDisplay(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "current"
	ctx.DefaultRevset = "default"
	model := New(ctx)
	test.SimulateModel(model, model.Init())
	test.SimulateModel(model, func() tea.Msg { return intents.CompletionMove{Delta: -1} })
	assert.Contains(t, test.RenderImmediate(model, 80, 5), "current")
}

func TestModel_View_DisplaysCurrentRevset(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "current"
	ctx.DefaultRevset = "default"
	model := New(ctx)
	assert.Contains(t, test.RenderImmediate(model, 80, 5), ctx.CurrentRevset)
}

func TestModel_ApplyCompletion(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)

	tests := []struct {
		name           string
		input          string
		item           CompletionItem
		expectedOutput string
	}{
		{
			name:           "function without parameters",
			input:          "al",
			item:           CompletionItem{Name: "all", Kind: KindFunction, HasParameters: false},
			expectedOutput: "all()",
		},
		{
			name:           "function with parameters",
			input:          "au",
			item:           CompletionItem{Name: "author", Kind: KindFunction, HasParameters: true},
			expectedOutput: "author(",
		},
		{
			name:           "history item",
			input:          "a",
			item:           CompletionItem{Name: "ancestors()", Kind: KindHistory},
			expectedOutput: "ancestors()",
		},
		{
			name:           "alias without parameters",
			input:          "my",
			item:           CompletionItem{Name: "myalias", Kind: KindAlias, HasParameters: false},
			expectedOutput: "myalias",
		},
		{
			name:           "function with context before",
			input:          "present(@) | au",
			item:           CompletionItem{Name: "author", Kind: KindFunction, HasParameters: true},
			expectedOutput: "present(@) | author(",
		},
		{
			name:           "parameterless function with context",
			input:          "empty() & ",
			item:           CompletionItem{Name: "all", Kind: KindFunction, HasParameters: false},
			expectedOutput: "empty() & all()",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := model.applyCompletion(test.input, test.item)
			assert.Equal(t, test.expectedOutput, result, "applyCompletion(%q, %v) should return %q", test.input, test.item, test.expectedOutput)
		})
	}
}
