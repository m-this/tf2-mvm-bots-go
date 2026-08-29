package navmesh

import (
	"fmt"
	"math"
)

// The numbers BuildStandPoint is written against, from source/redbots3/util.sp.
const (
	// BuildStandSearch is how far off the ring point the nav mesh may be,
	// BUILD_STAND_SEARCH.
	BuildStandSearch float32 = 120

	// BuildStandStorey is how much height still counts as beside the spot,
	// BUILD_STAND_STOREY. A ground further off than this is a storey away and
	// the side is refused.
	BuildStandStorey float32 = 100

	// BuildReach is the distance a building goes down in front of the man, and
	// so how far short of the spot he stands. The teleporter, the dispenser and
	// the sentry all use 90.
	BuildReach float32 = 90

	// BuildTryPoints is how many sides round a spot each of them tries.
	BuildTryPoints = 8
)

// StandRefusal says why BuildStandPoint returned false, or that it did not.
type StandRefusal uint8

// The two ways BuildStandPoint refuses a side, plus the accepting case.
const (
	// StandAccepted means the side produced a stand point.
	StandAccepted StandRefusal = iota

	// StandNoArea means no nav area lies within BuildStandSearch of the ring
	// point, so there is nowhere to stand on that side at all.
	StandNoArea

	// StandStorey means an area was found but its surface is more than
	// BuildStandStorey from the spot, so standing there is standing under or
	// over the spot rather than beside it.
	StandStorey
)

// String names the refusal for a report line.
func (r StandRefusal) String() string {
	switch r {
	case StandAccepted:
		return "accepted"
	case StandNoArea:
		return "no area within the search radius"
	case StandStorey:
		return "the ground found is a storey off the spot"
	default:
		return "?"
	}
}

// StandPoint is one side's answer from BuildStandPoint, with the working shown.
type StandPoint struct {
	// Attempt is which side round the spot this was, and Ring is the point that
	// side put in front of the spot before the mesh was asked anything.
	Attempt int
	Ring    Vec3

	// Area is the area the ring point snapped to, nil when nothing was within
	// BuildStandSearch. Stand is where on it he ends up.
	Area  *Area
	Stand Vec3

	// Refusal is why the side failed, or StandAccepted.
	Refusal StandRefusal
}

// OK reports whether this side produced a stand point.
func (s StandPoint) OK() bool { return s.Refusal == StandAccepted }

// BuildStandPoint is the port of the same function in source/redbots3/util.sp.
//
// A building goes down in front of the man and never under him, so the place to
// stand is a build's reach short of the spot with the spot in front of him.
// Attempt zero is the side he is coming from and each one after it is a step
// round the spot.
//
// The arithmetic is the plugin's, in the plugin's order, at the plugin's width.
// What differs is the mesh query underneath: see Mesh.NearestArea.
func (m *Mesh) BuildStandPoint(spot, from Vec3, attempt, attempts int, reach float32) StandPoint {
	away := from.Sub(spot)
	away.Z = 0

	if away.Length() < 1 {
		// He is standing on it, so any side will do to start from.
		away = Vec3{X: 1}
	} else {
		length := away.Length()
		away = Vec3{away.X / length, away.Y / length, 0}
	}

	yaw := math.Atan2(float64(away.Y), float64(away.X)) +
		float64(360.0/float32(attempts)*float32(attempt))*math.Pi/180

	ring := Vec3{
		X: spot.X + float32(math.Cos(yaw))*reach,
		Y: spot.Y + float32(math.Sin(yaw))*reach,
		Z: spot.Z,
	}

	out := StandPoint{Attempt: attempt, Ring: ring}

	area := m.NearestArea(ring, BuildStandSearch)
	if area == nil {
		out.Refusal = StandNoArea
		return out
	}

	ground := area.ClosestPoint(ring)
	out.Area = area
	out.Stand = ground

	if absf(ground.Z-spot.Z) > BuildStandStorey {
		out.Refusal = StandStorey
		return out
	}

	return out
}

// SnapVerdict is what a whole ring of BuildStandPoint attempts says about one
// declared spot: whether the sides that were accepted put the man on the spot's
// own ground or somewhere else.
//
// This is mvm-fgs made into a query. The bug is not that BuildStandPoint fails,
// it is that it succeeds onto the wrong ground: the ring point beside a spot on
// a rock snaps to the floor at the rock's foot, which is inside
// BuildStandStorey and so is accepted, and every side is then refused from down
// there for reasons the mesh cannot see.
type SnapVerdict struct {
	Spot Spot

	// Intended is the area the spot itself sits on, or nil when the spot is not
	// on the mesh at all within Tolerance.
	Intended  *Area
	Tolerance float32

	// Sides is every attempt round the spot, in order.
	Sides []StandPoint

	// Accepted, Elsewhere and OnIntended count the sides that produced a stand
	// point, of those how many landed on an area other than Intended, and how
	// many landed on Intended itself.
	Accepted   int
	Elsewhere  int
	OnIntended int

	// WorstDrop is the largest height by which an accepted side's ground sits
	// below the spot, and LeastDrop the smallest. Both are zero when no side
	// was accepted.
	//
	// LeastDrop is the one that separates a spot on a rock from a spot on
	// ordinary ground whose ring happens to reach the next tile along. A large
	// WorstDrop only says one side of the spot falls away. A large LeastDrop
	// says there is nowhere at the spot's own level to stand at all, which is
	// mvm-wxp: he takes the best of eight and still builds a storey down.
	WorstDrop float32
	LeastDrop float32
}

// Wrong reports whether every side that was accepted landed somewhere other than
// the spot's own area. That is the mvm-fgs shape exactly: he is offered ground,
// walks to it, and cannot build from it.
func (v SnapVerdict) Wrong() bool { return v.Accepted > 0 && v.OnIntended == 0 }

// Relocated reports whether the building silently moves: every accepted side
// landed on other ground, and even the shallowest of them is more than a step
// below the spot, so the man is never offered anything at the spot's own level.
//
// The height is what makes this a fault rather than a detail. Half the shipped
// dispenser spots have all eight sides land on a neighbouring area, because a
// ring ninety units out crosses a mesh cut on twenty-five; those sides are level
// with the spot and he builds where he was told. Wrong on its own does not
// separate the two, which is the mvm-z83.32 blind spot in its third instance.
func (v SnapVerdict) Relocated() bool { return v.Wrong() && v.LeastDrop > RaisedStep }

// Stranded reports whether no side was accepted at all.
func (v SnapVerdict) Stranded() bool { return v.Accepted == 0 }

// String is the verdict as one report line.
func (v SnapVerdict) String() string {
	intended := "off mesh"
	if v.Intended != nil {
		intended = fmt.Sprintf("area %d", v.Intended.ID)
	}
	return fmt.Sprintf("%s: intended %s, %d/%d sides accepted, %d elsewhere, drop %.0f to %.0f",
		v.Spot, intended, v.Accepted, len(v.Sides), v.Elsewhere, v.LeastDrop, v.WorstDrop)
}

// CheckSnap runs the whole ring of BuildStandPoint attempts against one spot and
// reports where they land.
//
// from is where the man is coming from, which only rotates the ring; passing the
// spot itself walks all eight sides from a fixed start, which is what a check
// over a config wants because it does not depend on where a bot happened to be.
func (m *Mesh) CheckSnap(spot Spot, from Vec3, attempts int, reach, tolerance float32) SnapVerdict {
	v := SnapVerdict{
		Spot:      spot,
		Intended:  m.AreaAt(spot.Origin, tolerance),
		Tolerance: tolerance,
		Sides:     make([]StandPoint, 0, attempts),
	}

	for attempt := range attempts {
		side := m.BuildStandPoint(spot.Origin, from, attempt, attempts, reach)
		v.Sides = append(v.Sides, side)

		if !side.OK() {
			continue
		}

		v.Accepted++
		if v.Intended != nil && side.Area.ID == v.Intended.ID {
			v.OnIntended++
		} else {
			v.Elsewhere++
		}

		drop := spot.Origin.Z - side.Stand.Z
		if drop > v.WorstDrop {
			v.WorstDrop = drop
		}
		if v.Accepted == 1 || drop < v.LeastDrop {
			v.LeastDrop = drop
		}
	}

	return v
}
