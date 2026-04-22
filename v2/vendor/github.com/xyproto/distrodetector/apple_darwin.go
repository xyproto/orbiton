//go:build darwin
// +build darwin

package distrodetector

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

type versionInfo struct {
	Name       string `xml:"name"`
	ConfigCode string `xml:"configCode"`
	Locale     string `xml:"locale"`
}

// codenameFromApple attempts to fetch the correct codename from Apple,
// given a version string. The URL that is used is:
// https://support-sp.apple.com/sp/product?edid=%s
//
// Only compiled on macOS so non-Apple targets don't pull in net/http.
func codenameFromApple(version string) (string, error) {
	URL := "https://support-sp.apple.com/sp/product?edid=" + version
	resp, err := http.Get(URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var vi versionInfo
	xml.Unmarshal(data, &vi)
	if vi.ConfigCode == "" {
		return "", errors.New("No codename returned from " + URL)
	}
	codename := vi.ConfigCode
	if strings.HasPrefix(codename, "macOS ") {
		return codename[6:], nil
	}
	if strings.HasPrefix(codename, "OS X ") {
		return codename[5:], nil
	}
	return codename, nil
}
