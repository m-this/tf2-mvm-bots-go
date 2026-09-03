package gosubset_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/gosubset"
)

func wrap(body string) string {
	return "package decisions\n\nfunc f(a int32, b float32) int32 {\n" + body + "\n\treturn a\n}\n"
}

// refusals is the negative space: every construct the subset must never let
// through, and the word the message has to contain for a reader to act on it.
func TestRefused(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want string
	}{
		{"goroutine", wrap("\tgo f(a, b)"), "goroutine"},
		{"defer", wrap("\tdefer f(a, b)"), "defer"},
		{"channel type", "package decisions\n\nfunc f(c chan int32) {}\n", "channel type"},
		{"channel receive", "package decisions\n\nfunc f(c chan int32) {\n\t_ = <-c\n}\n", "channel receive"},
		{"select", wrap("\tselect {}"), "select"},
		{"goto", "package decisions\n\nfunc f() {\nagain:\n\tgoto again\n}\n", "goto"},
		{"label", "package decisions\n\nfunc f() {\nagain:\n\tfor {\n\t\tbreak again\n\t}\n}\n", "label"},
		{"fallthrough", wrap("\tswitch a {\n\tcase 1:\n\t\tfallthrough\n\tcase 2:\n\t}"), "fallthrough"},
		{"tagless switch", wrap("\tswitch {\n\tcase a > 1:\n\t}"), "no value to switch on"},

		{"map type", "package decisions\n\ntype T struct {\n\tM map[int32]int32\n}\n", "map type"},
		{"slice type", "package decisions\n\ntype T struct {\n\tS []int32\n}\n", "slice type"},
		{"slice expr", "package decisions\n\nfunc f(x [4]int32) {\n\t_ = x[1:2]\n}\n", "slice expression"},
		{"interface type", "package decisions\n\ntype T interface{}\n", "interface type"},
		{"closure", wrap("\tg := func() int32 { return 1 }\n\t_ = g"), "function literal"},
		{"pointer param", "package decisions\n\nfunc f(p *int32) {}\n", "pointer type"},
		{"pointer result", "package decisions\n\nfunc f() *int32 { return nil }\n", "pointer type"},
		{"address of", wrap("\tp := &a\n\t_ = p"), "taking an address"},
		{"unsized int", "package decisions\n\nfunc f(x int) {}\n", "32 bits"},
		{"string result", "package decisions\n\nfunc f() string { return \"\" }\n", "no strings"},
		{"string field", "package decisions\n\ntype T struct {\n\tS string\n}\n", "no strings"},
		{"string literal", wrap("\t_ = \"medic\""), "string literal"},
		{"error type", "package decisions\n\nfunc f() error { return nil }\n", "no error interface"},
		{"any type", "package decisions\n\nfunc f(x any) {}\n", "concrete type"},
		{"make", wrap("\t_ = make([]int32, 4)"), "make"},
		{"new", wrap("\t_ = new(int32)"), "new"},
		{"append", "package decisions\n\nfunc f(s [4]int32) {\n\t_ = append(s, 1)\n}\n", "append"},
		{"panic", wrap("\tpanic(1)"), "panic"},
		{"recover", wrap("\t_ = recover()"), "recover"},
		{"type assertion", "package decisions\n\nfunc f(x any) {\n\t_ = x.(int32)\n}\n", "type assertion"},
		{"type switch", "package decisions\n\nfunc f(x any) {\n\tswitch x.(type) {\n\t}\n}\n", "type switch"},
		/* A method on a type this package declares is an enum struct's or a
		methodmap's, and is allowed

		SourcePawn writes both inside the braces of the type they hang off,
		so what is refused is a method on anything else. */
		{"method on a type from elsewhere", "package decisions\n\nimport \"time\"\n\nfunc (t time.Month) M() int32 { return 1 }\n", "import of \"time\""},

		{"generic function", "package decisions\n\nfunc f[T any](x T) T { return x }\n", "generic function"},
		{"computed package variable", "package decisions\n\nfunc n() int32 { return 1 }\n\nvar counter = n()\n", "package-level initialiser"},
		{"import", "package decisions\n\nimport \"os\"\n\nfunc f() { _ = os.Args }\n", "import of \"os\""},
		{"unknown call", wrap("\t_ = TF2_IsMiniBoss(a)"), "unknown function"},
		{"anonymous struct", "package decisions\n\nfunc f(x struct{ A int32 }) {}\n", "anonymous struct"},
		{"embedded field", "package decisions\n\ntype A struct{ X int32 }\ntype B struct {\n\tA\n\tY int32\n}\n", "embedded struct field"},
		{"variadic", "package decisions\n\nfunc f(xs ...int32) {}\n", "variadic"},
		{"type alias", "package decisions\n\ntype T = int32\n", "type alias"},
		{"local type", wrap("\ttype T struct{ A int32 }\n\tvar t T\n\t_ = t"), "type declared inside a function"},
		{"and not", wrap("\t_ = a &^ 3"), "&^"},
		{"init function", "package decisions\n\nfunc init() {}\n", "init function"},
		{"imaginary", wrap("\t_ = 3i"), "imaginary"},
		{"function type field", "package decisions\n\ntype T struct {\n\tF func(int32) int32\n}\n", "function type"},
		{"array length from the literal", wrap("\t_ = [...]int32{1, 2}"), "written as [...]"},
		{"bare expression statement", wrap("\ta + 1"), "evaluated for nothing"},
	}

	cfg := gosubset.DefaultConfig()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			diags := gosubset.CheckSource(tc.name+".go", tc.src, cfg)
			if len(diags) == 0 {
				t.Fatalf("accepted %s; it must be refused", tc.name)
			}
			joined := gosubset.Join(diags).Error()
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("refusal does not name %q:\n%s", tc.want, joined)
			}
			for _, d := range diags {
				if d.Pos.Line == 0 || d.Pos.Column == 0 {
					t.Errorf("refusal without a position: %s", d.Error())
				}
				if d.Fix == "" {
					t.Errorf("refusal without a fix: %s", d.Construct)
				}
			}
		})
	}
}

func TestAccepted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
	}{
		{"sized arithmetic", wrap("\ta = a*2 + int32(b)")},
		// An enum struct's method, which SourcePawn writes inside its
		// braces. The receiver is spelled this and the emitter is the
		// second gate on what the body does.
		{"method on a struct this package declares", "package decisions\n\ntype T struct{ A int32 }\n\nfunc (t *T) Reset() { t.A = 0 }\n"},
		// A methodmap's method, which SourcePawn writes inside its braces
		// the same way. //sp:methodmap is what says which of the two it is.
		{"method on a named integer this package declares", "package decisions\n\ntype T int32\n\nconst A T = 1\n\nfunc (t T) M() int32 { return 1 }\n"},
		// Text a function is given is const char[] in SourcePawn, which
		// the plugin's own helpers take: a reason to log, a name to
		// compare. Only as a parameter, and the emitter is the second
		// gate on what the body does with one.
		{"string parameter", "package decisions\n\nfunc f(reason string) {}\n"},
		// A call in a case reaches the emitter, which accepts it only when
		// it is a constant the extern package names and refuses the rest.
		// This checker cannot tell them apart.
		{"case naming an extern constant", wrap("\tswitch a {\n\tcase f(a, b):\n\t}")},
		// A method call reaches the emitter, which has the types to say
		// whether the receiver is an extern the engine declares. This
		// checker cannot: it would have to know what t is.
		{"method call", "package decisions\n\ntype T struct{ A int32 }\n\nfunc f(t T) int32 { return t.M() }\n"},
		{"package state", "package decisions\n\nvar touching bool\n\nvar counts [4]int32 = [4]int32{1, 2, 3, 4}\n\nfunc f() int32 {\n\tif touching {\n\t\tcounts[0]++\n\t}\n\treturn counts[0]\n}\n"},
		{"if else", wrap("\tif a > 1 {\n\t\ta = 1\n\t} else if a < 0 {\n\t\ta = 0\n\t} else {\n\t\ta = 2\n\t}")},
		{"three clause for", wrap("\tfor i := int32(0); i < 8; i++ {\n\t\ta += i\n\t}")},
		{"range over array", "package decisions\n\nfunc f(xs [4]int32) int32 {\n\ttotal := int32(0)\n\tfor _, x := range xs {\n\t\ttotal += x\n\t}\n\treturn total\n}\n"},
		{"range over int", wrap("\tfor i := range 8 {\n\t\ta += int32(i)\n\t}")},
		{"switch with several values", wrap("\tswitch a {\n\tcase 1, 2:\n\t\ta = 0\n\tdefault:\n\t\ta = 1\n\t}")},
		{"min and max and len", "package decisions\n\nfunc f(xs [4]int32) int32 {\n\treturn min(max(xs[0], xs[1]), int32(len(xs)))\n}\n"},
		{"several results", "package decisions\n\nfunc f(a int32) (int32, bool) {\n\treturn a, a > 0\n}\n"},
		{"struct of arrays", "package decisions\n\nconst n = 4\n\ntype T struct {\n\tOrigin [3]float32\n\tSeen   [n]bool\n}\n\nfunc f(t T) float32 { return t.Origin[0] }\n"},
		{"enum constants", "package decisions\n\ntype P int32\n\nconst (\n\tA P = iota\n\tB\n)\n\nfunc f(p P) bool { return p == B }\n"},
		{"composite literal", "package decisions\n\ntype T struct{ X [3]float32 }\n\nfunc f() T {\n\treturn T{X: [3]float32{1.0, 2.0, 3.0}}\n}\n"},
		{"discarded result", "package decisions\n\nfunc g(a int32) int32 { return a }\n\nfunc f(a int32) {\n\tg(a)\n}\n"},
		{"mapped math call", "package decisions\n\nimport \"math\"\n\nfunc f(x float64) float64 { return math.Abs(x) }\n"},
	}

	cfg := gosubset.DefaultConfig()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if diags := gosubset.CheckSource(tc.name+".go", tc.src, cfg); len(diags) > 0 {
				t.Fatalf("refused code that is in the subset:\n%s", gosubset.Join(diags))
			}
		})
	}
}

func TestUnmappedImportMember(t *testing.T) {
	t.Parallel()

	src := "package decisions\n\nimport \"math\"\n\nfunc f(x float64) float64 { return math.Gamma(x) }\n"
	diags := gosubset.CheckSource("m.go", src, gosubset.DefaultConfig())
	if len(diags) != 1 || !strings.Contains(diags[0].Construct, "math.Gamma") {
		t.Fatalf("math.Gamma must be refused by name, got %v", diags)
	}
}

// TestRealDecisionCode is the question the subset exists to answer: can the
// decision code that runs today be written in it?
func TestRealDecisionCode(t *testing.T) {
	t.Parallel()

	cfg := gosubset.DefaultConfig()
	cfg.Natives = []string{"sqrtFloat"}

	diags, err := gosubset.CheckDir("testdata", cfg)
	if err != nil {
		t.Fatalf("checking testdata: %v", err)
	}
	if len(diags) > 0 {
		t.Fatalf("the real decision code does not fit the subset:\n%s", gosubset.Join(diags))
	}
}

func TestPositionIsUsable(t *testing.T) {
	t.Parallel()

	src := "package decisions\n\nfunc f() {\n\tgo f()\n}\n"
	diags := gosubset.CheckSource("pos.go", src, gosubset.DefaultConfig())
	if len(diags) != 1 {
		t.Fatalf("want one refusal, got %d", len(diags))
	}
	if diags[0].Pos.Line != 4 || diags[0].Pos.Column != 2 {
		t.Fatalf("want pos.go:4:2, got %s", diags[0].Pos)
	}
	if got := diags[0].Error(); !strings.HasPrefix(got, "pos.go:4:2: ") {
		t.Fatalf("the message must lead with file:line:col, got %q", got)
	}
}

func TestNothingRefusedIsNilError(t *testing.T) {
	t.Parallel()

	if err := gosubset.Join(nil); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

// parseFiles turns named sources into one package's files, so a test can say
// what is declared where.
func parseFiles(t *testing.T, srcs map[string]string) (*token.FileSet, []*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(srcs))
	for _, name := range slices.Sorted(maps.Keys(srcs)) {
		f, err := parser.ParseFile(fset, name, srcs[name], parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		files = append(files, f)
	}
	return fset, files
}

// TestMultiFilePackage is the reason CheckDir exists: a package split the way
// a package should be split has to check clean. The fixture is the shipped
// action selection code in the three files it should have been.
func TestMultiFilePackage(t *testing.T) {
	t.Parallel()

	dir := filepath.Join("testdata", "actionsel")
	diags, err := gosubset.CheckDir(dir, gosubset.DefaultConfig())
	if err != nil {
		t.Fatalf("checking %s: %v", dir, err)
	}
	if len(diags) > 0 {
		t.Fatalf("a legitimately multi-file package was refused:\n%s", gosubset.Join(diags))
	}
}

// TestSingleFileCannotSeeThePackage pins the limit CheckFile documents: on its
// own, a file of a real package is refused for the names its siblings declare.
// It is what CheckDir exists to avoid, so it must stay observable.
func TestSingleFileCannotSeeThePackage(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "actionsel", "shipped.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	diags := gosubset.CheckFile(fset, f, gosubset.DefaultConfig())
	if len(diags) == 0 {
		t.Fatal("a single file of a package cannot know Class or shouldTakeUpPosition; it must refuse them")
	}
	joined := gosubset.Join(diags).Error()
	for _, want := range []string{"the unknown type Class", "the unknown function shouldTakeUpPosition"} {
		if !strings.Contains(joined, want) {
			t.Errorf("single-file refusal does not name %q:\n%s", want, joined)
		}
	}
}

// TestUnknownNamesAcrossFiles is the one that matters: collecting names over
// the whole directory must not turn the checker into a rubber stamp. Every
// case declares a two-file package where the second file reaches for a name no
// file declares, and every case must still be refused.
func TestUnknownNamesAcrossFiles(t *testing.T) {
	t.Parallel()

	const declared = "package decisions\n\ntype Known int32\n\nfunc known(x int32) int32 { return x }\n"

	tests := []struct {
		name string
		use  string
		want string
	}{
		{
			"call to a function no file declares",
			"package decisions\n\nfunc f(x int32) int32 { return unknownFn(x) }\n",
			"a call to the unknown function unknownFn",
		},
		{
			"parameter of a type no file declares",
			"package decisions\n\nfunc f(x Unknown) int32 { return 0 }\n",
			"the unknown type Unknown",
		},
		{
			"result of a type no file declares",
			"package decisions\n\nfunc f() Unknown { return 0 }\n",
			"the unknown type Unknown",
		},
		{
			"struct field of a type no file declares",
			"package decisions\n\ntype T struct {\n\tF Unknown\n}\n",
			"the unknown type Unknown",
		},
		{
			"conversion to a type no file declares",
			"package decisions\n\nfunc f(x int32) int32 { return int32(Unknown(x)) }\n",
			"a call to the unknown function Unknown",
		},
		{
			"composite literal of a type no file declares",
			"package decisions\n\nfunc f() Known { return Unknown{} }\n",
			"the unknown type Unknown",
		},
		{
			"local var of a type no file declares",
			"package decisions\n\nfunc f() {\n\tvar u Unknown\n\t_ = u\n}\n",
			"the unknown type Unknown",
		},
		{
			"array element of a type no file declares",
			"package decisions\n\nfunc f(xs [4]Unknown) Known { return Known(0) }\n",
			"the unknown type Unknown",
		},
		{
			"a name a sibling file declares is still not a licence for its neighbours",
			"package decisions\n\nfunc f(x Known) Known { return Known(known(int32(x))) }\n\nfunc g(y Unknown) {}\n",
			"the unknown type Unknown",
		},
		{
			"a refused construct is still refused in the second file",
			"package decisions\n\nfunc f(m map[int32]Known) {}\n",
			"a map type",
		},
	}

	cfg := gosubset.DefaultConfig()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fset, files := parseFiles(t, map[string]string{"a_declared.go": declared, "b_use.go": tc.use})
			diags := gosubset.CheckFiles(fset, files, cfg)
			if len(diags) == 0 {
				t.Fatalf("accepted %s; widening collection must not accept unknown names", tc.name)
			}
			joined := gosubset.Join(diags).Error()
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("refusal does not name %q:\n%s", tc.want, joined)
			}
		})
	}
}

// TestCrossFileNamesAreKnown is the positive half: the names a sibling file
// declares are accepted in call, type and conversion position.
func TestCrossFileNamesAreKnown(t *testing.T) {
	t.Parallel()

	srcs := map[string]string{
		"a_types.go": "package decisions\n\ntype Score int32\n\ntype Candidate struct {\n\tS Score\n}\n",
		"b_funcs.go": "package decisions\n\nfunc weigh(s Score) Score { return s * 2 }\n",
		"c_use.go":   "package decisions\n\nfunc pick(c Candidate) Score {\n\treturn weigh(Score(int32(c.S) + 1))\n}\n",
	}
	fset, files := parseFiles(t, srcs)
	if diags := gosubset.CheckFiles(fset, files, gosubset.DefaultConfig()); len(diags) > 0 {
		t.Fatalf("refused cross-file names that are in the subset:\n%s", gosubset.Join(diags))
	}
}

// TestImportsDoNotLeakBetweenFiles: an import name is file-scoped, so one
// file's `import "math"` must not license math.Abs in the next file.
func TestImportsDoNotLeakBetweenFiles(t *testing.T) {
	t.Parallel()

	srcs := map[string]string{
		"a_imports.go": "package decisions\n\nimport \"math\"\n\nfunc g(x float64) float64 { return math.Abs(x) }\n",
		"b_bare.go":    "package decisions\n\nfunc h(x float64) float64 { return math.Abs(x) }\n",
	}
	fset, files := parseFiles(t, srcs)
	diags := gosubset.CheckFiles(fset, files, gosubset.DefaultConfig())
	if len(diags) != 1 {
		t.Fatalf("want one refusal for the file that did not import math, got %d:\n%s", len(diags), gosubset.Join(diags))
	}
	if diags[0].Pos.Filename != "b_bare.go" {
		t.Fatalf("the refusal must land in b_bare.go, got %s", diags[0].Pos)
	}
}

// TestCheckDirRefusesTwoPackages: CheckDir merges package-level names, so it
// has to be looking at one package.
func TestCheckDirRefusesTwoPackages(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	write("a.go", "package one\n\nfunc f() {}\n")
	write("b.go", "package two\n\nfunc g() {}\n")

	if _, err := gosubset.CheckDir(dir, gosubset.DefaultConfig()); err == nil {
		t.Fatal("two package names in one directory must be an error, not a merged name set")
	}
}
