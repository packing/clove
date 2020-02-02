package utils

import (
	"math/rand"
	"time"
)

func randomIntSlice(vs []int64) []int64 {
	rand.Seed(time.Now().Unix())

	if len(vs) <= 0 {
		return vs
	}

	newVs := make([]int64, len(vs))
	copy(newVs, vs)

	for i := len(newVs) - 1; i > 0; i-- {
		num := rand.Intn(i + 1)
		newVs[i], newVs[num] = newVs[num], newVs[i]
	}

	return newVs
}

func CreateNumPool(s, e int64, random bool) []int64 {
	c := e - s + 1
	l := make([]int64, c)
	for i := s; i <= e; i ++ {
		l[i - s] = i
	}

	if random {
		return randomIntSlice(l)
	}
	return l
}