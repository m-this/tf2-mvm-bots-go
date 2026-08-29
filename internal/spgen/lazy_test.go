package spgen_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
)

// TestPredicateOrderMatchesTheDecision fails when a question is added to
// actionsel with no call written for it, which would otherwise emit a table
// whose ids mean something the edge does not know how to answer.
func TestPredicateOrderMatchesTheDecision(t *testing.T) {
	all := actionsel.Predicates()
	if len(all) != len(spgen.ActionSelPredicates) {
		t.Fatalf("actionsel asks %d questions, the edge answers %d", len(all), len(spgen.ActionSelPredicates))
	}
	for i, p := range all {
		if got := spgen.ActionSelPredicates[i]; got.Field != p.String() || got.Call != p.Call() {
			t.Errorf("predicate %d is %s/%s, the decision asks %s/%s", i, got.Field, got.Call, p, p.Call())
		}
	}
}

func actionSelTable(t *testing.T) spgen.Table {
	t.Helper()
	table, err := spgen.ActionSelTable()
	if err != nil {
		t.Fatal(err)
	}
	return table
}

// TestLazyWalkAgreesWithSelect walks the whole domain: the table has to answer
// what the decision answers, on every combination, or the plugin and the
// differential test are testing two different decisions.
func TestLazyWalkAgreesWithSelect(t *testing.T) {
	table := actionSelTable(t)

	compared := 0
	for p := range sweep {
		want := actionsel.Select(p.state, p.class, p.flags())
		got, _, err := table.Walk(p.axes(), func(id int32) bool { return p.bits&(1<<id) != 0 })
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if got != int32(want) {
			t.Fatalf("%s: the table walks to %d, Select answers %d", p, got, int32(want))
		}
		compared++
	}
	t.Logf("the lazy table agrees with Select on %d combinations", compared)
}

// TestTheWalkAsksNothingExtra is the behaviour-equivalence claim that matters
// for the port. Filling the predicates eagerly would call three that have side
// effects, so the walk has to ask a subsequence of what the decision itself
// reads on that same input: in the same order, and never something the
// decision never reached.
//
// The order it is held to comes from actionsel.AskOrder, which runs Select and
// records what it asks. It is the real evaluation order rather than one read
// off the source, so this is a stronger claim than it was.
func TestTheWalkAsksNothingExtra(t *testing.T) {
	table := actionSelTable(t)

	compared, extra := 0, 0
	for p := range sweep {
		source := actionsel.AskOrder(p.state, p.class, p.flags())
		_, asked, err := table.Walk(p.axes(), func(id int32) bool { return p.bits&(1<<id) != 0 })
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		walked := make([]actionsel.Predicate, 0, len(asked))
		for _, id := range asked {
			walked = append(walked, actionsel.Predicate(id))
		}
		if !spgen.IsSubsequence(walked, source) {
			extra++
			if extra <= 5 {
				t.Errorf("%s: the walk asks %v, the decision asks %v", p, walked, source)
			}
		}
		compared++
	}
	if extra > 5 {
		t.Errorf("%d further combinations where the walk asks for more than the decision", extra-5)
	}
	t.Logf("the walk asked no predicate the decision did not, over %d combinations", compared)
}
