package spgen

import (
	"fmt"
	"go/constant"
	"go/types"
)

// ActionSelConfig is how internal/actionsel is emitted. Every name carries the
// prefix, because Action, RoundState and Address are SourceMod's names too.
var ActionSelConfig = Config{Prefix: "ActionSel_", Guard: "actionsel"}

// ActionSelLazy is the decision the plugin walks. The axes are the two values
// the edge already holds when it is asked for an action, and the predicates
// are the fields of actionsel.Flags in declaration order.
var ActionSelLazy = LazyTable{
	Entry:      "Select",
	Axes:       []Axis{{Name: "RoundState", Size: 11}, {Name: "Class", Size: 10}},
	Predicates: predicateFields(),
}

func predicateFields() []string {
	out := make([]string, 0, len(ActionSelPredicates))
	for _, p := range ActionSelPredicates {
		out = append(out, p.Field)
	}
	return out
}

// ActionSel is the whole emission for one load of internal/actionsel: the pure
// file the plugin includes and the differential test compiles, and the edge
// that the plugin includes after it.
type ActionSel struct {
	Pure     string
	Dispatch string
	Table    Table
}

// EmitActionSel loads dir and emits both files.
func EmitActionSel(dir string) (ActionSel, error) {
	p, err := Load(dir)
	if err != nil {
		return ActionSel{}, err
	}
	if err := p.checkEdge(); err != nil {
		return ActionSel{}, err
	}
	pure, err := p.SourcePawn(ActionSelConfig)
	if err != nil {
		return ActionSel{}, err
	}
	table, err := p.Table(ActionSelLazy)
	if err != nil {
		return ActionSel{}, err
	}
	return ActionSel{
		Pure:     pure + table.SourcePawn(ActionSelConfig),
		Dispatch: Dispatch(ActionSelConfig, ActionSelPredicates, ActionSelOutcomes, "internal/actionsel"),
		Table:    table,
	}, nil
}

// checkEdge refuses an emission whose edge table has drifted from the Go it
// answers for: a predicate added to Flags with no call written for it, or an
// outcome added to the enum with no call site. Both would otherwise come out
// as a plausible file that decides the wrong thing.
func (p *Package) checkEdge() error {
	fields, err := p.StructFields("Flags")
	if err != nil {
		return err
	}
	if got, want := predicateFields(), fields; !equal(got, want) {
		return fmt.Errorf("spgen: the edge answers %v, but actionsel.Flags reads %v", got, want)
	}
	consts := p.ConstNames("Action")
	names := make([]string, 0, len(ActionSelOutcomes))
	for _, o := range ActionSelOutcomes {
		names = append(names, o.Const)
	}
	if !equal(names, consts) {
		return fmt.Errorf("spgen: the edge lists the outcomes %v, but actionsel.Action declares %v", names, consts)
	}
	return nil
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// StructFields is the field names of a package-level struct, in declaration
// order.
func (p *Package) StructFields(name string) ([]string, error) {
	obj := p.pkg.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("spgen: no type named %s", name)
	}
	st, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("spgen: %s is not a struct", name)
	}
	out := make([]string, 0, st.NumFields())
	for i := range st.NumFields() {
		out = append(out, st.Field(i).Name())
	}
	return out, nil
}

// ConstNames is every package-level constant of one named type, in the order
// its values run, which for an iota block is declaration order.
func (p *Package) ConstNames(typeName string) []string {
	byValue := map[int64]string{}
	highest := int64(-1)
	for _, name := range p.pkg.Scope().Names() {
		c, ok := p.pkg.Scope().Lookup(name).(*types.Const)
		if !ok {
			continue
		}
		named, ok := c.Type().(*types.Named)
		if !ok || named.Obj().Name() != typeName {
			continue
		}
		v, exact := constInt64(c)
		if !exact {
			continue
		}
		byValue[v] = name
		highest = max(highest, v)
	}
	out := make([]string, 0, len(byValue))
	for v := int64(0); v <= highest; v++ {
		if n, ok := byValue[v]; ok {
			out = append(out, n)
		}
	}
	return out
}

func constInt64(c *types.Const) (int64, bool) {
	return constant.Int64Val(c.Val())
}
