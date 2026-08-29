package spgen_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
)

var update = flag.Bool("update", false, "rewrite the golden SourcePawn from internal/actionsel")

// goldenData is the file the plugin includes and the differential test
// compiles. It is committed so that a change to the decision shows up as a
// diff in SourcePawn, which is the form the reviewer has to trust.
const (
	goldenData     = "testdata/actionsel.sp"
	goldenDispatch = "testdata/actionsel_dispatch.sp"
)

func emit(t *testing.T) spgen.ActionSel {
	t.Helper()
	out, err := spgen.EmitActionSel()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGoldenSourcePawnIsUpToDate(t *testing.T) {
	out := emit(t)
	for path, want := range map[string]string{goldenData: out.Data, goldenDispatch: out.Dispatch} {
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil { //nolint:gosec // G306: generated SourcePawn is source
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path) //nolint:gosec // G304: a path written in this file
		if err != nil {
			t.Fatalf("%v: run go test ./internal/spgen -update", err)
		}
		if string(got) != want {
			t.Errorf("%s is stale: run go test ./internal/spgen -update", path)
		}
	}
}

// TestGenerationIsReproducible fails if two runs disagree, which is what a map
// iterated without sorting looks like from the outside.
func TestGenerationIsReproducible(t *testing.T) {
	a, b := emit(t), emit(t)
	if a.Data != b.Data || a.Dispatch != b.Dispatch {
		t.Error("two builds of the same decision differ")
	}
}

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: a path written in this package's tests
	if err != nil {
		return "", err
	}
	return string(b), nil
}
