package revisions

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/ace_jump"
)

func (m *Model) IsAceJumping() bool {
	return m.aceJump != nil
}

func (m *Model) HandleAceJump(k tea.KeyMsg) tea.Cmd {
	if k.String() == tea.KeyEscape.String() {
		m.aceJump = nil
	} else if k.String() == tea.KeyEnter.String() {
		m.cursor = m.aceJump.First().RowIdx
		m.aceJump = nil
	} else if found := m.aceJump.Narrow(k); found != nil {
		m.cursor = found.RowIdx
		m.aceJump = nil
	}
	return nil
}

func (m *Model) findAceKeys() *ace_jump.AceJump {
	aj := ace_jump.NewAceJump()
	for i, row := range m.rows {
		c := row.Commit
		if c == nil {
			continue
		}
		aj.Append(i, c.CommitId, 0)
		if c.Hidden || c.IsConflicting() || c.IsRoot() {
			continue
		}
		aj.Append(i, c.ChangeId, 0)
	}
	return aj
}
