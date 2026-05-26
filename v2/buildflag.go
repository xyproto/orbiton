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
	label  string   // human-readable name, shown on stderr
	args   []string // arguments passed to the tool
}

// autoBuildCandidates returns the project-type candidates, in priority order.
// The first candidate whose marker file exists *and* whose build tool is on
// PATH is used. The list intentionally excludes C/C++ (handled by slay)
// and skips anything that requires a source filename -- only project-level
// builds go here, because `-b` is invoked without an explicit file.
func autoBuildCandidates() []autoBuildCandidate {
	return []autoBuildCandidate{
		{marker: "go.mod", tool: "go", label: "Go", args: []string{"build", "./..."}},
		{marker: "Cargo.toml", tool: "cargo", label: "Rust", args: []string{"build"}},
		{marker: "build.zig", tool: "zig", label: "Zig", args: []string{"build"}},
		{marker: "gleam.toml", tool: "gleam", label: "Gleam", args: []string{"build"}},
		{marker: "shard.yml", tool: "shards", label: "Crystal", args: []string{"build"}},
		{marker: "mix.exs", tool: "mix", label: "Elixir", args: []string{"compile"}},
		{marker: "dune-project", tool: "dune", label: "OCaml (dune)", args: []string{"build"}},
		{marker: "stack.yaml", tool: "stack", label: "Haskell (stack)", args: []string{"build"}},
		{marker: "package.json", tool: "npm", label: "Node.js", args: []string{"run", "build"}},
		{marker: "deno.json", tool: "deno", label: "Deno", args: []string{"task", "build"}},
		{marker: "pom.xml", tool: "mvn", label: "Maven", args: []string{"compile"}},
		{marker: "build.gradle", tool: "gradle", label: "Gradle", args: []string{"build"}},
		{marker: "build.gradle.kts", tool: "gradle", label: "Gradle (Kotlin)", args: []string{"build"}},
		{marker: "BUILD.bazel", tool: "bazel", label: "Bazel", args: []string{"build", "//..."}},
		{marker: "WORKSPACE", tool: "bazel", label: "Bazel", args: []string{"build", "//..."}},
		// Hare: has a build.ha convention but usually "hare build" works from
		// any directory containing .ha files, so match the module.ha marker.
		{marker: "module.ha", tool: "hare", label: "Hare", args: []string{"build"}},
		// Makefile last: many of the above also ship a Makefile wrapper and
		// we prefer the native tool when available.
		{marker: "Makefile", tool: "make", label: "make"},
		{marker: "makefile", tool: "make", label: "make"},
		{marker: "GNUmakefile", tool: "make", label: "GNU make"},
	}
}

// tryAutoBuild runs the first matching language-specific build in the current
// working directory. Returns (true, err) if a build was attempted. Returns
// (false, nil) when no candidate matched, meaning the caller should fall back
// to slay (C/C++) dispatch.
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
