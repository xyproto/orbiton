package guessica

import (
	"os"
	"strings"
)

// SpecificSites adds extra support for examining specific URLs at certain git repository sites
var SpecificSites = []string{"github.com", "gitlab.com", "sr.ht"}

// UpdateFile will try to update the version in a given PKGBUILD filename
func UpdateFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	pkgbuildContents := string(data)

	// Try to guess the new version number/tag
	var ver, sourceLine string
	for _, site := range SpecificSites {
		if strings.Contains(pkgbuildContents, site) {
			ver, sourceLine, err = GuessSourceString(pkgbuildContents, site)
			if err == nil {
				break
			}
		}
	}

	// NOTE: the guessica utility uses its own code for this!

	// Build the new PKGBUILD contents
	var sb strings.Builder
	for _, line := range strings.Split(pkgbuildContents, "\n") {
		if strings.HasPrefix(line, "pkgver=") {
			sb.WriteString("pkgver=" + ver + "\n")
		} else if strings.HasPrefix(line, "pkgrel=") {
			sb.WriteString("pkgrel=1\n")
		} else if strings.HasPrefix(line, "source=") {
			sb.WriteString(sourceLine + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}
	// Write changes
	return os.WriteFile(filename, []byte(strings.TrimSpace(sb.String())), 0664)
}
