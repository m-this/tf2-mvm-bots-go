package spgen

import (
	"fmt"

	"github.com/m-this/tf2-mvm-bots-go/internal/actionsel"
)

// Config is what the caller decides: the name every emitted identifier carries
// and the guard the emitted file uses.
type Config struct {
	// Prefix is prepended to every emitted type and constant name. It is not
	// optional: Action, RoundState and Plugin_Continue's Action are all
	// SourceMod names already, so an unprefixed emission collides.
	Prefix string
	// Guard is the include guard symbol, without underscores of its own.
	Guard string
}

// ActionSelConfig is how internal/actionsel is emitted. Every name carries the
// prefix, because Action, RoundState and Address are SourceMod's names too.
var ActionSelConfig = Config{Prefix: "ActionSel_", Guard: "actionsel"}

// ActionSel is the whole emission for one build of internal/actionsel: the
// data file the plugin includes and the differential test compiles, and the
// edge that the plugin includes after it.
type ActionSel struct {
	// Data is the enums and the decision table. It declares no function and
	// calls nothing.
	Data string
	// Dispatch is the edge: the predicate calls, the walk and the switch back
	// onto the shipped call sites.
	Dispatch string
	Table    Table
}

// EmitActionSel builds the decision table by running the decision, and emits
// both files.
//
// The table is extracted by actionsel.Explore, which runs Select against a
// Facts that refuses a question it has not been told about, answers it both
// ways and recurs. Nothing here reads the Go source: a table produced by
// running the decision cannot disagree with the decision.
func EmitActionSel() (ActionSel, error) {
	if err := checkEdge(); err != nil {
		return ActionSel{}, err
	}
	table, err := ActionSelTable()
	if err != nil {
		return ActionSel{}, err
	}
	data, err := Data(ActionSelConfig, table)
	if err != nil {
		return ActionSel{}, err
	}
	return ActionSel{
		Data:     data,
		Dispatch: Dispatch(ActionSelConfig, ActionSelPredicates, ActionSelOutcomes, "internal/actionsel"),
		Table:    table,
	}, nil
}

// ActionSelTable traces the decision once per round state and class, and
// shares the resulting trees into one graph.
func ActionSelTable() (Table, error) {
	b := &builder{seen: map[string]int32{}}
	states, classes := actionsel.RoundStates(), actionsel.Classes()
	roots := make([]int32, 0, len(states)*len(classes))
	for _, state := range states {
		for _, class := range classes {
			root, err := b.add(actionsel.Explore(state, class))
			if err != nil {
				return Table{}, fmt.Errorf("tracing %v/%v: %w", state, class, err)
			}
			roots = append(roots, root)
		}
	}
	return Table{
		Predicate:  b.predicate,
		WhenTrue:   b.whenTrue,
		WhenFalse:  b.whenFalse,
		Roots:      roots,
		Axes:       []Axis{{Name: "RoundState", Size: len(states)}, {Name: "Class", Size: len(classes)}},
		Predicates: predicateFields(),
	}, nil
}

// checkEdge refuses an emission whose edge table has drifted from the Go it
// answers for: an outcome added to the enum with no call site written for it
// would otherwise come out as a plausible file that decides the wrong thing.
// The predicates cannot drift, because they are read from actionsel.
func checkEdge() error {
	if got, want := len(ActionSelOutcomes), int(actionsel.ActionStrandedAsShipped)+1; got != want {
		return fmt.Errorf("spgen: the edge lists %d outcomes, actionsel.Action declares %d", got, want)
	}
	return nil
}
