// Read a whole sweep and say which map, and which class, is the weak one.
//
//	go run ./testbed/sweepreport testbed/results/sweep-night
//	go run ./testbed/sweepreport results/ab-x/on results/ab-x/off
//
// With two directories it compares them, the first being the arm under test.
//
// The per-run report answers "did this change help". This answers the two
// questions a sweep exists for and a single run cannot touch:
//
// Which map is wrong. Most of what an engineer does is a property of geometry,
// so a nest that never reaches level three or a dispenser at the far end of the
// map is a fact about that map's data or its ground, not about the mod. A map
// that produces those every wave wants somebody to walk it.
//
// Which class is carrying nothing. Waves cleared says the team held. It does
// not say who held it, and a class whose seat produces no damage, no kills and
// no deaths is a seat that would be better spent on another class.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var classes = []string{
	"scout", "soldier", "pyro", "demoman", "heavy",
	"engineer", "medic", "sniper", "spy",
}

type record struct {
	Event    string  `json:"event"`
	Map      string  `json:"map"`
	Wave     int     `json:"wave"`
	Result   string  `json:"result"`
	Duration float64 `json:"duration"`

	// wave_end
	Damage         int `json:"damage"`
	Healing        int `json:"healing"`
	DefenderDeaths int `json:"defender_deaths"`
	SentriesLost   int `json:"sentries_lost"`
	RobotKills     int `json:"robot_kills"`

	// perf
	Frames      int     `json:"frames"`
	FramesSlow  int     `json:"frames_slow"`
	FramesStall int     `json:"frames_stalled"`
	FrameMean   float64 `json:"frame_mean_ms"`
	FrameWorst  float64 `json:"frame_worst_ms"`

	// engineer
	When          string  `json:"when"`
	Who           string  `json:"who"`
	Sentry        int     `json:"sentry"`
	Dispenser     int     `json:"dispenser"`
	Entrance      int     `json:"entrance"`
	Exit          int     `json:"exit"`
	DispenserFrom float64 `json:"dispenser_from_sentry"`
	ExitFrom      float64 `json:"exit_from_sentry"`
	Samples       int     `json:"samples"`
	WithSentry    int     `json:"with_sentry"`
	WithLevel3    int     `json:"with_level3"`
	WithDispenser int     `json:"with_dispenser"`
}

// A dispenser is meant to be beside the sentry it feeds. This is generous: it
// is the distance at which "he put it somewhere else" stops being arguable.
const dispenserFarFromSentry = 600.0

type mapStats struct {
	name     string
	cleared  int
	lost     int
	seconds  float64
	deaths   int
	sentries int
	damage   int
	healing  int

	damageBy map[string]int
	killsBy  map[string]int
	deathsBy map[string]int

	// What the engineers had at the start of each wave, which is what the
	// between-rounds time was actually spent on
	engWaves      int
	engNoSentry   int
	engLowSentry  int
	engNoDispense int
	engDispFar    int
	engNoTele     int
	engHalfTele   int

	// How much of a wave the engineers actually had a nest, which is a
	// different complaint from never having built one
	samples       int
	withSentry    int
	withLevel3    int
	withDispenser int

	frames      int
	framesSlow  int
	framesStall int
	frameWorst  float64
	frameMeanWt float64
}

func newMapStats(name string) *mapStats {
	return &mapStats{
		name:     name,
		damageBy: map[string]int{},
		killsBy:  map[string]int{},
		deathsBy: map[string]int{},
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sweepreport <sweep directory>")
		os.Exit(2)
	}

	dir := os.Args[1]

	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil || len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no result files in %s\n", dir)
		os.Exit(1)
	}

	sort.Strings(files)

	order := make([]string, 0, len(files))
	byMap := map[string]*mapStats{}

	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".jsonl")

		stats := newMapStats(name)
		byMap[name] = stats
		order = append(order, name)

		if err := read(path, stats); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		}
	}

	printMaps(order, byMap)
	printEngineers(order, byMap)
	printClasses(order, byMap)
	printPerf(order, byMap)

	if len(os.Args) > 2 {
		baseOrder, baseByMap, err := loadDir(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		printComparison(order, byMap, baseOrder, baseByMap)
	}
}

func loadDir(dir string) ([]string, map[string]*mapStats, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil || len(files) == 0 {
		return nil, nil, fmt.Errorf("no result files in %s", dir)
	}

	sort.Strings(files)

	var order []string
	byMap := map[string]*mapStats{}

	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".jsonl")

		stats := newMapStats(name)
		byMap[name] = stats
		order = append(order, name)

		if err := read(path, stats); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		}
	}

	return order, byMap, nil
}

/* The first directory against the second, which is what an A/B run leaves behind.
 *
 * Damage per wave rather than damage, because the two arms do not play the same number of waves:
 * an arm that clears faster plays more of them in the same timeout, and totals would then reward
 * the arm that was slower.
 *
 * The waves line is the one to be careful with. Six waves an arm is a small sample and the bots
 * are not deterministic, so a difference of one cleared wave is noise and only a large move in
 * damage per wave is worth reading as anything. */
func printComparison(order []string, byMap map[string]*mapStats, baseOrder []string, baseByMap map[string]*mapStats) {
	fmt.Println("== A against B ==")

	a, aWaves := totals(order, byMap)
	b, bWaves := totals(baseOrder, baseByMap)

	cleared, lost := 0, 0
	for _, name := range order {
		cleared += byMap[name].cleared
		lost += byMap[name].lost
	}

	baseCleared, baseLost := 0, 0
	for _, name := range baseOrder {
		baseCleared += baseByMap[name].cleared
		baseLost += baseByMap[name].lost
	}

	fmt.Printf("waves      A %d cleared, %d lost      B %d cleared, %d lost\n\n",
		cleared, lost, baseCleared, baseLost)

	fmt.Printf("%-12s %12s %12s %10s %9s\n", "class", "A dmg/wave", "B dmg/wave", "change", "percent")

	sorted := append([]string{}, classes...)
	sort.Slice(sorted, func(i, j int) bool { return a[sorted[i]] > a[sorted[j]] })

	for _, c := range sorted {
		if aWaves[c] == 0 && bWaves[c] == 0 {
			continue
		}

		aPer, bPer := 0.0, 0.0

		if aWaves[c] > 0 {
			aPer = float64(a[c]) / float64(aWaves[c])
		}

		if bWaves[c] > 0 {
			bPer = float64(b[c]) / float64(bWaves[c])
		}

		pct := 0.0
		if bPer > 0 {
			pct = 100 * (aPer - bPer) / bPer
		}

		fmt.Printf("%-12s %12.0f %12.0f %10.0f %8.0f%%\n", c, aPer, bPer, aPer-bPer, pct)
	}

	fmt.Println()
}

func totals(order []string, byMap map[string]*mapStats) (map[string]int, map[string]int) {
	damage := map[string]int{}
	waves := map[string]int{}

	for _, name := range order {
		s := byMap[name]

		for _, c := range classes {
			damage[c] += s.damageBy[c]

			if s.damageBy[c] > 0 || s.killsBy[c] > 0 {
				waves[c] += s.cleared + s.lost
			}
		}
	}

	return damage, waves
}

func read(path string, stats *mapStats) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !strings.HasPrefix(line, "{") {
			continue
		}

		var r record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}

		// Wave zero is the tournament restart writing a result for a wave
		// nobody played, and it drags every average it appears in.
		if r.Wave == 0 {
			continue
		}

		switch r.Event {
		case "wave_end":
			var raw map[string]int
			_ = json.Unmarshal([]byte(line), &raw)

			if r.Result == "cleared" {
				stats.cleared++
			} else {
				stats.lost++
			}

			stats.seconds += r.Duration
			stats.deaths += r.DefenderDeaths
			stats.sentries += r.SentriesLost
			stats.damage += r.Damage
			stats.healing += r.Healing

			for _, c := range classes {
				stats.damageBy[c] += raw["damage_"+c]
				stats.killsBy[c] += raw["kills_"+c]
				stats.deathsBy[c] += raw["killedby_"+c]
			}

		case "perf":
			stats.frames += r.Frames
			stats.framesSlow += r.FramesSlow
			stats.framesStall += r.FramesStall
			stats.frameMeanWt += r.FrameMean * float64(r.Frames)

			if r.FrameWorst > stats.frameWorst {
				stats.frameWorst = r.FrameWorst
			}

		case "engineer":
			// The beginning of a wave is the question. What he has then is
			// what the between-rounds time bought; what he has at the end is
			// mostly a statement about the robots.
			// The uptime is only counted once the wave is over
			if r.When == "end" {
				stats.samples += r.Samples
				stats.withSentry += r.WithSentry
				stats.withLevel3 += r.WithLevel3
				stats.withDispenser += r.WithDispenser
			}

			if r.When != "begin" {
				continue
			}

			stats.engWaves++

			switch {
			case r.Sentry == 0:
				stats.engNoSentry++
			case r.Sentry < 3:
				stats.engLowSentry++
			}

			if r.Dispenser == 0 {
				stats.engNoDispense++
			} else if r.DispenserFrom > dispenserFarFromSentry {
				stats.engDispFar++
			}

			switch {
			case r.Entrance == 0 && r.Exit == 0:
				stats.engNoTele++
			case r.Entrance == 0 || r.Exit == 0:
				stats.engHalfTele++
			}
		}
	}

	return scanner.Err()
}

func printMaps(order []string, byMap map[string]*mapStats) {
	fmt.Println("== waves ==")
	fmt.Printf("%-18s %7s %5s %9s %8s %8s\n", "map", "cleared", "lost", "avg secs", "deaths", "sentries")

	for _, name := range order {
		s := byMap[name]
		waves := s.cleared + s.lost

		avg := 0.0
		if waves > 0 {
			avg = s.seconds / float64(waves)
		}

		fmt.Printf("%-18s %7d %5d %9.0f %8d %8d\n", short(name), s.cleared, s.lost, avg, s.deaths, s.sentries)
	}

	fmt.Println()
}

// The engineer table is the point of the sweep, so it says what is wrong rather
// than how much of it there was: counts of waves that began with a nest the
// engineer had not finished.
func printEngineers(order []string, byMap map[string]*mapStats) {
	fmt.Println("== engineers, at the start of each wave ==")
	fmt.Printf("%-18s %6s %9s %10s %10s %9s %7s %8s\n",
		"map", "waves", "no sentry", "sentry<3", "no disp", "disp far", "no tele", "half tele")

	for _, name := range order {
		s := byMap[name]

		if s.engWaves == 0 {
			fmt.Printf("%-18s %6s\n", short(name), "-")
			continue
		}

		fmt.Printf("%-18s %6d %9d %10d %10d %9d %7d %8d\n",
			short(name), s.engWaves, s.engNoSentry, s.engLowSentry,
			s.engNoDispense, s.engDispFar, s.engNoTele, s.engHalfTele)
	}

	fmt.Println()

	// A nest he never built and a nest he could not keep look the same in the
	// table above and are different complaints. This is the second one.
	fmt.Println("== how much of each wave the engineers had a nest standing ==")
	fmt.Printf("%-18s %9s %10s %10s %12s\n", "map", "samples", "sentry", "level 3", "dispenser")

	for _, name := range order {
		s := byMap[name]

		if s.samples == 0 {
			fmt.Printf("%-18s %9s\n", short(name), "-")
			continue
		}

		pct := func(n int) float64 { return 100 * float64(n) / float64(s.samples) }

		fmt.Printf("%-18s %9d %9.0f%% %9.0f%% %11.0f%%\n",
			short(name), s.samples, pct(s.withSentry), pct(s.withLevel3), pct(s.withDispenser))
	}

	fmt.Println()
}

func printClasses(order []string, byMap map[string]*mapStats) {
	total := map[string]int{}
	kills := map[string]int{}
	deaths := map[string]int{}
	seatWaves := map[string]int{}

	grand := 0

	for _, name := range order {
		s := byMap[name]

		for _, c := range classes {
			total[c] += s.damageBy[c]
			kills[c] += s.killsBy[c]
			deaths[c] += s.deathsBy[c]

			// A class only holds a seat on the maps whose composition names
			// it, so damage has to be read against the waves it played, not
			// against every wave in the sweep.
			if s.damageBy[c] > 0 || s.killsBy[c] > 0 {
				seatWaves[c] += s.cleared + s.lost
			}
		}

		grand += s.damage
	}

	fmt.Println("== what each class did with its seat, whole sweep ==")
	fmt.Printf("%-12s %10s %6s %8s %8s %12s\n", "class", "damage", "share", "kills", "waves", "dmg per wave")

	sorted := append([]string{}, classes...)
	sort.Slice(sorted, func(i, j int) bool { return total[sorted[i]] > total[sorted[j]] })

	for _, c := range sorted {
		if seatWaves[c] == 0 {
			continue
		}

		share := 0.0
		if grand > 0 {
			share = 100 * float64(total[c]) / float64(grand)
		}

		fmt.Printf("%-12s %10d %5.1f%% %8d %8d %12.0f\n",
			c, total[c], share, kills[c], seatWaves[c],
			float64(total[c])/float64(seatWaves[c]))
	}

	fmt.Println()
	fmt.Println("== what killed the defenders, whole sweep ==")

	sort.Slice(sorted, func(i, j int) bool { return deaths[sorted[i]] > deaths[sorted[j]] })

	for _, c := range sorted {
		if deaths[c] == 0 {
			continue
		}

		fmt.Printf("%-12s %6d\n", c, deaths[c])
	}

	fmt.Println()
}

// A mean frame time hides the thing worth finding. The counts are what matter:
// a run with no stalled frames is a run the mod fits inside a tick.
func printPerf(order []string, byMap map[string]*mapStats) {
	fmt.Println("== frames ==")
	fmt.Printf("%-18s %10s %9s %10s %9s %10s\n", "map", "frames", "mean ms", ">30ms", ">100ms", "worst ms")

	for _, name := range order {
		s := byMap[name]

		if s.frames == 0 {
			fmt.Printf("%-18s %10s\n", short(name), "-")
			continue
		}

		fmt.Printf("%-18s %10d %9.2f %10d %9d %10.0f\n",
			short(name), s.frames, s.frameMeanWt/float64(s.frames),
			s.framesSlow, s.framesStall, s.frameWorst)
	}

	fmt.Println()
}

func short(name string) string {
	return strings.TrimPrefix(name, "mvm_")
}
