// The spots the map configuration names, so a run can be asked whether the
// engineer used them.
//
// `configs/defenderbots/map/<map>.cfg` carries coordinates somebody walked the
// map to find: a dispenser goes behind that wall, a sentry on that roof. Nothing
// has ever checked that the engineer ended up there, so three separate rules
// that quietly rejected authored spots survived for weeks, and the only symptom
// was a player noticing a dispenser in the wrong place.
//
// This is a deliberately small reader. It wants the origins under two blocks and
// it does not care about the rest of the format.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type spot struct {
	name string
	at   [3]float64
}

// Which authored block a sampled building should be measured against
var spotBlockFor = map[string]string{
	"sentry":      "EngineerNest",
	"mini sentry": "EngineerNest",
	"dispenser":   "DispenserSpot",
}

func loadSpots(mapName string) map[string][]spot {
	path := filepath.Join("configs", "defenderbots", "map", mapName+".cfg")

	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	out := map[string][]spot{}
	block := ""
	depth := 0

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}

		switch line {
		case "{":
			depth++
			continue
		case "}":
			depth--
			if depth <= 1 {
				block = ""
			}
			continue
		}

		fields := quoted(line)

		if len(fields) == 1 && depth == 1 {
			block = fields[0]
			continue
		}

		if len(fields) == 2 && fields[0] == "origin" && block != "" {
			if at, ok := vector(fields[1]); ok {
				out[block] = append(out[block], spot{name: block, at: at})
			}
		}
	}

	return out
}

func quoted(line string) []string {
	var out []string
	var cur strings.Builder
	in := false

	for _, r := range line {
		if r == '"' {
			if in {
				out = append(out, cur.String())
				cur.Reset()
			}
			in = !in
			continue
		}

		if in {
			cur.WriteRune(r)
		}
	}

	return out
}

func vector(s string) ([3]float64, bool) {
	parts := strings.Fields(s)

	if len(parts) != 3 {
		return [3]float64{}, false
	}

	var out [3]float64

	for i, p := range parts {
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return [3]float64{}, false
		}
		out[i] = v
	}

	return out, true
}

func nearestSpot(spots []spot, at []float64) float64 {
	if len(spots) == 0 || len(at) != 3 {
		return -1
	}

	best := -1.0

	for _, s := range spots {
		d := 0.0

		for i := 0; i < 3; i++ {
			d += (s.at[i] - at[i]) * (s.at[i] - at[i])
		}

		d = sqrt(d)

		if best < 0 || d < best {
			best = d
		}
	}

	return best
}

func sqrt(v float64) float64 {
	if v <= 0 {
		return 0
	}

	x := v

	for i := 0; i < 40; i++ {
		x = 0.5 * (x + v/x)
	}

	return x
}

// How close each building ended up to the coordinate somebody walked the map to
// find. Silent when the map names no spots, which is most of them.
func printSpotUse(buildings []buildingSample) {
	if len(buildings) == 0 {
		return
	}

	spots := loadSpots(buildings[0].Map)

	if len(spots) == 0 {
		return
	}

	/* Per wave, and the typical sample rather than the best one

	The first version of this took the closest a building ever got and printed that. Wave one is
	the wave the engineer starts beside his nest with a whole break to build in, so it was almost
	always on the spot, and one good sample hid every wave after it. Reported from play as exactly
	that: right before wave one, wrong afterwards.

	So each wave gets its own answer, and the answer is the median of that wave's samples. A
	building that spent the wave in the wrong place cannot be rescued by the ten seconds it spent
	being carried past the right one. */
	type useKey struct {
		owner, kind string
		wave        int
	}

	seen := map[useKey][]float64{}
	kinds := map[string]bool{}

	for _, b := range buildings {
		block, wanted := spotBlockFor[buildingKind(b)]
		if !wanted {
			continue
		}

		d := nearestSpot(spots[block], b.At)

		if d < 0 {
			continue
		}

		k := useKey{b.Owner, buildingKind(b), b.Wave}
		seen[k] = append(seen[k], d)
		kinds[b.Owner+" "+buildingKind(b)] = true
	}

	if len(seen) == 0 {
		return
	}

	fmt.Printf("\n  how close the buildings got to the spots %s names\n", buildings[0].Map)

	waves := map[int]bool{}

	for k := range seen {
		waves[k.wave] = true
	}

	for _, who := range sortedKeys(kinds) {
		fmt.Printf("    %-28s", who)

		for wave := 1; wave <= len(waves)+8; wave++ {
			var found []float64

			for k, v := range seen {
				if k.owner+" "+k.kind == who && k.wave == wave {
					found = v
				}
			}

			if found == nil {
				continue
			}

			sort.Float64s(found)
			d := found[len(found)/2]

			flag := ""

			switch {
			case d > 600:
				flag = "!!"
			case d > 250:
				flag = "!"
			}

			fmt.Printf("  wave %d: %.0f%s", wave, d, flag)
		}

		fmt.Println()
	}

	fmt.Println("    (median units from the nearest authored spot; ! is off it, !! is nowhere near it)")
}
