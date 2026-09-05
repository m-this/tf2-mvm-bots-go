// Package wave reads what a run wrote down and compares two arms of one.
//
// Split out of the runner so the numbers a run prints and the numbers a report
// prints are the same numbers, read by the same code.
package wave

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// Result is one wave, as the statistics plugin wrote it.
type Result struct {
	Event      string `json:"event"`
	Map        string `json:"map"`
	Wave       int    `json:"wave"`
	Outcome    string `json:"result"`
	Duration   float64
	RobotKills int `json:"robot_kills"`
	Deaths     int `json:"defender_deaths"`
	Damage     int `json:"damage"`
}

// Begun reports whether the plugin has written a wave_begin, which is the only
// honest sign that a wave is running rather than a break being taken.
func Begun(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 0, 1<<20), 1<<22)
	for scan.Scan() {
		if strings.Contains(scan.Text(), `"event":"wave_begin"`) {
			return true
		}
	}
	return false
}

// Read is every wave result in one file. A file with none is not an error here:
// the caller knows whether that means a crash, a timeout or a stall, and says so.
func Read(path string) ([]Result, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var out []Result
	// A wave_end with no wave_begin before it in the file began under the
	// previous run's settings and finished under this one's. It is nobody's
	// wave and is not counted.
	begun := false
	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 0, 1<<20), 1<<22)
	for scan.Scan() {
		var r Result
		if err := json.Unmarshal(scan.Bytes(), &r); err != nil {
			continue
		}
		switch r.Event {
		case "wave_begin":
			begun = true
		case "wave_end":
			if begun {
				out = append(out, r)
			}
			begun = false
		}
	}
	return out, scan.Err()
}

// Arm is one side of a comparison: every wave of every attempt with one setting.
type Arm struct {
	Name     string
	Results  []Result
	Attempts int // runs started, which is not the number of waves they produced
	Crashes  int
	Empty    int // attempts that produced no wave at all
}

// Cleared is how many of the arm's waves ended in a win.
func (a Arm) Cleared() int {
	n := 0
	for _, r := range a.Results {
		if r.Outcome == "cleared" {
			n++
		}
	}
	return n
}

// Median of one column, and zero for an arm with nothing in it.
func (a Arm) Median(of func(Result) float64) float64 {
	if len(a.Results) == 0 {
		return 0
	}
	values := make([]float64, 0, len(a.Results))
	for _, r := range a.Results {
		values = append(values, of(r))
	}
	sort.Float64s(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

// Quartiles are the band this arm covers, which is what decides whether another
// arm has shown anything. docs/testbed-metrics.md is the rule: a median inside
// the other arm's quartiles has shown nothing.
func (a Arm) Quartiles(of func(Result) float64) (lo, hi float64) {
	if len(a.Results) < 4 {
		return math.NaN(), math.NaN()
	}
	values := make([]float64, 0, len(a.Results))
	for _, r := range a.Results {
		values = append(values, of(r))
	}
	sort.Float64s(values)
	return values[len(values)/4], values[len(values)*3/4]
}

var columns = []struct {
	name string
	of   func(Result) float64
}{
	{"robots killed", func(r Result) float64 { return float64(r.RobotKills) }},
	{"defenders died", func(r Result) float64 { return float64(r.Deaths) }},
	{"held for", func(r Result) float64 { return r.Duration }},
}

/*
Compare says what the two arms did, and whether the difference means anything.

The verdict is the band rule and nothing cleverer: an arm whose median falls
inside the other's quartiles has shown nothing. Four attempts is too few for
quartiles, and it says so rather than inventing a verdict.
*/
func Compare(treated, control Arm) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%-16s %18s %18s\n", "", treated.Name, control.Name)
	fmt.Fprintf(&b, "%-16s %18d %18d\n", "attempts", treated.Attempts, control.Attempts)
	fmt.Fprintf(&b, "%-16s %18d %18d\n", "waves", len(treated.Results), len(control.Results))
	fmt.Fprintf(&b, "%-16s %18d %18d\n", "waves cleared", treated.Cleared(), control.Cleared())
	fmt.Fprintf(&b, "%-16s %18d %18d\n", "crashes", treated.Crashes, control.Crashes)
	fmt.Fprintf(&b, "%-16s %18d %18d\n", "empty runs", treated.Empty, control.Empty)

	for _, c := range columns {
		lo, hi := control.Quartiles(c.of)
		got, want := treated.Median(c.of), control.Median(c.of)

		var note string
		switch {
		case got == 0 && want == 0:
			// A column nobody wrote to. Saying "inside the band" about two
			// zeros reads as a verdict, and it is an empty column.
			note = "  (not recorded)"
		case math.IsNaN(lo):
			note = "  (too few for a band)"
		case got >= lo && got <= hi:
			note = fmt.Sprintf("  inside %.0f to %.0f, so nothing shown", lo, hi)
		default:
			note = fmt.Sprintf("  outside %.0f to %.0f", lo, hi)
		}
		fmt.Fprintf(&b, "%-16s %18.1f %18.1f%s\n", c.name, got, want, note)
	}

	if treated.Crashes == 0 && control.Crashes > 0 {
		fmt.Fprintf(&b, "\nThe control crashed %d times and the treated arm did not.\n", control.Crashes)
	}
	return b.String()
}
