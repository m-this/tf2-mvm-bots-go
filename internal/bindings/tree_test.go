package bindings

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

// includeRoot is the include tree inside the plugin's test-bed build
// directory. The tree is resolved by internal/plugin, which reads a relative
// MVMBOTS_PLUGIN from the repository root rather than from this package: doing
// it here got it wrong, and these proofs skipped in silence.
func includeRoot(t *testing.T) string {
	t.Helper()

	return filepath.Join(plugin.SkipOrFail(t), "testbed", "build")
}

// includeFiles lists every .inc under the tree, skipping the prebuilt copies
// which duplicate src/ declaration for declaration.
func includeFiles(t *testing.T) []string {
	t.Helper()
	root := includeRoot(t)
	if _, err := os.Stat(root); err != nil {
		t.Skipf("include tree not present: %v", err)
	}
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "prebuilt" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".inc") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking include tree: %v", err)
	}
	return paths
}

func TestParseWholeTree(t *testing.T) {
	var cov Coverage
	for _, path := range includeFiles(t) {
		f, err := ParseFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		cov.Add(f)
	}
	if _, err := cov.WriteTo(os.Stdout); err != nil {
		t.Fatal(err)
	}
	for i, r := range cov.Refusals {
		if i >= 200 {
			break
		}
		t.Logf("refused %s", r)
	}
	// These are the ground-truth counts of `^native`, `^methodmap` and
	// `^property` lines in the tree. A drop means the parser started
	// swallowing declarations inside skipDeclaration.
	for _, want := range []struct {
		name string
		got  int
		min  int
	}{
		{"natives", cov.Natives, 1175},
		{"methodmaps", cov.Methodmaps, 112},
		{"properties", cov.Properties, 310},
		{"methods", cov.Methods, 1531},
		{"stocks", cov.Stocks, 914},
	} {
		if want.got < want.min {
			t.Errorf("%s = %d, want at least %d", want.name, want.got, want.min)
		}
	}
	if len(cov.Refusals) > 80 {
		t.Errorf("refusals = %d, want the known set to stay small", len(cov.Refusals))
	}
}

// TestEmitWholeTree proves the emitter produces syntactically valid Go for
// every include in the tree, and reports what it turned down.
func TestEmitWholeTree(t *testing.T) {
	byReason := map[string]int{}
	total := 0
	// The tree is emitted in path order, threading the constants each file
	// resolves into the next. It is the same job a generation driver does,
	// and it is what turns a cross-file `#define` from a refusal into a
	// constant.
	known := map[string]int64{}
	for _, path := range includeFiles(t) {
		f, err := ParseFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		out, err := Emit(f, Options{Package: "sp", Constants: known})
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		maps.Copy(known, out.Constants)
		for _, r := range out.Refusals {
			byReason[r.Kind+": "+r.Reason]++
			total++
		}
	}
	for _, reason := range slices.Sorted(maps.Keys(byReason)) {
		t.Logf("%5d  %s", byReason[reason], reason)
	}
	t.Logf("emit refusals %d", total)
	if total > 200 {
		t.Errorf("emit refusals = %d, want the known set to stay small", total)
	}
}
