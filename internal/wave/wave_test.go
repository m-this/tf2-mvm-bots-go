package wave

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// A file of samples with no wave in it is the shape a stalled mission writes,
// and it must read as no results rather than as an error nobody looks at.
func TestReadTakesOnlyWaveEnds(t *testing.T) {
	path := write(t,
		`{"event":"bot","class":"engineer"}`,
		`{"event":"wave_end","wave":1,"result":"lost","robot_kills":50}`,
		`not json at all`,
		`{"event":"wave_end","wave":2,"result":"cleared","robot_kills":80}`,
	)
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("read %d waves, want 2", len(got))
	}
	if got[1].Outcome != "cleared" || got[1].RobotKills != 80 {
		t.Errorf("second wave read as %+v", got[1])
	}
}

// The band rule from docs/testbed-metrics.md: a median inside the control's
// quartiles has shown nothing, and the report has to say so rather than leave
// the reader to eyeball two numbers.
func TestCompareSaysWhenNothingWasShown(t *testing.T) {
	control := Arm{Name: "off", Results: results(50, 55, 60, 65, 70, 75, 80, 85)}
	inside := Arm{Name: "on", Results: results(64, 66, 68)}

	if line := lineFor(Compare(inside, control), "robots killed"); !strings.Contains(line, "nothing shown") {
		t.Errorf("a median inside the band did not say so: %q", line)
	}

	outside := Arm{Name: "on", Results: results(140, 150, 160)}
	if line := lineFor(Compare(outside, control), "robots killed"); strings.Contains(line, "nothing shown") {
		t.Errorf("a median well outside the band was called nothing: %q", line)
	}
}

// Four attempts cannot carry quartiles, and inventing a verdict from three
// numbers is how a fix ships on noise.
func TestCompareRefusesABandItCannotHave(t *testing.T) {
	control := Arm{Name: "off", Results: results(50, 60)}
	if out := Compare(Arm{Name: "on", Results: results(55)}, control); !strings.Contains(out, "too few for a band") {
		t.Errorf("a two attempt control produced a verdict:\n%s", out)
	}
}

func TestQuartilesNeedFour(t *testing.T) {
	lo, _ := Arm{Results: results(1, 2, 3)}.Quartiles(func(r Result) float64 { return float64(r.RobotKills) })
	if !math.IsNaN(lo) {
		t.Errorf("three results produced a band of %v", lo)
	}
}

func results(kills ...int) []Result {
	out := make([]Result, 0, len(kills))
	for _, k := range kills {
		out = append(out, Result{Event: "wave_end", Outcome: "lost", RobotKills: k})
	}
	return out
}

// A column both arms left at zero is an empty column, not a verdict.
func TestCompareDoesNotJudgeAnEmptyColumn(t *testing.T) {
	control := Arm{Name: "off", Results: results(50, 55, 60, 65)}
	line := lineFor(Compare(Arm{Name: "on", Results: results(58)}, control), "defenders died")

	if !strings.Contains(line, "not recorded") {
		t.Errorf("an empty column reads %q", line)
	}
}

func lineFor(report, column string) string {
	for _, line := range strings.Split(report, "\n") {
		if strings.HasPrefix(line, column) {
			return line
		}
	}
	return ""
}
