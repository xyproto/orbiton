package gfx

import "testing"

func TestIM(t *testing.T) {
	u := V(1, 2)
	p := IM.Project(u)

	if u != p {
		t.Fatalf("%v != %v", u, p)
	}
}
