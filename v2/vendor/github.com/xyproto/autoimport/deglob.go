package autoimport

import (
	"strings"
)

// DeGlob takes a string like "import java.util.*; // ArrayList" and returns "import java.util.ArrayList",
// for each class/type name that is listed as a comma separated list after "//".
func DeGlob(imports string) []string {
	if !strings.Contains(imports, ".*") {
		return []string{imports}
	}
	fields := strings.SplitN(imports, ".*", 2)
	left := strings.TrimSpace(fields[0])
	right := strings.TrimSpace(strings.TrimPrefix(fields[1], ";"))
	right = strings.TrimSpace(strings.TrimPrefix(right, "//"))
	var deGlobbed []string
	for _, className := range strings.Split(right, ",") {
		deGlobbed = append(deGlobbed, left+"."+strings.TrimSpace(className)+";")
	}
	return deGlobbed
}
