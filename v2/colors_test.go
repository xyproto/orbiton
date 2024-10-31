package main

import (
	"fmt"
	"testing"
)

var ac = NewAttributeColor("Bright", "Blue")

func TestBackground(t *testing.T) {
	if BackgroundBlue.String() != Blue.Background().String() {
		fmt.Println("BLUE BG IS NOT BLUE BG")
		fmt.Println(BackgroundBlue.String() + "FIRST" + Stop())
		fmt.Println(Blue.Background().String() + "SECOND" + Stop())
		t.Fail()
	}
}

func TestInts(t *testing.T) {
	ai := BackgroundBlue.Ints()
	bi := Blue.Background().Ints()
	if len(ai) != len(bi) {
		fmt.Println("A", ai)
		fmt.Println("B", bi)
		fmt.Println("length mismatch")
		t.Fail()
	}
	for i := 0; i < len(ai); i++ {
		if ai[i] != bi[i] {
			fmt.Println("NO")
			t.Fail()
		}
	}
}

func BenchmarkNewAttributeColor(b *testing.B) {
	for n := 0; n < b.N; n++ {
		NewAttributeColor("Bright", "Blue")
	}
}

func BenchmarkHead(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Head()
	}
}

func BenchmarkTail(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Tail()
	}
}

func BenchmarkBackground(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Background()
	}
}

func BenchmarkStartStop(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.StartStop("test")
	}
}

func BenchmarkGet(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Get("test")
	}
}

func BenchmarkStart(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Start("test")
	}
}

func BenchmarkStop(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Stop("test")
	}
}

func BenchmarkCombine(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Combine(NewAttributeColor("Red"))
	}
}

func BenchmarkBright(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Bright()
	}
}

func BenchmarkInts(b *testing.B) {
	for n := 0; n < b.N; n++ {
		ac.Ints()
	}
}

func BenchmarkS2b(b *testing.B) {
	for n := 0; n < b.N; n++ {
		s2b("Blue")
	}
}

func BenchmarkB2s(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b2s(34) // 34 corresponds to "Blue" in s2b function
	}
}
