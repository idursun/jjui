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

func TestLogDoesNotOverrideWorkspaceColorsOrGraphOrdering(t *testing.T) {
	assert.NotContains(t, Log("all()", 10, "description"), "--config")
}
