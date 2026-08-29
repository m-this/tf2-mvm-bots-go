package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The shape of two breaks by one engineer: the first in the old order, the
// second with the entrance first and a teleport paying for it.
const setupLines = `{"event":"wave_begin","map":"mvm_decoy","wave":1}
{"event":"setup","map":"mvm_decoy","wave":1,"who":"Dell","break_s":60.0,"walked":4200,"teleports":0,"teleported":0,"sentry_at_s":12.0,"dispenser_at_s":20.0,"entrance_at_s":48.0,"exit_at_s":-1.0}
{"event":"setup","map":"mvm_decoy","wave":2,"who":"Dell","break_s":60.0,"walked":1400,"teleports":1,"teleported":3100,"sentry_at_s":14.0,"dispenser_at_s":22.0,"entrance_at_s":4.0,"exit_at_s":30.0}
`

func writeSetupFile(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "run.jsonl")

	if err := os.WriteFile(path, []byte(setupLines), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestLoadSetupTakesOnlySetupLines(t *testing.T) {
	setups, err := loadSetup(writeSetupFile(t))
	if err != nil {
		t.Fatal(err)
	}

	if len(setups) != 2 {
		t.Fatalf("read %d setup lines, want 2", len(setups))
	}

	if setups[1].EntranceAt != 4.0 {
		t.Errorf("entrance at %.1fs, want 4.0", setups[1].EntranceAt)
	}

	if setups[1].Teleports != 1 {
		t.Errorf("teleports %d, want 1", setups[1].Teleports)
	}
}

// A building that never went up carries -1 and must not be averaged in as a
// zero, which would read as the fastest build in the file.
func TestRollUpSetupSkipsBuildingsThatNeverStood(t *testing.T) {
	setups, err := loadSetup(writeSetupFile(t))
	if err != nil {
		t.Fatal(err)
	}

	rolled := rollUpSetup(setups)

	dell := rolled["Dell"]
	if dell == nil {
		t.Fatal("no rollup for Dell")
	}

	if dell.breaks != 2 {
		t.Fatalf("breaks %d, want 2", dell.breaks)
	}

	exit := dell.timing["exit"]
	if exit == nil || exit.built != 1 {
		t.Fatalf("exit built in %v breaks, want 1", exit)
	}

	if got := exit.mean(); got != 30.0 {
		t.Errorf("exit mean %.1fs, want 30.0", got)
	}

	if got := dell.timing["entrance"].mean(); got != 26.0 {
		t.Errorf("entrance mean %.1fs, want 26.0", got)
	}
}

func TestBuiltCountAndMeanSecondsSayNotBuiltRatherThanZero(t *testing.T) {
	empty := &setupRollup{timing: map[string]*buildTiming{}}

	if got := builtCount(empty, "exit"); got != 0 {
		t.Errorf("built count %d, want 0", got)
	}

	if got := meanSeconds(empty, "exit"); got != "not built" {
		t.Errorf("mean %q, want %q", got, "not built")
	}
}
