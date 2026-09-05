/*
Package adopt places the generated SourcePawn the plugin tree commits.

The plugin's build is a shell script and a compiler, so what it includes out of
a generated directory is committed there rather than produced at build time.
Files is the placement rule over the generator's whole output: make adopt
writes it and the drift test in internal/tables reads it, so a generated file
is shipped, or it is proof, and never quietly neither.
*/
package adopt

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/generated"
)

// pluginDir is where the plugin includes generated SourcePawn from.
var pluginDir = filepath.Join("source", "redbots3", "generated")

// testbedDir is where the test-bed's statistics plugin includes it from.
var testbedDir = filepath.Join("testbed", "stats", "generated")

// testbed names the output files that are the test-bed's and not the plugin's.
var testbed = map[string]bool{
	"sourcepawn/wave_write.sp": true,
}

// Files is every adopted file, keyed by its path inside the plugin tree.
func Files(root string) (map[string][]byte, error) {
	emitted, err := generated.Files(root)
	if err != nil {
		return nil, err
	}

	files := make(map[string][]byte, len(emitted))
	for name, source := range emitted {
		if !strings.HasPrefix(name, "sourcepawn/") || generated.Proof(name) {
			continue
		}
		dir := pluginDir
		if testbed[name] {
			dir = testbedDir
		}
		files[filepath.Join(dir, filepath.Base(name))] = source
	}

	return files, nil
}

// Write puts every adopted file into the plugin tree, and says which ones moved.
func Write(tree string, files map[string][]byte) ([]string, error) {
	moved := make([]string, 0, len(files))

	for name, want := range files {
		path := filepath.Join(tree, name)

		got, err := os.ReadFile(path) //nolint:gosec // the path is this repository's own tree
		if err == nil && string(got) == string(want) {
			continue
		}
		if err := os.WriteFile(path, want, 0o644); err != nil { //nolint:gosec // source is 0644
			return nil, fmt.Errorf("writing %s: %w", path, err)
		}
		moved = append(moved, name)
	}

	slices.Sort(moved)

	return moved, nil
}
