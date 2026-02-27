//go:build darwin

package orchideous

import (
	"fmt"
	"os"
)

// platformHints outputs macOS-specific hints for includes that require
// different header paths than their Linux equivalents.
func platformHints(missingIncludes []string) {
	for _, inc := range missingIncludes {
		switch inc {
		case "GL/glut.h":
			fmt.Fprintln(os.Stderr, `
NOTE: On macOS, include GLUT/glut.h instead of GL/glut.h.

Suggested code:

    #ifdef __APPLE__
    #include <GLUT/glut.h>
    #else
    #include <GL/glut.h>
    #endif`)
		case "GL/gl.h":
			fmt.Fprintln(os.Stderr, `
NOTE: On macOS, include OpenGL/gl.h instead of GL/gl.h.

Suggested code:

    #ifdef __APPLE__
    #include <OpenGL/gl.h>
    #else
    #include <GL/gl.h>
    #endif`)
		}
	}
}
