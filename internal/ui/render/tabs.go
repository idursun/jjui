package render

import "charm.land/lipgloss/v2"

const defaultTabWidth = 4

// ExpandTabs converts tabs to spaces using the same behavior as lipgloss and
// the preview pane.
func ExpandTabs(s string) string {
	return lipgloss.NewStyle().TabWidth(defaultTabWidth).Render(s)
}
