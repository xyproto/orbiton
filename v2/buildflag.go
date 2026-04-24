package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/xyproto/files"
)

// autoBuildCandidate describes a language/ecosystem detectable from a
// marker file in the working directory, together with the command used
// to build the project.
type autoBuildCandidate struct {
	marker string   // filename that must exist (case-sensitive)
	tool   string   // executable required on PATH
	args   []string // arguments passed to the tool
	label  string   // human-readable name, shown on stderr
}

// autoBuildCandidates returns the project-type candidates, in priority order.
// The first candidate whose marker file exists *and* whose build tool is on
// PATH is used. The list intentionally excludes C/C++ (handled by orchideous)
// and skips anything that requires a source filename — only project-level
// builds go here, because `-b` is invoked without an explicit file.
func autoBuildCandidates() []autoBuildCandidate {
	return []autoBuildCandidate{
		{"go.mod", "go", []string{"build", "./..."}, "Go"},
		{"Cargo.toml", "cargo", []string{"build"}, "Rust"},
		{"build.zig", "zig", []string{"build"}, "Zig"},
		{"gleam.toml", "gleam", []string{"build"}, "Gleam"},
		{"shard.yml", "shards", []string{"build"}, "Crystal"},
		{"mix.exs", "mix", []string{"compile"}, "Elixir"},
		{"dune-project", "dune", []string{"build"}, "OCaml (dune)"},
		{"stack.yaml", "stack", []string{"build"}, "Haskell (stack)"},
		{"package.json", "npm", []string{"run", "build"}, "Node.js"},
		{"deno.json", "deno", []string{"task", "build"}, "Deno"},
		{"pom.xml", "mvn", []string{"compile"}, "Maven"},
		{"build.gradle", "gradle", []string{"build"}, "Gradle"},
		{"build.gradle.kts", "gradle", []string{"build"}, "Gradle (Kotlin)"},
		{"BUILD.bazel", "bazel", []string{"build", "//..."}, "Bazel"},
		{"WORKSPACE", "bazel", []string{"build", "//..."}, "Bazel"},
		// Hare: has a build.ha convention but usually "hare build" works from
		// any directory containing .ha files, so match the module.ha marker.
		{"module.ha", "hare", []string{"build"}, "Hare"},
		// Makefile last: many of the above also ship a Makefile wrapper and
		// we prefer the native tool when available.
		{"Makefile", "make", nil, "make"},
		{"makefile", "make", nil, "make"},
		{"GNUmakefile", "make", nil, "GNU make"},
	}
}

// tryAutoBuild runs the first matching language-specific build in the current
// working directory. Returns (true, err) if a build was attempted. Returns
// (false, nil) when no candidate matched, meaning the caller should fall back
// to orchideous (C/C++) dispatch.
func tryAutoBuild() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	for _, cand := range autoBuildCandidates() {
		if !files.IsFile(filepath.Join(cwd, cand.marker)) {
			continue
		}
		if !has(cand.tool) {
			continue
		}
		fmt.Fprintf(os.Stderr, "%s project detected, building with %s\n", cand.label, cand.tool)
		cmd := exec.Command(cand.tool, cand.args...)
		cmd.Dir = cwd
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return true, cmd.Run()
	}
	return false, nil
}
