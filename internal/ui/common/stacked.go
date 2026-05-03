package common

// StackedModel is the contract for models presented in the stacked overlay.
type StackedModel interface {
	ImmediateModel
	ScopeProvider
}
