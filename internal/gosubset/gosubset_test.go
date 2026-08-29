package gosubset_test

import (
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
		{"non constant case", wrap("\tswitch a {\n\tcase f(a, b):\n\t}"), "not a constant"},
		{"map type", "package decisions\n\ntype T struct {\n\tM map[int32]int32\n}\n", "map type"},
		{"slice type", "package decisions\n\ntype T struct {\n\tS []int32\n}\n", "slice type"},
		{"slice expr", "package decisions\n\nfunc f(x [4]int32) {\n\t_ = x[1:2]\n}\n", "slice expression"},
		{"interface type", "package decisions\n\ntype T interface{}\n", "interface type"},
		{"closure", wrap("\tg := func() int32 { return 1 }\n\t_ = g"), "function literal"},
		{"pointer param", "package decisions\n\nfunc f(p *int32) {}\n", "pointer type"},
		{"pointer result", "package decisions\n\nfunc f() *int32 { return nil }\n", "pointer type"},
		{"address of", wrap("\tp := &a\n\t_ = p"), "taking an address"},
		{"unsized int", "package decisions\n\nfunc f(x int) {}\n", "32 bits"},
		{"string type", "package decisions\n\nfunc f(s string) {}\n", "no strings"},
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
		{"method receiver", "package decisions\n\ntype T struct{ A int32 }\n\nfunc (t T) M() int32 { return t.A }\n", "method receiver"},
		{"method call", "package decisions\n\ntype T struct{ A int32 }\n\nfunc f(t T) int32 { return t.M() }\n", "method call"},
		{"generic function", "package decisions\n\nfunc f[T any](x T) T { return x }\n", "generic function"},
		{"package variable", "package decisions\n\nvar counter int32\n", "package-level variable"},
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
