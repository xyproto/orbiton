package main

import (
	"runtime"
	"sort"
	"strings"

	"github.com/xyproto/files"
	"github.com/xyproto/globi"
)

// editPriority returns a sort weight for a filename when choosing which file to open.
// Lower values are preferred. Tiers:
//
//	0 – regular text file
//	1 – low-priority extension (.lock, .bak, …)
//	2 – Windows-only script on a non-Windows platform (.bat, .cmd)
//	3 – binary file
func editPriority(filename string) int {
	if files.IsBinaryAccurate(filename) {
		return 3
	}
	if runtime.GOOS != "windows" && hasSuffix(filename, []string{".bat", ".cmd"}) {
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
