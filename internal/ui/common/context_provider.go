package common

type ContextProvider interface {
	GetContext() map[string]string
}
