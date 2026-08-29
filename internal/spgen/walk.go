package spgen

import (
	"fmt"
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
	asked := make([]int32, 0, len(t.Predicates))
	for range len(t.Predicates) + 1 {
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
	return 0, asked, fmt.Errorf("spgen: the walk asked more than %d predicates: the graph has a cycle", len(t.Predicates))
}

func (t Table) root(point []int64) (int32, error) {
	if len(point) != len(t.Axes) {
		return 0, fmt.Errorf("spgen: the walk was given %d axis values, want %d", len(point), len(t.Axes))
	}
	index := 0
	for i, a := range t.Axes {
		v := int(point[i])
		if v < 0 || v >= a.Size {
			return 0, fmt.Errorf("spgen: %s is %d, outside 0..%d", a.Name, v, a.Size-1)
		}
		index = index*a.Size + v
	}
	return t.Roots[index], nil
}

// IsSubsequence says whether every element of short appears in long, in order.
// It is the behaviour-equivalence check for the lazy walk: the walk must never
// ask for a predicate the decision would not have asked for at that point.
func IsSubsequence[T comparable](short, long []T) bool {
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
