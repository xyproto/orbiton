package main

import (
	"log"
	"testing"
)

func TestKeyHistory(t *testing.T) {
	kh := NewKeyHistory()
	if kh.Prev() != "" || kh.PrevPrev() != "" {
		log.Fatalln(kh.String())
	}
	kh.Push("a", nil)
	kh.Push("b", nil)
	if kh.PrevPrev() != "a" || kh.Prev() != "b" {
		log.Fatalln(kh.String())
	}
	kh.Push("c", nil)
	if kh.PrevPrev() != "b" || kh.Prev() != "c" {
		log.Fatalln(kh.String())
	}
	if !kh.OnlyIn("a", "b", "c") {
		t.Fail()
	}
	if !kh.OnlyIn("a", "b", "c", "d") {
		t.Fail()
	}
	if !kh.OnlyIn("a", "b", "c", "d", "e", "f") {
		t.Fail()
	}

}
