package script

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	input := `
[script.nn]
name = "test"
steps = [
  { jj = ["commit", "-m", "Initial commit"] },
  { ui = { action = "select", params = { revision = "HEAD" } } },
  { jj = ["branch", "new-feature"] },
]
`
	script, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	assert.Equal(t, 3, len(script.Steps), "Expected 3 steps")
	assert.IsType(t, &JJStep{}, script.Steps[0])
	assert.IsType(t, &UIStep{}, script.Steps[1])
	assert.IsType(t, &JJStep{}, script.Steps[2])
}
