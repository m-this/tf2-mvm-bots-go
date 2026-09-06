package wave

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample(t float64, who string, at [3]float64, hp int, action string, pathing, failed int, pathLen float64) string {
	return fmt.Sprintf(`{"event":"bot","map":"m","wave":1,"t":%.1f,"who":%q,"class":"engineer","at":[%.0f,%.0f,%.0f],"hp":%d,"action":%q,"pathing":%d,"path_failed":%d,"path_len":%.0f}`,
		t, who, at[0], at[1], at[2], hp, action, pathing, failed, pathLen)
}

func writeTrace(t *testing.T, lines []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "r.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAPinnedBotIsNamedAndADeadOneIsNot(t *testing.T) {
	var lines []string
	for i := 0; i < 60; i++ {
		lines = append(lines, sample(float64(i)*0.5, "Fred", [3]float64{1014, 885, 274}, 125, "MainAction < TacticalMonitor < DefenderBuildSentrygun", 1, 0, 300))
		lines = append(lines, sample(float64(i)*0.5, "Corpse", [3]float64{10, 10, 10}, 0, "MainAction < TacticalMonitor < DefenderAttack", 0, 0, 0))
		lines = append(lines, sample(float64(i)*0.5, "Walker", [3]float64{float64(i) * 20, 0, 0}, 125, "MainAction < TacticalMonitor < DefenderAttack", 1, 0, 500))
	}
	got, err := Assert(writeTrace(t, lines))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Pins) != 1 || got.Pins[0].Who != "Fred" || got.Pins[0].Seconds < PinnedSeconds {
		t.Fatalf("pins %+v", got.Pins)
	}
	report := TraceReport(writeTrace(t, lines))
	if !strings.Contains(report, "Fred") || !strings.Contains(report, "pinned at 1014 885 274") {
		t.Errorf("report:\n%s", report)
	}
}

func TestAHuddleNeedsThreeForTenSeconds(t *testing.T) {
	var lines []string
	for i := 0; i < 30; i++ {
		for _, who := range []string{"A", "B", "C"} {
			lines = append(lines, sample(float64(i)*0.5, who, [3]float64{100 + float64(len(who)), 200, 50}, 100, "MainAction < TacticalMonitor < DefenderAttack", 0, 0, 0))
		}
		lines = append(lines, sample(float64(i)*0.5, "Far", [3]float64{900, 900, 50}, 100, "MainAction < TacticalMonitor < DefenderAttack", 0, 0, 0))
	}
	got, err := Assert(writeTrace(t, lines))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Huddles) != 1 || len(got.Huddles[0].Who) != 3 || got.Huddles[0].Seconds < HuddleSeconds {
		t.Fatalf("huddles %+v", got.Huddles)
	}
}

func TestPathSharesCountRefusedAndDrifting(t *testing.T) {
	var lines []string
	for i := 0; i < 20; i++ {
		failed := 0
		if i%2 == 0 {
			failed = 1
		}
		lines = append(lines, sample(float64(i)*0.5, "Lost", [3]float64{float64(i), 0, 0}, 100, "MainAction < TacticalMonitor < DefenderAttack", 1, failed, 0))
		lines = append(lines, sample(float64(i)*0.5, "Fine", [3]float64{float64(i), 5, 0}, 100, "MainAction < TacticalMonitor < DefenderAttack", 1, 0, 400))
		// Arrived: a zero length path held standing still is not drifting.
		lines = append(lines, sample(float64(i)*0.5, "Parked", [3]float64{700, 700, 0}, 100, "MainAction < TacticalMonitor < DefenderEngineerIdle", 1, 0, 0))
	}
	got, err := Assert(writeTrace(t, lines))
	if err != nil {
		t.Fatal(err)
	}
	byWho := map[string]PathShare{}
	for _, p := range got.Paths {
		byWho[p.Who] = p
	}
	if lost := byWho["Lost"]; lost.Bad() != 0.5 || lost.Failed != 10 || lost.Drifting != 10 {
		t.Errorf("Lost %+v; half refused outright and half measured nothing while moving", lost)
	}
	if fine := byWho["Fine"]; fine.Bad() != 0 {
		t.Errorf("Fine %+v", fine)
	}
	if parked := byWho["Parked"]; parked.Bad() != 0 || parked.Adrift() != 0 {
		t.Errorf("Parked %+v; a bot that has arrived is neither refused nor drifting", parked)
	}
	report := TraceReport(writeTrace(t, lines))
	if !strings.Contains(report, "Lost") || strings.Contains(report, "Fine") || strings.Contains(report, "Parked") {
		t.Errorf("report:\n%s", report)
	}
}
