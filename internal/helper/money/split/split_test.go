package split

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestRandomSplit(t *testing.T) {
	rng = rand.New(rand.NewSource(1))

	result := SplitRandom(20, 3)

	var sum int64
	for _, v := range result {
		sum += v
		if v < 6 || v > 7 {
			t.Fatalf("invalid split %d", v)
		}
	}

	if sum != 20 {
		t.Fatalf("expected sum 20, got %d", sum)
	}

	fmt.Println(result)
}
