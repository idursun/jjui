package context

type RevsetContext struct {
	DefaultRevset string
	CurrentRevset string
}

func NewRevsetContext() *RevsetContext {
	return &RevsetContext{
		DefaultRevset: "all()",
		CurrentRevset: "all()",
	}
}
