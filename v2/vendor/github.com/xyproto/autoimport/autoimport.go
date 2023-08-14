package autoimport

import (
	"sort"
	"strings"
)

// FindImports can find words that looks like classes, and then look up
// appropriate import package paths. Ignores "java.lang." classes.
func (impM *ImportMatcher) FindImports(sourceCode string) []string {
	var foundImports []string
	for _, word := range unique(extractWords(sourceCode)) {
		foundPath := impM.ImportPathExact(word)
		if foundPath == "" {
			// fmt.Fprintf(os.Stderr, "could not find an import path for this word: %s (could be fine)\n", word)
			continue
		}
		if foundPath != "" && !strings.HasPrefix(foundPath, "java.lang.") {
			if !hasS(foundImports, foundPath) {
				foundImports = append(foundImports, foundPath)
			}
		}
	}
	return foundImports
}

// OrganizedImports generates import statements for packages that belongs to classes
// that are found in the given source code. If onlyJava is true, a semicolon is added
// after each line, and Kotlin jar files are not considered.
func (impM *ImportMatcher) OrganizedImports(sourceCode string, onlyJava bool) string {
	var sb strings.Builder
	imports := impM.FindImports(sourceCode)
	sort.Strings(imports)
	for _, importPackage := range imports {
		sb.WriteString("import ")
		sb.WriteString(importPackage)
		if onlyJava {
			sb.WriteString(";\n")
		} else {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
