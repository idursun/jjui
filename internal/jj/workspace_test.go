package jj

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWorkspaces(t *testing.T) {
	assert.Equal(t, []Workspace{{Name: "default", Current: true}, {Name: "feature"}},
		ParseWorkspaces("default\ttrue\nfeature\tfalse\n\nmalformed\n"))
	assert.Empty(t, ParseWorkspaces(""))
}

func TestLogGraphPriorityIsExplicitAndOptional(t *testing.T) {
	assert.Equal(t, Log("all()", 10, "description"), Log("all()", 10, "description", ""))
	assert.Contains(t, Log("all()", 10, "description", "abc | def"), `revsets.log-graph-prioritize="abc | def"`)
}
