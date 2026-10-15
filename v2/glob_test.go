package main

import "testing"

func TestInGeneratedDir(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"/p/src/main/java/com/example/Mapper.java", false},
		{"/p/build/reports/checkstyle/Mapper.java.xml", true},
		{"/p/target/classes/Mapper.java", true},
		{"/p/.git/objects/Mapper.java", true},
		{"/p/node_modules/pkg/Mapper.java", true},
		{"/p/rebuilder/Mapper.java", false}, // only whole path elements count
	} {
		if got := inGeneratedDir(tc.path); got != tc.want {
			t.Errorf("inGeneratedDir(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// A generated report like "Mapper.java.xml" also matches a search for
// "Mapper.java", and sorts before "src" alphabetically, so it used to win.
func TestSortGlobMatchesPrefersRealSource(t *testing.T) {
	matches := []string{
		"/p/build/reports/checkstyle/PensionDecisionMapper.java.xml",
		"/p/src/main/java/com/example/PensionDecisionMapper.java",
	}
	sortGlobMatches(matches, "PensionDecisionMapper.java")
	if want := "/p/src/main/java/com/example/PensionDecisionMapper.java"; matches[0] != want {
		t.Errorf("got %q, want %q", matches[0], want)
	}
}

func TestSortGlobMatchesExactNameBeatsShallowerPath(t *testing.T) {
	matches := []string{
		"/p/Mapper.java.bak",
		"/p/src/Mapper.java",
	}
	sortGlobMatches(matches, "Mapper.java")
	if want := "/p/src/Mapper.java"; matches[0] != want {
		t.Errorf("got %q, want %q", matches[0], want)
	}
}

func TestSortGlobMatchesGeneratedStillReachable(t *testing.T) {
	matches := []string{"/p/build/reports/Mapper.java.xml"}
	sortGlobMatches(matches, "Mapper.java")
	if want := "/p/build/reports/Mapper.java.xml"; matches[0] != want {
		t.Errorf("got %q, want %q", matches[0], want)
	}
}

// With several equally good candidates the choice must stay deterministic.
func TestSortGlobMatchesDeterministic(t *testing.T) {
	matches := []string{
		"/p/b/src/Mapper.java",
		"/p/a/src/Mapper.java",
	}
	sortGlobMatches(matches, "Mapper.java")
	if want := "/p/a/src/Mapper.java"; matches[0] != want {
		t.Errorf("got %q, want %q", matches[0], want)
	}
}
