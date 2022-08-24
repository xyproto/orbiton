package guessica

// This file is extracted from the "getver" project, for automatically finding the newest
// version number for a given PKGBUILD file, by examining the corresponding web page.
// It has also been modified to fetch the latest git commit for the latest git version tag.
// This code is not particularly pretty and probably needs a good refactoring or two.

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// GuessSourceString is the function that is meant to be used from this source file
// It takes the contents of a PKGBUILD file and returns a new "source=" string.
// The new version number is guessed after looking online for a newer source.
// The git commit is included in the "source=" string, if possible.
// Returns the new pkgver and the new source.
// commonGitServer is often set to ie. "github.com"
func GuessSourceString(pkgbuildContents, commonGitServer string) (string, string, error) {
	lines := strings.Split(pkgbuildContents, "\n")
	var rawURL, rawSource string
	inSource := false
	for _, line := range lines {
		// First remove trailing comments
		if strings.Contains(line, " #") {
			parts := strings.SplitN(line, " #", 2)
			line = parts[0]
		}
		// Then check if we're in the source=() definition
		if inSource && len(strings.TrimSpace(line)) != 0 && !strings.Contains(line, "=") {
			rawSource += line
			continue
		} else {
			inSource = false
		}
		// Save url, pkgver, pkgrel and source
		if strings.HasPrefix(line, "url=") {
			rawURL = line[4:]
		} else if strings.HasPrefix(line, "source=") {
			rawSource = line[7:]
			inSource = true
		}
	}
	url := unquote(strings.TrimSpace(rawURL))

	if len(url) == 0 {
		return "", "", errors.New("found no URL definition")
	}

	if strings.Contains(url, commonGitServer+"/") && !strings.Contains(url, "/releases/") {
		if strings.HasSuffix(url, "/") {
			url += "releases/latest"
		} else {
			url += "/releases/latest"
		}
	}

	var (
		foundURL bool
		newVer   string
		err      error
	)

	// Should the source array URL be used instead of the "url=" field?
	if !strings.Contains(url, commonGitServer) && strings.Contains(rawSource, commonGitServer) {
		// Use the url from the source instead of the url field
		for _, sourceURL := range linkFinder.FindAllString(rawSource, -1) {
			if strings.Contains(sourceURL, "#") {
				sourceURL = strings.SplitN(sourceURL, "#", 2)[0]
			}
			sourceURL = strings.TrimSuffix(sourceURL, ".git")
			getverURL := sourceURL
			if strings.HasSuffix(getverURL, "/") {
				getverURL += "releases/latest"
			} else {
				getverURL += "/releases/latest"
			}
			newVer, err = getver(getverURL)
			if err == nil {
				// ok
				foundURL = true
				url = sourceURL
				break
			}
		}
	}

	if !foundURL {
		newVer, err = getver(url)
		if err != nil {
			return "", "", errors.New("could not guess a version number by visiting " + url)
		}
	}

	shortURL := url
	if strings.HasPrefix(url, "http://") {
		shortURL = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		shortURL = url[8:]
	}
	shortURL = strings.TrimSuffix(shortURL, "releases/latest")

	gotCommit := ""

	// git ls-remote https://github.com/xyproto/o 2.9.2

	tag := newVer
	cmd := exec.Command("git", "ls-remote", "-t", "https://"+shortURL, tag)
	data, err := cmd.CombinedOutput()
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		// Add a "v" in front of the tag
		cmd = exec.Command("git", "ls-remote", "-t", "https://"+shortURL, "v"+tag)
		data, err = cmd.CombinedOutput()
		if err != nil || len(bytes.TrimSpace(data)) == 0 {
			return "", "", errors.New("got no git commit hash from tag " + tag + " or tag v" + tag + " at " + shortURL)
		}
		gotCommit = strings.TrimSpace(string(data))
		tag = "v" + newVer
	} else {
		gotCommit = strings.TrimSpace(string(data))
	}

	if len(gotCommit) == 0 {
		return "", "", errors.New("got no git commit for tag " + tag + " or tag v" + tag)
	}

	fields := strings.Fields(gotCommit)
	if len(fields) > 0 {
		gotCommit = fields[0]
	}

	//fmt.Println("got commit: " + gotCommit)

	source := rawSource
	newSource := ""
	if len(gotCommit) != 0 && strings.Contains(source, "#commit=") {
		pos := strings.Index(source, "#commit=")
		if pos == -1 {
			return "", "", errors.New("found no #commit= in source")
		}
		pos += len("#commit=")
		if pos+len(gotCommit) < len(source) {
			// replace the existing commit hash, which is assumed to be as long as the new one
			newSource = source[:pos] + gotCommit + source[pos+len(gotCommit):]
		} else {
			// the existing commit has was too short, just replace the rest of the line
			newSource = source[:pos] + gotCommit + "\")"
		}
	}

	// add a tag commit
	if strings.HasSuffix(newSource, ")") {
		newSource += " # tag: " + tag
	}

	if len(newVer) == 0 {
		return "", "", errors.New("found no new version number")
	}

	return newVer, "source=" + newSource, nil
}
