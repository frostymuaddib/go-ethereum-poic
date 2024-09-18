package utils

import (
	"math/rand"
	"time"
)

//TODO: Proveriti logiku za radnom uint64!
const maxInt64 uint64 = 1<<63 - 1

func unint64n(n uint64, random *rand.Rand) uint64 {
	if n < maxInt64 {
		return uint64(random.Int63n(int64(n)))
	}
	x := random.Uint64()
	for x > n {
		x = rand.Uint64()
	}
	return x
}

// Int63n returns, as an int64, a non-negative pseudo-random number in [0,n).
// It panics if n <= 0.
// func unint64n(n uint64, random *rand.Rand) uint64 {
// 	if n&(n-1) == 0 { // n is power of two, can mask
// 		return random.Uint64() & (n - 1)
// 	}
// 	max := uint64((1 << 64) - 1 - uint64((1<<64)%n))
// 	v := random.Uint64()
// 	for v > max {
// 		v = random.Uint64()
// 	}
// 	return v % n
// }
type tableEl struct {
	X uint64
	Y uint64
}

// Perm returns, as a slice of n ints, a pseudo-random permutation of the integers [0,n).
func uintPerm(n uint64) []uint64 {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := make([]uint64, n)
	for i := uint64(0); i < n; i++ {
		j := unint64n(i+1, random)
		m[i] = m[j]
		m[j] = i
	}
	return m
}

func RandomPermutationFromRange(upperLimit uint64, numberOfElements uint64) []uint64 {
	perm := uintPerm(upperLimit)
	return perm[:numberOfElements]
}

func RandomPermutationFromRangeLessMemory(upperLimit uint64, numberOfElements uint64) []tableEl {

	count := numberOfElements

	result := make([]tableEl, numberOfElements)
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	cur := 0
	remaining := upperLimit
	for i := uint64(0); i < upperLimit && count > 0; i++ {
		probability := random.Float64()
		if probability < (float64(count) / float64(remaining)) {
			count--
			result[cur].X = i
			cur++
		}
		remaining--
	}
	return result
}
