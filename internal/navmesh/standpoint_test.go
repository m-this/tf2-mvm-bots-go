package navmesh

import "testing"

// bigrockTeleporterExit is the spot mvm-fgs is about, from
// configs/defenderbots/map/mvm_bigrock.cfg.
var bigrockTeleporterExit = Spot{Kind: TeleporterExit, Index: "1", Origin: Vec3{-178, 3921, 318}}

// TestBigrockExitSnapsToTheFloor is mvm-fgs, as a test rather than a wave.
//
// The bead says BuildStandPoint snaps to the nearest area within 120 units and
// gets the floor below the rock, which passes the storey check. That is what the
// mesh says, with one fact the bead does not have: there is no nav area on the
// rock at all. The spot is 68 units above the nearest surface anywhere, so there
// is no ground up there for the snap to find, for a path to end on, or for the
// climb the bead added to land on.
func TestBigrockExitSnapsToTheFloor(t *testing.T) {
	m := loadMap(t, "mvm_bigrock")
	spot := bigrockTeleporterExit.Origin

	if a := m.AreaAt(spot, StepHeight); a != nil {
		t.Fatalf("area %d is within a step of the exit spot, so the mesh has changed", a.ID)
	}

	nearest := m.NearestArea(spot, BuildStandSearch)
	if nearest == nil {
		t.Fatal("nothing within the build search radius of the exit spot")
	}
	if gap := spot.Z - nearest.ZAt(spot.X, spot.Y); gap < 60 || gap > 70 {
		t.Fatalf("the nearest surface is %.1f below the spot, want the rock's 60 to 70", gap)
	}

	v := m.CheckSnap(bigrockTeleporterExit, spot, BuildTryPoints, BuildReach, HalfHumanHeight)

	if v.Intended != nil {
		t.Fatalf("the spot resolved to area %d, so it is on the mesh after all", v.Intended.ID)
	}
	if v.Accepted != BuildTryPoints {
		t.Fatalf("%d of %d sides accepted, want all of them: the bug is that none is refused",
			v.Accepted, BuildTryPoints)
	}
	if !v.Wrong() {
		t.Fatal("no side landed away from the spot's own ground, so the snap is not the fault")
	}
	if v.WorstDrop <= BuildReach || v.WorstDrop >= BuildStandStorey {
		t.Fatalf("the worst side stands %.0f below the spot; the bug needs it under the %.0f storey limit",
			v.WorstDrop, BuildStandStorey)
	}
}

// TestStandPointRefusals covers what BuildStandPoint has to refuse, on ground
// where the mesh answers plainly.
func TestStandPointRefusals(t *testing.T) {
	m := loadMap(t, "mvm_bigrock")

	cases := []struct {
		name  string
		spot  Vec3
		reach float32
		want  StandRefusal
	}{
		{
			name:  "a spot in the sky has no ground within the search radius",
			spot:  Vec3{-178, 3921, 4000},
			reach: BuildReach,
			want:  StandNoArea,
		},
		{
			name:  "a spot far outside the map has no ground at all",
			spot:  Vec3{100000, 100000, 0},
			reach: BuildReach,
			want:  StandNoArea,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for attempt := range BuildTryPoints {
				got := m.BuildStandPoint(c.spot, c.spot, attempt, BuildTryPoints, c.reach)
				if got.Refusal != c.want {
					t.Fatalf("side %d refused with %q, want %q", attempt, got.Refusal, c.want)
				}
			}
		})
	}
}

// TestStandPointRingIsARing checks the arithmetic before the mesh query: eight
// attempts put eight points a reach away from the spot, evenly round it.
func TestStandPointRingIsARing(t *testing.T) {
	m := loadMap(t, "mvm_bigrock")
	spot := Vec3{-178, 3921, 318}

	for attempt := range BuildTryPoints {
		got := m.BuildStandPoint(spot, spot, attempt, BuildTryPoints, BuildReach)

		if d := got.Ring.Distance2D(spot); absf(d-BuildReach) > 0.01 {
			t.Errorf("side %d ring point is %.2f from the spot, want %.0f", attempt, d, BuildReach)
		}
		if got.Ring.Z != spot.Z {
			t.Errorf("side %d ring point is at z %g, want the spot's %g", attempt, got.Ring.Z, spot.Z)
		}
	}
}

// TestGroundSpotsOffTheMesh is the general form of mvm-fgs, run over every
// shipped map at once: a spot a building goes on has to be ground.
//
// The tolerance is a step. A spot further off the mesh than that is either
// authored at eye height by mistake or authored on geometry the mesh does not
// cover, and both end the same way: the engineer is sent to whatever the nearest
// area is instead.
//
// This asserts the set as it stands rather than that the set is empty. Six spots
// are off the mesh today and they are upstream's to fix, not this package's; the
// test exists so that fixing one, or adding a seventh, is noticed. The distance
// is the gap to the nearest surface anywhere, so it is the number to move the
// spot by.
func TestGroundSpotsOffTheMesh(t *testing.T) {
	type offender struct {
		mapName  string
		spot     string
		distance float32
	}

	// Measured, not chosen. Regenerate with the report golden.
	want := []offender{
		{"mvm_bigrock", "EngineerNest 1", 98},
		{"mvm_bigrock", "EngineerNest 2", 96},
		{"mvm_bigrock", "TeleporterExit 1", 68},
		{"mvm_coaltown", "DispenserSpot 2", 79},
		{"mvm_mannhattan", "EngineerNest 1", 19},
		{"mvm_mannhattan", "DispenserSpot 2", 23},
	}

	var got []offender
	for _, c := range loadConfigs(t) {
		if !haveNav(c.Map) {
			continue
		}

		m := loadMap(t, c.Map)
		for _, s := range c.Spots {
			if !s.Kind.IsGround() {
				continue
			}
			if m.AreaAt(s.Origin, StepHeight) != nil {
				continue
			}
			got = append(got, offender{
				mapName:  c.Map,
				spot:     string(s.Kind) + " " + s.Index,
				distance: roundf(m.CheckPoint(s.Origin).NearestDistance),
			})
		}
	}

	if len(got) != len(want) {
		t.Fatalf("%d ground spots are off the mesh, want %d:\n%+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("offender %d is %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestSniperSpotsAreEyeHeight is the control for the test above. Sniper spots
// are all off the mesh by construction, so a rule that failed on any spot off
// the mesh would fail on every map for no reason. The offset is a player's eye,
// crouched at 45 or standing at 68, and that is what this pins.
//
// One spot is not an eye above anything: Rottenburg's fourth, 213 units from the
// nearest surface on the map. That is not an authoring height, it is a spot no
// bot can stand on, and it is listed here rather than excluded so that fixing it
// is noticed.
func TestSniperSpotsAreEyeHeight(t *testing.T) {
	const unreachable = "SniperSpot 4 at 2238 122 -146"

	for _, c := range loadConfigs(t) {
		if !haveNav(c.Map) {
			continue
		}

		t.Run(c.Map, func(t *testing.T) {
			m := loadMap(t, c.Map)

			for _, s := range c.SpotsOf(SniperSpot) {
				d := m.CheckPoint(s.Origin).NearestDistance

				if c.Map == "mvm_rottenburg" && s.String() == unreachable {
					if d < 200 {
						t.Errorf("%s is now %.0f off the mesh, so it may have been fixed", s, d)
					}
					continue
				}

				if d < 30 || d > 80 {
					t.Errorf("%s is %.0f off the mesh, which is neither a crouched eye nor a standing one", s, d)
				}
			}
		})
	}
}

func roundf(f float32) float32 {
	return float32(int(f + 0.5))
}
