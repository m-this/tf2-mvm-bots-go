package bindings

import (
	"fmt"
	"strings"
)

// typeset emits a SourcePawn typeset as an opaque function reference plus one
// Go function type per signature.
//
// The reference type is what the declarations that mention the typeset need:
// a property setter takes the typeset by name, and without a type of that
// name nothing that touches it compiles. Its SourcePawn value really is a
// function reference, so an opaque handle is not a stand-in for something
// better, it is the value.
//
// The signatures are emitted beside it because a caller writing a callback
// has to know which of them it is writing. Collapsing them onto one would
// pick a shape wrong for every other callback the typeset serves.
func (e *emitter) typeset(ts Typeset) {
	name := goIdent(ts.Name)
	if !e.claim("", name, ts.Pos) {
		return
	}
	// The signatures are rendered before any is emitted, because a name is
	// chosen against the whole typeset rather than against what came before it.
	renders := make([]string, len(ts.Variants))
	params := make([]string, len(ts.Variants))
	results := make([]string, len(ts.Variants))
	errs := make([]error, len(ts.Variants))
	for i, v := range ts.Variants {
		params[i], results[i], errs[i] = signature(v.Params, v.Return)
		if errs[i] == nil {
			renders[i] = params[i] + "|" + results[i]
		}
	}
	/* A typeset can declare the same signature twice, and two of them do:
	NativeCall has (Handle, int) returning int under two different comments, and
	InputFuncCallback repeats the CBaseEntity-and-int shape. One signature is one
	Go type, so the repeat is dropped rather than refused. */
	kept := make([]int, 0, len(ts.Variants))
	first := map[string]int{}
	for i, r := range renders {
		if errs[i] == nil {
			if _, dup := first[r]; dup {
				continue
			}
			first[r] = i
		}
		kept = append(kept, i)
	}
	keptVariants := make([]TypesetVariant, len(kept))
	keptRenders := make([]string, len(kept))
	for j, i := range kept {
		keptVariants[j], keptRenders[j] = ts.Variants[i], renders[i]
	}
	names := variantNames(name, keptVariants, keptRenders)

	e.doc(ts.Doc)
	fmt.Fprintf(&e.b, "// %s is a function reference, with %d call signatures: %s.\n",
		name, len(names), strings.Join(names, ", "))
	fmt.Fprintf(&e.b, "type %s struct{ Ref int32 }\n\n", name)
	for j, i := range kept {
		v := ts.Variants[i]
		if errs[i] != nil {
			e.refuse(v.Pos, "typeset signature", names[j], errs[i].Error())
			continue
		}
		if !e.claim("", names[j], v.Pos) {
			continue
		}
		e.doc(v.Doc)
		fmt.Fprintf(&e.b, "type %s func(%s) %s\n\n", names[j], params[i], results[i])
	}
}
