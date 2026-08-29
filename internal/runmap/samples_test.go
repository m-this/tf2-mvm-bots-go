package runmap

import (
	"strings"
	"testing"
)

// A results file is a mixture: wave lines, bot samples, building samples, and
// whatever else the plugin writes. Only two of those are drawn.
const resultsFixture = `{"event":"wave_begin","map":"mvm_decoy","wave":1,"red":6,"bots":6}
{"event":"bot","wave":1,"t":0.5,"clock":10.5,"who":"Waldo","class":"engineer","at":[100,200,64],"action":"DefenderEngineerIdle","pathing":1}
{"event":"bot","wave":1,"t":1.0,"clock":11.0,"who":"Waldo","class":"engineer","at":[140,220,64],"action":"DefenderEngineerIdle","pathing":1}
{"event":"bot","wave":1,"t":0.5,"clock":10.5,"who":"Ada","class":"medic","at":[300,400,64],"action":"CTFBotMedicHeal","pathing":0}
{"event":"building","map":"mvm_decoy","wave":1,"t":1.0,"clock":11.0,"owner":"Waldo","type":"sentry","mode":0,"level":3,"at":[150,250,64]}
{"event":"bot","wave":2,"t":0.5,"clock":99.0,"who":"Waldo","class":"engineer","at":[160,260,64],"action":"DefenderEngineerIdle","pathing":1}
{"event":"stall","map":"mvm_decoy","wave":2,"ms":300}
not json at all
{"event":"bot","wave":2,"t":1.0,"clock":99
`

func read(t *testing.T) Run {
	t.Helper()

	run, err := Read(strings.NewReader(resultsFixture))
	if err != nil {
		t.Fatal(err)
	}

	return run
}

func TestReadSplitsWavesAndKeepsOrder(t *testing.T) {
	run := read(t)

	if run.Map != "mvm_decoy" {
		t.Errorf("map %q, want mvm_decoy", run.Map)
	}

	if len(run.Waves) != 2 {
		t.Fatalf("%d waves, want 2", len(run.Waves))
	}

	if run.Waves[0].Number != 1 || run.Waves[1].Number != 2 {
		t.Errorf("waves %d and %d, want 1 and 2", run.Waves[0].Number, run.Waves[1].Number)
	}
}

// A truncated last line is what a crashed run leaves behind, and it must not
// cost the file everything written before it.
func TestReadKeepsWhatCameBeforeATruncatedLine(t *testing.T) {
	run := read(t)

	second := run.Waves[1]
	if len(second.Tracks) != 1 {
		t.Fatalf("%d tracks in wave 2, want 1", len(second.Tracks))
	}

	if got := len(second.Tracks[0].Samples); got != 1 {
		t.Errorf("%d samples survived the truncation, want 1", got)
	}
}

func TestTracksAreGroupedByBotAndSortedByName(t *testing.T) {
	first := read(t).Waves[0]

	if len(first.Tracks) != 2 {
		t.Fatalf("%d tracks, want 2", len(first.Tracks))
	}

	if first.Tracks[0].Who != "Ada" || first.Tracks[1].Who != "Waldo" {
		t.Errorf("tracks %q and %q, want Ada then Waldo", first.Tracks[0].Who, first.Tracks[1].Who)
	}

	if got := len(first.Tracks[1].Samples); got != 2 {
		t.Errorf("Waldo has %d samples, want 2", got)
	}

	if first.Tracks[1].Class != "engineer" {
		t.Errorf("Waldo is a %q, want engineer", first.Tracks[1].Class)
	}
}

func TestBuildingsLandOnTheirOwnWave(t *testing.T) {
	run := read(t)

	if got := len(run.Waves[0].Buildings); got != 1 {
		t.Fatalf("%d buildings in wave 1, want 1", got)
	}

	if got := len(run.Waves[1].Buildings); got != 0 {
		t.Errorf("%d buildings in wave 2, want 0", got)
	}
}

func TestClassesAreListedOnce(t *testing.T) {
	got := read(t).Waves[0].Classes()

	if len(got) != 2 || got[0] != "engineer" || got[1] != "medic" {
		t.Errorf("classes %v, want [engineer medic]", got)
	}
}

// A sample with no position cannot be drawn and must not become one at the
// origin, which on a real map is a corner nobody stands in.
func TestASampleWithNoPositionIsDropped(t *testing.T) {
	run, err := Read(strings.NewReader(
		`{"event":"bot","wave":1,"who":"Waldo","class":"engineer","at":[]}` + "\n"))
	if err != nil {
		t.Fatal(err)
	}

	if len(run.Waves) != 0 {
		t.Errorf("%d waves, want none: the only sample had no position", len(run.Waves))
	}
}
