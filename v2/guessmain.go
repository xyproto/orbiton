package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/xyproto/files"
	"github.com/xyproto/orchideous"
)

var errNoMainFile = errors.New("no main file found")

// guessMainFileOfDirectory tries to detect the project type in the given directory
// and returns the path to the most likely "main" file to open for editing.
// It checks, in order:
//  1. C/C++ main source files (via orchideous)
//  2. Project-specific main files based on detected project markers
//  3. Common main source filenames (main.*, index.*, Program.*, app.*)
//  4. Build system files (Makefile, CMakeLists.txt, etc.)
//  5. README files
func guessMainFileOfDirectory(dir string) (string, error) {
	prevDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer os.Chdir(prevDir)

	// Try C/C++ detection via orchideous
	if mainSrc := orchideous.GetMainSourceFile(nil); mainSrc != "" {
		return filepath.Join(dir, mainSrc), nil
	}

	// Detect project type by marker files, then look for corresponding main files
	type projectHint struct {
		markers []string // files whose presence indicates this project type
		mains   []string // main source files to check, in priority order
	}

	projectHints := []projectHint{
		{[]string{"go.mod"}, []string{"main.go", "cmd/main.go"}},
		{[]string{"Cargo.toml"}, []string{"src/main.rs", "main.rs"}},
		{[]string{"package.json"}, []string{"index.ts", "index.js", "src/index.ts", "src/index.tsx", "src/index.js", "index.html"}},
		{[]string{"build.zig"}, []string{"src/main.zig", "main.zig"}},
		{[]string{"dub.json", "dub.sdl"}, []string{"source/app.d", "main.d"}},
		{[]string{"stack.yaml"}, []string{"app/Main.hs", "src/Main.hs", "Main.hs"}},
		{[]string{"pyproject.toml", "setup.py", "setup.cfg"}, []string{"main.py", "app.py", "__main__.py", "src/main.py"}},
		{[]string{"Gemfile"}, []string{"main.rb", "app.rb"}},
		{[]string{"mix.exs"}, []string{"lib/main.ex"}},
		{[]string{"pom.xml", "build.gradle", "build.gradle.kts"}, []string{"src/main/java/Main.java", "Main.java", "main.java"}},
		{[]string{"build.sbt"}, []string{"src/main/scala/Main.scala", "Main.scala", "main.scala"}},
		{[]string{"Package.swift"}, []string{"Sources/main.swift", "main.swift"}},
		{[]string{"Makefile.PL", "cpanfile"}, []string{"lib/main.pl", "main.pl"}},
	}

	for _, ph := range projectHints {
		markerFound := slices.ContainsFunc(ph.markers, files.Exists)
		if markerFound {
			for _, main := range ph.mains {
				if files.Exists(main) {
					return filepath.Join(dir, main), nil
				}
			}
		}
	}

	// Haskell: *.cabal as project marker
	if matches, _ := filepath.Glob("*.cabal"); len(matches) > 0 {
		for _, f := range []string{"app/Main.hs", "src/Main.hs", "Main.hs"} {
			if files.Exists(f) {
				return filepath.Join(dir, f), nil
			}
		}
	}

	// Pascal/Lazarus: *.lpr as main file
	if matches, _ := filepath.Glob("*.lpr"); len(matches) > 0 {
		return filepath.Join(dir, matches[0]), nil
	}

	// Generic main source filenames, regardless of project markers
	genericMains := []string{
		"main.go", "main.rs", "main.zig", "main.d",
		"main.c", "main.cpp", "main.cc", "main.cxx",
		"main.py", "main.rb", "main.java", "main.kt", "main.scala",
		"main.pas", "main.pp",
		"main.lua", "main.nim", "main.odin", "main.v",
		"main.swift", "main.m",
		"Main.hs",
		"Program.cs", "Program.fs",
		"index.html", "index.js", "index.ts",
		"app.py", "app.rb", "app.js", "app.ts",
	}
	for _, name := range genericMains {
		if files.Exists(name) {
			return filepath.Join(dir, name), nil
		}
	}

	// Build system files
	buildFiles := []string{
		"Makefile", "CMakeLists.txt", "meson.build",
		"Justfile", "Taskfile.yml",
		"Cargo.toml", "go.mod", "build.zig",
		"package.json", "pom.xml",
		"build.gradle", "build.gradle.kts", "build.sbt",
		"pyproject.toml", "setup.py",
		"Gemfile", "mix.exs",
		"dub.json", "dub.sdl",
	}
	for _, name := range buildFiles {
		if files.Exists(name) {
			return filepath.Join(dir, name), nil
		}
	}

	// README files
	readmeFiles := []string{"README.md", "README.txt", "README.rst", "README"}
	for _, name := range readmeFiles {
		if files.Exists(name) {
			return filepath.Join(dir, name), nil
		}
	}

	return "", errNoMainFile
}
