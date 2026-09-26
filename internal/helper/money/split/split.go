package split

import (
	"math/rand"
	"time"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func SplitRandom(amount int64, n int64) []int64 {
	if n <= 0 {
		panic("split by zero")
	}

	base := amount / n
	remainder := amount % n

	result := make([]int64, n)
	for i := int64(0); i < n; i++ {
		result[i] = base
	}

	indexes := make([]int, n)
	for i := 0; i < int(n); i++ {
		indexes[i] = i
	}

	rng.Shuffle(len(indexes), func(i, j int) {
		indexes[i], indexes[j] = indexes[j], indexes[i]
	})

	for i := int64(0); i < remainder; i++ {
		result[indexes[i]]++
	}

	return result
}
