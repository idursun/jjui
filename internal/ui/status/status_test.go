package status

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/help"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
)

func TestStatus_Update_ExecProcessCompletedMsg(t *testing.T) {
	cases := []struct {
		name           string
		msg            common.ExecProcessCompletedMsg
		expectedMode   string
		expectedPrompt string
		expectedInput  string
		shouldFocus    bool
		expectCmd      bool
	}{
		{
			name: "Execution failed, should restore input",
			msg: common.ExecProcessCompletedMsg{
				Err: errors.New("exit status 1"),
				Msg: common.ExecMsg{
					Line: "invalid command",
					Mode: common.ExecShell,
				},
			},
			expectedMode:   "exec sh",
			expectedPrompt: "$ ",
			expectedInput:  "invalid command",
			shouldFocus:    true,
			expectCmd:      true,
		},
		{
			name: "Execution succeeded",
			msg: common.ExecProcessCompletedMsg{
				Err: nil,
			},
			expectCmd: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &context.MainContext{
				Histories: config.NewHistories(),
			}
			m := New(ctx)
			cmd := m.Update(tt.msg)

			if tt.expectCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}

			if tt.shouldFocus {
				assert.True(t, m.IsFocused())
				assert.Equal(t, tt.expectedMode, m.mode)
				assert.Equal(t, tt.expectedPrompt, m.input.Prompt)
				assert.Equal(t, tt.expectedInput, m.input.Value())
			} else {
				assert.False(t, m.IsFocused())
			}
		})
	}
}

func TestModel_BuildHelpGrid_ColumnMajorOrder(t *testing.T) {
	m := &Model{}
	entries := []string{"A", "B", "C", "D", "E", "F"}

	lines := m.buildHelpGrid(entries, 1, 9)

	assert.Equal(t, []string{
		"A  C  E",
		"B  D  F",
	}, lines)
}

func TestModel_ViewStatusRectShowsHelpWhileInputIsFocused(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	m.SetHelp([]help.Entry{{Label: "enter", Desc: "apply"}})
	m.StartExec(common.ExecShell)

	dl := render.NewDisplayContext()
	m.ViewStatusRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 1)))
	rendered := dl.RenderToString(40, 1)

	assert.NotContains(t, rendered, "exec sh")
	assert.Contains(t, rendered, "enter")
	assert.Contains(t, rendered, "apply")
}

func TestModel_ViewStatusRectShowsMode(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	m.SetMode("diff")

	dl := render.NewDisplayContext()
	m.ViewStatusRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 1)))
	rendered := dl.RenderToString(40, 1)

	assert.Contains(t, rendered, "diff")
}

func TestModel_ViewStatusRectExpandedHelpLeavesMainViewVisible(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	entries := make([]help.Entry, 20)
	for i := range entries {
		entries[i] = help.Entry{Label: "k", Desc: "entry"}
	}
	m.SetHelp(entries)
	m.SetStatusExpanded(true)

	dl := render.NewDisplayContext()
	m.ViewStatusRect(dl, layout.NewBox(layout.Rect(0, 4, 20, 1)))
	rendered := dl.RenderToString(20, 5)
	lines := strings.Split(rendered, "\n")

	assert.NotContains(t, lines[0], "entry")
	assert.Contains(t, rendered, "full help")
}

func TestModel_ViewInputRectUsesFullWidth(t *testing.T) {
	ctx := &context.MainContext{Histories: config.NewHistories()}
	m := New(ctx)
	m.StartExec(common.ExecShell)

	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 40, 1))
	m.ViewInputRect(dl, box)

	assert.Equal(t, 27, m.input.Width())
	assert.NotNil(t, dl.Cursor())
}

func TestStatus_Update_FileSearchFocusesAndTypingUpdatesInput(t *testing.T) {
	ctx := &context.MainContext{
		Histories: config.NewHistories(),
	}
	m := New(ctx)

	cmd := m.Update(common.FileSearchMsg{
		Revset:     "@",
		Commit:     &jj.Commit{ChangeId: "abc123", CommitId: "def456"},
		RawFileOut: []byte("a.txt\nb.txt"),
	})
	assert.NotNil(t, cmd)
	assert.Equal(t, FocusFileSearch, m.FocusKind())
	assert.True(t, m.IsFocused())

	m.Update(tea.KeyPressMsg{Text: "x", Code: 'x'})
	assert.Equal(t, "x", m.InputValue())
}
