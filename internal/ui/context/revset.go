package context

import "strings"

// NarrowedRevset adds condition to the current revset. The default revset is
// replaced instead, since what the condition matches often lies outside it.
func (ctx *MainContext) NarrowedRevset(condition string) string {
	current := strings.TrimSpace(ctx.CurrentRevset)
	if current == "" || current == ctx.DefaultRevset {
		return condition
	}
	// | binds looser than &, ranges and :: bind tighter.
	if strings.Contains(current, "|") {
		current = "(" + current + ")"
	}
	return current + " & " + condition
}
