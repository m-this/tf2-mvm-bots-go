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

	src := exchangeNames(t, string(tables.SourcePawnFeatures()), "watch_idle_bots", "ammo_failover")

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
// swapOnce replaces from with to, and fails if from is not unique: a mutation
// that landed somewhere other than where it was aimed proves nothing.
func swapOnce(t *testing.T, src, from, to string) string {
	t.Helper()

	if n := strings.Count(src, from); n != 1 {
		t.Fatalf("%q appears %d times, wanted 1", from, n)
	}
	return strings.Replace(src, from, to, 1)
}

/*
	exchangeNames swaps two names in FEATURE_NAME and leaves the enum alone

Matched as a whole entry line, because the names also appear in the file's own
comment about this very bug. Both have to be there exactly once, which is what
makes the swap a swap rather than a rename, and a placeholder carries the
exchange across so neither half is caught by the other's replacement.
*/
func exchangeNames(t *testing.T, src, a, b string) string {
	t.Helper()

	entry := func(name string) string { return "\t\"" + name + "\",\n" }
	for _, name := range []string{a, b} {
		if n := strings.Count(src, entry(name)); n != 1 {
			t.Fatalf("%q appears %d times as an entry line, wanted 1", name, n)
		}
	}
	src = strings.Replace(src, entry(a), entry("\x00"), 1)
	src = strings.Replace(src, entry(b), entry(a), 1)
	return strings.Replace(src, entry("\x00"), entry(b), 1)
}
