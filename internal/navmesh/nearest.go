package navmesh

// The engine's body measurements, from nav.h. They decide which area a position
// is standing on, so they belong to the mesh model rather than to any one query.
const (
	// StepHeight is the rise a body walks up without jumping.
	StepHeight float32 = 18

	// HumanHeight is a standing player, and HalfHumanHeight the tolerance
	// GetNavArea allows for an area above the position it is given.
	HumanHeight     float32 = 72
	HalfHumanHeight float32 = 36

	// JumpCrouchHeight is the rise a crouch jump clears, which is the ceiling
	// the engineer climb in mvm-fgs is written against.
	JumpCrouchHeight float32 = 64

	// BeneathLimit is how far below a position GetNavArea will still call an
	// area the ground under it. This is the default of
	// CNavMesh::GetNavArea, and it is the number that lets a spot on a
	// seventy-unit rock resolve to the floor at its foot.
	BeneathLimit float32 = 120
)

// AreaUnder is the area a position is standing on: the highest area whose
// footprint contains it, whose surface is no more than HalfHumanHeight above it
// and no more than beneathLimit below. This is CNavMesh::GetNavArea.
//
// It is the first thing GetNearestNavArea tries, and on a raised spot with floor
// underneath it is also the last, because the floor qualifies.
func (m *Mesh) AreaUnder(pos Vec3, beneathLimit float32) *Area {
	var best *Area
	bestZ := float32(0)

	m.grid.at(pos.X, pos.Y, func(a *Area) {
		if !a.Contains2D(pos.X, pos.Y) {
			return
		}

		z := a.ZAt(pos.X, pos.Y)
		if z > pos.Z+HalfHumanHeight {
			return
		}
		if z < pos.Z-beneathLimit {
			return
		}
		if best == nil || z > bestZ {
			best, bestZ = a, z
		}
	})

	return best
}

// NearestArea is the area a position snaps to, within maxDist.
//
// It is CNavMesh::GetNearestNavArea with checkGround set, which is how every
// caller in the plugin asks: the area under the feet if there is one, and
// otherwise the area whose surface has the nearest point in three dimensions.
//
// What is not modelled is the line of sight and ground traces the engine can
// also be asked for. The plugin passes checkLOS false everywhere, so the only
// missing test is the ground trace inside GetNavArea, which rejects an area
// whose surface is separated from the position by world geometry. Nothing in a
// nav file describes that geometry, so this answer is the engine's answer or a
// superset of it, never a narrower one.
func (m *Mesh) NearestArea(pos Vec3, maxDist float32) *Area {
	if a := m.AreaUnder(pos, BeneathLimit); a != nil {
		return a
	}

	var best *Area
	bestDistSq := maxDist * maxDist

	m.grid.near(pos.X, pos.Y, maxDist, func(a *Area) {
		d := a.ClosestPoint(pos).Sub(pos)
		distSq := d.X*d.X + d.Y*d.Y + d.Z*d.Z
		if distSq >= bestDistSq {
			return
		}
		best, bestDistSq = a, distSq
	})

	return best
}

// AreaAt is the area a declared spot names: the one whose surface the spot sits
// on or just above, preferring a footprint that contains it and falling back to
// the nearest surface within tolerance.
//
// This is what "the intended area" means for a config spot. It is deliberately
// not NearestArea: NearestArea is what the plugin computes and is the thing
// under test, so a check that used it to define the right answer would agree
// with itself.
func (m *Mesh) AreaAt(pos Vec3, tolerance float32) *Area {
	var best *Area
	bestGap := tolerance

	m.grid.at(pos.X, pos.Y, func(a *Area) {
		if !a.Contains2D(pos.X, pos.Y) {
			return
		}
		gap := absf(a.ZAt(pos.X, pos.Y) - pos.Z)
		if gap > bestGap {
			return
		}
		best, bestGap = a, gap
	})
	if best != nil {
		return best
	}

	bestDist := tolerance
	m.grid.near(pos.X, pos.Y, tolerance, func(a *Area) {
		d := a.Distance(pos)
		if d > bestDist {
			return
		}
		best, bestDist = a, d
	})

	return best
}

// AreasWithin lists every area with a point of its surface inside radius of pos,
// nearest first.
func (m *Mesh) AreasWithin(pos Vec3, radius float32) []*Area {
	var out []*Area
	seen := make(map[AreaID]bool)

	m.grid.near(pos.X, pos.Y, radius, func(a *Area) {
		if seen[a.ID] || a.Distance(pos) > radius {
			return
		}
		seen[a.ID] = true
		out = append(out, a)
	})

	sortAreasByDistance(out, pos)

	return out
}

func absf(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}
