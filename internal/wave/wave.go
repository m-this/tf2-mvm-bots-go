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
	"strconv"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/machine"
)

// Result is one wave, as the statistics plugin wrote it.
type Result struct {
	Event    string `json:"event"`
	Map      string `json:"map"`
	Wave     int    `json:"wave"`
	Outcome  string `json:"result"`
	Duration float64
	// FeaturesFired is name:count pairs, comma separated: which features
	// answered true during the wave and the break before it.
	FeaturesFired string `json:"features_fired"`
	RobotKills    int    `json:"robot_kills"`
	Deaths        int    `json:"defender_deaths"`
	Damage        int    `json:"damage"`
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
	/* The crash columns, and why there are two of them.

	The same argument was had in four sessions: an arm crashed, the other did
	not, and nobody could say whether the change or the bed did it. This bed has
	a crash rate of its own, so a crashed attempt is replayed once on the same
	arm. Crashes is the ones that crashed again, which are the arm's. BedCrashes
	is the ones that did not, which are the bed's and are not charged to
	anybody. See mvm-9yn. */
	Crashes    int
	BedCrashes int
	Empty      int // attempts that produced no wave at all
	// Machines is what each attempt was played on, in order. Two arms
	// whose machines differ are refused a comparison: see machine.Comparable.
	Machines []machine.Machine
	// Armed is the features this arm switched on. One that never fired in
	// any of the arm's waves makes the arm a refusal rather than a result.
	Armed []string
}

// Fired is how many times the named feature answered true over the arm's waves.
func (a Arm) Fired(feature string) int {
	total := 0
	for _, r := range a.Results {
		for pair := range strings.SplitSeq(r.FeaturesFired, ",") {
			name, count, found := strings.Cut(pair, ":")
			if !found || name != feature {
				continue
			}
			n, err := strconv.Atoi(count)
			if err == nil {
				total += n
			}
		}
	}
	return total
}

// NeverFired is every armed feature with no firing in any wave, which is an
// arm that did not run the code it was testing.
func (a Arm) NeverFired() []string {
	var out []string
	for _, feature := range a.Armed {
		if a.Fired(feature) == 0 {
			out = append(out, feature)
		}
	}
	return out
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

// Waves are the wave numbers this arm played, in ascending order and each once.
func (a Arm) Waves() []int {
	seen := map[int]bool{}
	out := []int{}
	for _, r := range a.Results {
		if seen[r.Wave] {
			continue
		}
		seen[r.Wave] = true
		out = append(out, r.Wave)
	}
	sort.Ints(out)
	return out
}

// AtWave is this arm holding one wave number, which is what a band belongs to.
func (a Arm) AtWave(n int) Arm {
	kept := Arm{Name: a.Name, Attempts: a.Attempts, Crashes: a.Crashes, Empty: a.Empty}
	for _, r := range a.Results {
		if r.Wave == n {
			kept.Results = append(kept.Results, r)
		}
	}
	return kept
}

/*
foldHidesAFactor says two of the arm's waves never overlap.

Wave 1 of a mission is not wave 2 of it, and a median over both is a median over
two different problems. mvm-k57 is what that cost: defenders died read 9.0
against 5.5 and was called inside the band, while wave 2 alone was 17 and 16
against 9 and 8 and did not overlap at all.

The test is each wave's own spread against the others, and not against the band
the fold produced: the disagreement is what inflated that band, so measuring
against it is measuring against itself. Ranges rather than quartiles, because
three attempts a wave cannot carry quartiles and can still be plainly apart. Two
attempts a wave is the floor: one is a point, and two points that differ at all
would read as apart.

Whether folding is honest is a property of the mission rather than of the switch
being measured, so an arm is asked about itself.
*/
func foldHidesAFactor(a Arm, of func(Result) float64) bool {
	waves := a.Waves()
	if len(waves) < 2 {
		return false
	}

	lows := make([]float64, 0, len(waves))
	highs := make([]float64, 0, len(waves))
	for _, w := range waves {
		low, high, enough := spread(a.AtWave(w), of)
		if !enough {
			return false
		}
		lows, highs = append(lows, low), append(highs, high)
	}

	for i := range waves {
		for j := i + 1; j < len(waves); j++ {
			if highs[i] < lows[j] || highs[j] < lows[i] {
				return true
			}
		}
	}
	return false
}

// spread is the lowest and highest this arm read for a column, and whether
// there were enough readings for the pair to mean anything.
func spread(a Arm, of func(Result) float64) (low, high float64, enough bool) {
	const readingsMin = 2
	if len(a.Results) < readingsMin {
		return 0, 0, false
	}
	low, high = math.Inf(1), math.Inf(-1)
	for _, r := range a.Results {
		v := of(r)
		low, high = math.Min(low, v), math.Max(high, v)
	}
	return low, high, true
}

var columns = []struct {
	name string
	of   func(Result) float64
}{
	{"robots killed", func(r Result) float64 { return float64(r.RobotKills) }},
	{"defenders died", func(r Result) float64 { return float64(r.Deaths) }},
	{"held for", func(r Result) float64 { return r.Duration }},
}

// verdict is the band rule read off the control, and the reasons there is no
// verdict to read.
func verdict(treated, control Arm, of func(Result) float64) string {
	got, want := treated.Median(of), control.Median(of)
	lo, hi := control.Quartiles(of)

	switch {
	case got == 0 && want == 0:
		// A column nobody wrote to. Saying "inside the band" about two zeros
		// reads as a verdict, and it is an empty column.
		return "  (not recorded)"
	case math.IsNaN(lo):
		return "  (too few for a band)"
	case foldHidesAFactor(control, of), foldHidesAFactor(treated, of):
		return "  folded over waves whose spreads do not overlap, so nothing is claimed"
	case got >= lo && got <= hi:
		return fmt.Sprintf("  inside %.0f to %.0f, so nothing shown", lo, hi)
	default:
		return fmt.Sprintf("  outside %.0f to %.0f", lo, hi)
	}
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
	if treated.BedCrashes+control.BedCrashes > 0 {
		fmt.Fprintf(&b, "%-16s %18d %18d\n", "bed crashes", treated.BedCrashes, control.BedCrashes)
	}
	fmt.Fprintf(&b, "%-16s %18d %18d\n", "empty runs", treated.Empty, control.Empty)
	if treated.BedCrashes+control.BedCrashes > 0 {
		b.WriteString("\nThe bed crashes did not crash again on their replay, so they are the bed's and are charged to neither arm.\n")
	}

	waves := control.Waves()

	for _, c := range columns {
		folded := verdict(treated, control, c.of)
		fmt.Fprintf(&b, "%-16s %18.1f %18.1f%s\n",
			c.name, treated.Median(c.of), control.Median(c.of), folded)

		// The waves the fold was over. Once it is refused they carry the
		// verdict instead, one band per wave, which is what a band belongs to.
		// A column nobody wrote to is skipped: a row of zeros per wave is not
		// a spread, it is noise on every report that does not use the column.
		if len(waves) < 2 || strings.Contains(folded, "not recorded") {
			continue
		}
		for _, w := range waves {
			one, other := treated.AtWave(w), control.AtWave(w)
			note := ""
			if strings.Contains(folded, "nothing is claimed") {
				note = verdict(one, other, c.of)
			}
			fmt.Fprintf(&b, "  %-14s %18.1f %18.1f%s\n",
				fmt.Sprintf("wave %d", w), one.Median(c.of), other.Median(c.of), note)
		}
	}

	if treated.Crashes == 0 && control.Crashes > 0 {
		fmt.Fprintf(&b, "\nThe control crashed %d times and the treated arm did not.\n", control.Crashes)
	}
	return b.String()
}
