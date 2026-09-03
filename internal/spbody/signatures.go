package spbody

import (
	"fmt"
	"go/token"
	"go/types"
)

/*
Package-level signatures of the extern package, so an extern can be held against
the Go it claims

ExternsFromDir reads the directives off the syntax and stops there, because that
is all the emitter needs: a directive says how SourcePawn writes the call, and
the call site's types come from the body's own type check. What it does not give
is the extern's own signature, and without that a //sp:body extern is a name and
nothing else. It says the port generates a function of that name; it does not say
the port generates a function of that shape, and SourcePawn cannot say either.
int, float, every tag and every handle are one cell there, so a threshold
declared where an entity index is passed compiles and misbehaves in a wave.

So this type checks the extern package once and hands back the shapes.
*/

// SignaturesFromDir type checks the package in dir and returns the signature of
// every function it declares, keyed the way ExternsFromDir keys its externs: by
// package name and function name, with the receiver type in between for a
// method.
func SignaturesFromDir(dir string) (map[string]*types.Signature, error) {
	fset := token.NewFileSet()
	files, _, err := parseDir(fset, dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("spbody: %s holds no Go file", dir)
	}
	imp, err := newModuleImporter(fset, dir)
	if err != nil {
		return nil, err
	}
	conf := types.Config{Importer: imp}
	pkg, err := conf.Check(dir, fset, files, nil)
	if err != nil {
		return nil, fmt.Errorf("spbody: %s does not type check: %w", dir, err)
	}

	out := make(map[string]*types.Signature)
	local := pkg.Name()
	for _, name := range pkg.Scope().Names() {
		switch obj := pkg.Scope().Lookup(name).(type) {
		case *types.Func:
			out[local+"."+name] = obj.Signature()
		case *types.TypeName:
			named, isNamed := types.Unalias(obj.Type()).(*types.Named)
			if !isNamed {
				continue
			}
			for m := range named.Methods() {
				out[local+"."+name+"."+m.Name()] = m.Signature()
			}
		}
	}
	return out, nil
}

// optionalParams marks each parameter a caller need not write, by the four
// directives that make one.
func optionalParams(sig *types.Signature, defaults map[string]string, byrefs, mutates map[string]bool, lengths map[string]string) []bool {
	buffers := make(map[string]bool, 2*len(lengths))
	for buffer, size := range lengths {
		buffers[buffer], buffers[size] = true, true
	}
	out := make([]bool, sig.Params().Len())
	for i := range out {
		name := sig.Params().At(i).Name()
		_, given := defaults[name]
		out[i] = given || byrefs[name] || mutates[name] || buffers[name]
	}
	return out
}

/*
Allowance is what an extern's own directive supplies, so a difference it
explains is not a difference.

Trail are the constant arguments written after the Go ones, which the generated
function declares and the extern does not. Buffer says the extern's last result
is text the generated function fills through a parameter instead of returning,
which is the sized and fills form: two trailing parameters, the buffer and its
size, in place of one result.
*/
type Allowance struct {
	Trail  int
	Buffer bool
}

/*
SameShape says whether an extern declares the same shape as the function it
names, and names the first difference when it does not.

The two come from separate type checks -- the extern package's own, and the body
package's, which imported it through an importer of its own -- so their named
types are different *types.Named values standing for the same declaration, and
types.Identical says no to every one of them. The comparison is therefore on the
written form, qualified by package name, which is the form both sides agree on
and the form a reader would compare by hand.

The two are not required to be identical, because SourcePawn does not require
it. A trailing parameter the generated function declares may be left off, taken
back as a result, or supplied by the directive; which of the three is what
optional and the allowance say. Everything the extern does write has to match.
*/
func SameShape(declared, generated *types.Signature, optional []bool, a Allowance) (difference string, same bool) {
	if declared.Variadic() != generated.Variadic() {
		return "one is variadic and the other is not", false
	}
	/*
		The trailing parameters an extern may account for another way,
		newest last: the ones the directive supplies, and then the run of
		optional ones in front of those. The run stops at the first
		parameter that is not optional, because a parameter a caller
		cannot leave off does not become optional by having one behind
		it. Anything in front of the run the extern has to write, in
		order, and every one of these it does write it has to write in
		order too: the shortening only ever comes off the end.
	*/
	spare := a.Trail
	for i := len(optional) - a.Trail - 1; i >= 0 && optional[i]; i-- {
		spare++
	}
	fixed := generated.Params().Len() - spare
	if declared.Params().Len() < fixed {
		return fmt.Sprintf("the extern takes %d arguments and the function takes %d, of which the last %d need not be written",
			declared.Params().Len(), generated.Params().Len(), spare), false
	}
	if declared.Params().Len() > generated.Params().Len() {
		return fmt.Sprintf("the extern takes %d arguments and the function takes %d", declared.Params().Len(), generated.Params().Len()), false
	}
	for i := range declared.Params().Len() {
		want, got := declared.Params().At(i).Type(), generated.Params().At(i).Type()
		if !sameType(want, got) {
			return fmt.Sprintf("argument %d is %s in the extern and %s in the function", i+1, shape(want), shape(got)), false
		}
	}

	/*
		The extern's results are the function's, and then whatever it
		took back off the end of the parameter list: a by-reference
		answer, or the buffer a filling call writes into. They come back
		in the order they were declared in, so the first extra result is
		the first parameter the extern left off.

		Fewer is allowed too, and means the same thing it means in
		SourcePawn: the caller does not want the answer. Every result the
		extern does declare still has to be the one the function returns
		in that position, so this only ever comes off the end.
	*/
	dropped := generated.Params().Len() - declared.Params().Len()
	extra := declared.Results().Len() - generated.Results().Len()
	if extra > dropped {
		return fmt.Sprintf("the extern returns %d values and the function returns %d, having left off %d arguments",
			declared.Results().Len(), generated.Results().Len(), dropped), false
	}
	for i := range declared.Results().Len() {
		want := declared.Results().At(i).Type()
		var got types.Type
		switch {
		case i < generated.Results().Len():
			got = generated.Results().At(i).Type()
		case a.Buffer:
			// The whole point of the form: the extern returns the
			// text the function was handed a buffer for.
			continue
		default:
			got = generated.Params().At(declared.Params().Len() + i - generated.Results().Len()).Type()
		}
		if !sameType(want, got) {
			return fmt.Sprintf("result %d is %s in the extern and %s in the function", i+1, shape(want), shape(got)), false
		}
	}
	return "", true
}

// shape is a type written the way both type checks spell it: by package name,
// never by import path, because the two sides hold different *types.Package
// values for the same package and only the name is shared.
func shape(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}

/*
sameType is equality with the one normalisation SourcePawn forces.

Text there is char[], and that is the whole of it: a literal and a buffer are
the same thing and either goes where the other is expected. Go has to
distinguish them -- a string is not an array -- so the same SourcePawn parameter
is written string in one declaration and engine.Text or [16]byte in another, and
neither is wrong. //sp:same exists at call sites for exactly this.

Nothing else is normalised. Two tagged ints are one cell in SourcePawn and the
comparison keeps them apart on purpose, because a cell is what makes the
difference invisible everywhere else.
*/
func sameType(a, b types.Type) bool {
	return shape(a) == shape(b) || (isText(a) && isText(b))
}

// isText is a Go string, or the [N]byte the subset spells char[N] as. The
// underlying type and not the name, because Text is an alias and a body that
// declares its own buffer writes the array out.
func isText(t types.Type) bool {
	switch u := types.Unalias(t).Underlying().(type) {
	case *types.Basic:
		return u.Kind() == types.String
	case *types.Array:
		elem, isBasic := u.Elem().Underlying().(*types.Basic)
		return isBasic && elem.Kind() == types.Byte
	}
	return false
}
