package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xyproto/env"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// TemplateProgram represents a string and cursor movement up, and then to the right
// which can be used to position the cursor after inserting a string.
type TemplateProgram struct {
	text  string
	right int
	up    int
}

var templatePrograms = map[mode.Mode]TemplateProgram{
	mode.Agda: {
		"module FILENAME where\n\nopen import Agda.Builtin.IO using (IO)\nopen import Agda.Builtin.Unit using (⊤)\nopen import Agda.Builtin.String using (String)\n\npostulate putStrLn : String → IO ⊤\n{-# FOREIGN GHC import qualified Data.Text as T #-}\n{-# COMPILE GHC putStrLn = putStrLn . T.unpack #-}\n\nmain : IO ⊤\nmain = putStrLn \"Hello, World!\"\n",
		17,
		1,
	},
	mode.C: {
		"#include <stdio.h>\n#include <stdlib.h>\n\nint main(int argc, char* argv[])\n{\n\tprintf(\"%s\\n\", \"Hello, World!\");\n\treturn EXIT_SUCCESS;\n}\n",
		8,
		3,
	},
	mode.Cpp: {
		"#include <cstdlib>\n#include <iostream>\n#include <string>\n\nusing namespace std::string_literals;\n\nint main(int argc, char** argv)\n{\n    std::cout << \"Hello, World!\"s << std::endl;\n    return EXIT_SUCCESS;\n}\n",
		14,
		3,
	},
	mode.Clojure: {
		"(ns example.hello\n  (:gen-class))\n\n(defn hello-world []\n  (println \"Hello, World!\"))\n\n(hello-world)\n",
		10,
		3,
	},
	mode.Crystal: {
		"class Greeter\n  def initialize(@name : String)\n  end\n\n  def greet\n    puts \"Hello, #{@name}!\"\n  end\nend\n\nGreeter.new(\"World\").greet\n",
		6,
		5,
	},
	mode.CS: {
		"using System;\n\nclass Greeter {\n    public static void Main(string[] args) {\n        Console.WriteLine(\"Hello, World!\");\n    }\n}\n",
		19,
		3,
	},
	mode.D: {
		"module main;\n\nimport std.stdio;\n\nvoid main(string[] args) {\n    writeln(\"Hello, World!\");\n}\n",
		9,
		2,
	},
	mode.Email: {
		"Hello ,\n\nBest regards,\n" + getFullName() + "\n",
		6,
		5,
	},
	mode.Erlang: {
		"-module(hello).\n-export([hello_world/0]).\n\nhello_world() -> io:fwrite(\"hello, world\\n\").\n",
		29,
		2,
	},
	mode.Garnet: {
		"fn main(): {} =\n    __print_str(\"Hello, World!\")\nend\n",
		10,
		2,
	},
	mode.GDScript: { // WIP
		"extends Node\n\nexport var x = 42\nvar y = 64\n\nfunc _ready():\n\ty = 96\n\nfunc _process(delta):\n\ty = 128\n\n",
		6,
		5,
	},
	mode.Go: {
		"package main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n",
		13,
		2,
	},
	mode.Haxe: {
        "class Main {\n    static public function main():Void {\n        trace(\"Hello, World!\");\n    }\n}",
		7,
		2,
	},
	mode.Hare: {
		"use fmt;\n\nexport fn main() void = {\n	fmt::println(\"Hello, World!\")!;\n};\n",
		14,
		2,
	},
	mode.Haskell: {
		"main :: IO ()\nmain = putStrLn \"Hello, World!\"\n",
		17,
		1,
	},
	mode.HTML: {
		"<!doctype html>\n<html lang=\"en\">\n  <head>\n    <meta charset=\"utf-8\">\n    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n    <title>Hello</title>\n    <meta name=\"description\" content=\"Hello\">\n    <link rel=\"shortcut icon\" href=\"https://www.iconsdb.com/icons/download/orange/teapot-16.png\">\n    <link rel=\"stylesheet\" href=\"https://unpkg.com/@picocss/pico@latest/css/pico.classless.min.css\">\n  </head>\n  <body>\n    <header>\n      <hgroup>\n        <h1>Greetings</h1>\n        <h2>About to greet <code>the world</code></h2>\n      </hgroup>\n    </header>\n    <main>\n      <section id=\"hello\">\n        <h2>Hello, World!</h2>\n        Task completed.\n      </section>\n    </main>\n    <footer>\n      All done.\n    </footer>\n  </body>\n</html>\n",
		4,
		9,
	},
	mode.Jakt: {
		"function main() {\n    println(\"Hello, World!\")\n}\n",
		9,
		2,
	},
	mode.Java: {
		"class Greeter {\n    public static void main(String[] args) {\n        System.out.println(\"Hello, World!\");\n    }\n}\n",
		20,
		3,
	},
	mode.JavaScript: {
		"console.log('Hello, World!');\n",
		13,
		1,
	},
	mode.Koka: {
		"fun main() {\n  println(\"Hello, World!\")\n}\n",
		9,
		2,
	},
	mode.Kotlin: {
		"fun main() {\n    println(\"Hello, World!\")\n}\n",
		9,
		2,
	},
	mode.Lua: {
		"print(\"Hello, World!\")\n",
		7,
		1,
	},
	mode.Nim: {
		"echo \"Hello, World!\"\n",
		6,
		1,
	},
	mode.ObjectPascal: {
		"program Hello;\nconst\n  greeting = 'Hello, World!';\nbegin\n  writeln(greeting);\nend.\n",
		12,
		4,
	},
	mode.OCaml: {
		"print_string \"Hello world!\\n\"",
		14,
		2,
	},
	mode.Odin: {
		"package main\n\nimport \"core:fmt\"\n\nmain :: proc() {\n    fmt.println(\"Hello, World!\");\n}\n",
		13,
		2,
	},
	mode.Perl: {
		"#!/usr/bin/env perl\n\nuse strict;\nuse utf8;\nuse warnings;\n\nbinmode(\\*STDOUT, \":utf8\");\nbinmode(\\*STDIN,  \":utf8\");\nbinmode(\\*STDERR, \":utf8\");\n\nsub main {\n  print(\"Hello, World!\\n\");\n}\n\nmain();\n",
		7,
		4,
	},
	mode.Python: {
		"#!/usr/bin/env python\n# -*- coding: utf-8 -*-\n\ndef main():\n    print(\"Hello, World!\")\n\n\nif __name__ == \"__main__\":\n    main()\n",
		7,
		5,
	},
	mode.R: {
		"print(\"Hello, World!\", quote=FALSE)",
		7,
		1,
	},
	mode.Rust: {
		"fn main() {\n    println!(\"Hello, World!\");\n}\n",
		10,
		2,
	},
	mode.Scala: {
		"object Hello {\n\tdef main(args: Array[String]) = {\n\t\tprintln(\"Hello, World!\")\n\t}\n}\n",
		9,
		3,
	},
	mode.Shell: {
		"# Maintainer: " + getFullName() + " <" + env.Str("EMAIL", "email") + ">\n\npkgname=\npkgver=1.0.0\npkgrel=1\npkgdesc='Example application'\narch=(x86_64)\nurl='https://github.com/example/application'\nlicense=(BSD3)\nmakedepends=(git go)\nsource=(\"git+$url#commit=asdf\") # tag: v1.0.0\nb2sums=(SKIP)\n\nbuild() {\n  cd $pkgname\n  go build -v -mod=vendor -buildmode=pie -trimpath -ldflags=\"-s -w -extldflags \\\"${LDFLAGS}\\\"\"\n}\n\npackage() {\n  install -Dm755 $pkgname/$pkgname \"$pkgdir/usr/bin/$pkgname\"\n  install -Dm644 $pkgname/LICENSE \"$pkgdir/usr/share/licenses/$pkgname/LICENSE\"\n}\n",
		8,
		20,
	},
	// This one is a bit more elaborate than strictly needed
	mode.StandardML: {
		"let\n  val name = \"World\"\nin\n  map (fn x => (print (\"Hello, \" ^ x ^ \"!\\n\"))) [name]\nend;",
		22,
		1,
	},
	mode.Teal: {
		"print(\"Hello, World!\")\n",
		7,
		1,
	},
	mode.TypeScript: {
		"console.log('Hello, World!');\n",
		13,
		1,
	},
	mode.V: {
		"fn main() {\n    name := 'World'\n    println('Hello, $name!')\n}\n",
		9,
		2,
	},
	mode.Zig: {
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

// BaseFilenameWithoutExtension returns the base filename, without the extension
// For instance, "/some/where/main.c" becomes just "main".
func (e *Editor) BaseFilenameWithoutExtension() string {
	baseFilename := filepath.Base(e.filename)
	ext := filepath.Ext(baseFilename)
	return strings.TrimSuffix(baseFilename, ext)
}

// InsertTemplateProgram will insert a template program at the current cursor position,
// if available. It will then reposition the cursor at an appropriate place in the template.
func (e *Editor) InsertTemplateProgram(c *vt100.Canvas) error {
	prog, found := templatePrograms[e.mode]
	if !found {
		return fmt.Errorf("could not find a template program for %s", e.mode)
	}

	baseFilenameWithoutExt := e.BaseFilenameWithoutExtension()

	// If the mode is Go and this is a test file, insert a test template instead
	if e.mode == mode.Go && strings.HasSuffix(baseFilenameWithoutExt, "_test") {
		prog = TemplateProgram{
			"package main\n\nimport (\n\t\"testing\"\n)\n\nfunc TestSomething(t *testing.T) {\n\tt.Fail()\n}\n",
			7,
			2,
		}
	}

	// Replace FILENAME with the base filename without extension, introduced because of Agda.
	prog.text = strings.ReplaceAll(prog.text, "FILENAME", e.BaseFilenameWithoutExtension())

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
