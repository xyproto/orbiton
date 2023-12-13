package main

import (
	"fmt"
)

func ExampleEditor_SortStrings() {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "example=(o a f g e b c d q h)")
	currentLine := e.CurrentLine()
	err := e.SortStrings()
	if err != nil {
		panic(err)
	}
	sortedLine := e.CurrentLine()
	fmt.Println(currentLine)
	fmt.Println(sortedLine)
	// Output:
	// example=(o a f g e b c d q h)
	// example=(a b c d e f g h o q)
}

func Example_sortStrings() {
	inputString := "example=(o a f g e b c d q h)"
	fmt.Println(inputString)
	sorted, err := sortStrings(inputString)
	if err != nil {
		panic(err)
	}
	fmt.Println(sorted)
	// Output:
	// example=(o a f g e b c d q h)
	// example=(a b c d e f g h o q)
}

func Example_sortStrings_2() {
	inputString := "example={o a f g e b c d q h}"
	fmt.Println(inputString)
	sorted, err := sortStrings(inputString)
	if err != nil {
		panic(err)
	}
	fmt.Println(sorted)
	// Output:
	// example={o a f g e b c d q h}
	// example={a b c d e f g h o q}
}

func Example_sortStrings_3() {
	inputString := "example={\"o\" a 'f' g 'e' b \"c\" d 'q' h}"
	fmt.Println(inputString)
	sorted, err := sortStrings(inputString)
	if err != nil {
		panic(err)
	}
	fmt.Println(sorted)
	// Output:
	// example={"o" a 'f' g 'e' b "c" d 'q' h}
	// example={a b "c" d 'e' 'f' g h "o" 'q'}
}

func Example_sortStrings_4() {
	inputString := "o a 'f' g 'e' b \"c\" d 'q' h"
	fmt.Println(inputString)
	sorted, err := sortStrings(inputString)
	if err != nil {
		panic(err)
	}
	fmt.Println(sorted)
	// Output:
	// o a 'f' g 'e' b "c" d 'q' h
	// a b "c" d 'e' 'f' g h o 'q'
}

func Example_sortStrings_5() {
	inputString := `addKeywords = []string{"z", "--force", "-f", "cmake", "configure", "fdisk", "gdisk", "install", "make", "mv", "ninja", "rm", "rmdir"}`
	fmt.Println(inputString)
	sorted, err := sortStrings(inputString)
	if err != nil {
		panic(err)
	}
	fmt.Println(sorted)
	// Output:
	// addKeywords = []string{"z", "--force", "-f", "cmake", "configure", "fdisk", "gdisk", "install", "make", "mv", "ninja", "rm", "rmdir"}
	// addKeywords = []string{"--force", "-f", "cmake", "configure", "fdisk", "gdisk", "install", "make", "mv", "ninja", "rm", "rmdir", "z"}
}
