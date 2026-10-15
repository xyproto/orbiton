package main

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/xyproto/files"
	"github.com/xyproto/globi"
)

// editPriority returns a sort weight for a filename when choosing which file to open.
// Lower values are preferred. Tiers:
//
//	0 - regular text file
//	1 - low-priority extension (.lock, .bak, ...)
//	2 - Windows-only script on a non-Windows platform (.bat, .cmd)
//	3 - binary file
func editPriority(filename string) int {
	if files.IsBinaryAccurate(filename) {
		return 3
	}
	if !isWindows && hasSuffix(filename, []string{".bat", ".cmd"}) {
		return 2
	}
	if hasSuffix(filename, probablyDoesNotWantToEditExtensions) || !strings.Contains(filename, ".") {
		return 1
	}
	return 0
}

// sortByEditPriority sorts filenames so the most desirable file to edit comes first.
// Within the same priority tier the names are ordered alphabetically.
func sortByEditPriority(matches []string) {
	sort.SliceStable(matches, func(i, j int) bool {
		pi, pj := editPriority(matches[i]), editPriority(matches[j])
		if pi != pj {
			return pi < pj
		}
		return matches[i] < matches[j]
	})
}

// generatedDirs are directories that usually hold build output, dependencies or
// VCS data, so files below them are rarely the ones the user meant to edit.
var generatedDirs = []string{".git", ".gradle", ".hg", ".idea", ".mvn", ".svn", ".tox", ".vscode", "__pycache__", "build", "dist", "node_modules", "out", "target", "vendor"}

// inGeneratedDir reports whether any element of the path is a generated directory
func inGeneratedDir(path string) bool {
	for part := range strings.SplitSeq(filepath.ToSlash(path), "/") {
		if slices.Contains(generatedDirs, part) {
			return true
		}
	}
	return false
}

// sortGlobMatches orders the matches from a recursive search so that the file the
// user most likely meant comes first. files.FindFile matches every filename that
// *contains* the given string, so searching for "Mapper.java" also matches a
// generated "Mapper.java.xml" report. Prefer an exact filename, then a path
// outside build output, then an editable file, then the shallowest path, and
// finally the alphabetically first one so the choice stays deterministic.
func sortGlobMatches(matches []string, substring string) {
	want := strings.ToLower(filepath.Base(substring))
	exact := func(path string) bool {
		return strings.ToLower(filepath.Base(path)) == want
	}
	depth := func(path string) int {
		return strings.Count(filepath.ToSlash(path), "/")
	}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if ea, eb := exact(a), exact(b); ea != eb {
			return ea
		}
		if ga, gb := inGeneratedDir(a), inGeneratedDir(b); ga != gb {
			return gb
		}
		if pa, pb := editPriority(a), editPriority(b); pa != pb {
			return pa < pb
		}
		if da, db := depth(a), depth(b); da != db {
			return da < db
		}
		return a < b
	})
}

// approximateFilename tries to find an existing file that matches the given
// incomplete filename, using glob expansion and weighted sorting.
// It returns the best match, or the original name if no match is found.
func approximateFilename(filename string) string {
	if strings.HasSuffix(filename, ".") {
		// Tab-completion left a trailing dot: glob for everything that starts with this prefix
		matches, err := globi.Glob(filename + "*")
		if err == nil && len(matches) > 0 {
			sortByEditPriority(matches)
			if len(matches[0]) > 0 {
				return matches[0]
			}
		}
	} else if !strings.Contains(filename, ".") && allLower(filename) {
		// No dot, all lowercase: more than one file may start with this name
		matches, err := globi.Glob(filename + "*")
		if err == nil && len(matches) > 1 {
			sortByEditPriority(matches)
			return matches[0]
		}
	} else {
		// Also match eg. "PKGBUILD" if just "Pk" was entered
		matches, err := globi.Glob(strings.ToTitle(filename) + "*")
		if err == nil && len(matches) >= 1 {
			sortByEditPriority(matches)
			return matches[0]
		}
	}
	return filename
}
