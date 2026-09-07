package navmesh

import (
	"fmt"
	"sort"
)

/*
Which dispenser spot each authored nest ends up with.

The engineer takes the named dispenser spot that belongs to his nest, and
belonging is decided by coordinates somebody wrote down after walking a map. The
way that goes wrong is silent: the spot is ignored, the dispenser goes in the
nest area instead, and the run looks fine while the walking is thrown away.
Mannhattan is the example, where all three spots sit the better part of a storey
below the nests nearest to them.

This reads the configs and nothing else, so it costs nothing and needs no
server. It was a Python script beside the compose file, which is the last of the
Python the test-bed shipped: see mvm-qbi.
*/

// nestKinds are the three keys a nest can be written under. All three are the
// same thing to the engineer; they differ only in when the nest is used.
var nestKinds = []SpotKind{EngineerNest, NestTankOnly, NestNoTank}

// Pairing is one nest and the dispenser spot it takes, or the fall back it
// takes instead.
type Pairing struct {
	Nest Spot
	// Spot is the dispenser spot the nest takes, and Taken says whether it took
	// one at all. A nest that took none puts its dispenser in the nest area.
	Spot  Spot
	Taken bool
	// RangeUnits is how far the spot is from the nest, in Hammer units.
	RangeUnits float64
}

/*
PairDispensers is every nest in a config with the spot it would take.

The rule is the plugin's: a nest takes a spot written in its own zone if the map
put one there, and otherwise takes the nearest spot no zone claims. A nest with
neither is a fall back, which is what this exists to find.
*/
func PairDispensers(c *MapConfig) []Pairing {
	nests := make([]Spot, 0, len(c.Spots))
	for _, kind := range nestKinds {
		nests = append(nests, c.SpotsOf(kind)...)
	}
	spots := c.SpotsOf(DispenserSpot)
	if len(nests) == 0 || len(spots) == 0 {
		return nil
	}

	out := make([]Pairing, 0, len(nests))
	for _, nest := range nests {
		out = append(out, pair(nest, spots))
	}
	return out
}

func pair(nest Spot, spots []Spot) Pairing {
	owned := inZone(spots, nest.Zone)
	// A nest whose zone claims no spot falls back on the ones no zone claims,
	// which is how a map that names zones for some nests and not others works.
	if len(owned) == 0 && nest.Zone != "" {
		owned = inZone(spots, "")
	}
	if len(owned) == 0 {
		return Pairing{Nest: nest}
	}

	sort.SliceStable(owned, func(i, j int) bool {
		return spotRange(nest, owned[i]) < spotRange(nest, owned[j])
	})
	return Pairing{Nest: nest, Spot: owned[0], Taken: true, RangeUnits: spotRange(nest, owned[0])}
}

func inZone(spots []Spot, zone string) []Spot {
	var out []Spot
	for _, s := range spots {
		if s.Zone == zone {
			out = append(out, s)
		}
	}
	return out
}

func spotRange(a, b Spot) float64 { return float64(b.Origin.Sub(a.Origin).Length()) }

// DispenserReport is the pairings of one config as a person reads them, empty
// for a config that declares no nest or no spot.
func DispenserReport(c *MapConfig) string {
	pairs := PairDispensers(c)
	if len(pairs) == 0 {
		return ""
	}
	out := fmt.Sprintf("%s: %d nests, %d dispenser spots\n", c.Map, len(pairs), len(c.SpotsOf(DispenserSpot)))
	for i, p := range pairs {
		if !p.Taken {
			out += fmt.Sprintf("  nest %d %s zone=%q -> falls back to the nest area\n", i+1, p.Nest, p.Nest.Zone)
			continue
		}
		out += fmt.Sprintf("  nest %d %s zone=%q -> %s at %.0fu\n", i+1, p.Nest, p.Nest.Zone, p.Spot, p.RangeUnits)
	}
	return out
}
