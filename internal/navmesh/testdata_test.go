package navmesh

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

// shippedMaps is every Valve MvM map the mod has a config for and whose nav
// mesh is in the game files. The nav files come out of tf2_misc_dir.vpk, gzipped
// so the test data is a fifth of its size; they are the game's own, unedited.
var shippedMaps = []string{
	"mvm_bigrock",
	"mvm_coaltown",
	"mvm_decoy",
	"mvm_ghost_town",
	"mvm_mannhattan",
	"mvm_mannworks",
	"mvm_rottenburg",
}

func loadMap(t *testing.T, name string) *Mesh {
	t.Helper()

	m, err := LoadFile(filepath.Join("testdata", name+".nav.gz"))
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}

	return m
}

// configDir is the mod's map config directory in the upstream repository. The
// configs are not copied here: they are the plugin's and a copy would be a
// second source of truth for the same spots.
func configDir(t *testing.T) string {
	t.Helper()

	dir := filepath.Join(plugin.SkipOrFail(t), "configs", "defenderbots", "map")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no map configs at %s", dir)
	}

	return dir
}

func loadConfigs(t *testing.T) []*MapConfig {
	t.Helper()

	cfgs, err := LoadMapConfigs(configDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfgs) == 0 {
		t.Fatal("no map configs found")
	}

	return cfgs
}

// haveNav reports whether a map config has a nav mesh in the test data. Twenty
// of the twenty-seven configs are for community maps whose .nav files are not in
// the game install, so they are not checkable here at all.
func haveNav(name string) bool {
	_, err := os.Stat(filepath.Join("testdata", name+".nav.gz"))
	return err == nil
}
