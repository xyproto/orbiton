package megafile

// Action is an "enum" for Orbiton to request an action via stderr
type Action int

const (
	// Returned/requested actions

	// NoAction - take no action
	NoAction = iota
	// NextFile - go to the next file
	NextFile
	// PreviousFile - go to the previous file
	PreviousFile
	// StopParent - stop the parent process
	StopParent
)
