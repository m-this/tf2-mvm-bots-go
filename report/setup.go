package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// What one engineer's break bought, as the statistics plugin writes it at the
// start of every wave. The timestamps are seconds into the break and -1 for a
// building that never went up.
type setupSample struct {
	Map   string  `json:"map"`
	Wave  int     `json:"wave"`
	Who   string  `json:"who"`
	Break float64 `json:"break_s"`

	Walked     float64 `json:"walked"`
	Teleports  int     `json:"teleports"`
	Teleported float64 `json:"teleported"`

	SentryAt    float64 `json:"sentry_at_s"`
	DispenserAt float64 `json:"dispenser_at_s"`
	EntranceAt  float64 `json:"entrance_at_s"`
	ExitAt      float64 `json:"exit_at_s"`
}

func loadSetup(path string) ([]setupSample, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var setups []setupSample

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !strings.Contains(line, `"event":"setup"`) {
			continue
		}

		var s setupSample
		if json.Unmarshal([]byte(line), &s) == nil && s.Wave > 0 {
			setups = append(setups, s)
		}
	}

	return setups, scanner.Err()
}

// The four building timestamps in the order the report prints them
var setupBuildings = []struct {
	name string
	at   func(setupSample) float64
}{
	{"entrance", func(s setupSample) float64 { return s.EntranceAt }},
	{"sentry", func(s setupSample) float64 { return s.SentryAt }},
	{"dispenser", func(s setupSample) float64 { return s.DispenserAt }},
	{"exit", func(s setupSample) float64 { return s.ExitAt }},
}

// A building that stood, averaged over the breaks it stood in
type buildTiming struct {
	built int
	total float64
}

func (b buildTiming) mean() float64 {
	if b.built == 0 {
		return -1
	}

	return b.total / float64(b.built)
}

type setupRollup struct {
	breaks     int
	breakTotal float64
	walked     float64
	teleports  int
	teleported float64
	timing     map[string]*buildTiming
}

func rollUpSetup(setups []setupSample) map[string]*setupRollup {
	rolled := map[string]*setupRollup{}

	for _, s := range setups {
		r := rolled[s.Who]
		if r == nil {
			r = &setupRollup{timing: map[string]*buildTiming{}}
			rolled[s.Who] = r
		}

		r.breaks++
		r.breakTotal += s.Break
		r.walked += s.Walked
		r.teleports += s.Teleports
		r.teleported += s.Teleported

		for _, b := range setupBuildings {
			at := b.at(s)
			if at < 0 {
				continue
			}

			t := r.timing[b.name]
			if t == nil {
				t = &buildTiming{}
				r.timing[b.name] = t
			}

			t.built++
			t.total += at
		}
	}

	return rolled
}

// The order the engineer built in and what it cost him in ground.
//
// This is the measurement mvm-dh8 is judged on. The entrance going up before
// the sentry is the change; the walk coming down is the point of it. A build
// that never happens is louder than either, so a building that stood in fewer
// breaks than there were is printed with the count rather than hidden behind
// an average of the times it did. That count is not a failure on its own: a
// teleporter still standing from the last wave is not built again.
func printSetup(setups []setupSample) {
	rolled := rollUpSetup(setups)

	if len(rolled) == 0 {
		return
	}

	fmt.Printf("\n  the break, per engineer\n")

	who := make([]string, 0, len(rolled))
	for name := range rolled {
		who = append(who, name)
	}
	sort.Strings(who)

	for _, name := range who {
		r := rolled[name]

		fmt.Printf("    %-16s %d breaks, %.0fs each, walked %.0f units",
			name, r.breaks, r.breakTotal/float64(r.breaks), r.walked/float64(r.breaks))

		if r.teleports > 0 {
			fmt.Printf(", teleported %.0f over %d %s",
				r.teleported/float64(r.breaks), r.teleports, plural(r.teleports, "jump"))
		}

		fmt.Printf("\n")

		for _, b := range setupBuildings {
			t := r.timing[b.name]

			if t == nil {
				fmt.Printf("      %-10s not built in any break\n", b.name)
				continue
			}

			fmt.Printf("      %-10s %.0fs in", b.name, t.mean())

			// A teleporter that survived the last wave is standing and is not
			// rebuilt, so this is not the same as a build that failed.
			if t.built < r.breaks {
				fmt.Printf(", built in %d of %d breaks", t.built, r.breaks)
			}

			fmt.Printf("\n")
		}
	}
}

// Side by side with the run before it, which is the only way the numbers mean
// anything: a walk of four thousand units is neither good nor bad on its own.
func compareSetup(now, then []setupSample) {
	nowRolled, thenRolled := rollUpSetup(now), rollUpSetup(then)

	if len(nowRolled) == 0 || len(thenRolled) == 0 {
		return
	}

	fmt.Printf("\n  the break\n")

	nowAll, thenAll := foldSetup(nowRolled), foldSetup(thenRolled)

	fmt.Printf("    walked per break %.0f -> %.0f units\n",
		thenAll.walked/float64(thenAll.breaks), nowAll.walked/float64(nowAll.breaks))
	fmt.Printf("    teleports        %d -> %d\n", thenAll.teleports, nowAll.teleports)

	for _, b := range setupBuildings {
		built := builtCount(nowAll, b.name)

		fmt.Printf("    %-10s built %d -> %d %s, at %s -> %s\n", b.name,
			builtCount(thenAll, b.name), built, plural(built, "time"),
			meanSeconds(thenAll, b.name), meanSeconds(nowAll, b.name))
	}
}

// Every engineer in one, because which of the two built the entrance is not
// the question the arms are being compared on.
func foldSetup(rolled map[string]*setupRollup) *setupRollup {
	all := &setupRollup{timing: map[string]*buildTiming{}}

	for _, r := range rolled {
		all.breaks += r.breaks
		all.breakTotal += r.breakTotal
		all.walked += r.walked
		all.teleports += r.teleports
		all.teleported += r.teleported

		for name, t := range r.timing {
			into := all.timing[name]
			if into == nil {
				into = &buildTiming{}
				all.timing[name] = into
			}

			into.built += t.built
			into.total += t.total
		}
	}

	return all
}

func builtCount(r *setupRollup, name string) int {
	if t := r.timing[name]; t != nil {
		return t.built
	}

	return 0
}

func meanSeconds(r *setupRollup, name string) string {
	t := r.timing[name]

	if t == nil || t.built == 0 {
		return "not built"
	}

	return fmt.Sprintf("%.0fs", t.mean())
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}

	return word + "s"
}
