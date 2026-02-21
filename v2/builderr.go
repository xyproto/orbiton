package main

import "errors"

// BuildError holds one build error message and if the source location was found.
type BuildError struct {
	message        string
	jumpedToSource bool
}

// Error returns the wrapped build error message.
func (be *BuildError) Error() string {
	return be.message
}

// JumpedToSource returns true if error parsing could jump to a source line.
func (be *BuildError) JumpedToSource() bool {
	return be.jumpedToSource
}

// newBuildError creates one build error.
func newBuildError(message string, jumpedToSource bool) error {
	return &BuildError{
		message:        message,
		jumpedToSource: jumpedToSource,
	}
}

// buildErrorJumpedToSource checks if the given error is a BuildError with source jump info.
func buildErrorJumpedToSource(err error) bool {
	var buildErr *BuildError
	return errors.As(err, &buildErr) && buildErr.JumpedToSource()
}
