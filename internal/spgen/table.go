package spgen

import (
	"fmt"
	"strings"
)

// A LazyTable is the decision the plugin walks instead of calling the pure
// function.
//
// The plugin cannot fill a struct of predicates and then decide, because three
// of the predicates have side effects: CTFBotAttackTank_SelectTarget writes
// m_iTankTarget, CTFBotCollectNearMoney_SelectTarget writes m_iCurrencyPack,
// and CTFBotDefenderAttack_SelectTarget consumes randomness. Asking all of
// them up front would set state and burn draws the shipped chain never
// touches, and the A/B would then measure the edge rather than the decision.
//
// So the decision comes out as data. Each node is a predicate and the two
// nodes to go to; the edge asks for a predicate only when the walk arrives at
// it, and a predicate is asked at most once per walk because its answer is
// what chose the next node. The walk is a loop of about six lines and the
// table never calls anything, so the never-call-back rule holds.
type LazyTable struct {
	// Entry is the function to walk, taking the axes and then the struct.
	Entry string
	// Axes are the parameters the edge already has in hand, in signature
	// order. They cost nothing to read, so the walk indexes on them rather
	// than asking about them.
	Axes []Axis
	// Predicates are the struct's fields, in declaration order. The index is
	// the id the walk hands the edge.
	Predicates []string
}

// Axis is one free parameter and the values it takes, which are 0 to Size-1
// because the enums it stands for are declared with iota.
type Axis struct {
	Name string
	Size int
}

// Table is the built decision, a directed acyclic graph with one root per
// point of the axes. A node with Predicate -1 is a leaf and WhenTrue is the
// outcome.
type Table struct {
	Predicate []int32
	WhenTrue  []int32
	WhenFalse []int32
	Roots     []int32
	Lazy      LazyTable
}

// leaf marks a node that answers rather than asks.
const leaf = int32(-1)

// maxNodes bounds the build. The graph is shared, so a body this size is a
// sign the decision is not a decision.
const maxNodes = 1 << 16

// Table builds the lazy decision for one entry function.
func (p *Package) Table(lazy LazyTable) (Table, error) {
	if len(lazy.Predicates) == 0 || len(lazy.Axes) == 0 {
		return Table{}, fmt.Errorf("spgen: a lazy table needs at least one axis and one predicate")
	}
	b := &builder{in: newInterp(p), lazy: lazy, seen: map[string]int32{}}
	roots := make([]int32, 0, axesSize(lazy.Axes))
	for _, point := range axesPoints(lazy.Axes) {
		root, err := b.build(point, map[string]bool{})
		if err != nil {
			return Table{}, err
		}
		roots = append(roots, root)
	}
	return Table{Predicate: b.predicate, WhenTrue: b.whenTrue, WhenFalse: b.whenFalse, Roots: roots, Lazy: lazy}, nil
}

func axesSize(axes []Axis) int {
	n := 1
	for _, a := range axes {
		n *= a.Size
	}
	return n
}

// axesPoints walks the axes row-major, so the emitted roots index the same way
// a nested SourcePawn array does.
func axesPoints(axes []Axis) [][]int64 {
	points := [][]int64{{}}
	for _, a := range axes {
		next := make([][]int64, 0, len(points)*a.Size)
		for _, p := range points {
			for v := range a.Size {
				next = append(next, append(append([]int64{}, p...), int64(v)))
			}
		}
		points = next
	}
	return points
}

type builder struct {
	in   *interp
	lazy LazyTable
	seen map[string]int32

	predicate []int32
	whenTrue  []int32
	whenFalse []int32
}

func (b *builder) build(point []int64, known map[string]bool) (int32, error) {
	args := make([]value, 0, len(point)+1)
	for _, v := range point {
		args = append(args, value{kind: kindInt, i: v})
	}
	args = append(args, value{kind: kindStruct, fields: &fieldSet{known: known}})

	out, stopped, err := b.in.run(b.lazy.Entry, args)
	if err != nil {
		return 0, err
	}
	if stopped == "" {
		return b.node(leaf, int32(out.i), leaf)
	}
	id := b.predicateID(stopped)
	if id < 0 {
		return 0, fmt.Errorf("spgen: %s read the field %s, which is not one of the predicates", b.lazy.Entry, stopped)
	}
	yes, err := b.build(point, with(known, stopped, true))
	if err != nil {
		return 0, err
	}
	no, err := b.build(point, with(known, stopped, false))
	if err != nil {
		return 0, err
	}
	return b.node(id, yes, no)
}

// node hash-conses, so two branches that decide the same thing are one node
// and the graph stays the size of the decision rather than of the domain.
func (b *builder) node(pred, yes, no int32) (int32, error) {
	key := fmt.Sprintf("%d/%d/%d", pred, yes, no)
	if i, ok := b.seen[key]; ok {
		return i, nil
	}
	if len(b.predicate) >= maxNodes {
		return 0, fmt.Errorf("spgen: the decision graph passed %d nodes", maxNodes)
	}
	i := int32(len(b.predicate)) //nolint:gosec // G115: bounded by maxNodes above
	b.predicate = append(b.predicate, pred)
	b.whenTrue = append(b.whenTrue, yes)
	b.whenFalse = append(b.whenFalse, no)
	b.seen[key] = i
	return i, nil
}

func (b *builder) predicateID(field string) int32 {
	for i, p := range b.lazy.Predicates {
		if p == field {
			return int32(i) //nolint:gosec // G115: an index into a list written by hand
		}
	}
	return -1
}

func with(known map[string]bool, field string, v bool) map[string]bool {
	out := make(map[string]bool, len(known)+1)
	for k, val := range known {
		out[k] = val
	}
	out[field] = v
	return out
}

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
	return asked, int32(out.i), nil
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
