/*
Package adopt is the list of generated files the plugin tree includes.

The plugin's build is a shell script and a compiler, so what it includes out of
a generated directory is committed there rather than produced at build time.
This is the one place that says which files those are: make adopt writes them
and the drift test in internal/tables reads the same list, so neither can
forget a file the other knows about.
*/
package adopt

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

// generated is where the plugin includes generated bodies from.
var generated = filepath.Join("source", "redbots3", "generated")

/*
Files is every adopted file, keyed by its path inside the plugin tree.

The behaviours and the bodies come from the lists rather than being named here,
so a port that adds one cannot forget to ship it. roster is the generator's
proof and ships nowhere, so it is skipped.
*/
func Files(root string) (map[string][]byte, error) {
	bodies, err := body.Generate(root)
	if err != nil {
		return nil, fmt.Errorf("generating the bodies: %w", err)
	}

	files := map[string][]byte{
		filepath.Join(generated, "features.sp"):                         tables.SourcePawnFeatures(),
		filepath.Join(generated, "threat_priority.sp"):                  spgen.EmitThreatPriority(),
		filepath.Join(generated, "scan.sp"):                             bodies["sourcepawn/scan.sp"],
		filepath.Join(generated, "spysap.sp"):                           bodies["sourcepawn/spysap.sp"],
		filepath.Join(generated, "collectnearmoney.sp"):                 bodies["sourcepawn/collectnearmoney.sp"],
		filepath.Join("testbed", "stats", "generated", "wave_write.sp"): tables.SourcePawnWaveWriter(),
	}

	for _, b := range slices.Concat(body.Actions, body.All) {
		if b.Out == "" {
			continue
		}
		name := filepath.Base(b.Out)
		if strings.HasPrefix(name, "roster") {
			continue
		}
		files[filepath.Join(generated, name)] = bodies[b.Out]
	}

	return files, nil
}

// Write puts every adopted file into the plugin tree, and says which ones moved.
func Write(pluginDir string, files map[string][]byte) ([]string, error) {
	moved := make([]string, 0, len(files))

	for name, want := range files {
		path := filepath.Join(pluginDir, name)

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
