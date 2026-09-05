package adopt_test

import (
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/adopt"
)

// TestFilesPlacesEveryShippedOutput pins the placement rule: the plugin's files
// under its generated directory, the test-bed's under its own, and the proof
// packages nowhere.
func TestFilesPlacesEveryShippedOutput(t *testing.T) {
	t.Parallel()

	files, err := adopt.Files("../..")
	if err != nil {
		t.Fatalf("listing the adopted files: %v", err)
	}

	plugin := filepath.Join("source", "redbots3", "generated")
	testbed := filepath.Join("testbed", "stats", "generated")

	for _, name := range []string{"attributes.sp", "dispatch.sp", "scan.sp", "threat_priority.sp", "upgrade_rank.sp"} {
		if _, ok := files[filepath.Join(plugin, name)]; !ok {
			t.Errorf("%s is shipped and not placed", name)
		}
	}
	if _, ok := files[filepath.Join(testbed, "wave_write.sp")]; !ok {
		t.Error("wave_write.sp is the test-bed's and not placed there")
	}
	for _, name := range []string{"roster.sp", "roster_dhooks.sp", "actionsel.sp", "actionsel_dispatch.sp"} {
		if _, ok := files[filepath.Join(plugin, name)]; ok {
			t.Errorf("%s is proof and must not ship", name)
		}
	}
	for name := range files {
		if filepath.Ext(name) != ".sp" {
			t.Errorf("%s is not SourcePawn and has no place in the plugin tree", name)
		}
	}
}
