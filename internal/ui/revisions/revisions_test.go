package revisions

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common/models"
	"github.com/stretchr/testify/assert"
)

func TestModel_highlightChanges(t *testing.T) {
	model := Model{
		rows: []models.Row{
			{Commit: &jj.Commit{ChangeId: "someother"}},
			{Commit: &jj.Commit{ChangeId: "nyqzpsmt"}},
		},
		output: `
Absorbed changes into these revisions:
  nyqzpsmt 8b1e95e3 change third file
Working copy now at: okrwsxvv 5233c94f (empty) (no description set)
Parent commit      : nyqzpsmt 8b1e95e3 change third file
`, err: nil}
	_ = model.highlightChanges()
	assert.False(t, model.rows[0].IsAffected)
	assert.True(t, model.rows[1].IsAffected)
}
