// What the bots were actually doing, as opposed to what the wave line says they
// achieved.
//
// The wave line is a scoreboard: cleared or not, who did the damage, what killed
// whom. It is silent about the thing that has produced every reported fault so
// far, which is a bot in the wrong place doing nothing. Five of those were found
// by a person watching one of six bots at a time and guessing at the cause.
//
// These are the samples the plugin takes every few seconds of every bot and every
// building. Facts go in the file; the verdicts are here, so changing one's mind
// about what "a useless dispenser" means costs a recompile rather than a run.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// The events the two sample lines carry. Named once, because the selector in
// report/select.go picks lines by them too and a second spelling is a filter
// that silently matches nothing.
const (
	botEvent      = "bot"
	buildingEvent = "building"
)

type botSample struct {
	Event        string    `json:"event"`
	Wave         int       `json:"wave"`
	T            float64   `json:"t"`
	Clock        float64   `json:"clock"`
	Who          string    `json:"who"`
	Class        string    `json:"class"`
	At           []float64 `json:"at"`
	HP           int       `json:"hp"`
	MaxHP        int       `json:"maxhp"`
	Weapon       string    `json:"weapon"`
	Slot         int       `json:"slot"`
	Charge       float64   `json:"charge"`
	Deploying    int       `json:"deploying"`
	Between      int       `json:"between"`
	Healing      string    `json:"healing"`
	Wants        string    `json:"wants"`
	Action       string    `json:"action"`
	NearestEnemy float64   `json:"nearest_enemy"`
	Aim          string    `json:"aim"`
	AimRange     float64   `json:"aim_range"`
	Firing       int       `json:"firing"`
	PathLen      float64   `json:"path_len"`
	Pathing      int       `json:"pathing"`
	PathFailed   int       `json:"path_failed"`
	PathFailures int       `json:"path_failures"`
	RepairStalls int       `json:"repair_stalls"`
}

type buildingSample struct {
	Event         string    `json:"event"`
	Map           string    `json:"map"`
	Wave          int       `json:"wave"`
	T             float64   `json:"t"`
	Clock         float64   `json:"clock"`
	Owner         string    `json:"owner"`
	Type          string    `json:"type"`
	Mode          int       `json:"mode"`
	Level         int       `json:"level"`
	HP            int       `json:"hp"`
	MaxHP         int       `json:"maxhp"`
	At            []float64 `json:"at"`
	Disposable    int       `json:"disposable"`
	Kills         int       `json:"kills"`
	EnemiesSeen   int       `json:"enemies_seen"`
	TeammatesNear int       `json:"teammates_near"`
	Sapped        int       `json:"sapped"`
}

// One bot's share of the samples, which is the only honest way to say "he spent
// the wave doing nothing": a fraction of observations, not a stopwatch.
type botRollup struct {
	class    string
	samples  int
	slots    map[int]int
	actions  map[string]int
	beaming  int // medic only: samples with the medigun actually on somebody
	trigger  int // medic only: samples with the trigger held, connected or not
	patients map[string]int
	wanted   map[string]int // medic only: who the ranking chose, which is not who the beam reached
	reached  int            // samples where the ranking's choice and the beam agree
	chose    int            // samples where the ranking chose anybody at all

	// What the medigun had in it. A wave total of zero ubers cannot say whether
	// the medic never had a charge or never spent one, and these two say which:
	// held counts a full charge doing nothing, and spent counts it going off.
	charged  int // samples with a full charge in the medigun
	chargeOn int // full charge and a patient on the beam, which is a charge going unspent
	spent    int // samples inside a deploy
	breakOn  int // samples during a break with the beam on somebody
	breakAll int // samples during a break
	hurt     int // samples below four fifths health
	stillFor map[string]int
}

// A building is judged by who it was near, which is the whole of what a
// dispenser is for and most of what a sentry is for. A sentry that never saw a
// robot is in the wrong place however healthy it is.
type buildingRollup struct {
	kind      string
	samples   int
	level     int
	levelSum  int
	sawEnemy  int
	enemySum  int
	nearSum   int
	sapped    int
	positions map[string]int
}

func rollupBots(samples []botSample) map[string]*botRollup {
	out := map[string]*botRollup{}

	for _, s := range samples {
		r := out[s.Who]

		if r == nil {
			r = &botRollup{
				class: s.Class, slots: map[int]int{},
				actions: map[string]int{}, stillFor: map[string]int{},
				patients: map[string]int{}, wanted: map[string]int{},
			}
			out[s.Who] = r
		}

		r.samples++
		r.slots[s.Slot]++
		r.actions[innermost(s.Action)]++

		if s.Healing != "" {
			r.beaming++
			r.patients[s.Healing]++
		}

		if s.Wants != "" {
			r.chose++
			r.wanted[s.Wants]++

			if s.Wants == s.Healing {
				r.reached++
			}
		}

		if s.Firing != 0 {
			r.trigger++
		}

		if s.Between != 0 {
			r.breakAll++

			if s.Healing != "" {
				r.breakOn++
			}
		}

		if s.Deploying != 0 {
			r.spent++
		}

		if s.Charge >= 1.0 {
			r.charged++

			if s.Healing != "" {
				r.chargeOn++
			}
		}

		if s.MaxHP > 0 && float64(s.HP) < 0.8*float64(s.MaxHP) {
			r.hurt++
		}
	}

	return out
}

func rollupBuildings(samples []buildingSample) map[string]*buildingRollup {
	out := map[string]*buildingRollup{}

	for _, s := range samples {
		key := s.Owner + " " + buildingKind(s)
		r := out[key]

		if r == nil {
			r = &buildingRollup{kind: buildingKind(s), positions: map[string]int{}}
			out[key] = r
		}

		r.samples++
		r.levelSum += s.Level
		r.enemySum += s.EnemiesSeen
		r.nearSum += s.TeammatesNear

		if s.EnemiesSeen > 0 {
			r.sawEnemy++
		}

		if s.Sapped != 0 {
			r.sapped++
		}

		if s.Level > r.level {
			r.level = s.Level
		}
	}

	return out
}

func buildingKind(s buildingSample) string {
	switch s.Type {
	case "obj_sentrygun":
		if s.Disposable != 0 {
			return "mini sentry"
		}
		return "sentry"
	case "obj_dispenser":
		return "dispenser"
	case "obj_teleporter":
		if s.Mode == 1 {
			return "teleporter exit"
		}
		return "teleporter entrance"
	}

	return s.Type
}

// The scaffolding every bot is inside no matter what it is doing. Two of these
// are the game's, two are ours, and none of them is an answer to "what was he
// doing".
var scaffolding = map[string]bool{
	"MainAction": true, "TacticalMonitor": true,
	"ScenarioMonitor": true, "Heal": true, "": true,
}

// The most specific thing the bot was doing.
//
// Not simply the first or last name: ActionsManager.Iterator does not hand them
// back in a dependable stack order, and both ends were tried. A run reported
// every bot as "MainAction 100%" and then as "ScenarioMonitor 100%", which are
// columns of numbers that say nothing while looking like they say something.
//
// The mod's own actions all begin "Defender", so the deepest of those is the
// answer whenever there is one. Failing that, anything that is not scaffolding.
// Failing that, the scaffolding itself, which does mean idle.
func innermost(stack string) string {
	if stack == "" {
		return "none"
	}

	// Ranked rather than positional. The iterator's order is not a dependable
	// nesting order: DefenderGotoUpgrade has been seen listed before its own
	// parent ScenarioMonitor, so both "first Defender" and "last Defender" pick
	// the wrong one some of the time. That misread an engineer sat inside a
	// build action as sitting in the idle action that suspended for it, which
	// is the difference between "he never tried" and "he tried and it never
	// finished".
	parts := strings.Split(stack, " < ")
	answer := ""
	rank := 0

	for _, p := range parts {
		var r int

		switch {
		case scaffolding[p]:
			r = 0
		case p == "DefenderEngineerIdle" || p == "DefenderSpyLurkMvM":
			r = 1 // a holding pattern that suspends for the real work
		case strings.HasPrefix(p, "Defender"):
			r = 2
		default:
			r = 1
		}

		if r >= rank {
			rank = r
			answer = p
		}
	}

	if answer != "" && rank > 0 {
		return answer
	}

	for _, p := range parts {
		if !scaffolding[p] {
			answer = p
		}
	}

	if answer != "" {
		return answer
	}

	return parts[len(parts)-1]
}

func pct(part, whole int) int {
	if whole == 0 {
		return 0
	}

	return part * 100 / whole
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func loadTelemetry(path string) ([]botSample, []buildingSample, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = file.Close() }()

	var bots []botSample
	var buildings []buildingSample

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !strings.HasPrefix(line, "{") {
			continue
		}

		switch {
		case strings.Contains(line, `"event":"`+botEvent+`"`):
			var s botSample
			if json.Unmarshal([]byte(line), &s) == nil && s.Wave > 0 {
				bots = append(bots, s)
			}
		case strings.Contains(line, `"event":"`+buildingEvent+`"`):
			var s buildingSample
			if json.Unmarshal([]byte(line), &s) == nil && s.Wave > 0 {
				buildings = append(buildings, s)
			}
		}
	}

	return bots, buildings, scanner.Err()
}

// What each bot spent the run doing, and whether each building was worth its
// metal. Printed only when there are samples, so an older results file still
// reads as it did.
func printTelemetry(bots []botSample, buildings []buildingSample) {
	if len(bots) == 0 && len(buildings) == 0 {
		return
	}

	if len(bots) > 0 {
		fmt.Printf("\n  what the bots were doing (%d samples)\n", len(bots))

		rolled := rollupBots(bots)

		for _, who := range sortedKeys(rolled) {
			r := rolled[who]

			fmt.Printf("    %-16s %-9s %s\n", who, r.class, topShares(r.actions, r.samples))

			// Holding the trigger and healing somebody are different things. The
			// medigun reaches about 450 units; a medic aiming at a patient 926
			// away with the button down is delivering nothing, and every number
			// except this one counts it as a medic at work.
			if r.class == "medic" {
				fmt.Printf("    %-16s %-9s beam on somebody %d%% of the time, trigger held %d%% (%s of that connected)\n",
					"", "", pct(r.beaming, r.samples), pct(r.trigger, r.samples),
					percent(r.beaming, r.trigger))

				/* Who, and not only how often.

				"Beam on somebody 27% of the time" cannot tell a medic who
				pocketed one body all wave from one who shared it out, and it
				cannot say whether a call was answered: both of those are the
				patient's name and nothing else. A run that seats a puppet reads
				this line, because the puppet is named after itself. */
				if len(r.patients) > 0 {
					fmt.Printf("    %-16s %-9s beam went to %s\n",
						"", "", topShares(r.patients, r.beaming))
				}

				/* The ranking's own answer, and how often the beam got there

				Who the medigun is worth the most on and who it reached are two questions, and only
				the second is visible in the line above. A choice the beam never reaches is a medic
				stood in the wrong place rather than a ranking that picked the wrong man, and the
				two want different fixes. See mvm-eil. */
				if len(r.wanted) > 0 {
					fmt.Printf("    %-16s %-9s ranking chose %s, and the beam was on that man in %d%% of those\n",
						"", "", topShares(r.wanted, r.chose), pct(r.reached, r.chose))
				}

				/* What was in the medigun, which is the half "ubers: 0" cannot say

				A full charge sitting on a live beam is inventory: the medic had one, had somebody
				to spend it on, and did not. A wave total of zero cannot tell that from never having
				built one, and they are different faults with different fixes.

				The two shares do not add up and must not be read as if they did. m_flChargeLevel
				drains while the charge releases, so a deploying sample is almost never a full
				charge as well; counting "full and deploying" reads as nearly zero however eager the
				rule is, which is a metric that cannot move. Whether the charge goes off at all is
				the wave line's ubers, counted off the game's own event.

				The break line is the other end of it, because uber is built by healing and the
				break is the quiet time to build one. */
				fmt.Printf("    %-16s %-9s a full charge sat on a live beam through %d%% of the wave, and he was inside a charge for %d%%\n",
					"", "", pct(r.chargeOn, r.samples), pct(r.spent, r.samples))

				if r.breakAll > 0 {
					fmt.Printf("    %-16s %-9s between waves: beam on somebody %d%% of %d samples\n",
						"", "", pct(r.breakOn, r.breakAll), r.breakAll)
				}
			}

			fmt.Printf("    %-16s %-9s %s, hurt %d%% of the time\n", "", "", slotShare(r.slots, r.samples), pct(r.hurt, r.samples))
		}
	}

	if len(buildings) == 0 {
		return
	}

	fmt.Printf("\n  what the buildings were worth (%d samples)\n", len(buildings))

	rolled := rollupBuildings(buildings)

	for _, key := range sortedKeys(rolled) {
		r := rolled[key]

		// A sentry is judged on whether robots ever came into its view, a
		// dispenser on whether anybody stood in it. Both are the question the
		// hand-walked spots in configs/ are trying to answer.
		fmt.Printf("    %-28s level %d, saw a robot %d%% of samples (%.1f at a time), %.1f teammates in range",
			key, r.level, pct(r.sawEnemy, r.samples),
			float64(r.enemySum)/float64(max(r.samples, 1)),
			float64(r.nearSum)/float64(max(r.samples, 1)))

		if r.sapped > 0 {
			fmt.Printf(", sapped %d%% of samples", pct(r.sapped, r.samples))
		}

		fmt.Println()
	}

	printSpotUse(buildings)
}

// The three actions a bot spent most of its samples in, which is the shape of
// "he was stood in a house" without anybody having to be in the house.
// topShares is the three biggest names in a tally, each as a share of the
// samples it was counted over.
func topShares(counts map[string]int, samples int) string {
	type pair struct {
		name  string
		count int
	}

	pairs := make([]pair, 0, len(counts))

	for name, count := range counts {
		pairs = append(pairs, pair{name, count})
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })

	var parts []string

	for i, p := range pairs {
		if i == 3 {
			break
		}

		parts = append(parts, fmt.Sprintf("%s %d%%", p.name, pct(p.count, samples)))
	}

	return strings.Join(parts, ", ")
}

var slotNames = map[int]string{0: "primary", 1: "secondary", 2: "melee"}

func slotShare(slots map[int]int, samples int) string {
	var parts []string

	for slot := 0; slot <= 2; slot++ {
		if slots[slot] == 0 {
			continue
		}

		parts = append(parts, fmt.Sprintf("%s %d%%", slotNames[slot], pct(slots[slot], samples)))
	}

	if len(parts) == 0 {
		return "no weapon out"
	}

	return strings.Join(parts, "/")
}
