package spgen

import (
	"fmt"
	"math"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
)

// A Table is the decision the plugin walks instead of calling the decision.
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
//
// A node with Predicate -1 is a leaf and WhenTrue is the outcome.
type Table struct {
	Predicate []int32
	WhenTrue  []int32
	WhenFalse []int32
	Roots     []int32

	// Axes are the values the edge already holds when it is asked for an
	// action, in signature order. They cost nothing to read, so the table
	// indexes on them rather than asking about them.
	Axes []Axis
	// Predicates are the questions, in the order of actionsel.Predicate. The
	// index is the id the walk hands the edge.
	Predicates []string
}

// Axis is one free parameter and the values it takes, which are 0 to Size-1
// because the enums it stands for are declared with iota.
type Axis struct {
	Name string
	Size int
}

// leaf marks a node that answers rather than asks.
const leaf = int32(-1)

// maxNodes bounds the build. The graph is shared, so a body this size is a
// sign the decision is not a decision.
const maxNodes = 1 << 16

// build turns one decision tree per point of the axes into the shared graph.
// Points come in row-major order, so the emitted roots index the same way a
// nested SourcePawn array does.
type builder struct {
	seen map[string]int32

	predicate []int32
	whenTrue  []int32
	whenFalse []int32
}

// add numbers one traced tree into the graph, children first, so a node is
// written only once both its answers exist.
func (b *builder) add(n *actionsel.Node) (int32, error) {
	if n == nil {
		return 0, fmt.Errorf("spgen: the trace has a nil node")
	}
	if n.Leaf {
		outcome, ok := asCell(int64(n.Action))
		if !ok {
			return 0, fmt.Errorf("spgen: the outcome %d does not fit a cell", int64(n.Action))
		}
		return b.node(leaf, outcome, leaf)
	}
	yes, err := b.add(n.True)
	if err != nil {
		return 0, err
	}
	no, err := b.add(n.False)
	if err != nil {
		return 0, err
	}
	id := int32(n.Ask) //nolint:gosec // G115: a Predicate is an index into a list written by hand
	if id < 0 || int(id) >= len(actionsel.Predicates()) {
		return 0, fmt.Errorf("spgen: the trace asks %v, which is not one of the predicates", n.Ask)
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

// asCell narrows a decision's answer to the 32 bits SourcePawn has.
func asCell(v int64) (int32, bool) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0, false
	}
	return int32(v), true
}
