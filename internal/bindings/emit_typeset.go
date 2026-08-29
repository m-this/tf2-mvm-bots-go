package bindings

import "fmt"

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
	e.doc(ts.Doc)
	fmt.Fprintf(&e.b, "// %s is a function reference. Its call signatures are %sSig1..%sSig%d.\n",
		name, name, name, len(ts.Variants))
	fmt.Fprintf(&e.b, "type %s struct{ Ref int32 }\n\n", name)
	for i, v := range ts.Variants {
		e.typesetVariant(name, i+1, v)
	}
}

func (e *emitter) typesetVariant(owner string, n int, v TypesetVariant) {
	sig := fmt.Sprintf("%sSig%d", owner, n)
	params, result, err := signature(v.Params, v.Return)
	if err != nil {
		e.refuse(v.Pos, "typeset signature", sig, err.Error())
		return
	}
	if !e.claim("", sig, v.Pos) {
		return
	}
	e.doc(v.Doc)
	fmt.Fprintf(&e.b, "type %s func(%s) %s\n\n", sig, params, result)
}
