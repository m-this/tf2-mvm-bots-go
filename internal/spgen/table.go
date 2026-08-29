package spgen

import "fmt"

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
		outcome, ok := asCell(out.i)
		if !ok {
			return 0, fmt.Errorf("spgen: %s answered %d, which does not fit a cell", b.lazy.Entry, out.i)
		}
		return b.node(leaf, outcome, leaf)
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
