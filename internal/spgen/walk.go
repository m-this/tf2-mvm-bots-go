package spgen

import (
	"fmt"
	"math"
	"strings"
)

// Walk is the Go twin of the loop the plugin runs: it asks for a predicate
// only when the walk reaches it, and it returns the outcome and the ids it
// asked for, in order.
func (t Table) Walk(point []int64, ask func(id int32) bool) (int32, []int32, error) {
	node, err := t.root(point)
	if err != nil {
		return 0, nil, err
	}
	asked := make([]int32, 0, len(t.Lazy.Predicates))
	for range len(t.Lazy.Predicates) + 1 {
		if t.Predicate[node] == leaf {
			return t.WhenTrue[node], asked, nil
		}
		id := t.Predicate[node]
		asked = append(asked, id)
		if ask(id) {
			node = t.WhenTrue[node]
		} else {
			node = t.WhenFalse[node]
		}
	}
	return 0, asked, fmt.Errorf("spgen: the walk asked more than %d predicates: the graph has a cycle", len(t.Lazy.Predicates))
}

func (t Table) root(point []int64) (int32, error) {
	if len(point) != len(t.Lazy.Axes) {
		return 0, fmt.Errorf("spgen: the walk was given %d axis values, want %d", len(point), len(t.Lazy.Axes))
	}
	index := 0
	for i, a := range t.Lazy.Axes {
		v := int(point[i])
		if v < 0 || v >= a.Size {
			return 0, fmt.Errorf("spgen: %s is %d, outside 0..%d", a.Name, v, a.Size-1)
		}
		index = index*a.Size + v
	}
	return t.Roots[index], nil
}

// Asked runs the entry function over a fully known input and reports the
// predicates it read, in order, with repeats collapsed. It is the ground truth
// the walk is held to: the walk may ask a subsequence of this and nothing more.
func (p *Package) Asked(lazy LazyTable, point []int64, known map[string]bool) ([]string, int32, error) {
	in := newInterp(p)
	asked := []string{}
	args := make([]value, 0, len(point)+1)
	for _, v := range point {
		args = append(args, value{kind: kindInt, i: v})
	}
	args = append(args, value{kind: kindStruct, fields: &fieldSet{known: known, asked: &asked}})
	out, stopped, err := in.run(lazy.Entry, args)
	if err != nil {
		return nil, 0, err
	}
	if stopped != "" {
		return nil, 0, fmt.Errorf("spgen: %s stopped on %s with every predicate known", lazy.Entry, stopped)
	}
	outcome, ok := asCell(out.i)
	if !ok {
		return nil, 0, fmt.Errorf("spgen: %s answered %d, which does not fit a cell", lazy.Entry, out.i)
	}
	return asked, outcome, nil
}

// asCell narrows an interpreted result to the 32 bits SourcePawn has.
func asCell(v int64) (int32, bool) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0, false
	}
	return int32(v), true
}

// IsSubsequence says whether every id in short appears in long, in order. It
// is the behaviour-equivalence check for the lazy walk: the walk must never
// ask for a predicate the source would not have asked for at that point.
func IsSubsequence(short, long []string) bool {
	i := 0
	for _, s := range long {
		if i < len(short) && short[i] == s {
			i++
		}
	}
	return i == len(short)
}

func (t Table) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d nodes, %d roots\n", len(t.Predicate), len(t.Roots))
	return b.String()
}
