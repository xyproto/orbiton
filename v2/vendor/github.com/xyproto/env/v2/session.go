package env

import "strings"

// WaylandSession returns true of XDG_SESSION_TYPE is "wayland" or if
// DESKTOP_SESSION contains "wayland".
func WaylandSession() bool {
	return Str("XDG_SESSION_TYPE") == "wayland" || strings.Contains(Str("DESKTOP_SESSION"), "wayland")
}

// XSession returns true if DISPLAY is set and WaylandSession() returns false.
func XSession() bool {
	return Has("DISPLAY") && !WaylandSession()
}

// XOrWaylandSession returns true if DISPLAY is set or WaylandSession() returns true.
func XOrWaylandSession() bool {
	return Has("DISPLAY") || WaylandSession()
}
