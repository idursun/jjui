package details

import tea "charm.land/bubbletea/v2"

func detailsObjectCursor() *tea.Cursor {
	cursor := tea.NewCursor(0, 0)
	cursor.Shape = tea.CursorBlock
	return cursor
}
