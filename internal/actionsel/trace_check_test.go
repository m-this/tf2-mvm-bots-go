package actionsel_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
)

func TestTableFromTracingAgreesWithSelect(t *testing.T) {
	compared := 0
	for _, st := range actionsel.RoundStates() {
		for _, cl := range actionsel.Classes() {
			root := actionsel.Explore(st, cl)
			for bits := 0; bits < actionsel.AllFlags; bits++ {
				f := actionsel.FromBits(bits)
				if !actionsel.Reachable(cl, f) {
					continue
				}
				n := root
				for !n.Leaf {
					if f.Ask(n.Ask) {
						n = n.True
					} else {
						n = n.False
					}
				}
				if got, want := n.Action, actionsel.Select(st, cl, f); got != want {
					t.Fatalf("%v/%v/%+v: table says %v, Select says %v", st, cl, f, got, want)
				}
				compared++
			}
		}
	}
	t.Logf("compared %d reachable combinations", compared)
}
