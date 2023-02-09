package main

import "github.com/xyproto/env"

// Load environment variables into the env package cache before other "var" statements.
// The "init" function loads after all the global variables has been initialized, so this is used instead.
// This function must exist in the file that comes first in the alphabet, hence the "aardvark".
var _ bool = func() bool {
	env.Load()
	return true
}()
