package spgen_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
)

// TestPredicateOrderMatchesTheStruct fails when a field is added to
// actionsel.Flags without a call written for it, which would otherwise emit a
// table whose ids mean something the edge does not know how to answer.
func TestPredicateOrderMatchesTheStruct(t *testing.T) {
	p, err := spgen.Load("../actionsel")
	if err != nil {
		t.Fatal(err)
	}
	fields, err := p.StructFields("Flags")
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != len(spgen.ActionSelPredicates) {
		t.Fatalf("Flags has %d fields, the edge answers %d", len(fields), len(spgen.ActionSelPredicates))
	}
	for i, f := range fields {
		if spgen.ActionSelPredicates[i].Field != f {
			t.Errorf("predicate %d is %s, the struct's field is %s", i, spgen.ActionSelPredicates[i].Field, f)
		}
	}
}

// TestLazyWalkAgreesWithSelect walks the whole domain: the table has to answer
// what the pure function answers, on every combination, or the plugin and the
// differential test are testing two different decisions.
func TestLazyWalkAgreesWithSelect(t *testing.T) {
	pkg, err := spgen.Load("../actionsel")
	if err != nil {
		t.Fatal(err)
	}
	table, err := pkg.Table(spgen.ActionSelLazy)
	if err != nil {
		t.Fatal(err)
	}

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
// for the port. Filling the struct eagerly would call three predicates that
// have side effects, so the walk has to ask for a subsequence of what the
// source itself reads on that same input: in the same order, and never for
// something the source never reached.
func TestTheWalkAsksNothingExtra(t *testing.T) {
	pkg, err := spgen.Load("../actionsel")
	if err != nil {
		t.Fatal(err)
	}
	table, err := pkg.Table(spgen.ActionSelLazy)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(spgen.ActionSelPredicates))
	for _, p := range spgen.ActionSelPredicates {
		names = append(names, p.Field)
	}

	compared, extra := 0, 0
	for p := range sweep {
		source, out, err := pkg.Asked(spgen.ActionSelLazy, p.axes(), knownOf(p.bits))
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if out != int32(actionsel.Select(p.state, p.class, p.flags())) {
			t.Fatalf("%s: the interpreter answers %d, the compiled Go answers %d", p, out, int32(actionsel.Select(p.state, p.class, p.flags())))
		}
		_, asked, err := table.Walk(p.axes(), func(id int32) bool { return p.bits&(1<<id) != 0 })
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		walked := make([]string, 0, len(asked))
		for _, id := range asked {
			walked = append(walked, names[id])
		}
		if !spgen.IsSubsequence(walked, source) {
			extra++
			if extra <= 5 {
				t.Errorf("%s: the walk asks %v, the source asks %v", p, walked, source)
			}
		}
		compared++
	}
	if extra > 5 {
		t.Errorf("%d further combinations where the walk asks for more than the source", extra-5)
	}
	t.Logf("the walk asked no predicate the source did not, over %d combinations", compared)
}
