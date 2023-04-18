//go:build !tinygo
// +build !tinygo

package gfx

import "math/rand"

// RandSeed uses the provided seed value to initialize the default Source to a
// deterministic state. If Seed is not called, the generator behaves as
// if seeded by Seed(1). Seed values that have the same remainder when
// divided by 2^31-1 generate the same pseudo-random sequence.
// RandSeed, unlike the Rand.Seed method, is safe for concurrent use.
func RandSeed(seed int64) {
	rand.Seed(seed)
}

// RandIntn returns, as an int, a non-negative pseudo-random number in [0,n)
// from the default Source.
// It panics if n <= 0.
func RandIntn(n int) int { return rand.Intn(n) }

// RandFloat64 returns, as a float64, a pseudo-random number in [0.0,1.0)
// from the default Source.
func RandFloat64() float64 { return rand.Float64() }
