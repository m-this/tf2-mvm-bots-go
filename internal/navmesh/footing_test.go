package navmesh

import "testing"

// TestFootingSeparatesRockTopsFromHoles is mvm-z83.32.
//
// The old rule called a coordinate with no area under it "in a hole in the
// mesh", whatever surrounded it, and reported all seven of these the same way.
// Three of them are spots on top of rocks, which is deliberate authoring, and
// four are gaps in the ground. Nothing in the footprint separates them; the
// height of the surround does, and by a wide margin.
func TestFootingSeparatesRockTopsFromHoles(t *testing.T) {
	cases := []struct {
		mapName string
		pos     Vec3
		want    Footing
		note    string
	}{
		{"mvm_bigrock", Vec3{-552, 4266, 250}, FootingRaised, "EngineerNest 1, on a rock"},
		{"mvm_bigrock", Vec3{-423, 4579, 260}, FootingRaised, "EngineerNest 2, on a rock"},
		{"mvm_bigrock", Vec3{-178, 3921, 318}, FootingRaised, "TeleporterExit 1, mvm-fgs's rock"},
		{"mvm_mannhattan", Vec3{214, -1319, 204}, FootingPocket, "EngineerNest 1"},
		{"mvm_mannhattan", Vec3{-723, -629, -68}, FootingPocket, "DispenserSpot 2"},
		{"mvm_mannhattan", Vec3{-414, -3222, -240}, FootingPocket, "DispenserSpot 3"},
		{"mvm_mannworks", Vec3{1014, 885, 256}, FootingPocket, "EngineerNest 4, mvm-wb0's wedge"},
		{"mvm_rottenburg", Vec3{2238, 122, -146}, FootingOffMesh, "SniperSpot 4, mvm-0dn"},
	}

	for _, c := range cases {
		t.Run(c.note, func(t *testing.T) {
			v := loadMap(t, c.mapName).CheckPoint(c.pos)
			if v.Footing != c.want {
				t.Fatalf("%s is %q, want %q (%.0f over the ground round it)",
					c.note, v.Footing, c.want, v.SurroundHeight)
			}
		})
	}
}

// TestFootingReadingIsNotBorderline bounds how close the shipped data comes to
// the line the verdict is drawn on. A discriminator that separates two classes
// by a couple of units is a coin toss dressed as a measurement, so this states
// the empty band round RaisedStep rather than assuming there is one.
//
// It reads the coordinates the old rule could not tell apart: every declared
// spot with no nav area under it. The highest ground-level hole and the lowest
// rock top have to leave the line room on both sides.
func TestFootingReadingIsNotBorderline(t *testing.T) {
	const band float32 = 40

	pocketMax, raisedMin := float32(0), float32(0)
	pockets, raised := 0, 0

	for _, c := range loadConfigs(t) {
		if !haveNav(c.Map) {
			continue
		}
		m := loadMap(t, c.Map)

		for _, s := range c.Spots {
			v := m.CheckPoint(s.Origin)
			if v.Under != nil {
				continue
			}
			switch v.Footing {
			case FootingPocket:
				if pockets == 0 || v.SurroundHeight > pocketMax {
					pocketMax = v.SurroundHeight
				}
				pockets++
			case FootingRaised:
				if raised == 0 || v.SurroundHeight < raisedMin {
					raisedMin = v.SurroundHeight
				}
				raised++
			case FootingGround, FootingOffMesh:
			}
		}
	}

	if pockets == 0 || raised == 0 {
		t.Fatalf("%d holes and %d rock tops, so the two classes are not both present to separate", pockets, raised)
	}
	if pocketMax >= RaisedStep || raisedMin <= RaisedStep {
		t.Fatalf("the line at %.0f is not between the highest hole at %.0f and the lowest rock top at %.0f",
			RaisedStep, pocketMax, raisedMin)
	}
	if raisedMin-pocketMax < band {
		t.Errorf("only %.0f units separate the highest hole from the lowest rock top, which is too close to read",
			raisedMin-pocketMax)
	}
}

// TestRaisedIsNotSuspicious keeps the correction from being undone. A spot on a
// rock is ordinary geometry, and a sweep that flags it will be ignored on the
// case that matters.
func TestRaisedIsNotSuspicious(t *testing.T) {
	m := loadMap(t, "mvm_bigrock")

	v := m.CheckPoint(Vec3{-178, 3921, 318})
	if v.Suspicious() {
		t.Errorf("the rock top at %v is still flagged: %v", v.Pos, v)
	}
}
