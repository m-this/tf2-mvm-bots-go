package bindings

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

// goldenCases pin the emitter against real includes: the two extension
// headers the plugin leans on hardest, the processors header whose two
// typesets carry every behaviour callback the plugin writes, a CBaseNPC
// header full of inheriting methodmaps, and a plain SourceMod one.
var goldenCases = []struct {
	name    string
	include string
}{
	{"tf2utils", "src/tf2utils/scripting/include/tf2utils.inc"},
	{"actions", "src/actions/sourcemod/include/actions.inc"},
	{"actions_processors", "src/actions/sourcemod/include/actions_processors.inc"},
	{"locomotion", "src/cbasenpc/scripting/include/cbasenpc/nextbot/locomotion.inc"},
	{"tf2", "spcomp/addons/sourcemod/scripting/include/tf2.inc"},
}

func TestGolden(t *testing.T) {
	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(includeRoot(t), tc.include)
			if _, err := os.Stat(path); err != nil {
				t.Skipf("include not present: %v", err)
			}
			f, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parsing: %v", err)
			}
			// The generated header records the include's path. Pin it to the
			// tree-relative one so the golden does not depend on where the
			// plugin repository sits.
			f.Path = tc.include
			out, err := Emit(f, Options{Package: "sp"})
			if err != nil {
				t.Fatalf("emitting: %v", err)
			}
			compareGolden(t, tc.name+".go.txt", string(out.Source))
			compareGolden(t, tc.name+".refusals.txt", refusalReport(f.Refusals, out.Refusals))
		})
	}
}

func refusalReport(parse, emit []Refusal) string {
	var b strings.Builder
	b.WriteString("# refused while parsing\n")
	for _, r := range parse {
		b.WriteString(r.String() + "\n")
	}
	b.WriteString("\n# refused while emitting\n")
	for _, r := range emit {
		b.WriteString(r.String() + "\n")
	}
	return b.String()
}

func compareGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden (run go test -update): %v", err)
	}
	if string(want) != got {
		t.Errorf("%s differs from the golden file; run go test -update and read the diff", name)
	}
}
