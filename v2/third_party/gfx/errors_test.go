package gfx

import (
	"fmt"
	"testing"
)

func TestErrorf(t *testing.T) {
	err := Errorf("foo %d and bar %s", 123, "abc")

	if got, want := err.Error(), "foo 123 and bar abc"; got != want {
		t.Fatalf("err.Error() = %q, want %q", got, want)
	}
}

func ExampleErrorf() {
	err := Errorf("foo %d and bar %s", 123, "abc")

	fmt.Println(err)

	// Output:
	// foo 123 and bar abc
}
