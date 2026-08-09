package main

import "testing"

// Function keys that are not bound to anything must be recognized, or they are
// inserted into the document as literal text by the paste branch in the key loop
func TestIsFunctionKey(t *testing.T) {
	for _, key := range []string{"F1", "F5", "F9", "F10", "F12"} {
		if !isFunctionKey(key) {
			t.Errorf("%q should be a function key", key)
		}
	}
	for _, key := range []string{"", "F", "F0", "F13", "F123", "Fx", "f1", "c:1", "Foo", "↑"} {
		if isFunctionKey(key) {
			t.Errorf("%q should not be a function key", key)
		}
	}
}
