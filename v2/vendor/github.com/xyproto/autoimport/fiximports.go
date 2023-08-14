package autoimport

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// ForEachByteLineInData splits data on '\n' and iterates over the byte slices
func ForEachByteLineInData(data []byte, process func([]byte)) {
	byteLines := bytes.Split(data, []byte{'\n'})
	for _, byteLine := range byteLines {
		process(byteLine)
	}
}

// ForEachLineInData splits data on '\n' and iterates over the lines.
// The callback function will be given each line and trimmed line as the function iterates.
func ForEachLineInData(data []byte, process func(string, string)) {
	ForEachByteLineInData(data, func(byteLine []byte) {
		process(string(byteLine), string(bytes.TrimSpace(byteLine)))
	})
}

// FileImports generates sorted "import" lines for a .java or .kotlin file
// (the ImportMatcher should be configured to be either for Java or Kotlin as well)
func (ima *ImportMatcher) FixImports(data []byte, verbose bool) ([]byte, error) {
	importMap := make(map[string]string) // from import path to comment (including "// ")
	skipWords := []string{"package", "public", "private", "protected"}
	var inComment bool
	ForEachLineInData(data, func(line, trimmedLine string) {
		if strings.HasPrefix(trimmedLine, "//") {
			return // continue
		}
		for _, skipWord := range skipWords {
			if strings.HasPrefix(trimmedLine, skipWord) {
				return // continue
			}
		}
		if !inComment && strings.Contains(trimmedLine, "/*") && !strings.Contains(trimmedLine, "*/") {
			inComment = true
		} else if inComment && strings.Contains(trimmedLine, "*/") && !strings.Contains(trimmedLine, "/*") {
			inComment = false
		}
		if inComment {
			return // continue
		}
		words := strings.Fields(trimmedLine)
		for _, word := range words {
			if strings.Contains(word, "(") {
				fields := strings.SplitN(word, "(", 2)
				word = strings.TrimSpace(fields[0])
			}
			if strings.Contains(word, "<") {
				fields := strings.SplitN(word, "<", 2)
				word = strings.TrimSpace(fields[0])
			}
			if word == "" {
				continue
			}
			foundImport := ima.StarPathExact(word)
			if foundImport == "java.lang.*" {
				continue
			}
			if foundImport != "" {
				key := "import " + foundImport + "; // "
				value := word
				if verbose {
					fmt.Printf("%s\t->\t%s%s\n", word, key, value)
				}
				if v, found := importMap[key]; found {
					if !strings.Contains(v, value) {
						newValues := v + ", " + value
						fields := strings.Split(newValues, ", ")
						sort.Strings(fields)
						importMap[key] = strings.Join(fields, ", ")
					}
				} else {
					importMap[key] = value
				}
			}
		}
	})
	var importLines []string
	for k, v := range importMap {
		importLines = append(importLines, k+v)
	}
	sort.Strings(importLines)
	if verbose {
		fmt.Println()
	}
	importBlock := strings.Join(importLines, "\n")

	// Imports are found, now modify the given source code and return it

	hasImports := bytes.Contains(data, []byte("\nimport "))

	importsDone := false
	var sb strings.Builder
	ForEachLineInData(data, func(line, trimmedLine string) {
		if hasImports && strings.HasPrefix(trimmedLine, "import ") {
			if !importsDone {
				sb.WriteString(importBlock + "\n")
				importsDone = true
			} // else ignore this "import" line
		} else if !hasImports && strings.HasPrefix(trimmedLine, "package ") {
			sb.WriteString(line + "\n\n")
			sb.WriteString(importBlock + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	})

	s := sb.String()

	// Remove the final blank line if the input bytes does not have one
	if !bytes.HasSuffix(data, []byte{'\n', '\n'}) && strings.HasSuffix(s, "\n\n") {
		s = s[:len(s)-1]
	}

	return []byte(s), nil
}
