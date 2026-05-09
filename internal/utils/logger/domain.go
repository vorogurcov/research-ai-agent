package logger

type Logger interface {
	WriteNonPrettified(text string) error
	WritePrettified(text string) error
}
