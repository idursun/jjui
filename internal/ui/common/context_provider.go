package common

type ContextProvider interface {
	Read(value string) string
}
