package gfx

import (
	"math/rand"
	"sync"
)

var (
	randMu  sync.Mutex
	randGen = rand.New(rand.NewSource(1))
)

// RandSeed uses the provided seed value to initialize the gfx random number
// generator to a deterministic state. If RandSeed is not called, the
// generator behaves as if seeded by RandSeed(1).
//
// RandSeed is safe for concurrent use.
func RandSeed(seed int64) {
	randMu.Lock()
	randGen.Seed(seed)
	randMu.Unlock()
}

// RandIntn returns, as an int, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
func RandIntn(n int) int {
	randMu.Lock()
	defer randMu.Unlock()
	return randGen.Intn(n)
}

// RandFloat64 returns, as a float64, a pseudo-random number in [0.0,1.0).
func RandFloat64() float64 {
	randMu.Lock()
	defer randMu.Unlock()
	return randGen.Float64()
}
