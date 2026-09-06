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
	// RootedUnits is how little ground a live defender may cover across a
	// whole wave before it counts as never having left where it started.
	RootedUnits = 300.0
	// RootedSeconds is how long the wave has to have run for that to mean
	// anything: a wave lost in twenty seconds moves nobody far.
	RootedSeconds = 90.0
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

A refused path is the reliable signal and it is what this flags on. A zero
length path is not: a bot that has arrived has nowhere left to go, and at a five
second sample "arrived somewhere during the last five seconds" and "moving with
no path" look the same from the position alone. Measured on Coaltown 2026-09-06,
an engineer wrenching his own sentry read half his samples that way.

So Drifting is counted and printed, because mvm-zx0 is about exactly that
shape, and it is not folded into what raises the report. Telling the two apart
wants the action the bot is running, or a faster sample, and neither is here.
*/
type PathShare struct {
	Who, Class string
	Pathing    int
	// Failed is a path the engine refused outright.
	Failed int
	// Drifting is a zero length path the bot moved on anyway.
	Drifting int
}

// Bad is the share of pathing samples the engine refused outright.
func (p PathShare) Bad() float64 {
	if p.Pathing == 0 {
		return 0
	}
	return float64(p.Failed) / float64(p.Pathing)
}

// Adrift is the share that measured nothing while the bot moved. Read beside
// Bad rather than added to it: see the type's own note.
func (p PathShare) Adrift() float64 {
	if p.Pathing == 0 {
		return 0
	}
	return float64(p.Drifting) / float64(p.Pathing)
}

// Rooted is one defender that covered almost no ground across a whole wave.
type Rooted struct {
	Who, Class string
	Wave       int
	Covered    float64
	Seconds    float64
}

// Traces is what the passes found in one file.
type Traces struct {
	Pins    []Pin
	Huddles []Huddle
	Paths   []PathShare
	Rooted  []Rooted
}

// Assert runs every pass over a results file.
func Assert(path string) (Traces, error) {
	samples, err := readSamples(path)
	if err != nil {
		return Traces{}, err
	}
	return Traces{
		Pins:    pins(samples),
		Huddles: huddles(samples),
		Paths:   pathShares(samples),
		Rooted:  rooted(samples),
	}, nil
}

/*
rooted finds a defender who never went anywhere for a whole wave.

Reported on Area 52 and Thriller and seen on Valve maps too: the bots are
prepared, they have shopped, and they stand in spawn while the mission runs.
That is mvm-78m, and it leaves no other trace: no watchdog arms for it, the
wave numbers read as a team that fought badly, and the samples are the only
place it shows.

Ground covered rather than distance from a spawn polygon, because the polygon
would want the map's nav mesh and a results file does not carry one. A bot that
holds a doorway all wave covers more than this; one that never left where it
started covers almost nothing.
*/
func rooted(samples []Sample) []Rooted {
	type track struct {
		class       string
		first, last float64
		at          []float64
		covered     float64
		alive       bool
	}
	byBot := map[string]*track{}
	var order []string

	for _, s := range samples {
		key := fmt.Sprintf("%d/%s", s.Wave, s.Who)

		t := byBot[key]
		if t == nil {
			t = &track{class: s.Class, first: s.Time, at: s.At}
			byBot[key] = t
			order = append(order, key)
		}
		t.last = s.Time
		if s.Health > 0 {
			t.alive = true
		}
		t.covered += distance(t.at, s.At)
		t.at = s.At
	}

	var found []Rooted
	for _, key := range order {
		t := byBot[key]
		if !t.alive || t.last-t.first < RootedSeconds || t.covered >= RootedUnits {
			continue
		}

		wave, who := 0, key
		if n, name, ok := strings.Cut(key, "/"); ok {
			who = name
			_, _ = fmt.Sscanf(n, "%d", &wave)
		}
		found = append(found, Rooted{
			Who: who, Class: t.class, Wave: wave,
			Covered: t.covered, Seconds: t.last - t.first,
		})
	}
	return found
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
	for _, r := range t.Rooted {
		lines = append(lines, fmt.Sprintf("  %-20s %-8s covered %.0f units in %.0fs of wave %d, so it never left where it started",
			r.Who, r.Class, r.Covered, r.Seconds, r.Wave))
	}
	for _, p := range t.Paths {
		if p.Bad() < PathFailShareWorthReporting || p.Pathing < 10 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %-20s %-8s %3.0f%% of %d path requests refused, and %.0f%% measured nothing while moving",
			p.Who, p.Class, p.Bad()*100, p.Pathing, p.Adrift()*100))
	}
	if len(lines) == 0 {
		return ""
	}
	return "the traces say:\n" + strings.Join(lines, "\n")
}
