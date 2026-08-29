package navmesh

import "fmt"

// Heights a fall is judged against.
//
// Source accelerates a player at sv_gravity, 800 units per second squared, and
// Team Fortress 2 starts charging fall damage above a landing speed of 650 units
// per second. v squared over twice g puts that at 264 units of free fall, and
// the damage rises with the speed from there, so a drop of about twice that
// kills a 125 health class outright.
//
// These are what a report sorts by. Every query returns the measured descent as
// well, so a reader can disagree with the thresholds without disagreeing with
// the numbers.
const (
	// FallDamageHeight is the drop at which a fall begins to hurt.
	FallDamageHeight float32 = 264

	// FallLethalHeight is the drop at which a fall kills a light class.
	FallLethalHeight float32 = 520

	// BrinkProbe is how far past an unguarded edge the ground is looked for. It
	// is a nav grid cell, which is the smallest step the mesh is cut on.
	BrinkProbe float32 = 25

	// BrinkMinWidth is the narrowest unguarded span worth calling a way off the
	// edge. Anything narrower is a seam between two areas rather than a gap a
	// body walks through.
	BrinkMinWidth float32 = 24
)

// Fall is one place the ground goes down: an edge of an area with lower ground
// on the other side.
//
// There are two kinds and the difference is the whole point. A connected fall is
// a link the mesh holds, so the path finder will route a bot down it on purpose.
// An unconnected one is an edge the mesh simply stops at, with ground far below
// on the other side: no bot is ever routed off it, and anything that walks,
// blast-jumps or is knocked over it falls.
//
// mvm-0am and mvm-778 are the second kind. A teleporter exit beside one is a
// building that puts the team down at speed next to an edge nothing in the mesh
// warns about.
type Fall struct {
	From      AreaID
	Direction Direction

	// At is where on From's edge the drop begins, and Descent how far below it
	// the ground on the other side is.
	At      Vec3
	Descent float32

	// To is what you land on, or zero when nothing in the mesh is underneath.
	// Nothing underneath is a wall or a pit; the mesh does not distinguish them
	// and this does not pretend to.
	To AreaID

	// Connected is true when the mesh holds a link across this drop, and OneWay
	// that the link has no return.
	Connected bool
	OneWay    bool

	// Width is how much of the edge is unguarded, for an unconnected fall.
	Width float32
}

// String is the fall as one report line.
func (f Fall) String() string {
	kind := "unguarded edge"
	switch {
	case f.Connected && f.OneWay:
		kind = "one-way link"
	case f.Connected:
		kind = "two-way link"
	}

	onto := "nothing below"
	if f.To != 0 {
		onto = fmt.Sprintf("onto area %d", f.To)
	}

	return fmt.Sprintf("%.0f down off area %d %s at %.0f %.0f %.0f, %s, %s",
		f.Descent, f.From, f.Direction, f.At.X, f.At.Y, f.At.Z, onto, kind)
}

// Falls lists every way down off one area, deepest first. A descent no larger
// than StepHeight is a step and is left out.
func (m *Mesh) Falls(id AreaID) []Fall {
	a := m.byID[id]
	if a == nil {
		return nil
	}

	var out []Fall
	for d := range NumDirections {
		dir := Direction(d)
		out = append(out, m.connectedFalls(a, dir)...)
		out = append(out, m.unguardedFalls(a, dir)...)
	}

	sortFalls(out)

	return out
}

func (m *Mesh) connectedFalls(a *Area, dir Direction) []Fall {
	var out []Fall

	for _, toID := range a.Connections[dir] {
		to := m.byID[toID]
		if to == nil {
			continue
		}

		at, descent := edgeDescent(a, to, dir)
		if descent <= StepHeight {
			continue
		}

		out = append(out, Fall{
			From:      a.ID,
			Direction: dir,
			At:        at,
			Descent:   descent,
			To:        to.ID,
			Connected: true,
			OneWay:    !m.ConnectsTo(to.ID, a.ID),
		})
	}

	return out
}

// unguardedFalls walks the parts of one side that no connection covers, and asks
// what is under the ground just past each of them.
func (m *Mesh) unguardedFalls(a *Area, dir Direction) []Fall {
	lo, hi := edgeSpan(a, dir)

	covered := make([][2]float32, 0, len(a.Connections[dir]))
	for _, toID := range a.Connections[dir] {
		to := m.byID[toID]
		if to == nil {
			continue
		}
		bLo, bHi := edgeSpan(to, opposite(dir))
		covered = append(covered, [2]float32{maxf(lo, bLo), minf(hi, bHi)})
	}

	var out []Fall
	for _, gap := range uncovered(lo, hi, covered) {
		width := gap[1] - gap[0]
		if width < BrinkMinWidth {
			continue
		}

		mid := (gap[0] + gap[1]) * 0.5
		at, outside := edgePoints(a, dir, mid)

		below, descent := m.groundBelow(outside, at.Z)
		if descent <= StepHeight {
			continue
		}

		f := Fall{
			From:      a.ID,
			Direction: dir,
			At:        at,
			Descent:   descent,
			Width:     width,
		}
		if below != nil {
			f.To = below.ID
		}
		out = append(out, f)
	}

	return out
}

// groundBelow is the highest surface under a point that is at least StepHeight
// below fromZ, and how far below fromZ it is.
//
// It answers nothing in two cases, and they are the limit of what a nav file
// can say. If the point has walkable ground at about the same height, the edge
// is a wall between two floors rather than a drop. If it has no ground at all,
// the mesh does not know whether that is a wall, a pit nobody meshed, or the
// edge of the world, and this refuses to guess: a query that called every wall
// a lethal fall would report every area on every map.
//
// So this under-reports. A pit with no nav in the bottom of it is invisible
// here, and that is the honest failure direction for a check whose output is
// meant to be acted on.
func (m *Mesh) groundBelow(p Vec3, fromZ float32) (*Area, float32) {
	var best *Area
	bestZ := float32(0)
	wall := false

	m.grid.at(p.X, p.Y, func(a *Area) {
		if wall || !a.Contains2D(p.X, p.Y) {
			return
		}

		z := a.ZAt(p.X, p.Y)
		if z > fromZ-StepHeight {
			// Ground level with the edge, so the edge is a wall.
			if z >= fromZ-HalfHumanHeight && z <= fromZ+HumanHeight {
				wall = true
			}
			return
		}
		if best == nil || z > bestZ {
			best, bestZ = a, z
		}
	})

	if wall || best == nil {
		return nil, 0
	}

	return best, fromZ - bestZ
}

// edgeSpan is the interval one side of an area covers, in x for the north and
// south sides and in y for the east and west.
func edgeSpan(a *Area, dir Direction) (float32, float32) {
	if dir == North || dir == South {
		return a.NorthWest.X, a.SouthEast.X
	}
	return a.NorthWest.Y, a.SouthEast.Y
}

// edgePoints is the point on one side of an area at position t along it, and the
// point one probe step past it.
func edgePoints(a *Area, dir Direction, t float32) (on, outside Vec3) {
	switch dir {
	case North:
		on = Vec3{t, a.NorthWest.Y, a.ZAt(t, a.NorthWest.Y)}
		outside = Vec3{t, a.NorthWest.Y - BrinkProbe, on.Z}
	case South:
		on = Vec3{t, a.SouthEast.Y, a.ZAt(t, a.SouthEast.Y)}
		outside = Vec3{t, a.SouthEast.Y + BrinkProbe, on.Z}
	case East:
		on = Vec3{a.SouthEast.X, t, a.ZAt(a.SouthEast.X, t)}
		outside = Vec3{a.SouthEast.X + BrinkProbe, t, on.Z}
	default:
		on = Vec3{a.NorthWest.X, t, a.ZAt(a.NorthWest.X, t)}
		outside = Vec3{a.NorthWest.X - BrinkProbe, t, on.Z}
	}
	return on, outside
}

func opposite(d Direction) Direction {
	switch d {
	case North:
		return South
	case South:
		return North
	case East:
		return West
	default:
		return East
	}
}

// uncovered is the parts of [lo, hi] no interval in spans covers.
func uncovered(lo, hi float32, spans [][2]float32) [][2]float32 {
	sortSpans(spans)

	var out [][2]float32
	at := lo

	for _, s := range spans {
		if s[1] <= at {
			continue
		}
		if s[0] > at {
			out = append(out, [2]float32{at, minf(s[0], hi)})
		}
		at = maxf(at, s[1])
		if at >= hi {
			return out
		}
	}

	if at < hi {
		out = append(out, [2]float32{at, hi})
	}

	return out
}

// edgeDescent measures the step from one area to a connected neighbour at the
// middle of the boundary they share, and returns the point on the leaving edge.
// A large area's centre says nothing about where its edge is, which is why this
// works on the boundary.
func edgeDescent(from, to *Area, dir Direction) (Vec3, float32) {
	aLo, aHi := edgeSpan(from, dir)
	bLo, bHi := edgeSpan(to, opposite(dir))
	mid := overlapMid(aLo, aHi, bLo, bHi)

	at, _ := edgePoints(from, dir, mid)

	var toZ float32
	switch dir {
	case North:
		toZ = to.ZAt(mid, to.SouthEast.Y)
	case South:
		toZ = to.ZAt(mid, to.NorthWest.Y)
	case East:
		toZ = to.ZAt(to.NorthWest.X, mid)
	default:
		toZ = to.ZAt(to.SouthEast.X, mid)
	}

	return at, at.Z - toZ
}

// overlapMid is the middle of the span two intervals share, or the middle of the
// nearer ends when they share none.
func overlapMid(aLo, aHi, bLo, bHi float32) float32 {
	lo := maxf(aLo, bLo)
	hi := minf(aHi, bHi)
	if lo > hi {
		lo, hi = hi, lo
	}
	return (lo + hi) * 0.5
}

// DropVerdict is what the mesh says about the ground around one declared spot.
//
// This is mvm-0am and mvm-778 made into a query. A teleporter exit is a spot the
// whole team arrives at without looking, at speed, so what matters is not the
// spot's own area but every fall within a step or two of it.
type DropVerdict struct {
	Spot   Spot
	Radius float32

	// OnMesh is false when no area's surface is within Tolerance of the spot,
	// which is its own fault and makes the rest of this a reading about ground
	// near the spot rather than under it.
	OnMesh    bool
	Tolerance float32

	// Falls is every way down off an area within Radius of the spot, deepest
	// first, and Worst is the first of them.
	Falls []Fall
	Worst Fall
}

// Hurts reports whether the deepest fall beside the spot is far enough to cost
// health.
func (v DropVerdict) Hurts() bool { return v.Worst.Descent >= FallDamageHeight }

// Kills reports whether the deepest fall beside the spot is far enough to kill a
// light class.
func (v DropVerdict) Kills() bool { return v.Worst.Descent >= FallLethalHeight }

// String is the verdict as one report line.
func (v DropVerdict) String() string {
	if v.Worst.Descent == 0 {
		return fmt.Sprintf("%s: no fall over %.0f units within %.0f", v.Spot, StepHeight, v.Radius)
	}
	return fmt.Sprintf("%s: %s", v.Spot, v.Worst)
}

// CheckDrop reports the falls around one declared spot.
//
// radius is how far from the spot counts as beside it. A teleporter exit puts a
// player down facing an arbitrary way and already moving, so the honest radius
// is wider than the building.
func (m *Mesh) CheckDrop(spot Spot, radius, tolerance float32) DropVerdict {
	v := DropVerdict{Spot: spot, Radius: radius, Tolerance: tolerance}
	v.OnMesh = m.AreaAt(spot.Origin, tolerance) != nil

	for _, a := range m.AreasWithin(spot.Origin, radius) {
		v.Falls = append(v.Falls, m.Falls(a.ID)...)
	}

	sortFalls(v.Falls)
	if len(v.Falls) > 0 {
		v.Worst = v.Falls[0]
	}

	return v
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
