// Command gen writes every generated file. Nothing it writes is committed, and
// nothing it writes is edited by hand: make check regenerates and fails if the
// output moved.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

func main() {
	out := flag.String("out", "gen", "directory to write generated files into")
	flag.String("upstream", "../tf2-mvm-bots", "the plugin repository, read for round-trip checks")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

// files is the whole output. A generator that emits a file not listed here does
// not exist as far as the reproducibility check is concerned.
func files() map[string][]byte {
	return map[string][]byte{
		"sourcepawn/features.sp":   tables.SourcePawnFeatures(),
		"sourcepawn/wave_write.sp": tables.SourcePawnWaveWriter(),
		"go/arms/arms.go":          tables.GoFeatureArms("arms"),
		"go/wave/wave.go":          tables.GoWaveParser("wave"),
	}
}

func run(out string) error {
	if err := os.RemoveAll(out); err != nil {
		return fmt.Errorf("clearing %s: %w", out, err)
	}
	for name, body := range files() {
		path := filepath.Join(out, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("making %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return nil
}
