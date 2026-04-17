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

// AppleCodename returns a codename, or an empty string.
// Will first use the lookup table, and then try to fetch it from Apple over HTTP.
func AppleCodename(version string) string {
	// See also: https://en.wikipedia.org/wiki/MacOS_version_history#Releases
	var appleCodeNames = map[string]string{
		"10.0":  "Cheetah",
		"10.1":  "Puma",
		"10.2":  "Jaguar",
		"10.3":  "Panther",
		"10.4":  "Tiger",
		"10.5":  "Leopard",
		"10.6":  "Snow Leopard",
		"10.7":  "Lion",
		"10.8":  "Mountain Lion",
		"10.9":  "Mavericks",
		"10.10": "Yosemite",
		"10.11": "El Capitan",
		"10.12": "Sierra",
		"10.13": "High Sierra",
		"10.14": "Mojave",
		"10.15": "Catalina",
		"11.0":  "Big Sur",
		"12.0":  "Monterey",
		"13.0":  "Ventura",
	}
	// Search the keys, longest keys first
	for keyLength := 5; keyLength >= 4; keyLength-- {
		for k, v := range appleCodeNames {
			if len(k) == keyLength {
				if strings.HasPrefix(version, k) {
					return v
				}
			}
		}
	}
	// No codename found, use one with a matching major version number
	majorVersionAndDot := version
	if strings.Contains(version, ".") {
		fields := strings.SplitN(version, ".", 2)
		majorVersionAndDot = fields[0] + "."
	}
	for k, v := range appleCodeNames {
		if strings.HasPrefix(k, majorVersionAndDot) {
			return v
		}
	}
	// No codename found
	return ""
}

// codenameFromApple attempts to fetch the correct codename from Apple,
// given a version string. The URL that is used is:
// https://support-sp.apple.com/sp/product?edid=%s
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
	//fmt.Println(string(data))
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
