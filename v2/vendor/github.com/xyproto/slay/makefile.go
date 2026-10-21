package slay

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// When the project directory contains a Makefile with a "run:" rule whose
// dependencies are file targets, the build is delegated to make with those
// targets. This respects how the project splits its executables (the default
// goal may be an install target that must not run), and gives the executables
// the names the project author intended. The recipe of the "run:" rule also
// points out which executable is the one to run.

var (
	assignmentRe = regexp.MustCompile(`^[ \t]*[A-Za-z_][A-Za-z0-9_.-]*[ \t]*(::|\+=|\?=|!=|:=)?[ \t]*=`)
	targetRe     = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.+-]*$`)
	runTokenRe   = regexp.MustCompile(`\./([A-Za-z0-9_.+-]+)`)
)

// makeRule is one rule from a Makefile.
type makeRule struct {
	targets []string
	deps    []string
	recipe  []string
}

// parseMakeRules parses the targets and recipe lines of a Makefile.
// Only the dependencies of .PHONY are needed, returned separately.
func parseMakeRules(dir, filename string) (rules []makeRule, phony map[string]bool) {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return nil, nil
	}
	phony = map[string]bool{}
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, "\t") {
			// Recipe line for the latest rule
			if len(rules) > 0 {
				rules[len(rules)-1].recipe = append(rules[len(rules)-1].recipe, strings.TrimSpace(line))
			}
			continue
		}
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if assignmentRe.MatchString(s) {
			// Variable assignment, not a rule
			continue
		}
		// A rule line: "targets : deps". Multi-target rules are common.
		var targets, deps []string
		if idx := strings.Index(s, ":"); idx > 0 {
			left := strings.Fields(s[:idx])
			deps = strings.Fields(strings.SplitN(s[idx+1:], ";", 2)[0])
			if len(left) == 1 && left[0] == ".PHONY" {
				for _, dep := range deps {
					phony[dep] = true
				}
				continue
			}
			for _, f := range left {
				if !targetRe.MatchString(f) {
					targets = nil
					break
				}
				targets = append(targets, f)
			}
		}
		if len(targets) > 0 {
			rules = append(rules, makeRule{targets: targets, deps: deps})
		}
	}
	return rules, phony
}

// findMakefile returns the name of the Makefile-like file in the given
// directory, if any.
func findMakefile(dir string) string {
	for _, mf := range []string{"Makefile", "makefile", "GNUmakefile"} {
		if fileExists(filepath.Join(dir, mf)) {
			return mf
		}
	}
	return ""
}

// runTargets returns the file targets (non-phony) that the "run:" rule
// depends on, if the Makefile has a "run:" rule.
func runTargets(rules []makeRule, phony map[string]bool) []string {
	var targets []string
	for _, rule := range rules {
		if rule.targets[0] != "run" {
			continue
		}
		for _, dep := range rule.deps {
			if targetRe.MatchString(dep) && !phony[dep] {
				targets = append(targets, dep)
			}
		}
	}
	return targets
}

// runProgram scans the recipes of the parsed rules for the first
// "./program name" token, in the order the rules appear.
func runProgram(rules []makeRule) string {
	for _, rule := range rules {
		for _, line := range rule.recipe {
			if m := runTokenRe.FindStringSubmatch(line); len(m) > 1 {
				return m[1]
			}
		}
	}
	return ""
}

// newestBuiltFile returns the newest regular file in the current directory
// that is not an object or dependency file, and that was modified no earlier
// than startTime. Used as the last fallback for the produced executable.
func newestBuiltFile(startTime time.Time) string {
	var best string
	var bestMod time.Time
	entries, err := os.ReadDir(".")
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(name, ".o") || strings.HasSuffix(name, ".d") {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.ModTime().Before(startTime) {
			continue
		}
		if best == "" || info.ModTime().After(bestMod) {
			best = name
			bestMod = info.ModTime()
		}
	}
	return best
}

// buildWithMakefile invokes make with the targets that the "run:" rule depends
// on, and returns the combined output, the commands that were run and the
// produced executable. Returns ok=false when the route is not applicable, in
// which case the regular build should be performed instead.
func buildWithMakefile() (output []byte, cmds []string, exe string, ok bool, err error) {
	if _, err := exec.LookPath("make"); err != nil {
		return nil, nil, "", false, nil
	}
	makefile := findMakefile("")
	if makefile == "" {
		return nil, nil, "", false, nil
	}
	rules, phony := parseMakeRules("", makefile)
	targets := runTargets(rules, phony)
	if len(targets) == 0 {
		return nil, nil, "", false, nil
	}

	startTime := time.Now()
	args := append([]string{"-j" + strconv.Itoa(runtime.NumCPU())}, targets...)
	cmd := exec.Command("make", args...)
	out, runErr := cmd.CombinedOutput()
	cmds = []string{"make " + strings.Join(args, " ")}

	candidates := []string{runProgram(rules), firstExisting(targets)}
	if exe = firstExisting(candidates); exe == "" && runErr == nil {
		exe = newestBuiltFile(startTime)
	}
	return out, cmds, exe, true, runErr
}

// firstExisting returns the first name that exists as a file.
func firstExisting(names []string) string {
	for _, name := range names {
		if fileExists(name) {
			return name
		}
	}
	return ""
}

// extraMainName returns the executable name for an extra main source file.
func extraMainName(extraMain string) string {
	return strings.TrimSuffix(filepath.Base(extraMain), filepath.Ext(extraMain))
}

// buildExtraMains builds each additional source file that contains its own
// main( function as a separate executable. Each extra main is first linked by
// itself, then together with the shared dependency sources if needed.
func buildExtraMains(proj Project, flags BuildFlags) error {
	if proj.HasWin64 || proj.HasWatcom {
		return nil
	}
	for _, extraMain := range proj.ExtraMains {
		name := extraMainName(extraMain)
		if name == executableName() {
			continue
		}
		var err error
		for _, srcs := range extraMainAttempts(extraMain, proj) {
			compilerOutput.Reset()
			err = compileSources(srcs, name, flags)
			if err == nil {
				break
			}
			if retryFlags, retryOk := flagsForMissingHeaders(compilerOutput.String(), flags); retryOk {
				fmt.Println("Found pkg-config flags for the missing headers, compiling again")
				compilerOutput.Reset()
				if err = compileSources(srcs, name, retryFlags); err == nil {
					break
				}
			}
		}
		if err != nil {
			recommendPackage(proj.Includes)
			platformHints(proj.Includes)
			return fmt.Errorf("building %s: %w", name, err)
		}
	}
	return nil
}

// buildExtraMainsCaptured is like buildExtraMains, but captures all build
// output instead of printing it.
func buildExtraMainsCaptured(proj Project, flags BuildFlags) ([]byte, []string, error) {
	if proj.HasWin64 || proj.HasWatcom {
		return nil, nil, nil
	}
	var output []byte
	var cmds []string
	for _, extraMain := range proj.ExtraMains {
		name := extraMainName(extraMain)
		if name == executableName() {
			continue
		}
		var err error
		for _, srcs := range extraMainAttempts(extraMain, proj) {
			out, extraCmds, attemptErr := compileSourcesCaptured(srcs, name, flags)
			output = append(output, out...)
			cmds = append(cmds, extraCmds...)
			err = attemptErr
			if err == nil {
				break
			}
		}
		if err != nil {
			recommendPackage(proj.Includes)
			return output, cmds, fmt.Errorf("building %s: %w", name, err)
		}
	}
	return output, cmds, nil
}

// extraMainAttempts lists the ways an extra main can be linked: first by
// itself, since the shared sources may conflict with definitions that come
// with the extra main (like a duplicated data array), then together with the
// shared dependency sources.
func extraMainAttempts(extraMain string, proj Project) [][]string {
	return [][]string{
		{extraMain},
		append([]string{extraMain}, proj.DepSources...),
	}
}

// MakefileRunTargets returns the file targets that the "run:" rule of the
// Makefile in the current directory depends on. Returns ok=false when there
// is no Makefile with such a rule, in which case the default goal applies.
func MakefileRunTargets(dir string) ([]string, bool) {
	makefile := findMakefile(dir)
	if makefile == "" {
		return nil, false
	}
	rules, phony := parseMakeRules(dir, makefile)
	targets := runTargets(rules, phony)
	if len(targets) == 0 {
		return nil, false
	}
	return targets, true
}

// MakefileRunProgram returns the program that the recipe of the "run:" rule
// starts, like "preview" for a recipe that runs "./preview". If the recipe
// does not run a program from the current directory, the first existing
// run: target is returned instead.
func MakefileRunProgram(dir string) string {
	makefile := findMakefile(dir)
	if makefile == "" {
		return ""
	}
	rules, phony := parseMakeRules(dir, makefile)
	if program := runProgram(rules); program != "" {
		return program
	}
	targets := runTargets(rules, phony)
	if len(targets) == 0 {
		return ""
	}
	return firstExisting(targets)
}
