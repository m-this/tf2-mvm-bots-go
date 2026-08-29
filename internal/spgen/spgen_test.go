package spgen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
)

func generate(t *testing.T, src string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "body.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	return spgen.GenerateDir(dir, spgen.Config{Prefix: "Gen_", Guard: "body"})
}

// TestRefusals covers the negative space: every construct the generator has no
// translation for has to come back named, with a position, rather than as
// plausible SourcePawn that decides something else.
func TestRefusals(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "a float64, which does not fit a cell",
			src:  "package body\n\nfunc F(x float64) float64 { return x }\n",
			want: "does not fit a 32-bit cell",
		},
		{
			name: "several results, which spgen does not turn into parameters yet",
			src:  "package body\n\nfunc F(x int32) (int32, bool) { return x, true }\n",
			want: "more than one result",
		},
		{
			name: "an import, because the package has to be self-contained",
			src:  "package body\n\nimport \"math\"\n\nfunc F(x float32) float32 { return float32(math.Abs(float64(x))) }\n",
			want: "an import",
		},
		{
			name: "a slice, refused by the subset before spgen sees it",
			src:  "package body\n\nfunc F(xs []int32) int32 { return xs[0] }\n",
			want: "a slice type",
		},
		{
			name: "a map, refused by the subset",
			src:  "package body\n\nfunc F(m map[int32]int32) int32 { return m[0] }\n",
			want: "a map type",
		},
		{
			name: "a rounding conversion, which is a decision the author owes",
			src:  "package body\n\nfunc F(x float32) int32 { return int32(x) }\n",
			want: "a conversion between float and int",
		},
		{
			name: "a for with no condition, because every loop needs a bound",
			src:  "package body\n\nfunc F(x int32) int32 {\n\tfor {\n\t\tx++\n\t}\n}\n",
			want: "no bound at all",
		},
		{
			name: "a package-level variable, because a body owns no state",
			src:  "package body\n\nvar n int32\n\nfunc F() int32 { return n }\n",
			want: "package-level variable",
		},
		{
			name: "an if with an init statement, which SourcePawn has no place for",
			src:  "package body\n\nfunc F(x int32) int32 {\n\tif y := x; y > 0 {\n\t\treturn y\n\t}\n\treturn 0\n}\n",
			want: "an if with an init statement",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := generate(t, tc.src)
			if err == nil {
				t.Fatalf("it generated:\n%s", out)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal reads %q, wanted it to name %q", err, tc.want)
			}
		})
	}
}

// TestAnEmptyPrefixIsRefused: Action and RoundState are SourceMod's names, so
// an unprefixed emission collides with the plugin it is included into.
func TestAnEmptyPrefixIsRefused(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "body.go"), []byte("package body\n\nfunc F() int32 { return 0 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := spgen.GenerateDir(dir, spgen.Config{Guard: "body"}); err == nil {
		t.Fatal("an empty prefix was accepted")
	}
}

// TestTranslations covers the constructs actionsel does not use but the
// generator claims: arrays, loops, locals and the compound operators.
func TestTranslations(t *testing.T) {
	src := `package body

// Weight is a named type over a cell, so it comes out as an enum.
type Weight int32

const (
	WeightLow Weight = iota
	WeightHigh
)

// Sample is a struct of cells, so it comes out as an enum struct.
type Sample struct {
	Count  int32
	Score  float32
	Live   bool
	Recent [4]int32
}

func Total(s Sample, class int32) int32 {
	total := int32(0)
	for i := int32(0); i < int32(len(s.Recent)); i++ {
		total += s.Recent[i]
	}
	total <<= 1
	if s.Live {
		total -= class
	}
	switch class {
	case 0, 1:
		total = total * int32(WeightHigh)
	default:
		total = total + int32(WeightLow)
	}
	return total
}
`
	out, err := generate(t, src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"enum Gen_Weight\n{\n\tGen_WeightLow = 0,\n\tGen_WeightHigh = 1\n};",
		"enum struct Gen_Sample\n{\n\tint Count;\n\tfloat Score;\n\tbool Live;\n\tint Recent[4];\n}",
		"stock int Gen_Total(Gen_Sample s, int class_)",
		"for (int i = 0; (i < sizeof(s.Recent)); i++)",
		"total <<= 1;",
		"case 0, 1:",
		"default:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the output has no %q. It reads:\n%s", want, out)
		}
	}
}
