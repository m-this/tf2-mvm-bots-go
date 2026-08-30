// Command gen writes every generated file. Nothing it writes is committed, and
// nothing it writes is edited by hand: make check regenerates and fails if the
// output moved.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	mvmbots "github.com/m-this/tf2-mvm-bots-go"
	"github.com/m-this/tf2-mvm-bots-go/internal/bindgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
	"github.com/m-this/tf2-mvm-bots-go/internal/upgrade"
)

func main() {
	out := flag.String("out", "gen", "directory to write generated files into")
	upstream := flag.String("upstream", "../tf2-mvm-bots", "the plugin repository, read for the include tree")
	flag.Parse()

	if err := run(*out, *upstream); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

// files is the whole output. A generator that emits a file not listed here does
// not exist as far as the reproducibility check is concerned.
func files(root string) (map[string][]byte, error) {
	sel, err := spgen.EmitActionSel()
	if err != nil {
		return nil, fmt.Errorf("emitting action selection: %w", err)
	}
	bodies, err := body.Generate(root)
	if err != nil {
		return nil, fmt.Errorf("emitting bodies: %w", err)
	}
	out := map[string][]byte{
		"sourcepawn/actionsel.sp":          []byte(sel.Data),
		"sourcepawn/actionsel_dispatch.sp": []byte(sel.Dispatch),
		"sourcepawn/attributes.sp":         tables.SourcePawnAttributes(),
		"sourcepawn/features.sp":           tables.SourcePawnFeatures(),
		"sourcepawn/threat_priority.sp":    spgen.EmitThreatPriority(),
		"sourcepawn/upgrade_rank.sp":       upgrade.SourcePawnRanking(),
		"sourcepawn/weapon_tuning.sp":      tables.SourcePawnTuning(),
		"sourcepawn/wave_write.sp":         tables.SourcePawnWaveWriter(),
		"go/arms/arms.go":                  tables.GoFeatureArms("arms"),
		"go/attr/attr.go":                  tables.GoAttributes("attr"),
		"go/wave/wave.go":                  tables.GoWaveParser("wave"),
	}
	for name, source := range bodies {
		if _, taken := out[name]; taken {
			return nil, fmt.Errorf("two generators write %s", name)
		}
		out[name] = source
	}
	return out, nil
}

/*
	The include tree the bindings are generated from

It lives inside the plugin's test-bed build directory because that is what
build.sh already downloads and caches. Bindings generated from includes older
than the compiler are bindings for the wrong API, which is mvm-z83.21.
*/
func includeRoot(upstream string) string {
	return filepath.Join(upstream, "testbed", "build")
}

// writeBindings emits the SourceMod API as Go. Absent includes are not an
// error: a fresh clone has not run build.sh yet, and the rest of the output
// does not depend on them.
func writeBindings(out, upstream string) error {
	root := includeRoot(upstream)
	if !isDir(root) {
		fmt.Fprintf(os.Stderr, "gen: no include tree at %s, skipping bindings\n", root)
		return nil
	}

	res, err := bindgen.Generate(bindgen.Options{Root: root, Package: "sm"})
	if err != nil {
		return fmt.Errorf("generating bindings: %w", err)
	}
	dir := filepath.Join(out, "go", "sm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("making %s: %w", dir, err)
	}
	if err := bindgen.Write(dir, res); err != nil {
		return fmt.Errorf("writing bindings: %w", err)
	}
	fmt.Fprintf(os.Stderr, "gen: %d binding files, %d refusals\n", len(res.Files), len(res.Refusals))
	return nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func run(out, upstream string) error {
	if err := os.RemoveAll(out); err != nil {
		return fmt.Errorf("clearing %s: %w", out, err)
	}
	// The bodies are generated from Go source, which is in the working
	// directory in a checkout and in the binary anywhere else.
	sources, done, err := mvmbots.SourceRoot(".")
	if err != nil {
		return fmt.Errorf("resolving the generator's own sources: %w", err)
	}
	defer done()

	emitted, err := files(sources)
	if err != nil {
		return err
	}
	for name, body := range emitted {
		path := filepath.Join(out, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("making %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return writeBindings(out, upstream)
}
