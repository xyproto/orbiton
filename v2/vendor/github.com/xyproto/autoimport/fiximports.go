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

// ImportBlock generates "import" lines for the given Java or Kotlin source code
func (ima *ImportMatcher) ImportBlock(data []byte, verbose bool) ([]byte, error) {
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
			if strings.HasPrefix(foundImport, "java.desktop.java.") {
				foundImport = strings.TrimPrefix(foundImport, "java.desktop.")
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
	var importLine string
	for k, v := range importMap {
		importLine = k + v
		if importLine != "" {
			importLines = append(importLines, importLine)
		}
	}
	sort.Strings(importLines)
	importBlock := strings.Join(importLines, "\n")
	return []byte(importBlock), nil
}

// FixImports generates sorted "import" lines for a .java or .kotlin file
// (the ImportMatcher should be configured to be either for Java or Kotlin as well).
// The existing imports (if any) are the replaced with the generated imports.
func (ima *ImportMatcher) FixImports(data []byte, verbose bool) ([]byte, error) {
	importBlockBytes, err := ima.ImportBlock(data, verbose)
	if err != nil {
		return nil, err
	}

	// Imports are found, now modify the given source code and return it

	hasImports := bytes.Contains(data, []byte("\nimport "))

	if hasImports && !ima.removeExistingImports {
		importMap := make(map[string]string)
		ForEachLineInData(data, func(line, trimmedLine string) {
			if strings.HasPrefix(trimmedLine, "import ") {
				key := trimmedLine
				if strings.Contains(key, ";") {
					fields := strings.SplitN(key, ";", 2)
					key = fields[0]
				}
				importMap[key] = trimmedLine
			}
		})
		if verbose {
			fmt.Println("Existing imports:")
			for _, v := range importMap {
				fmt.Println(v)
			}
		}
		ForEachLineInData(importBlockBytes, func(line, trimmedLine string) {
			if trimmedLine == "" {
				return // continue
			}
			key := trimmedLine
			if strings.Contains(key, ";") {
				fields := strings.SplitN(key, ";", 2)
				key = fields[0]
			}
			importMap[key] = trimmedLine
		})
		if verbose {
			fmt.Println("Existing and new imports:")
			for _, v := range importMap {
				fmt.Println(v)
			}
		}
		var importLines []string
		for _, trimmedLine := range importMap {
			importLines = append(importLines, trimmedLine)
		}
		sort.Strings(importLines)
		if verbose {
			fmt.Println("Existing and new imports, sorted:")
			for _, importLine := range importLines {
				fmt.Println(importLine)
			}
		}
		// We now have a new import block that keeps the old imports, but not duplicates
		importBlockBytes = []byte(strings.Join(importLines, "\n"))
	}

	// Now replace/insert the newly organized import statements

	var (
		sb               strings.Builder
		importsDone      bool
		ignoreBlankLines int
	)
	ForEachLineInData(data, func(line, trimmedLine string) {
		if ignoreBlankLines > 0 {
			if trimmedLine == "" {
				ignoreBlankLines--
				return // continue
			}
			ignoreBlankLines = 0
		}
		if hasImports && strings.HasPrefix(trimmedLine, "import ") {
			if !importsDone {
				sb.Write(importBlockBytes)
				sb.WriteString("\n")
				importsDone = true
				ignoreBlankLines = 2
			} // else ignore this "import" line
		} else if !hasImports && strings.HasPrefix(trimmedLine, "package ") {
			sb.WriteString(line + "\n")
			sb.Write(importBlockBytes)
			sb.WriteString("\n")
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
