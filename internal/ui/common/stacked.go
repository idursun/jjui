package common

// StackedModel is the contract for models presented in the stacked overlay.
// It includes the action owner so UI scope resolution can dispatch correctly.
type StackedModel interface {
	ImmediateModel
	StackedActionOwner() string
}
