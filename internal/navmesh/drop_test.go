package navmesh

import "testing"

// ExitDropRadius is how far from a teleporter exit counts as beside it. A player
// arrives facing an arbitrary way and already moving, so it is wider than the
// building.
const ExitDropRadius float32 = 150

// TestFallsOnKnownGeometry pins the two kinds of fall on ground picked because
// the answer can be read off the mesh by hand.
func TestFallsOnKnownGeometry(t *testing.T) {
	m := loadMap(t, "mvm_decoy")

	var connected, unguarded, oneWay int
	for _, a := range m.Areas {
		for _, f := range m.Falls(a.ID) {
			if f.Descent <= StepHeight {
				t.Fatalf("%v is a step, not a fall", f)
			}
			if f.From != a.ID {
				t.Fatalf("%v is attributed to area %d", f, a.ID)
			}

			switch {
			case f.Connected && f.OneWay:
				if m.ConnectsTo(f.To, f.From) {
					t.Fatalf("%v is called one-way but the return link exists", f)
				}
				oneWay++
				connected++
			case f.Connected:
				if !m.ConnectsTo(f.From, f.To) {
					t.Fatalf("%v is called connected but there is no link", f)
				}
				connected++
			default:
				if f.Width < BrinkMinWidth {
					t.Fatalf("%v is narrower than the minimum span", f)
				}
				unguarded++
			}
		}
	}

	if connected == 0 || unguarded == 0 || oneWay == 0 {
		t.Fatalf("decoy has %d connected falls, %d unguarded, %d one-way; want some of each",
			connected, unguarded, oneWay)
	}
}

// TestUncovered is the interval arithmetic the unguarded-edge query rests on,
// which is the one piece of this that is not a lookup.
func TestUncovered(t *testing.T) {
	cases := []struct {
		name   string
		lo, hi float32
		spans  [][2]float32
		want   [][2]float32
	}{
		{"nothing covered", 0, 100, nil, [][2]float32{{0, 100}}},
		{"fully covered", 0, 100, [][2]float32{{0, 100}}, nil},
		{"covered in two pieces", 0, 100, [][2]float32{{0, 50}, {50, 100}}, nil},
		{"a gap in the middle", 0, 100, [][2]float32{{0, 40}, {60, 100}}, [][2]float32{{40, 60}}},
		{"a gap at each end", 0, 100, [][2]float32{{20, 80}}, [][2]float32{{0, 20}, {80, 100}}},
		{"overlapping spans", 0, 100, [][2]float32{{0, 60}, {20, 80}}, [][2]float32{{80, 100}}},
		{"spans out of order", 0, 100, [][2]float32{{60, 100}, {0, 40}}, [][2]float32{{40, 60}}},
		{"a span wider than the edge", 0, 100, [][2]float32{{-50, 150}}, nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := uncovered(c.lo, c.hi, c.spans)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("got %v, want %v", got, c.want)
				}
			}
		})
	}
}

// TestTeleporterExitsBesideADrop is mvm-0am and mvm-778, run across every
// shipped map config at once rather than one map at a time.
//
// The result contradicts the bead, and the contradiction is the finding.
// mvm-0am says Rottenburg's exit is beside a fall a player dies on. The mesh
// says Rottenburg's declared exit has no fall over a step within 150 units of
// it, while four of the other five maps' exits do, and Mannhattan's first exit
// has one at exactly the height fall damage starts.
func TestTeleporterExitsBesideADrop(t *testing.T) {
	type verdict struct {
		mapName string
		spot    string
		descent float32
	}

	// Measured. The number is the deepest fall within ExitDropRadius of the
	// declared exit, rounded, and zero for none.
	want := []verdict{
		{"mvm_bigrock", "TeleporterExit 1", 0},
		{"mvm_coaltown", "TeleporterExit 1", 262},
		{"mvm_decoy", "TeleporterExit 1", 207},
		{"mvm_mannhattan", "TeleporterExit 1", 264},
		{"mvm_mannhattan", "TeleporterExit 2", 0},
		{"mvm_mannhattan", "TeleporterExit 3", 186},
		{"mvm_mannworks", "TeleporterExit 1", 249},
		{"mvm_rottenburg", "TeleporterExit 1", 0},
	}

	var got []verdict
	for _, c := range loadConfigs(t) {
		if !haveNav(c.Map) {
			continue
		}

		m := loadMap(t, c.Map)
		for _, s := range c.SpotsOf(TeleporterExit) {
			v := m.CheckDrop(s, ExitDropRadius, StepHeight)
			got = append(got, verdict{c.Map, string(s.Kind) + " " + s.Index, roundf(v.Worst.Descent)})
		}
	}

	if len(got) != len(want) {
		t.Fatalf("%d declared exits, want %d:\n%+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("exit %d is %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestRottenburgExitIsNotBesideADrop is mvm-0am stated as narrowly as the mesh
// allows, because it is the bead's own claim and it does not hold for the
// declared spot.
//
// Rottenburg turns MovingNests on, so the engineer relocates and the exit is
// placed on a ring round wherever the nest ended up rather than on the config's
// spot. Every fall on Rottenburg deep enough to hurt is above z 0, and every
// spot the config declares is below z minus 140, so the ground the player fell
// off is not ground this config names.
func TestRottenburgExitIsNotBesideADrop(t *testing.T) {
	m := loadMap(t, "mvm_rottenburg")
	exit := Spot{Kind: TeleporterExit, Index: "1", Origin: Vec3{1723, -199, -407}}

	for _, radius := range []float32{ExitDropRadius, 300, 500} {
		if v := m.CheckDrop(exit, radius, StepHeight); v.Worst.Descent > StepHeight {
			t.Fatalf("a %.0f unit fall is within %.0f of the declared exit: %v", v.Worst.Descent, radius, v.Worst)
		}
	}

	var deepest Fall
	for _, a := range m.Areas {
		for _, f := range m.Falls(a.ID) {
			if f.Descent > deepest.Descent {
				deepest = f
			}
		}
	}

	if !hurts(deepest) {
		t.Fatalf("rottenburg's deepest fall is only %.0f, so this map has no drop to find", deepest.Descent)
	}
	if deepest.At.Z < 0 {
		t.Fatalf("rottenburg's deepest fall is at z %.0f, which is the half of the map the config names", deepest.At.Z)
	}
}

func hurts(f Fall) bool { return f.Descent >= FallDamageHeight }
