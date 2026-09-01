package plugin_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

/*
	TestThePluginCompiles is the gate's only whole-plugin proof

Everything else here checks a piece: a body against the file it replaces, a
generated file against the generator, a table against its golden. None of that
notices an unbalanced #if, a name the port took away that something else still
calls, or an include left pointing at a file that is gone. The port does all
three every time it cuts a function out, and until this existed the answer came
from tf2-archipelago's build, days later and somewhere else.

It found one the moment it was written: cutting PluginBot_SimulateFrame took its
opening #if and left the #endif, and the plugin had not compiled since.

Skipped when the staged include tree is not there, because building it fetches
seven projects. plugin/testbed/build.sh makes it.
*/
func TestThePluginCompiles(t *testing.T) {
	dir, err := plugin.Dir()
	if err != nil {
		t.Skipf("no plugin tree: %v", err)
	}

	build := filepath.Join(dir, "testbed", "build")
	spcomp := filepath.Join(build, "spcomp", "addons", "sourcemod", "scripting", "spcomp64")

	if _, err := os.Stat(spcomp); err != nil {
		t.Skipf("no staged compiler at %s: run plugin/testbed/build.sh", spcomp)
	}

	src := filepath.Join(build, "src")
	sm := filepath.Dir(spcomp)

	includes := []string{
		filepath.Join(sm, "include"),
		filepath.Join(src, "stocklib"),
		filepath.Join(src, "stocksoup-root"),
		filepath.Join(src, "cbasenpc", "scripting", "include"),
		filepath.Join(src, "actions", "sourcemod", "include"),
		filepath.Join(build, "ripext", "addons", "sourcemod", "scripting", "include"),
		filepath.Join(src, "tf2attributes", "scripting", "include"),
		filepath.Join(src, "tf_econ_data", "scripting", "include"),
		filepath.Join(src, "tf2utils", "scripting", "include"),
	}

	for _, name := range []string{
		filepath.Join("source", "tf2_defenderbots.sp"),
		filepath.Join("testbed", "stats", "mvmbots_stats.sp"),
		filepath.Join("testbed", "stats", "mvmbots_host.sp"),
		filepath.Join("testbed", "stats", "mvmbots_refund.sp"),
	} {
		t.Run(filepath.Base(name), func(t *testing.T) {
			path := filepath.Join(dir, name)

			args := make([]string, 0, len(includes)+3)
			for _, inc := range includes {
				args = append(args, "-i"+inc)
			}
			args = append(args,
				"-i"+filepath.Dir(path),
				"-o"+filepath.Join(t.TempDir(), "out.smx"),
				path,
			)

			out, err := exec.Command(spcomp, args...).CombinedOutput()
			if err != nil {
				t.Fatalf("compiling %s:\n%s", name, errorLines(string(out)))
			}
		})
	}
}

// errorLines is spcomp's output without the warnings, because a plugin this old
// carries hundreds of them and the error is what somebody has to read.
func errorLines(out string) string {
	var kept []string

	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "error") || strings.Contains(line, "Errors") {
			kept = append(kept, line)
		}
	}

	if len(kept) == 0 {
		return out
	}

	return strings.Join(kept, "\n")
}
