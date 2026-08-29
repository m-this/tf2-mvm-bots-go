package tables_test

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

// TestFeatureProofCatchesASwappedName is the proof about the proof. The bug
// that started this was two names exchanged in the array, so the test that says
// the table matches features.sp is worth nothing unless it fails when they are.
func TestFeatureProofCatchesASwappedName(t *testing.T) {
	t.Parallel()

	src := swapOnce(t, string(tables.SourcePawnFeatures()), "\t\"watch_idle_bots\",\n", "\t\"ammo_failover\",\n")
	src = swapOnce(t, src, "\t\"ammo_failover\",\n\t\"engineer_entrance_first\",", "\t\"watch_idle_bots\",\n\t\"engineer_entrance_first\",")

	mutated := parseFeaturesSP(t, src)

	same := true
	for i, f := range tables.Features {
		if mutated[i].name != f.Name || mutated[i].convar != f.ConVar() {
			same = false
		}
	}
	if same {
		t.Fatal("two exchanged names in FEATURE_NAME read back as the table, so the round trip proves nothing")
	}
}

// TestWaveProofCatchesARenamedField. A renamed field used to read as a zero;
// here it has to read as a failure.
func TestWaveProofCatchesARenamedField(t *testing.T) {
	t.Parallel()

	src := swapOnce(t, string(tables.SourcePawnWaveWriter()), `\"robot_kills\"`, `\"robots_killed\"`)
	mutated := parseWaveWriter(t, src)

	if len(mutated) != len(tables.WaveRecord) {
		t.Fatalf("mutation changed the field count: %d against %d", len(mutated), len(tables.WaveRecord))
	}

	same := true
	for i := range mutated {
		if mutated[i] != tables.WaveRecord[i] {
			same = false
		}
	}
	if same {
		t.Fatal("a renamed field read back as the table, so the round trip proves nothing")
	}
}

// swapOnce replaces from with to and insists it happened exactly once, so a
// mutation that quietly matched nothing cannot pass for a mutation.
func swapOnce(t *testing.T, src, from, to string) string {
	t.Helper()

	if n := strings.Count(src, from); n != 1 {
		t.Fatalf("%q appears %d times, wanted 1", from, n)
	}
	return strings.Replace(src, from, to, 1)
}
