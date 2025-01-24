package globi

// equalStringSlices checks if two given string string slices are equal or not
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 { // lenb must also be 0 at this point
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
