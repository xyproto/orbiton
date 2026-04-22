package distrodetector

import "strings"

// AppleCodename returns a codename, or an empty string.
// Uses the lookup table below. On macOS, callers may additionally try
// codenameFromApple (see apple_darwin.go) to query Apple over HTTP.
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
		"14.0":  "Sonoma",
		"15.0":  "Sequoia",
		"26.0":  "Tahoe",
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
