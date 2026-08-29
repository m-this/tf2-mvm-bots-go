package wave

import (
	"os"
	"path/filepath"
	"testing"
)

/*
The three stacks that matter, from Peppy's server.

The middle one is why the leaf cannot be the test on its own: a working engineer
carries ScenarioMonitor last, and an earlier rule that read only the leaf called
every engineer of every run broken.
*/
func TestOnlyMonitorsMeansNothingToDo(t *testing.T) {
	for stack, want := range map[string]bool{
		"MainAction < TacticalMonitor < ScenarioMonitor":                        true,
		"MainAction < TacticalMonitor < DefenderEngineerIdle < ScenarioMonitor": false,
		"MainAction < TacticalMonitor < ScenarioMonitor < SniperLurk":           false,
		"MainAction < Taunt < TacticalMonitor < ScenarioMonitor":                true,
		"": true,
	} {
		if got := leafless(stack); got != want {
			t.Errorf("%q read as nothing-to-do=%v, wanted %v", stack, got, want)
		}
	}
}

// The stalled sniper and the working one, side by side, as the plugin writes them.
func TestIdleShareSeparatesTheTwoSnipers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.jsonl")
	lines := `{"event":"bot","who":"stalled","class":"sniper","at":[1,2,3],"firing":0,"action":"MainAction < TacticalMonitor < ScenarioMonitor"}
{"event":"bot","who":"stalled","class":"sniper","at":[1,2,3],"firing":0,"action":"MainAction < TacticalMonitor < ScenarioMonitor"}
{"event":"bot","who":"working","class":"sniper","at":[1,2,3],"firing":1,"action":"MainAction < TacticalMonitor < ScenarioMonitor < SniperLurk"}
{"event":"bot","who":"working","class":"sniper","at":[9,9,9],"firing":1,"action":"MainAction < TacticalMonitor < ScenarioMonitor < SniperLurk"}
{"event":"wave_end","result":"lost"}
`
	if err := os.WriteFile(path, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}

	bots, err := IdleShare(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(bots) != 2 {
		t.Fatalf("read %d bots, wanted 2", len(bots))
	}

	// Sorted worst first, so the stalled one leads.
	if bots[0].Who != "stalled" || bots[0].Leafless() != 1 {
		t.Errorf("the stalled sniper read as %s at %.2f", bots[0].Who, bots[0].Leafless())
	}
	if bots[1].Leafless() != 0 {
		t.Errorf("the working sniper read as idle at %.2f", bots[1].Leafless())
	}
	if bots[0].Positions != 1 {
		t.Errorf("the stalled sniper moved through %d positions, wanted 1", bots[0].Positions)
	}

	if IdleReport(path) == "" {
		t.Error("a sniper idle for the whole run was not reported")
	}
}
