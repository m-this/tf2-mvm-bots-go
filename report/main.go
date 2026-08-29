// Read the wave lines a run produced and say what happened.
//
//	go run ./report results/batch/current-1.jsonl
//	go run ./report results/after.jsonl results/before.jsonl
//
// With two files it compares them, the first being the run under test.
//
// The interesting numbers are not the ones that say whether the team held.
// Waves cleared is almost always the same for two builds of the same mod;
// what separates them is who did the work and what killed them.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	waveline "github.com/m-this/tf2-mvm-bots-go/gen/go/wave"
)

/*
One wave, as the statistics plugin writes it.

The record is generated from internal/tables, which is the same table the
plugin's FormatEx is generated from, so a field this report reads is a field the
plugin writes by construction. It used to be a hand written subset of 38 of the
112, and a field renamed on one side read as zero on the other with nothing said.

The maps below are not in the file. The plugin writes the per class numbers as
flat keys because it has one format string and no allocation, and they are
nested here because that is how they are read.
*/
type wave struct {
	waveline.Record

	HealingBy map[string]int

	SelfDamageBy map[string]int
	SelfDeathsBy map[string]int
	DamageBy     map[string]int
	KillsBy      map[string]int
	GiantsBy     map[string]int
	KilledByBy   map[string]int
	CauseBy      map[string]int
}

var classes = []string{
	"scout", "soldier", "pyro", "demoman", "heavy",
	"engineer", "medic", "sniper", "spy",
}

// How a defender died, as opposed to who killed him
var causes = []string{
	"bullet", "explosion", "fire", "melee", "backstab",
	"headshot", "fall", "other",
}

// The per-class fields are flat keys rather than nested objects, because the
// plugin writes them with one format string and no allocation.
func (w *wave) unpackClasses(raw map[string]int) {
	w.DamageBy = map[string]int{}
	w.KillsBy = map[string]int{}
	w.GiantsBy = map[string]int{}
	w.KilledByBy = map[string]int{}

	w.SelfDamageBy = map[string]int{}
	w.SelfDeathsBy = map[string]int{}
	w.HealingBy = map[string]int{}

	for _, c := range classes {
		w.HealingBy[c] = raw["healing_"+c]
		w.SelfDamageBy[c] = raw["selfdamage_"+c]
		w.SelfDeathsBy[c] = raw["selfdeaths_"+c]
		w.DamageBy[c] = raw["damage_"+c]
		w.KillsBy[c] = raw["kills_"+c]
		w.GiantsBy[c] = raw["giantkills_"+c]
		w.KilledByBy[c] = raw["killedby_"+c]
	}

	w.KilledByBy["sentry"] = raw["killedby_sentry"]
	w.KilledByBy["tank"] = raw["killedby_tank"]

	w.CauseBy = map[string]int{}

	for _, c := range causes {
		w.CauseBy[c] = raw["cause_"+c]
	}
}

func load(path string) ([]wave, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var waves []wave

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !strings.HasPrefix(line, "{") {
			continue
		}

		var w wave
		if err := json.Unmarshal([]byte(line), &w); err != nil {
			continue
		}

		// Wave zero is the tournament restart writing a result for a wave
		// nobody played, and it drags every average it appears in.
		if w.Event != "wave_end" || w.Wave == 0 {
			continue
		}

		var raw map[string]int
		_ = json.Unmarshal([]byte(line), &raw)
		w.unpackClasses(raw)

		waves = append(waves, w)
	}

	return waves, scanner.Err()
}

type summary struct {
	waves, cleared                        int
	kills, giants, deaths, stabs          int
	sentriesLost, busters                 int
	repaired, buildingDamage              int
	healScoreboard                        int
	healingBy                             map[string]int
	damage, tankDamage, sentryDamage      int
	demoPipe, demoSticky, demoMelee       int
	solRocket, solOther                   int
	firedSol, hitSol, firedDemo, hitDemo  int
	jars                                  int
	healing, ubers                        int
	damageBy, killsBy, giantsBy, killedBy map[string]int
	causeBy                               map[string]int
	selfDamageBy, selfDeathsBy            map[string]int
}

func summarise(waves []wave) summary {
	s := summary{
		damageBy: map[string]int{}, killsBy: map[string]int{},
		giantsBy: map[string]int{}, killedBy: map[string]int{},
		causeBy:      map[string]int{},
		healingBy:    map[string]int{},
		selfDamageBy: map[string]int{}, selfDeathsBy: map[string]int{},
	}

	for _, w := range waves {
		s.waves++
		if w.Result == "cleared" {
			s.cleared++
		}

		s.kills += w.RobotKills
		s.giants += w.GiantKills
		s.deaths += w.DefenderDeaths
		s.stabs += w.Backstabs
		s.sentriesLost += w.SentriesLost
		s.repaired += w.BuildingRepaired
		s.buildingDamage += w.BuildingDamage
		s.busters += w.BusterDetonations
		s.damage += w.Damage
		s.demoPipe += w.DemoPipeDamage
		s.demoSticky += w.DemoStickyDamage
		s.demoMelee += w.DemoMeleeDamage
		s.solRocket += w.SoldierRocketDamage
		s.solOther += w.SoldierOtherDamage
		s.firedSol += w.FiredSoldier
		s.hitSol += w.HitSoldier
		s.firedDemo += w.FiredDemoman
		s.hitDemo += w.HitDemoman
		s.jars += w.JarsThrown
		s.tankDamage += w.TankDamage
		s.sentryDamage += w.SentryDamage
		s.healing += w.Healing
		s.healScoreboard += w.HealingScoreboard

		for k, v := range w.HealingBy {
			s.healingBy[k] += v
		}
		s.ubers += w.Ubers

		for k, v := range w.DamageBy {
			s.damageBy[k] += v
		}
		for k, v := range w.KillsBy {
			s.killsBy[k] += v
		}
		for k, v := range w.GiantsBy {
			s.giantsBy[k] += v
		}
		for k, v := range w.KilledByBy {
			s.killedBy[k] += v
		}
		for k, v := range w.SelfDamageBy {
			s.selfDamageBy[k] += v
		}
		for k, v := range w.SelfDeathsBy {
			s.selfDeathsBy[k] += v
		}
		for k, v := range w.CauseBy {
			s.causeBy[k] += v
		}
	}

	return s
}

// Descending by value, dropping zeroes, so a line only names what happened
func ranked(m map[string]int) string {
	type pair struct {
		name  string
		count int
	}

	var pairs []pair
	for k, v := range m {
		if v > 0 {
			pairs = append(pairs, pair{k, v})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].name < pairs[j].name
	})

	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, fmt.Sprintf("%s %d", p.name, p.count))
	}

	if len(parts) == 0 {
		return "nothing"
	}

	return strings.Join(parts, "  ")
}

func percent(part, whole int) string {
	if whole == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(part)/float64(whole)*100)
}

func report(name string, s summary) {
	fmt.Printf("\n%s\n", name)
	fmt.Printf("  waves played      %d\n", s.waves)
	fmt.Printf("  cleared           %d (%s)\n", s.cleared, percent(s.cleared, s.waves))
	fmt.Printf("  robots killed     %d, of them %d giants\n", s.kills, s.giants)
	fmt.Printf("  defenders died    %d\n", s.deaths)
	fmt.Printf("  sentries lost     %d\n", s.sentriesLost)

	// An engineer who never swings and one who repairs perfectly look the same
	// from uptime and sentries lost. This is the difference between them.
	if s.repaired > 0 || s.buildingDamage > 0 {
		fmt.Printf("  buildings took    %d, engineer put back %d (%s)\n",
			s.buildingDamage, s.repaired, percent(s.repaired, s.buildingDamage))
	}

	if s.damage == 0 {
		fmt.Printf("  (no contribution numbers in this file)\n")
		return
	}

	fmt.Printf("  damage dealt      %d\n", s.damage)
	fmt.Printf("  of it, sentries   %d (%s)\n", s.sentryDamage, percent(s.sentryDamage, s.damage))
	fmt.Printf("  damage to tanks   %d\n", s.tankDamage)
	fmt.Printf("  healing done      %d, %d ubers\n", s.healing, s.ubers)

	// Who did the healing, and whether the two ways of counting it agree. The
	// engineer's share is his dispenser: player_healed names him as the healer
	// for it, so it has always been inside the total and never visible.
	if len(s.healingBy) > 0 || s.healScoreboard > 0 {
		fmt.Printf("  healing by class  %s\n", ranked(s.healingBy))

		// Every class, so the parts add up to the whole. The first cut printed
		// only the medic and the engineer and they came to 1757 of a total of
		// 7246: five and a half thousand points of healing done by classes the
		// report did not think could heal.
		counted := 0

		for _, v := range s.healingBy {
			counted += v
		}

		if gap := s.healing - counted; gap != 0 {
			fmt.Printf("  unattributed      %d\n", gap)
		}

		fmt.Printf("  scoreboard says   %d", s.healScoreboard)

		// Two routes to one quantity. A gap is not automatically a bug, but it
		// is always worth a look, and silence about it is how a broken counter
		// survives.
		if gap := s.healScoreboard - s.healing; gap > s.healing/10 || gap < -s.healing/10 {
			fmt.Printf("   (the event sum says %d, which is a %d gap)", s.healing, gap)
		}

		fmt.Println()
	}
	fmt.Printf("  damage by class   %s\n", ranked(s.damageBy))
	fmt.Printf("  kills by class    %s\n", ranked(s.killsBy))
	fmt.Printf("  giants by class   %s\n", ranked(s.giantsBy))
	fmt.Printf("  killed us         %s\n", ranked(s.killedBy))
	fmt.Printf("  died to           %s\n", ranked(s.causeBy))

	// The demoman's two weapons apart, because "he is the weakest class" does
	// not say whether the pipes miss or the stickies are never detonated.
	if s.demoPipe+s.demoSticky+s.demoMelee > 0 {
		fmt.Printf("  demoman damage    pipes %d, stickies %d, bottle %d\n",
			s.demoPipe, s.demoSticky, s.demoMelee)
	}

	if s.solRocket+s.solOther > 0 {
		fmt.Printf("  soldier damage    rockets %d, everything else %d\n", s.solRocket, s.solOther)
	}

	// The question a damage total cannot answer: is he not shooting, or is he
	// shooting and missing?
	// A jar held is not a jar thrown, and the weapon share cannot tell them apart
	if s.jars > 0 {
		fmt.Printf("  jars thrown       %d\n", s.jars)
	}

	if s.firedSol+s.firedDemo > 0 {
		fmt.Printf("  projectiles       soldier %d fired, %d hit (%d%%);  demoman %d fired, %d hit (%d%%)\n",
			s.firedSol, s.hitSol, pct(s.hitSol, s.firedSol),
			s.firedDemo, s.hitDemo, pct(s.hitDemo, s.firedDemo))
	}

	// Printed only when it happened, because a zero here is the normal case and a
	// line of zeroes in every report is a line nobody reads.
	if self := ranked(s.selfDamageBy); self != "" {
		fmt.Printf("  hurt themselves   %s\n", self)
	}

	if self := ranked(s.selfDeathsBy); self != "" {
		fmt.Printf("  killed themselves %s\n", self)
	}
}

func compare(now, then summary) {
	fmt.Printf("\ncompared\n")
	fmt.Printf("  cleared           %s -> %s\n", percent(then.cleared, then.waves), percent(now.cleared, now.waves))
	fmt.Printf("  defenders died    %d -> %d\n", then.deaths, now.deaths)
	fmt.Printf("  damage dealt      %d -> %d\n", then.damage, now.damage)
	fmt.Printf("  sentry damage     %d -> %d\n", then.sentryDamage, now.sentryDamage)
	fmt.Printf("  healing done      %d -> %d\n", then.healing, now.healing)
	fmt.Printf("  of it, the medic  %d -> %d\n", then.healingBy["medic"], now.healingBy["medic"])
	fmt.Printf("  of it, dispensers %d -> %d\n", then.healingBy["engineer"], now.healingBy["engineer"])
	fmt.Printf("  sentries lost     %d -> %d\n", then.sentriesLost, now.sentriesLost)
	fmt.Printf("  repaired          %d -> %d\n", then.repaired, now.repaired)
	fmt.Printf("  backstabbed       %d -> %d\n", then.causeBy["backstab"], now.causeBy["backstab"])
}

/* How much a wave of this mission varies on its own, so a difference can be read
 *
 * Every total above is a sum over waves, and a sum hides its spread. A change was called a
 * forty four per cent longer hold here on eight waves against eight, and the same change built a
 * second way came back at the baseline: the arm that looked like a discovery had waves of 88 and
 * 282 seconds in it and the difference was inside its own scatter.
 *
 * So the per-wave spread is printed beside the middle of it. A difference between two arms that
 * does not clear the quartiles of either is a story, not a result.
 */
func spread(label string, waves []wave, pick func(wave) int) {
	if len(waves) < 4 {
		return
	}

	values := make([]int, 0, len(waves))
	for _, w := range waves {
		values = append(values, pick(w))
	}
	sort.Ints(values)

	n := len(values)
	fmt.Printf("    %-16s median %5d   quartiles %d to %d   range %d to %d   over %d waves\n",
		label, values[n/2], values[n/4], values[(3*n)/4], values[0], values[n-1], n)
}

func printSpread(waves []wave) {
	fmt.Printf("\n  how much one wave varies from the next\n")
	spread("held for", waves, func(w wave) int { return int(w.Duration) })
	spread("defenders died", waves, func(w wave) int { return w.DefenderDeaths })
	spread("robots killed", waves, func(w wave) int { return w.RobotKills })
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 || len(args) > 2 {
		fmt.Fprintln(os.Stderr, "usage: report <after.jsonl> [before.jsonl]")
		return
	}

	after, err := load(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(after) == 0 {
		fmt.Printf("%s\n  no wave results in this file\n", args[0])
		return
	}

	now := summarise(after)
	report(args[0], now)

	if bots, buildings, err := loadTelemetry(args[0]); err == nil {
		printTelemetry(bots, buildings)
		printStanding(bots, buildings)
		printBreak(bots)
	}

	nowSetup, err := loadSetup(args[0])
	if err == nil {
		printSetup(nowSetup)
	}

	printSpread(after)

	if len(args) == 1 {
		return
	}

	before, err := load(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	then := summarise(before)
	report(args[1], then)
	compare(now, then)

	if thenSetup, err := loadSetup(args[1]); err == nil {
		compareSetup(nowSetup, thenSetup)
	}
}
