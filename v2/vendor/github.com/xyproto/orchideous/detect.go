package orchideous

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SourceExts are the recognized C/C++ source file extensions.
var SourceExts = []string{".cpp", ".cc", ".cxx", ".c"}

// localIncludePaths are relative paths searched for project headers.
var localIncludePaths = []string{".", "include", "Include", "..", "../include", "../Include", "common", "Common", "../common", "../Common"}

// localCommonPaths are relative paths searched for shared source files.
var localCommonPaths = []string{"common", "Common", "../common", "../Common"}

// stdHeaders that should be skipped during dependency discovery.
var stdHeaders = map[string]bool{
	"assert.h": true, "complex.h": true, "ctype.h": true, "errno.h": true,
	"fenv.h": true, "float.h": true, "inttypes.h": true, "iso646.h": true,
	"limits.h": true, "locale.h": true, "math.h": true, "setjmp.h": true,
	"signal.h": true, "stdalign.h": true, "stdarg.h": true, "stdatomic.h": true,
	"stdbool.h": true, "stddef.h": true, "stdint.h": true, "stdio.h": true,
	"stdlib.h": true, "stdnoreturn.h": true, "string.h": true, "tgmath.h": true,
	"threads.h": true, "time.h": true, "uchar.h": true, "wchar.h": true,
	"wctype.h": true, "cstdlib": true, "csignal": true, "csetjmp": true,
	"cstdarg": true, "typeinfo": true, "typeindex": true, "type_traits": true,
	"bitset": true, "functional": true, "utility": true, "ctime": true,
	"chrono": true, "cstddef": true, "initializer_list": true, "tuple": true,
	"any": true, "optional": true, "variant": true, "new": true,
	"memory": true, "scoped_allocator": true, "memory_resource": true,
	"climits": true, "cfloat": true, "cstring": true, "cctype": true,
	"cstdint": true, "cinttypes": true, "limits": true, "exception": true,
	"stdexcept": true, "cassert": true, "system_error": true, "cerrno": true,
	"array": true, "vector": true, "deque": true, "list": true,
	"forward_list": true, "set": true, "map": true, "unordered_set": true,
	"unordered_map": true, "stack": true, "queue": true, "algorithm": true,
	"execution": true, "iterator": true, "cmath": true, "complex": true,
	"valarray": true, "random": true, "numeric": true, "ratio": true,
	"cfenv": true, "iosfwd": true, "ios": true, "istream": true,
	"ostream": true, "iostream": true, "fstream": true, "sstream": true,
	"iomanip": true, "streambuf": true, "cstdio": true, "locale": true,
	"clocale": true, "regex": true, "atomic": true, "thread": true,
	"mutex": true, "shared_mutex": true, "future": true,
	"condition_variable": true, "filesystem": true, "compare": true,
	"charconv": true, "syncstream": true, "strstream": true, "codecvt": true,
	"string": true, "windows.h": true, "format": true, "version": true,
	"source_location": true, "span": true, "ranges": true, "bit": true,
	"numbers": true, "stop_token": true, "semaphore": true, "latch": true,
	"barrier": true, "concepts": true, "coroutine": true, "stacktrace": true,
	"dlfcn.h": true, "pthread.h": true, "glibc": true,
}

// Project holds all detected project information.
type Project struct {
	MainSource    string
	DepSources    []string
	TestSources   []string
	Includes      []string // external includes from source files
	BoostLibs     []string
	IsC           bool // true if main source is a .c file
	HasOpenMP     bool
	HasBoost      bool
	HasQt6        bool
	HasMathLib    bool
	HasFS         bool
	HasThreads    bool
	HasWin64      bool // detected from #include <windows.h>
	HasGLFWVulkan bool // detected from #define GLFW_INCLUDE_VULKAN
	HasDlopen     bool // detected from #include <dlfcn.h>
}

// detectProject scans the current directory to detect the project layout.
func detectProject() Project {
	var p Project
	p.TestSources = getTestSources()
	p.MainSource = GetMainSourceFile(p.TestSources)
	p.DepSources = getDepSources(p.MainSource, p.TestSources)
	if strings.HasSuffix(p.MainSource, ".c") {
		p.IsC = true
	}

	// Scan ALL source files for special flags (not just main)
	allSources := []string{}
	if p.MainSource != "" {
		allSources = append(allSources, p.MainSource)
	}
	allSources = append(allSources, p.DepSources...)
	allSources = append(allSources, p.TestSources...)
	for _, src := range allSources {
		scanSourceForFlags(src, &p)
	}

	// Resolve common/ sources from includes (iteratively)
	p.resolveCommonDeps()

	// Collect external includes from all sources
	allSrcs := []string{}
	if p.MainSource != "" {
		allSrcs = append(allSrcs, p.MainSource)
	}
	allSrcs = append(allSrcs, p.DepSources...)
	allSrcs = append(allSrcs, p.TestSources...)
	p.Includes = collectExternalIncludes(allSrcs, p.HasWin64)

	return p
}

func scanSourceForFlags(filename string, p *Project) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.Contains(line, "#pragma omp") {
			p.HasOpenMP = true
		}
		if strings.Contains(line, "#include <boost/") {
			p.HasBoost = true
			// Try to detect boost library name from the include path
			// e.g. #include <boost/filesystem.hpp> -> boost_filesystem
			if _, after, ok := strings.Cut(line, "<boost/"); ok {
				rest := after
				if end := strings.IndexAny(rest, "./>"); end >= 0 {
					libName := "boost_" + rest[:end]
					p.BoostLibs = appendUnique(p.BoostLibs, libName)
				}
			}
		}
		if strings.Contains(line, "#include <QApplication") {
			p.HasQt6 = true
		}
		if strings.Contains(line, "#include <filesystem>") {
			p.HasFS = true
		}
		if trimmed == "#include <cmath>" || trimmed == `#include "math.h"` || trimmed == "#include <math.h>" {
			p.HasMathLib = true
		}
		if trimmed == "#include <thread>" || trimmed == "#include <pthread.h>" ||
			trimmed == "#include <mutex>" || trimmed == "#include <future>" ||
			trimmed == "#include <condition_variable>" || trimmed == "#include <shared_mutex>" {
			p.HasThreads = true
		}
		if trimmed == "#include <dlfcn.h>" {
			p.HasDlopen = true
		}
		// Detect win64 from includes
		for _, wh := range []string{`#include <windows.h>`, `#include "windows.h"`, `#include<windows.h>`} {
			if strings.Contains(line, wh) {
				p.HasWin64 = true
				break
			}
		}
		if strings.Contains(line, "#define GLFW_INCLUDE_VULKAN") {
			p.HasGLFWVulkan = true
		}
	}
}

// getTestSources returns all test source files.
func getTestSources() []string {
	var tests []string
	searchDirs := append([]string{"."}, localCommonPaths...)
	for _, dir := range searchDirs {
		for _, ext := range SourceExts {
			matches, _ := filepath.Glob(filepath.Join(dir, "*_test"+ext))
			tests = append(tests, matches...)
		}
		for _, ext := range SourceExts {
			name := filepath.Join(dir, "test"+ext)
			if fileExists(name) {
				tests = append(tests, name)
				break
			}
		}
	}
	return uniqueStrings(tests)
}

// GetMainSourceFile finds the main C/C++ source file in the current directory.
func GetMainSourceFile(testSrcs []string) string {
	// Check for explicit main.* files
	for _, ext := range SourceExts {
		name := "main" + ext
		if fileExists(name) {
			return name
		}
	}

	testMap := toSet(testSrcs)
	var allSrcs []string
	for _, ext := range SourceExts {
		matches, _ := filepath.Glob("*" + ext)
		for _, m := range matches {
			if !testMap[m] && !isTestFile(m) {
				allSrcs = append(allSrcs, m)
			}
		}
	}

	if len(allSrcs) == 0 {
		// Fallback: check src/ subdirectory
		for _, ext := range SourceExts {
			name := filepath.Join("src", "main"+ext)
			if fileExists(name) {
				return name
			}
		}
		for _, ext := range SourceExts {
			matches, _ := filepath.Glob(filepath.Join("src", "*"+ext))
			for _, m := range matches {
				if !isTestFile(m) {
					allSrcs = append(allSrcs, m)
				}
			}
		}
		if len(allSrcs) == 0 {
			return ""
		}
	}
	if len(allSrcs) == 1 {
		if containsMain(allSrcs[0]) {
			return allSrcs[0]
		}
		return ""
	}

	// Multiple candidates: pick the one containing main(
	for _, src := range allSrcs {
		if containsMain(src) {
			return src
		}
	}
	return ""
}

// getDepSources returns non-main, non-test source files.
func getDepSources(mainSrc string, testSrcs []string) []string {
	testMap := toSet(testSrcs)
	var deps []string
	for _, ext := range SourceExts {
		matches, _ := filepath.Glob("*" + ext)
		for _, m := range matches {
			if m != mainSrc && !testMap[m] && !isTestFile(m) {
				deps = append(deps, m)
			}
		}
	}
	// Also include common/ sources (excluding test files)
	for _, cp := range localCommonPaths {
		for _, ext := range SourceExts {
			matches, _ := filepath.Glob(filepath.Join(cp, "*"+ext))
			for _, m := range matches {
				if !isTestFile(m) {
					deps = append(deps, m)
				}
			}
		}
	}
	return uniqueStrings(deps)
}

// isTestFile returns true if the filename matches *_test.* or test.* patterns.
func isTestFile(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return strings.HasSuffix(name, "_test") || name == "test"
}

// containsMain checks if a source file contains a main function.
func containsMain(filename string) bool {
	data, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip single-line comments
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(line, " main(") || strings.HasPrefix(trimmed, "main(") ||
			strings.Contains(line, " SDL_main(") || strings.HasPrefix(trimmed, "SDL_main(") ||
			strings.Contains(line, " main (") || strings.HasPrefix(trimmed, "main (") {
			return true
		}
	}
	return false
}

// collectExternalIncludes parses source files for #include <...> directives
// and returns those that are not standard library or local headers.
// It first tries CPP preprocessing to resolve conditional includes,
// then falls back to direct text scanning.
func collectExternalIncludes(sourceFiles []string, win64 bool) []string {
	seen := make(map[string]bool)
	var result []string

	for _, sf := range sourceFiles {
		if sf == "" {
			continue
		}
		lines := cppPreprocessIncludes(sf)
		if lines == nil {
			// Fallback: scan directly
			lines = directScanIncludes(sf)
		}
		for _, inc := range lines {
			if stdHeaders[inc] {
				continue
			}
			if win64 && win64SkipHeaders[inc] {
				continue
			}
			if isLocalInclude(inc) {
				continue
			}
			if !seen[inc] {
				seen[inc] = true
				result = append(result, inc)
			}
		}
	}
	return result
}

// cppPreprocessIncludes runs the C preprocessor on a source file to resolve
// conditional includes, then extracts #include directives.
func cppPreprocessIncludes(filename string) []string {
	// Use the same trick as build.py: replace #include with a marker before cpp,
	// then restore after preprocessing to get the includes that survive conditionals.
	marker := "@@@@@"
	cmd := fmt.Sprintf(
		"LC_CTYPE=C LANG=C sed 's/^#include/%sinclude/g' < %q | cpp -E -P -w -pipe 2>/dev/null | sed 's/^%sinclude/#include/g'",
		marker, filename, marker)
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return nil
	}
	var includes []string
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#include") {
			continue
		}
		inc := ""
		if idx := strings.Index(line, "<"); idx >= 0 {
			if end := strings.Index(line[idx:], ">"); end >= 0 {
				inc = line[idx+1 : idx+end]
			}
		} else if strings.Count(line, "\"") >= 2 {
			parts := strings.SplitN(line, "\"", 3)
			if len(parts) >= 2 {
				inc = parts[1]
			}
		}
		if inc != "" {
			includes = append(includes, inc)
		}
	}
	return includes
}

// directScanIncludes scans a file directly for #include <...> directives.
func directScanIncludes(filename string) []string {
	f, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer f.Close()
	var includes []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#include") {
			continue
		}
		if idx := strings.Index(line, "<"); idx >= 0 {
			if end := strings.Index(line[idx:], ">"); end >= 0 {
				includes = append(includes, line[idx+1:idx+end])
			}
		}
	}
	return includes
}

// isLocalInclude checks if the include refers to a local project file.
func isLocalInclude(inc string) bool {
	for _, lp := range localIncludePaths {
		if fileExists(filepath.Join(lp, inc)) {
			return true
		}
	}
	return false
}

// resolveCommonDeps iteratively finds source files in common/ that correspond
// to included headers. Repeats until no new deps are discovered.
func (p *Project) resolveCommonDeps() {
	if p.MainSource == "" {
		return
	}
	for {
		allFiles := append([]string{p.MainSource}, p.DepSources...)
		allIncludes := collectLocalIncludes(allFiles)
		existingDeps := toSet(p.DepSources)
		foundNew := false

		for _, inc := range allIncludes {
			base := strings.TrimSuffix(inc, filepath.Ext(inc))
			for _, cp := range localCommonPaths {
				for _, ext := range SourceExts {
					candidate := filepath.Join(cp, base+ext)
					if fileExists(candidate) && !existingDeps[candidate] {
						p.DepSources = append(p.DepSources, candidate)
						existingDeps[candidate] = true
						foundNew = true
					}
				}
			}
		}
		if !foundNew {
			break
		}
	}
}

// collectLocalIncludes extracts #include "..." from source files and their included headers.
func collectLocalIncludes(files []string) []string {
	seen := make(map[string]bool)
	var result []string
	examined := make(map[string]bool)

	// Iteratively discover local includes from source and header files
	queue := make([]string, len(files))
	copy(queue, files)

	for len(queue) > 0 {
		sf := queue[0]
		queue = queue[1:]
		if sf == "" || examined[strings.ToLower(sf)] {
			continue
		}
		examined[strings.ToLower(sf)] = true

		f, err := os.Open(sf)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#include \"") {
				parts := strings.SplitN(line, "\"", 3)
				if len(parts) >= 2 {
					inc := parts[1]
					if !seen[inc] {
						seen[inc] = true
						result = append(result, inc)
						// Also scan the included header itself
						for _, lp := range localIncludePaths {
							headerPath := filepath.Join(lp, inc)
							if fileExists(headerPath) {
								queue = append(queue, headerPath)
								break
							}
						}
					}
				}
			}
		}
		f.Close()
	}
	return result
}

// executableName returns the name for the output executable.
func executableName() string {
	dir, err := os.Getwd()
	if err != nil {
		return "main"
	}
	name := filepath.Base(dir)
	if name == "src" {
		return "main"
	}
	return name
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func toSet(strs []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range strs {
		m[s] = true
	}
	return m
}

func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
