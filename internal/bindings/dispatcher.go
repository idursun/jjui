package bindings

// Resolve finds the last matching binding for the given action whose condition
// matches the provided state. Bindings are scanned from the end to give later
// entries (user config) precedence.
func Resolve(action string, bindings []KeyBinding, state map[string]any) (KeyBinding, bool) {
	for i := len(bindings) - 1; i >= 0; i-- {
		b := bindings[i]
		if b.Action != action {
			continue
		}
		if b.When != "" && !b.Condition.Eval(state) {
			continue
		}
		return b, true
	}
	return KeyBinding{}, false
}
