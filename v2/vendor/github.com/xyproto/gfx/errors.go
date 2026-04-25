package gfx

// ErrDone can for example be returned when you are done rendering.
var ErrDone = Error("done")

// Error is a string that implements the error interface.
type Error string

// Error implements the error interface.
func (e Error) Error() string {
	return string(e)
}

// Errorf constructs a formatted error.
func Errorf(format string, a ...interface{}) error {
	return Error(Sprintf(format, a...))
}
