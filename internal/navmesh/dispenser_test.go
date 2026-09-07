package navmesh

import "testing"

const zonedConfig = `"MapConfig"
{
	"EngineerNest"
	{
		"1" { "origin" "0 0 0" "zone" "inside" }
		"2" { "origin" "1000 0 0" "zone" "outside" }
		"3" { "origin" "0 2000 0" }
	}
	"DispenserSpot"
	{
		"1" { "origin" "100 0 0" "zone" "inside" }
		"2" { "origin" "1200 0 0" "zone" "outside" }
		"3" { "origin" "0 2100 0" }
	}
}`

/*
A nest takes the spot written in its own zone, and the nearest unclaimed one if
its zone claims none.

The rule is the plugin's, and getting it wrong is silent: the spot is ignored,
the dispenser goes in the nest area instead, and a day of walking a map is
thrown away with nothing said.
*/
func TestNestTakesItsOwnZone(t *testing.T) {
	c, err := ParseMapConfig(zonedConfig)
	if err != nil {
		t.Fatal(err)
	}
	pairs := PairDispensers(c)
	if len(pairs) != 3 {
		t.Fatalf("paired %d nests, want 3", len(pairs))
	}
	for i, want := range []struct {
		zone  string
		spot  string
		units float64
	}{
		{"inside", "1", 100},
		{"outside", "2", 200},
		{"", "3", 100},
	} {
		got := pairs[i]
		if !got.Taken {
			t.Errorf("nest %d in zone %q took no spot", i+1, want.zone)
			continue
		}
		if got.Spot.Index != want.spot || got.RangeUnits != want.units {
			t.Errorf("nest %d took spot %s at %gu, want spot %s at %gu",
				i+1, got.Spot.Index, got.RangeUnits, want.spot, want.units)
		}
	}
}

// A nest whose zone claims nothing falls back on the spots no zone claims,
// which is how a map that names zones for some nests and not others works.
func TestZonedNestFallsBackOnUnclaimedSpots(t *testing.T) {
	c, err := ParseMapConfig(`"MapConfig"
{
	"EngineerNest" { "1" { "origin" "0 0 0" "zone" "attic" } }
	"DispenserSpot" { "1" { "origin" "50 0 0" } }
}`)
	if err != nil {
		t.Fatal(err)
	}
	pairs := PairDispensers(c)
	if len(pairs) != 1 || !pairs[0].Taken || pairs[0].RangeUnits != 50 {
		t.Fatalf("got %+v, want the unclaimed spot at 50u", pairs)
	}
}

// A nest with no spot to take puts its dispenser in the nest area, which is the
// finding this exists to print. Mannhattan does it on every nest.
func TestNestWithNoSpotFallsBack(t *testing.T) {
	c, err := ParseMapConfig(`"MapConfig"
{
	"EngineerNest" { "1" { "origin" "0 0 0" "zone" "attic" } }
	"DispenserSpot" { "1" { "origin" "50 0 0" "zone" "cellar" } }
}`)
	if err != nil {
		t.Fatal(err)
	}
	pairs := PairDispensers(c)
	if len(pairs) != 1 || pairs[0].Taken {
		t.Fatalf("got %+v, want a fall back", pairs)
	}
}

// A config with no nest or no spot has nothing to say, and says nothing.
func TestNothingToPairIsNoReport(t *testing.T) {
	c, err := ParseMapConfig(`"MapConfig" { "SniperSpot" { "1" { "origin" "0 0 0" } } }`)
	if err != nil {
		t.Fatal(err)
	}
	if report := DispenserReport(c); report != "" {
		t.Fatalf("a config with no nests reported %q", report)
	}
}
