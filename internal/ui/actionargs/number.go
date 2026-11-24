package actionargs

func NumberArg(args map[string]any, name string, fallback float64) float64 {
	if args == nil {
		return fallback
	}
	switch v := args[name].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	default:
		return fallback
	}
}
