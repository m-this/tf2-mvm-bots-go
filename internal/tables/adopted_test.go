package tables_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/plugin"
	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
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

	bodies, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}

	// Every generated file the plugin includes, whether it came from a table,
	// a body or an action. Adding one here is what keeps it from drifting.
	adopted := map[string][]byte{
		filepath.Join("source", "redbots3", "generated", "features.sp"):         tables.SourcePawnFeatures(),
		filepath.Join("source", "redbots3", "generated", "threat_priority.sp"):  spgen.EmitThreatPriority(),
		filepath.Join("source", "redbots3", "generated", "scan.sp"):             bodies["sourcepawn/scan.sp"],
		filepath.Join("source", "redbots3", "generated", "spysap.sp"):           bodies["sourcepawn/spysap.sp"],
		filepath.Join("source", "redbots3", "generated", "collectnearmoney.sp"): bodies["sourcepawn/collectnearmoney.sp"],
		filepath.Join("testbed", "stats", "generated", "wave_write.sp"):         tables.SourcePawnWaveWriter(),
	}

	// The behaviours and the bodies come from the lists rather than being
	// named here, so a port that adds one cannot forget to guard it. scan.sp
	// is already named above; roster is the generator's proof and ships
	// nowhere, so it is skipped.
	for _, b := range slices.Concat(body.Actions, body.All) {
		name := filepath.Base(b.Out)
		if strings.HasPrefix(name, "roster") {
			continue
		}
		adopted[filepath.Join("source", "redbots3", "generated", name)] = bodies[b.Out]
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
