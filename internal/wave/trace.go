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

/*
Assertions over every recorded trace.

A fault that is a document somebody reads is found once. A fault that is a
test over the samples every run writes is found every time it comes back. The
three below are the shapes the beads describe: a bot pinned to one spot past
the watchdog's window (mvm-wb0, mvm-ipf), several bots in one small cube for
a while (mvm-wb0's four in a doorway), and paths that fail or measure nothing
(mvm-zx0). Each is a pass over the file, and each says who and where.
*/

const (
	// PinnedSeconds is how long a live bot may hold one exact position
	// before it is called pinned. The stuck watchdog fires at twelve, so a
	// pin that lasts this long is one the watchdog did not clear.
	PinnedSeconds = 20.0
	// HuddleRadius is how close bots have to stand to be in each other's way.
	HuddleRadius = 32.0
	// HuddleSeconds is how long they have to stand there before it is a huddle.
	HuddleSeconds = 10.0
	// PathFailShareWorthReporting is the share of a bot's pathing samples
	// that may come back failed or empty before it is named.
	PathFailShareWorthReporting = 0.25
)

// Pin is one bot that did not move for a while.
type Pin struct {
	Who, Class string
	At         []float64
	Seconds    float64
	Action     string
}

// Huddle is several bots in one small cube for a while.
type Huddle struct {
	Who     []string
	At      []float64
	Seconds float64
}

/*
PathShare is how a bot's path requests went.

Drifting is a zero length path the bot is moving on, and it is not the same
thing as a zero length path standing still: a bot that has arrived has nowhere
left to go, and its path is empty for the right reason. Measured on Coaltown
2026-09-06, an engineer wrenching his own sentry reads pathing with a zero
length path in half his samples, which is him standing at it. So the position
has to move for the sample to count, which is what mvm-zx0 describes: drifting
on a zero length path until the pack expires.
*/
type PathShare struct {
	Who, Class string
	Pathing    int
	// Failed is a path the engine refused outright.
	Failed int
	// Drifting is a zero length path the bot moved on anyway.
	Drifting int
}

// Bad is the share of pathing samples that were refused or drifted.
func (p PathShare) Bad() float64 {
	if p.Pathing == 0 {
		return 0
	}
	return float64(p.Failed+p.Drifting) / float64(p.Pathing)
}

// Traces is what the three passes found in one file.
type Traces struct {
	Pins    []Pin
	Huddles []Huddle
	Paths   []PathShare
}

// Assert runs every pass over a results file.
func Assert(path string) (Traces, error) {
	samples, err := readSamples(path)
	if err != nil {
		return Traces{}, err
	}
	return Traces{Pins: pins(samples), Huddles: huddles(samples), Paths: pathShares(samples)}, nil
}

func readSamples(path string) ([]Sample, error) {
	file, err := os.Open(path) //nolint:gosec // a results file the caller named
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var out []Sample
	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 0, 1<<20), 1<<22)
	for scan.Scan() {
		line := scan.Bytes()
		if !strings.Contains(string(line), `"event":"bot"`) {
			continue
		}
		var s Sample
		if err := json.Unmarshal(line, &s); err != nil || len(s.At) != 3 {
			continue
		}
		out = append(out, s)
	}
	return out, scan.Err()
}

// pins finds a bot at one exact position for PinnedSeconds while it had a
// path, which is the watchdog's own reading of wanting to be elsewhere. A dead
// bot lies still, a camper stands still on purpose, and a bot with nothing to
// do stands still with no path; the wedge is the one that is trying to walk.
func pins(samples []Sample) []Pin {
	type run struct {
		start, last float64
		at          []float64
		wave        int
		action      string
	}
	open := map[string]*run{}
	var found []Pin
	seen := map[string]bool{}

	for _, s := range samples {
		r := open[s.Who]
		alive := s.Health > 0 && s.Pathing == 1 && !leafless(s.Action)
		if r != nil && (r.wave != s.Wave || !samePlace(r.at, s.At) || !alive) {
			open[s.Who] = nil
			r = nil
		}
		if !alive {
			continue
		}
		if r == nil {
			open[s.Who] = &run{start: s.Time, last: s.Time, at: s.At, wave: s.Wave, action: s.Action}
			continue
		}
		r.last = s.Time
		key := fmt.Sprintf("%s@%v/%d", s.Who, r.at, r.wave)
		if r.last-r.start >= PinnedSeconds && !seen[key] {
			seen[key] = true
			found = append(found, Pin{Who: s.Who, Class: s.Class, At: r.at, Seconds: r.last - r.start, Action: r.action})
		}
	}
	for _, p := range found {
		if r := open[p.Who]; r != nil && samePlace(r.at, p.At) {
			p.Seconds = r.last - r.start
		}
	}
	return found
}

// huddles groups the bots alive at each sample time and reports a group of
// three or more inside HuddleRadius that lasts HuddleSeconds.
func huddles(samples []Sample) []Huddle {
	byTime := map[string][]Sample{}
	var order []string
	for _, s := range samples {
		if s.Health <= 0 {
			continue
		}
		key := fmt.Sprintf("%d/%.1f", s.Wave, s.Time)
		if _, ok := byTime[key]; !ok {
			order = append(order, key)
		}
		byTime[key] = append(byTime[key], s)
	}

	type standing struct {
		start, last float64
		at          []float64
	}
	open := map[string]*standing{}
	var found []Huddle
	seen := map[string]bool{}

	for _, key := range order {
		group := byTime[key]
		present := map[string]bool{}
		for i := range group {
			var names []string
			for j := range group {
				if distance(group[i].At, group[j].At) <= HuddleRadius {
					names = append(names, group[j].Who)
				}
			}
			if len(names) < 3 {
				continue
			}
			sort.Strings(names)
			id := strings.Join(names, ",")
			present[id] = true
			st := open[id]
			if st == nil {
				open[id] = &standing{start: group[i].Time, last: group[i].Time, at: group[i].At}
				continue
			}
			st.last = group[i].Time
			if st.last-st.start >= HuddleSeconds && !seen[id] {
				seen[id] = true
				found = append(found, Huddle{Who: names, At: st.at, Seconds: st.last - st.start})
			}
		}
		for id := range open {
			if !present[id] {
				delete(open, id)
			}
		}
	}
	return found
}

func distance(a, b []float64) float64 {
	if len(a) != 3 || len(b) != 3 {
		return math.Inf(1)
	}
	dx, dy, dz := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// pathShares counts, per bot, the samples taken while pathing and how many of
// those were refused or spent moving with no path.
func pathShares(samples []Sample) []PathShare {
	byBot := map[string]*PathShare{}
	last := map[string][]float64{}

	for _, s := range samples {
		before := last[s.Who]
		last[s.Who] = s.At

		if s.Pathing == 0 {
			continue
		}

		p := byBot[s.Who]
		if p == nil {
			p = &PathShare{Who: s.Who, Class: s.Class}
			byBot[s.Who] = p
		}
		p.Pathing++

		switch {
		case s.PathFailed == 1:
			p.Failed++
		case s.PathLen == 0 && before != nil && !samePlace(before, s.At):
			p.Drifting++
		}
	}
	out := make([]PathShare, 0, len(byBot))
	for _, p := range byBot {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bad() > out[j].Bad() })
	return out
}

// TraceReport names what the passes found, and says nothing when they found
// nothing worth a line.
func TraceReport(path string) string {
	t, err := Assert(path)
	if err != nil {
		return ""
	}
	var lines []string
	for _, p := range t.Pins {
		lines = append(lines, fmt.Sprintf("  %-20s %-8s pinned at %.0f %.0f %.0f for %.0fs doing %s",
			p.Who, p.Class, p.At[0], p.At[1], p.At[2], p.Seconds, p.Action))
	}
	for _, h := range t.Huddles {
		lines = append(lines, fmt.Sprintf("  %d bots within %.0f units at %.0f %.0f %.0f for %.0fs: %s",
			len(h.Who), HuddleRadius, h.At[0], h.At[1], h.At[2], h.Seconds, strings.Join(h.Who, ", ")))
	}
	for _, p := range t.Paths {
		if p.Bad() < PathFailShareWorthReporting || p.Pathing < 10 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %-20s %-8s %3.0f%% of %d pathing samples refused or drifted (%d refused, %d drifting)",
			p.Who, p.Class, p.Bad()*100, p.Pathing, p.Failed, p.Drifting))
	}
	if len(lines) == 0 {
		return ""
	}
	return "the traces say:\n" + strings.Join(lines, "\n")
}
