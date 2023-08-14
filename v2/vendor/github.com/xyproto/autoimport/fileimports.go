package autoimport

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ForEachByteLine splits a file on '\n' and iterates over the byte slices
func ForEachByteLine(filename string, process func([]byte)) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read %s: %v", filename, err)
	}
	byteLines := bytes.Split(data, []byte{'\n'})
	for _, byteLine := range byteLines {
		process(byteLine)
	}
	return nil
}

// ForEachLine splits a file on '\n' and iterates over the lines.
// The callback function will be given each line and trimmed line as the function iterates.
func ForEachLine(filename string, process func(string, string)) error {
	return ForEachByteLine(filename, func(byteLine []byte) {
		process(string(byteLine), string(bytes.TrimSpace(byteLine)))
	})
}

// FileImports generates sorted "import" lines for a .java or .kotlin file
// (the ImportMatcher should be configured to be either for Java or Kotlin as well)
func (ima *ImportMatcher) FileImports(filename string, verbose bool) (string, error) {
	importMap := make(map[string]string) // from import path to comment (including "// ")
	skipWords := []string{"package", "public", "private", "protected"}
	var inComment bool
	if err := ForEachLine(filename, func(line, trimmedLine string) {
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
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	var importLines []string
	for k, v := range importMap {
		importLines = append(importLines, k+v)
	}
	sort.Strings(importLines)
	if verbose {
		fmt.Println()
	}
	return strings.Join(importLines, "\n"), nil
}
