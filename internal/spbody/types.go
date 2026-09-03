package spbody

import (
	"fmt"
	"go/types"
	"strings"
)

// spType is the SourcePawn tag for a Go type, and the array dimensions that
// follow the declared name.
//
// Every integer width becomes int, which is the cell SourcePawn has. The subset
// accepts int64 and float64 and SourcePawn has neither, so both narrow here;
// gosubset's SUBSET.md says so and says to write int32 and float32.
func (e *emitter) spType(t types.Type) (tag string, dims []int64, err error) {
	switch t := t.(type) {
	case *types.Basic:
		tag, err = basicTag(t)
		return tag, nil, err
	case *types.Named:
		return e.namedTag(t)
	case *types.Array:
		// A byte array is a string buffer. SourcePawn spells that char,
		// and nothing else in the subset uses uint8, so the mapping is
		// unambiguous: byte means text.
		if b, ok := types.Unalias(t.Elem()).(*types.Basic); ok && b.Kind() == types.Byte {
			return "char", []int64{t.Len()}, nil
		}
		tag, inner, err := e.spType(t.Elem())
		if err != nil {
			return "", nil, err
		}
		return tag, append([]int64{t.Len()}, inner...), nil
	case *types.Alias:
		return e.spType(types.Unalias(t))
	default:
		return "", nil, fmt.Errorf("the type %s has no SourcePawn; the subset has sized integers, float32, bool, arrays of those and named structs", t)
	}
}

func basicTag(t *types.Basic) (string, error) {
	switch {
	case t.Info()&types.IsBoolean != 0:
		return "bool", nil
	case t.Info()&types.IsInteger != 0:
		return "int", nil
	case t.Info()&types.IsFloat != 0:
		return "float", nil
	}
	return "", fmt.Errorf("the type %s has no SourcePawn; write a sized integer, float32 or bool", t)
}

func (e *emitter) namedTag(t *types.Named) (string, []int64, error) {
	name := t.Obj().Name()
	if pkg := t.Obj().Pkg(); pkg != nil && pkg != e.pkg {
		// A type the extern package declares stands for a SourcePawn tag
		// that already exists. Emitting our own enum for it would give
		// the plugin two names for one thing.
		tag, declared := e.cfg.Tags[pkg.Name()+"."+name]
		if !declared {
			return "", nil, fmt.Errorf("the type %s.%s has no //sp:tag saying what it is called in SourcePawn", pkg.Name(), name)
		}
		// A named array carries its length with it, which is how a text
		// buffer is one type rather than one per size.
		if arr, isArray := t.Underlying().(*types.Array); isArray {
			return tag, []int64{arr.Len()}, nil
		}
		return tag, nil, nil
	}
	switch u := t.Underlying().(type) {
	case *types.Struct:
		/* The name the port claimed, if it claimed one

		A record other files declare variables of keeps the plugin's name,
		and typeSpec already writes it out that way. A variable of that
		type has to agree, or it names a type nothing declares. */
		if claimed, ok := e.typeNames[name]; ok {
			return claimed, nil, nil
		}
		return e.cfg.Prefix + name, nil, nil
	case *types.Basic:
		if u.Info()&types.IsInteger == 0 {
			// A named float or bool has no tag of its own to
			// declare, so it would emit as its underlying type and
			// the name would be lost in the output.
			return "", nil, fmt.Errorf("the named type %s is not an integer; SourcePawn tags what it can enumerate, so write the underlying type", name)
		}
		if !e.hasConstants(t) {
			return "", nil, fmt.Errorf("the named type %s declares no constants, so there is no SourcePawn enum to emit; declare its constants, or write int32", name)
		}
		return e.cfg.Prefix + name, nil, nil
	case *types.Array:
		return "", nil, fmt.Errorf("the named array type %s has no SourcePawn; write the array type out", name)
	default:
		return "", nil, fmt.Errorf("the named type %s has no SourcePawn", name)
	}
}

func (e *emitter) hasConstants(t *types.Named) bool {
	scope := e.pkg.Scope()
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if ok && types.Identical(c.Type(), t) {
			return true
		}
	}
	return false
}

// declare writes a SourcePawn declaration: the tag, the name, and the array
// dimensions that in SourcePawn follow the name rather than precede it.
func declare(tag, name string, dims []int64) string {
	var b strings.Builder
	b.WriteString(tag)
	b.WriteByte(' ')
	b.WriteString(name)
	for _, d := range dims {
		fmt.Fprintf(&b, "[%d]", d)
	}
	return b.String()
}
