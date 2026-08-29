package main

import (
	"slices"
	"testing"
)

// Every arm has to lead as often as it follows, or the crash column reads arm
// order instead of the feature. See mvm-p4x.
func TestNoArmKeepsTheFirstSlot(t *testing.T) {
	const arms, rounds = 2, 6

	led := make([]int, arms)
	for round := 1; round <= rounds; round++ {
		order := roundOrder(arms, round)

		if len(order) != arms {
			t.Fatalf("round %d played %d arms, not %d", round, len(order), arms)
		}
		for at := range arms {
			if !slices.Contains(order, at) {
				t.Fatalf("round %d never played arm %d", round, at)
			}
		}
		led[order[0]]++
	}

	for at, times := range led {
		if times != rounds/arms {
			t.Errorf("arm %d led %d rounds of %d, so the arms are not level", at, times, rounds)
		}
	}
}
