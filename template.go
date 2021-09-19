package main

import (
	"fmt"
	"strings"

	"github.com/xyproto/env"
	"github.com/xyproto/vt100"
)

// TemplateProgram represents a string and cursor movement up, and then to the right
// which can be used to position the cursor after inserting a string.
type TemplateProgram struct {
	text  string
	right int
	up    int
}

var templatePrograms = map[Mode]TemplateProgram{
	modeC: {
		"#include <stdio.h>\n#include <stdlib.h>\n\nint main(int argc, char* argv[])\n{\n\tprintf(\"Hello, World!\");\n\treturn EXIT_SUCCESS;\n}\n",
		8,
		3,
	},
	modeCpp: {
		"#include <cstdlib>\n#include <iostream>\n#include <string>\n\nusing namespace std::string_literals;\n\nint main(int argc, char** argv)\n{\n    std::cout << \"Hello, World!\"s << std::endl;\n    return EXIT_SUCCESS;\n}\n",
		14,
		3,
	},
	modeClojure: {
		"(ns example.hello\n  (:gen-class))\n\n(defn hello-world []\n  (println \"Hello, World!\"))\n\n(hello-world)\n",
		10,
		3,
	},
	modeCrystal: {
		"class Greeter\n  def initialize(@name : String)\n  end\n\n  def greet\n    puts \"Hello, #{@name}!\"\n  end\nend\n\nGreeter.new(\"World\").greet\n",
		6,
		5,
	},
	modeD: {
		"module main;\n\nimport std.stdio;\n\nvoid main(string[] args) {\n    writeln(\"Hello, World!\");\n}\n",
		2,
		1,
	},
	modeGo: {
		"package main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n",
		13,
		2,
	},
	modeObjectPascal: {
		"program Hello;\nconst\n  greeting = 'Hello, World!';\nbegin\n  writeln(greeting);\nend.\n",
		12,
		4,
	},
	modeOdin: {
		"package main\n\nimport \"core:fmt\"\n\nmain :: proc() {\n    fmt.println(\"Hello, World!\");\n}\n",
		13,
		2,
	},
	modePython: {
		"#!/usr/bin/env python\n# -*- coding: utf-8 -*-\n\ndef main():\n    print(\"Hello, World!\")\n\n\nif __name__ == \"__main__\":\n    main()\n",
		7,
		5,
	},
	modeRust: {
		"fn main() {\n    println!(\"Hello, World!\");\n}\n",
		10,
		2,
	},
	modeScala: {
		"object Hello {\n\tdef main(args: Array[String]) = {\n\t\tprintln(\"Hello, World!\")\n\t}\n}\n",
		9,
		3,
	},
	modeShell: {
		"# Maintainer: " + strings.Title(env.Str("LOGNAME", "name")) + " <" + env.Str("EMAIL", "email") + ">\n\npkgname=\npkgver=1.0.0\npkgrel=1\npkgdesc='Example application'\narch=(x86_64)\nurl='https://github.com/example/application'\nlicense=(BSD3)\nmakedepends=(git go)\nsource=(\"git+$url#commit=asdf\") # tag: v1.0.0\nb2sums=(SKIP)\n\nbuild() {\n  cd $pkgname\n  go build -v -mod=vendor -buildmode=pie -trimpath -ldflags=\"-s -w -extldflags \\\"${LDFLAGS}\\\"\"\n}\n\npackage() {\n  install -Dm755 $pkgname/$pkgname \"$pkgdir/usr/bin/$pkgname\"\n  install -Dm644 $pkgname/LICENSE \"$pkgdir/usr/share/licenses/$pkgname/LICENSE\"\n}\n",
		8,
		20,
	},
	modeV: {
		"fn main() {\n    name := 'World'\n    println('Hello, $name!')\n}\n",
		9,
		2,
	},
	modeZig: {
		"const std = @import(\"std\");\n\npub fn main() !void {\n    const stdout = std.io.getStdOut().writer();\n    try stdout.print(\"Hello, World!\\n\", .{});\n}\n",
		18,
		2,
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
