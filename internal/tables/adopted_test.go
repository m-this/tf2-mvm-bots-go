package tables_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/adopt"
	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
)

/*
	TestAdoptedFilesMatchTheGenerator

The plugin's build is a shell script and a compiler, so the generated files it
includes are committed there rather than produced at build time. The cost of
that is drift, and until now nothing paid it: threat_priority.sp has been a
committed copy that nobody checked, which is exactly the two-places-for-one-fact
this repository exists to end.

This is the check. Every file the plugin includes out of a generated directory
has to be byte for byte what the generator writes today, and the failure says
which one and what to run.
*/
func TestAdoptedFilesMatchTheGenerator(t *testing.T) {
	t.Parallel()

	root := plugin.SkipOrFail(t)

	adopted, err := adopt.Files("../..")
	if err != nil {
		t.Fatalf("listing the adopted files: %v", err)
	}

	for name, want := range adopted {
		t.Run(name, func(t *testing.T) {
			got, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatalf("%v: run make adopt", err)
			}
			if string(got) != string(want) {
				t.Errorf("%s has drifted from the generator; run make adopt", name)
			}
		})
	}
}
