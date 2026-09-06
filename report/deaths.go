package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// A defender's death, as the statistics plugin writes it: who, and what did it.
type defenderDeath struct {
	Event  string  `json:"event"`
	Wave   int     `json:"wave"`
	At     float64 `json:"at"`
	Who    string  `json:"who"`
	Class  string  `json:"class"`
	Killer string  `json:"killer"`
	Giant  bool    `json:"giant"`
	Weapon string  `json:"weapon"`
	Cause  string  `json:"cause"`
}

// One purchase at the station. A refund is a negative count.
type upgradePurchase struct {
	Event   string `json:"event"`
	Wave    int    `json:"wave"`
	Who     string `json:"who"`
	Class   string `json:"class"`
	Slot    int    `json:"slot"`
	Upgrade int    `json:"upgrade"`
	Count   int    `json:"count"`
}

func loadDeathsAndUpgrades(path string) ([]defenderDeath, []upgradePurchase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = file.Close() }()

	var deaths []defenderDeath
	var upgrades []upgradePurchase

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, `{"event":"defender_death"`):
			var d defenderDeath
			if json.Unmarshal([]byte(line), &d) == nil && d.Wave > 0 {
				deaths = append(deaths, d)
			}
		case strings.HasPrefix(line, `{"event":"upgrade"`):
			var u upgradePurchase
			if json.Unmarshal([]byte(line), &u) == nil && u.Wave > 0 {
				upgrades = append(upgrades, u)
			}
		}
	}
	return deaths, upgrades, scanner.Err()
}

/*
printDeaths says what killed whom, class against killer, and how.

The wave line has deaths by killer and deaths by cause as two separate totals,
and the question a resistance raises needs them together: a Scout dying to a
giant Heavy's bullets is one problem, a Scout dying to a Soldier's rockets
another, and the two totals cannot say which.
*/
func printDeaths(deaths []defenderDeath) {
	if len(deaths) == 0 {
		return
	}
	fmt.Printf("\n  who killed whom (%d defender deaths)\n", len(deaths))

	type key struct{ class, killer, cause string }
	counts := map[key]int{}
	giants := map[key]int{}
	for _, d := range deaths {
		k := key{d.Class, d.Killer, d.Cause}
		counts[k]++
		if d.Giant {
			giants[k]++
		}
	}
	keys := make([]key, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i].class+keys[i].killer < keys[j].class+keys[j].killer
	})
	for _, k := range keys {
		giant := ""
		if giants[k] > 0 {
			giant = fmt.Sprintf(", %d by a giant", giants[k])
		}
		fmt.Printf("    %-10s died to %-10s by %-9s %3d%s\n", k.class, k.killer, k.cause, counts[k], giant)
	}
}

// printUpgradeTiers says how far each class took each upgrade: the purchases
// summed, refunds included, and the highest any one bot of the class reached.
func printUpgradeTiers(upgrades []upgradePurchase) {
	if len(upgrades) == 0 {
		return
	}
	type key struct {
		who, class    string
		slot, upgrade int
	}
	tier := map[key]int{}
	for _, u := range upgrades {
		tier[key{u.Who, u.Class, u.Slot, u.Upgrade}] += u.Count
	}

	type line struct {
		class         string
		slot, upgrade int
	}
	best := map[line]int{}
	for k, t := range tier {
		l := line{k.class, k.slot, k.upgrade}
		if t > best[l] {
			best[l] = t
		}
	}
	lines := make([]line, 0, len(best))
	for l := range best {
		lines = append(lines, l)
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].class != lines[j].class {
			return lines[i].class < lines[j].class
		}
		if lines[i].slot != lines[j].slot {
			return lines[i].slot < lines[j].slot
		}
		return lines[i].upgrade < lines[j].upgrade
	})

	fmt.Printf("\n  how far the upgrades got (%d purchases; slot -1 is the player)\n", len(upgrades))
	for _, l := range lines {
		fmt.Printf("    %-10s slot %2d upgrade %3d  tier %d\n", l.class, l.slot, l.upgrade, best[l])
	}
}
