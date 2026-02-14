package actionargs

func BoolArg(args map[string]any, name string, fallback bool) bool {
	if args == nil {
		return fallback
	}
	raw, ok := args[name]
	if !ok {
		return fallback
	}
	v, ok := raw.(bool)
	if !ok {
		return fallback
	}
	return v
}
