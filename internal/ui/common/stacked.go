package common

import "github.com/idursun/jjui/internal/ui/routing"

// StackedModel is the contract for models presented in the stacked overlay.
type StackedModel interface {
	ImmediateModel
	routing.ScopeProvider
}
