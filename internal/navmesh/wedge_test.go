package navmesh

import "testing"

// mannworksWedge is the coordinate mvm-wb0 and mvm-ipf both name: where an
// engineer jams, ten times across five bots in one player's logs, and where two
// of the three crashes in that bundle landed.
var mannworksWedge = Vec3{1014, 885, 274}

// TestMannworksWedgeIsADeclaredNest answers mvm-wb0's own question, "what the
// nest picker offers that leads him through it".
//
// It does not lead him through it. It sends him to it. The fourth EngineerNest
// spot in configs/defenderbots/map/mvm_mannworks.cfg is 1014 885 256, which is
// the wedge coordinate one step lower: the config names the spot the engineer
// gets stuck on.
func TestMannworksWedgeIsADeclaredNest(t *testing.T) {
	cfgs := loadConfigs(t)

	var mannworks *MapConfig
	for _, c := range cfgs {
		if c.Map == "mvm_mannworks" {
			mannworks = c
		}
	}
	if mannworks == nil {
		t.Fatal("no mannworks config")
	}

	var nearest Spot
	best := float32(-1)
	for _, s := range mannworks.Spots {
		if d := s.Origin.Distance(mannworksWedge); best < 0 || d < best {
			nearest, best = s, d
		}
	}

	if best > StepHeight {
		t.Fatalf("the nearest declared spot to the wedge is %s, %.0f away", nearest, best)
	}
	if nearest.Kind != EngineerNest {
		t.Fatalf("the spot on the wedge is a %s, want an %s", nearest.Kind, EngineerNest)
	}
}

// TestMannworksWedgeIsInAHole reads the coordinate itself.
//
// A nav mesh does not model props, so it cannot say a body wedges. It says
// something better here: there is no nav area over 1014 885 at all. The
// coordinate sits in a hole in the mesh, walled on every side by ground within
// twenty-five units of it, and the config's fourth nest spot is inside the same
// hole.
//
// That is a bot sent to stand where the mesh says there is nowhere to stand.
func TestMannworksWedgeIsInAHole(t *testing.T) {
	m := loadMap(t, "mvm_mannworks")

	v := m.CheckPoint(mannworksWedge)
	if v.Under != nil {
		t.Fatalf("the wedge coordinate stands on area %d, so the hole has been filled", v.Under.ID)
	}
	if v.Footing != FootingPocket {
		t.Fatalf("the wedge is %s, %.0f over the ground round it", v.Footing, v.SurroundHeight)
	}
	if !v.Suspicious() {
		t.Fatal("the mesh has nothing to say about the wedge")
	}

	// The hole is small: every side of it is meshed ground a step or so away.
	if v.NearestDistance > pinchWidth {
		t.Errorf("the nearest surface is %.0f away, which is a gap rather than a hole", v.NearestDistance)
	}

	// And the declared nest spot is inside it too, not merely near it.
	nest := Vec3{1014, 885, 256}
	if nv := m.CheckPoint(nest); nv.Under != nil {
		t.Errorf("the declared nest spot stands on area %d after all", nv.Under.ID)
	}
}

// TestSuspiciousIsRare is what makes the verdict worth reading. A signal that
// fires on most of a map is not a signal, so this bounds how much of each mesh
// the shapes it looks for actually cover.
func TestSuspiciousIsRare(t *testing.T) {
	for _, name := range shippedMaps {
		t.Run(name, func(t *testing.T) {
			m := loadMap(t, name)

			flagged := 0
			for _, a := range m.Areas {
				if m.CheckPoint(a.Center()).Suspicious() {
					flagged++
				}
			}

			share := float64(flagged) / float64(len(m.Areas))
			if share > 0.03 {
				t.Errorf("%.1f%% of areas are suspicious, which makes the verdict useless", share*100)
			}
		})
	}
}

// TestCheckPointOffMesh covers the case a wedged bot can also be in: not on the
// mesh at all.
func TestCheckPointOffMesh(t *testing.T) {
	m := loadMap(t, "mvm_mannworks")

	v := m.CheckPoint(Vec3{100000, 100000, 100000})
	if v.Under != nil {
		t.Fatalf("a coordinate outside the map stands on area %d", v.Under.ID)
	}
	if v.Nearest != nil {
		t.Fatalf("a coordinate outside the map snapped to area %d", v.Nearest.ID)
	}
	if v.NearestDistance < 10000 {
		t.Fatalf("the nearest surface is %.0f away, which is not outside the map", v.NearestDistance)
	}
}
