package main

import (
	"fmt"

	"github.com/xyproto/vt100"
)

// TemplateProgram represents a string and cursor movement up, and then to the right
// which can be used to position the cursor after inserting a string.
type TemplateProgram struct {
	text  string
	up    int
	right int
}

var templatePrograms = map[Mode]TemplateProgram{
	modeGo: {
		"package main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n",
		2,
		13,
	},
	modeC: {
		"#include <stdio.h>\n#include <stdlib.h>\n\nint main(int argc, char* argv[])\n{\n\tprintf(\"Hello, World!\");\n\treturn EXIT_SUCCESS;\n}\n",
		3,
		8,
	},
	modeCpp: {
		"#include <cstdlib>\n#include <iostream>\n#include <string>\n\nusing namespace std::string_literals;\n\nint main(int argc, char** argv)\n{\n    std::cout << \"Hello, World!\"s << std::endl;\n    return EXIT_SUCCESS;\n}\n",
		3,
		14,
	},
	modeOdin: {
		"package main\n\nimport \"core:fmt\"\n\nmain :: proc() {\n    fmt.println(\"Hello, World!\");\n}\n",
		2,
		13,
	},
}

// HasTemplateProgram checks if a template is available for the current
// programming language, by looking at e.mode.
func (e *Editor) HasTemplateProgram() bool {
	_, found := templatePrograms[e.mode]
	return found
}

// InsertTemplateProgram will insert a template program at the current cursor position,
// if available. It will then reposition the cursor at an appropriate place in the template.
func (e *Editor) InsertTemplateProgram(c *vt100.Canvas) error {
	prog, found := templatePrograms[e.mode]
	if !found {
		return fmt.Errorf("could not find a template program for %s", e.mode)
	}

	// Insert the template program
	e.InsertStringAndMove(c, prog.text)

	// Move the cursor up and to the right
	for x := 0; x < prog.up; x++ {
		e.Up(c, nil)
	}
	for x := 0; x < prog.right; x++ {
		e.Next(c)
	}

	return nil
}
