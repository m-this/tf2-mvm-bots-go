package spbody_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spbody"
)

// signatures type checks one source file and hands back the signature of every
// function in it, so a case below reads as the two declarations it is about.
func signatures(t *testing.T, source string) map[string]*types.Signature {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "shapes.go", source, 0)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("shapes", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatalf("type checking: %v", err)
	}
	out := make(map[string]*types.Signature)
	for _, name := range pkg.Scope().Names() {
		if fn, isFunc := pkg.Scope().Lookup(name).(*types.Func); isFunc {
			out[name] = fn.Signature()
		}
	}
	return out
}

const shapes = `package shapes

type Team int32
type Text = [512]byte

func Declared(client int32, position [3]float32) bool                 { return false }
func Same(client int32, position [3]float32) bool                     { return false }
func Tagged(client Team, position [3]float32) bool                    { return false }
func Widened(client int32, position [3]float32) float32               { return 0 }
func ByRef(client int32, position [3]float32) (bool, float32)         { return false, 0 }
func Longer(client int32, position [3]float32, extra float32) bool    { return false }
func Shorter(client int32) bool                                       { return false }
func Ignoring(client int32, position [3]float32)                      {}
func Filling(client int32, buffer Text, maxlength int32)              {}
func Returning(client int32) Text                                     { return Text{} }
func Literal(name string) bool                                        { return false }
func Buffered(name Text) bool                                         { return false }
`

// TestSameShapeAcceptsWhatSourcePawnAllows covers the differences a directive
// explains. Each of these is a real extern's shape, reduced.
func TestSameShapeAcceptsWhatSourcePawnAllows(t *testing.T) {
	sig := signatures(t, shapes)
	none := []bool{false, false}
	cases := []struct {
		name              string
		declared, defined string
		optional          []bool
		allow             spbody.Allowance
	}{
		{"identical", "Same", "Declared", none, spbody.Allowance{}},
		{
			// SourcePawn has one char[], so a literal and a buffer are
			// the same parameter written two ways.
			"text either way", "Literal", "Buffered", []bool{false}, spbody.Allowance{},
		},
		{
			// //sp:default: the caller may leave the last one off.
			"a defaulted argument left off", "Same", "Longer",
			[]bool{false, false, true},
			spbody.Allowance{},
		},
		{
			// //sp:byref: the caller takes it as an answer instead.
			"a by-reference argument taken as a result", "ByRef", "Longer",
			[]bool{false, false, true},
			spbody.Allowance{},
		},
		{
			// The trailing constants the directive itself writes.
			"an argument the directive supplies", "Same", "Longer",
			none,
			spbody.Allowance{Trail: 1},
		},
		{
			// sized: the buffer and its length, in place of a result.
			"a filled buffer returned instead", "Returning", "Filling",
			[]bool{false, true, true},
			spbody.Allowance{Buffer: true},
		},
		{
			// Calling it as a statement, which SourcePawn allows.
			"a result the extern ignores", "Ignoring", "Declared", none, spbody.Allowance{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if difference, same := spbody.SameShape(sig[c.declared], sig[c.defined], c.optional, c.allow); !same {
				t.Errorf("%s against %s is refused: %s", c.declared, c.defined, difference)
			}
		})
	}
}

/*
TestSameShapeRefusesWhatSourcePawnCannotSee is the point of the comparison.

Every case here compiles. int, float, every tag and every handle are one cell in
SourcePawn, so its own declaration cannot tell a threshold from an entity index,
and the mistake arrives as a bot doing the wrong thing rather than as an error.
*/
func TestSameShapeRefusesWhatSourcePawnCannotSee(t *testing.T) {
	sig := signatures(t, shapes)
	none := []bool{false, false}
	cases := []struct {
		name              string
		declared, defined string
		optional          []bool
	}{
		{"a tagged argument against a plain one", "Tagged", "Declared", none},
		{"a different result type", "Widened", "Declared", none},
		{"an argument that is not there", "Longer", "Declared", none},
		{"a missing argument with nothing optional", "Shorter", "Declared", none},
		{"an argument left off that was not optional", "Shorter", "Declared", []bool{false, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			difference, same := spbody.SameShape(sig[c.declared], sig[c.defined], c.optional, spbody.Allowance{})
			if same {
				t.Fatalf("%s against %s is accepted", c.declared, c.defined)
			}
			if difference == "" {
				t.Error("the refusal does not say what differs")
			}
		})
	}
}

// TestSignaturesFromDirReadsTheExternPackage proves the key is the one
// ExternsFromDir uses, because holding one against the other is the whole job.
func TestSignaturesFromDirReadsTheExternPackage(t *testing.T) {
	const dir = "../engine"
	declared, err := spbody.ExternsFromDir(dir)
	if err != nil {
		t.Fatalf("reading the externs: %v", err)
	}
	sigs, err := spbody.SignaturesFromDir(dir)
	if err != nil {
		t.Fatalf("reading the signatures: %v", err)
	}
	for qualified, x := range declared.Funcs {
		if x.Method {
			// A method is keyed by its receiver, and the emitter
			// resolves it that way; the shape check does not use it.
			continue
		}
		if _, found := sigs[qualified]; !found {
			t.Errorf("%s is declared as an extern and has no signature", qualified)
		}
	}
}
