package tables_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
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

	root := upstream.SkipOrFail(t)

	adopted := map[string][]byte{
		filepath.Join("source", "redbots3", "generated", "features.sp"):        tables.SourcePawnFeatures(),
		filepath.Join("source", "redbots3", "generated", "threat_priority.sp"): spgen.EmitThreatPriority(),
		filepath.Join("testbed", "stats", "generated", "wave_write.sp"):        tables.SourcePawnWaveWriter(),
	}

	for name, want := range adopted {
		t.Run(name, func(t *testing.T) {
			got, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatalf("%v: run make gen and copy it across", err)
			}
			if string(got) == string(want) {
				return
			}
			t.Errorf("%s has drifted from the generator; refresh it with\n"+
				"\tmake -C tf2-mvm-bots-go gen && cp tf2-mvm-bots-go/gen/sourcepawn/%s %s",
				name, filepath.Base(name), name)
		})
	}
}
