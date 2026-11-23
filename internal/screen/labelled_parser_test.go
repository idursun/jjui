package screen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Empty spans are kept so their labels still describe the line.
func TestParseLabelledFromReader(t *testing.T) {
	in := "\x1b[1m<<log node::◆>>\x1b[0m  <<log empty::(empty)>><<log:: >><<log description::a >> b << c::d>><<log::>>\n" +
		"│  <<log description::x>>y>>\n"
	var got []string
	for s := range ParseLabelledFromReader(strings.NewReader(in)) {
		got = append(got, s.Text+"|"+strings.Join(s.Labels, ","))
	}
	assert.Equal(t, []string{
		"◆|log,node",
		"  |",
		"(empty)|log,empty",
		" |log",
		"a >> b << c::d|log,description",
		"|log",
		"\n│  |",
		"x>>y|log,description",
		"\n|",
	}, got)
}

func TestStripLabels(t *testing.T) {
	in := "<<error heading::Error: >><<error::The working copy is stale.>>\n" +
		"<<hint heading::Hint: >><<hint::Run `jj workspace update-stale` to update it.>>\n"
	assert.Equal(t, "Error: The working copy is stale.\nHint: Run `jj workspace update-stale` to update it.\n", StripLabels(in))
	assert.Equal(t, "plain a << b", StripLabels("plain a << b"))
}
