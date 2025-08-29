package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/fullname"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// TemplateProgram represents a string and cursor movement up, and then to the right
// which can be used to position the cursor after inserting a string.
type TemplateProgram struct {
	text  string
	right int
	up    int
}

// TemplatePrograms maps from editor mode to simple example programs.
type TemplatePrograms map[mode.Mode]TemplateProgram

var templatePrograms TemplatePrograms

// GetTemplatePrograms will return a map from editor mode to per-programming-language template program.
// It is done this way to only initialize the map once, but not at the time when the program starts.
func GetTemplatePrograms() TemplatePrograms {
	if templatePrograms == nil {
		fullName := fullname.Get()
		// NOTE: Cursor coordinates are (X, -Y)
		templatePrograms = TemplatePrograms{
			mode.ABC: {
				"X:1\nT:ABC Template\nM:4/4\nL:1/8\nQ:1/4=80\nK:C\n%%MIDI program 62  % Synth Brass 1\nC C/2 C/2 E E/2 E/2 G G/2 G/2 c2\n",
				0,
				1,
			},
			mode.Agda: {
				"module FILENAME where\n\nopen import Agda.Builtin.IO using (IO)\nopen import Agda.Builtin.Unit using (⊤)\nopen import Agda.Builtin.String using (String)\n\npostulate putStrLn : String → IO ⊤\n{-# FOREIGN GHC import qualified Data.Text as T #-}\n{-# COMPILE GHC putStrLn = putStrLn . T.unpack #-}\n\nmain : IO ⊤\nmain = putStrLn \"Hello, World!\"\n",
				17,
				1,
			},
			mode.Algol68: {
				"print((\"Hello, World!\", newline))\n",
				8,
				1,
			},
			mode.Arduino: {
				"void setup() {\n  Serial.begin(9600);\n  Serial.println(\"Hello, World!\");\n}\n\nvoid loop() {\n}\n",
				16,
				5,
			},
			mode.C: {
				"#include <stdio.h>\n#include <stdlib.h>\n\nint main(int argc, char* argv[])\n{\n\tprintf(\"%s\\n\", \"Hello, World!\");\n\treturn EXIT_SUCCESS;\n}\n",
				8,
				3,
			},
			mode.C3: {
				"import std::io;\n\nfn void main()\n{\n    io::printn(\"Hello, World!\");\n}\n\n",
				12,
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
			mode.CMake: {
				"cmake_minimum_required(VERSION 3.5)\n\nproject(cmake-project-template)\n\nset(CMAKE_CXX_STANDARD 17)\nset(CMAKE_CXX_FLAGS \"${CMAKE_CXX_FLAGS} -std=c++17 -O2\")\n\nset(CMAKE_INSTALL_PREFIX ${PROJECT_SOURCE_DIR})\n\nset(DIVISIBLE_INSTALL_INCLUDE_DIR ${PROJECT_SOURCE_DIR}/include)\nset(DIVISIBLE_INSTALL_BIN_DIR ${PROJECT_SOURCE_DIR}/bin)\nset(DIVISIBLE_INSTALL_LIB_DIR ${PROJECT_SOURCE_DIR}/lib)\n\nset(DIVISION_HEADERS_DIR ${PROJECT_SOURCE_DIR}/src/division)\n\ninclude_directories(${DIVISIBLE_INSTALL_INCLUDE_DIR})\ninclude_directories(${DIVISION_HEADERS_DIR})\n\nadd_subdirectory(src)\nadd_subdirectory(test)\n",
				8,
				18,
			},
			mode.CS: {
				"using System;\n\nclass Greeter {\n    public static void Main(string[] args) {\n        Console.WriteLine(\"Hello, World!\");\n    }\n}\n",
				19,
				3,
			},
			mode.CSS: {
				"body, h1, h2, h3, h4, h5, h6, p, ul, ol, li, figure, figcaption, blockquote, dl, dd {\n  margin: 0;\n  padding: 0;\n}\n\nbody {\n  font-family: -apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, Oxygen-Sans, Ubuntu, Cantarell, \"Helvetica Neue\", sans-serif;\n  font-size: 16px; /* Good for readability */\n  line-height: 1.6; /* Good for readability */\n  color: #333; /* Sufficient contrast for readability */\n  background-color: #fff; /* Light background to ensure contrast */\n}\n\na, button {\n  color: #0066cc; /* Color should meet WCAG contrast ratio */\n  text-decoration: none; /* Underlines can be confusing for some dyslexic users */\n}\n\na:hover, a:focus, button:hover, button:focus {\n  text-decoration: underline; /* Indicate interactivity */\n  outline: none; /* Custom focus styles are more visually appealing */\n}\n\n.container {\n  width: 100%;\n  margin-right: auto;\n  margin-left: auto;\n  padding-right: 15px;\n  padding-left: 15px;\n}\n\n@media (min-width: 576px) { .container { max-width: 540px; } }\n@media (min-width: 768px) { .container { max-width: 720px; } }\n@media (min-width: 992px) { .container { max-width: 960px; } }\n@media (min-width: 1200px) { .container { max-width: 1140px; } }\n\n.padding-1 { padding: 0.25rem; }\n.padding-2 { padding: 0.5rem; }\n.padding-3 { padding: 1rem; }\n.margin-1 { margin: 0.25rem; }\n.margin-2 { margin: 0.5rem; }\n.margin-3 { margin: 1rem; }\n\n.flex-row {\n  display: flex;\n  flex-direction: row;\n}\n\n.flex-column {\n  display: flex;\n  flex-direction: column;\n}\n\n.justify-center {\n  justify-content: center;\n}\n\n.align-center {\n  align-items: center;\n}\n\nh1 { font-size: 2.25rem; }\nh2 { font-size: 1.8rem; }\nh3 { font-size: 1.5rem; }\nh4 { font-size: 1.2rem; }\nh5 { font-size: 1rem; }\nh6 { font-size: 0.875rem; }\n\n@media (max-width: 768px) {\n  h1 { font-size: 2rem; }\n  h2 { font-size: 1.75rem; }\n  h3 { font-size: 1.5rem; }\n}\n\n:focus-visible {\n  outline: 3px solid #ffbf47;\n  outline-offset: 3px;\n}\n\nlabel {\n  display: block;\n  margin-bottom: .5rem;\n}\n\ninput, select, textarea, button {\n  font: inherit; /* Ensure inputs use the same font */\n  color: inherit; /* Ensure inputs use the same color */\n  padding: .5rem;\n  margin-bottom: 1rem; /* Space out elements */\n  border: 1px solid #ccc; /* Slight border for definition */\n  border-radius: 4px; /* Modern look with rounded corners */\n}\n\ninput:focus, select:focus, textarea:focus, button:focus {\n  border-color: #0066cc;\n  box-shadow: 0 0 0 3px rgba(0,102,204,0.25);\n}\n\nimg, video {\n  max-width: 100%;\n  height: auto;\n}\n",
				1,
				1,
			},
			mode.D: {
				"module main;\n\nimport std.stdio;\n\nvoid main(string[] args) {\n    writeln(\"Hello, World!\");\n}\n",
				9,
				2,
			},
			mode.Dart: {
				"void main() {\n  print('Hello, World!');\n}\n",
				7,
				2,
			},
			mode.Email: {
				"Hello ,\n\nBest regards,\n" + fullName + "\n",
				6,
				5,
			},
			mode.Erlang: {
				"-module(hello).\n-export([hello_world/0]).\n\nhello_world() -> io:fwrite(\"hello, world\\n\").\n",
				29,
				2,
			},
			mode.Fortran77: {
				"* Fortran 77\n       PRINT *, 'Hello, World!'\n       END\n",
				10,
				2,
			},
			mode.Fortran90: {
				"program hello\n  ! Output the message\n  print *, 'Hello, World!'\nend program hello\n",
				10,
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
			mode.Ignore: {
				"# Binaries for programs and plugins\n*.exe\n*.dll\n*.so\n*.dylib\n*.test\n*.out\n\n# Object files and cached files\n*.o\n*.a\n*.bak\n\n# Build directories\nbin/\n\n# Logs and temp files\n*.log\n*.tmp\n*.swp\n*.bak\n*.log\n.DS_Store\n\n# Ignore editor or IDE directories\n.idea/\n.vscode/\n*.iml\n\n# cmd directory: ignore everything except Go source files and documentation\ncmd/*/*\n!cmd/*/*.go\n!cmd/*/*.txt\n!cmd/*/*.md\n!cmd/*/*.rst\n!cmd/*/*.asciidoc\n!cmd/*/*.1*\n\n# images not in img/\n*.bmp\n*.gif\n*.jpeg\n*.jpg\n*.png\n*.webp\n*.qoi\n!img/*\n\n# Orbiton include.txt file\ninclude.txt\n\n# Java and Kotlin\nbuild/\n.gradle/\n.kotlin/\n\n# Python\n__pycache__/\n*.py[cod]\nvenv/\n.venv/\n\n",
				0,
				0,
			},
			mode.Inko: {
				"import std.stdio.STDOUT\n\nclass async Main {\n  fn async main {\n    STDOUT.new.print('Hello, World!')\n  }\n}\n",
				18,
				3,
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
			mode.Just: {
				"#!/usr/bin/env just --justfile\n\nhello:\n\t@echo \"Hello, World!\"\n",
				7,
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
			mode.Lilypond: {
				"\\version \"2.24.2\"\n\n\\score {\n  \\relative c' {\n    c d e f g a b c\n  }\n}\n",
				0,
				3,
			},
			mode.Lua: {
				"print(\"Hello, World!\")\n",
				7,
				1,
			},
			mode.Markdown: {
				"# Title\n\n## Subtitle\n\ntext\n",
				4,
				1,
			},
			mode.Mojo: {
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
				"# Maintainer: " + fullName + " <" + env.Str("EMAIL", "email") + ">\n\npkgname=\npkgver=1.0.0\npkgrel=1\npkgdesc='Example application'\narch=(x86_64)\nurl='https://github.com/example/application'\nlicense=(BSD3)\nmakedepends=(git go)\nsource=(\"git+$url#commit=asdf\") # tag: v1.0.0\nb2sums=(SKIP)\n\nbuild() {\n  cd $pkgname\n  go build -v -mod=vendor -buildmode=pie -trimpath -ldflags=\"-s -w -extldflags \\\"${LDFLAGS}\\\"\"\n}\n\npackage() {\n  install -Dm755 $pkgname/$pkgname \"$pkgdir/usr/bin/$pkgname\"\n  install -Dm644 $pkgname/LICENSE \"$pkgdir/usr/share/licenses/$pkgname/LICENSE\"\n}\n",
				8,
				20,
			},
			// This one is a bit more elaborate than strictly needed
			mode.StandardML: {
				"let\n  val name = \"World\"\nin\n  map (fn x => (print (\"Hello, \" ^ x ^ \"!\\n\"))) [name]\nend;\n",
				22,
				2,
			},
			mode.SuperCollider: {
				"// Drum example from https://supercollider.github.io/examples.html\n{\n  var bdrum, hihat, snare;\n  var tempo = 4;\n  tempo = Impulse.ar(tempo);\n  snare = WhiteNoise.ar(Decay2.ar(PulseDivider.ar(tempo, 4, 2), 0.005, 0.5));\n  bdrum = SinOsc.ar(Line.ar(120,60, 1), 0, Decay2.ar(PulseDivider.ar(tempo, 4, 0), 0.005, 0.5));\n  hihat = HPF.ar(WhiteNoise.ar(1), 10000) * Decay2.ar(tempo, 0.005, 0.5);  \n  Out.ar(0, (snare + bdrum + hihat) * 0.4 ! 2)\n}.play\n",
				0,
				2,
			},
			mode.Swift: {
				"print(\"Hello, World!\")\n",
				7,
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
	}
	return templatePrograms
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
func (e *Editor) InsertTemplateProgram(c *vt.Canvas) error {
	prog, found := GetTemplatePrograms()[e.mode]
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
