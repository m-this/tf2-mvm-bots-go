package spbody_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spbody"
	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// goldens are the fixtures whose emitted SourcePawn is pinned. shapes is one of
// everything the subset has; the other two are the shapes that took a design
// decision and are worth reading on their own.
var goldens = []string{"shapes", "handle", "variadic"}

// TestShapesGolden emits each and compares it with the file a reviewer reads.
// The golden file is the review: a change to the emitter that nobody meant
// shows up here as a diff and nowhere else.
func TestShapesGolden(t *testing.T) {
	declared, err := spbody.ExternsFromDir("../engine")
	if err != nil {
		t.Fatalf("reading the extern declarations: %v", err)
	}
	for _, name := range goldens {
		t.Run(name, func(t *testing.T) {
			goldenOf(t, name, spbody.Config{
				Prefix: "Go_", Externs: declared.Funcs, Tags: declared.Tags,
			})
		})
	}
}

func goldenOf(t *testing.T, name string, cfg spbody.Config) {
	t.Helper()
	g, err := spbody.GenerateDir("testdata/"+name, cfg)
	if err != nil {
		t.Fatalf("generating %s: %v", name, err)
	}
	golden := "testdata/" + name + ".sp"
	if *update {
		if err := os.WriteFile(golden, []byte(g.Source), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v: run go test ./internal/spbody -update", err)
	}
	if string(want) != g.Source {
		t.Errorf("the emitted SourcePawn moved:\n--- %s\n+++ emitted\n%s", golden, diff(string(want), g.Source))
	}
}

// TestTheGoldenOutputCompiles is the second half of the golden test. A golden
// file only says the output did not move; compiling says it is SourcePawn.
func TestTheGoldenOutputCompiles(t *testing.T) {
	tc := spshell.ForTest(t)
	g, err := spbody.GenerateDir("testdata/shapes", spbody.Config{Prefix: "Go_"})
	if err != nil {
		t.Fatalf("generating the shapes body: %v", err)
	}
	if err := tc.Compile(t.Context(), "testdata/shapes_smoke.sp", map[string]string{"shapes.sp": g.Source}); err != nil {
		t.Fatalf("compiling the emitted SourcePawn: %v", err)
	}
}

// The refusals. Each fixture holds one construct that type checks as Go and has
// no faithful SourcePawn, and the test is that the refusal names it. A
// generator that emitted something plausible for any of these would produce a
// plugin that compiles and is wrong.
func TestRefusals(t *testing.T) {
	cases := []struct {
		dir  string
		want string
	}{
		{"write_to_array_param", "which SourcePawn passes by reference and Go copies"},
		{"reserved_word", "is a SourcePawn keyword"},
		{"struct_literal", "SourcePawn has no struct literal"},
		{"unknown_extern", "is not an extern this emission was given"},
		{"enum_without_constants", "declares no constants"},
		{"array_call_as_value", "used as a value"},
		{"default_before_plain", "has no default and follows one that does"},
		{"unclosed_handle", "is a handle and nothing closes it"},
		{"handle_in_return", "is returned or read by the return that closes it"},
	}
	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			declared, err := spbody.ExternsFromDir("../engine")
			if err != nil {
				t.Fatalf("reading the extern declarations: %v", err)
			}
			_, err = spbody.GenerateDir(filepath.Join("testdata", tc.dir), spbody.Config{
				Prefix: "Go_", Externs: declared.Funcs, Tags: declared.Tags,
			})
			if err == nil {
				t.Fatal("the construct was translated; it has no faithful SourcePawn")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal says %q, and does not name %q", err, tc.want)
			}
		})
	}
}

// diff is enough to see which line moved. A real diff library for a golden file
// nobody edits by hand would be a dependency bought for nothing.
func diff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	for i := range max(len(w), len(g)) {
		wl, gl := "", ""
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl == gl {
			continue
		}
		b.WriteString("-" + wl + "\n+" + gl + "\n")
	}
	return b.String()
}
