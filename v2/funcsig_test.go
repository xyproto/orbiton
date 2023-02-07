package main

import (
	"testing"

	"github.com/xyproto/mode"
)

func TestFindFunctionSignatures(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Go

	foundSignatures := e.FindFunctionSignatures()

	// fmt.Println("Found these:")
	// for _, funcSig := range foundSignatures {
	// fmt.Println(funcSig)
	// }

	if len(foundSignatures) == 0 {
		t.Fail()
	}
}
