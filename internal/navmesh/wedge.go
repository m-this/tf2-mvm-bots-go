package navmesh

import "fmt"

// PointVerdict is everything the mesh has to say about one world coordinate.
//
// This is mvm-wb0 and mvm-ipf made answerable. Neither is a spot in a config;
// both are a coordinate a bot kept ending up jammed at. A nav mesh does not
// model props, so it cannot say a body wedges there. What it can say is whether
// the ground at that coordinate has the shape ground gets stuck on: a pocket
// small enough to stand across, ground stacked over ground, an area you can
// walk into and not out of, or one whose only ways out are falls.
type PointVerdict struct {
	Pos Vec3

	// Under is the area the coordinate stands on, by the same rule the plugin's
	// own nav queries use. Nearest is what a 120 unit snap would return, which
	// is the same thing whenever Under is set.
	Under   *Area
	Nearest *Area

	// NearestDistance is how far the coordinate is from the nearest surface. A
	// large value on a coordinate a bot was standing at means the bot was off
	// the mesh.
	NearestDistance float32

	// Stacked is every area whose footprint contains the coordinate at any
	// height, lowest first. More than one means ground over ground: a ledge, a
	// walkway, a doorway with a floor above it.
	Stacked []*Area

	// OutDegree and InDegree are the connections leaving and entering Under.
	OutDegree int
	InDegree  int

	// NarrowestSide is the shorter of Under's two dimensions. The mesh is cut on
	// a 25 unit grid and a player is 49 units wide, so an area under about fifty
	// is ground nobody fits across squarely.
	NarrowestSide float32

	// Exits is every way down off Under, deepest first: the links the mesh
	// routes bots along and the edges it simply stops at.
	Exits []Fall

	// AllExitsFall is true when every connection leaving Under is a drop, so
	// the only way the mesh routes a bot out of this ground is downwards.
	AllExitsFall bool

	// DeadEnd is true when Under has no outgoing connections at all: ground the
	// mesh lets a bot walk into and never routes it out of.
	DeadEnd bool

	// Footing is where the coordinate stands relative to the ground round it:
	// on it, over it, in a hole in it, or nowhere near it. SurroundHeight is
	// the height that decides it, and it is read even when Footing is
	// FootingGround because a spot can sit on one area and still stand a storey
	// over everything else near it.
	Footing        Footing
	SurroundHeight float32
}

// PlayerWidth is a Team Fortress 2 player's bounding box across, from the
// standing hull. An area narrower than this is ground a body overhangs.
const PlayerWidth float32 = 49

// The thresholds Suspicious uses. They are set where the shape stops being
// ordinary: the mesh is cut on a 25 unit grid, so an area narrower than one cell
// is a sliver rather than ground, and areas overlap in pairs everywhere a level
// meets another, so three deep is the first count that means anything. Under
// these, between nought and seventeen areas per shipped map are flagged; a size
// that would flag half the mesh was tried first and thrown out.
const (
	pinchWidth  float32 = 25
	stackedDeep int     = 2
)

// Suspicious reports whether any of the shapes this verdict looks for is
// present. It is a filter for a sweep, not a diagnosis: a suspicious coordinate
// is one worth looking at, and an unsuspicious one is only a statement that the
// mesh has nothing to say.
func (v PointVerdict) Suspicious() bool {
	return v.Footing == FootingPocket || v.DeadEnd || v.AllExitsFall || len(v.Stacked) > stackedDeep ||
		(v.Under != nil && v.NarrowestSide < pinchWidth)
}

// String is the verdict as one report line.
func (v PointVerdict) String() string {
	if v.Under == nil {
		return fmt.Sprintf("%.0f %.0f %.0f: %s, %.0f over the ground round it, nearest surface %.0f away",
			v.Pos.X, v.Pos.Y, v.Pos.Z, v.Footing, height(v.SurroundHeight), v.NearestDistance)
	}
	return fmt.Sprintf("%.0f %.0f %.0f: area %d, %.0fx%.0f, %.0f over the ground round it, %d in %d out, %d stacked, %d falls out%s",
		v.Pos.X, v.Pos.Y, v.Pos.Z, v.Under.ID, v.Under.SizeX(), v.Under.SizeY(),
		height(v.SurroundHeight), v.InDegree, v.OutDegree, len(v.Stacked), len(v.Exits), deadEndNote(v))
}

func deadEndNote(v PointVerdict) string {
	switch {
	case v.DeadEnd:
		return ", dead end"
	case v.AllExitsFall:
		return ", every way out is a fall"
	default:
		return ""
	}
}

// CheckPoint reads one coordinate against the mesh.
func (m *Mesh) CheckPoint(pos Vec3) PointVerdict {
	v := PointVerdict{Pos: pos}
	v.Under = m.AreaUnder(pos, BeneathLimit)
	v.Nearest = m.NearestArea(pos, BuildStandSearch)

	if v.Nearest != nil {
		v.NearestDistance = v.Nearest.Distance(pos)
	} else {
		v.NearestDistance = nearestSurfaceDistance(m, pos)
	}

	m.grid.at(pos.X, pos.Y, func(a *Area) {
		if a.Contains2D(pos.X, pos.Y) {
			v.Stacked = append(v.Stacked, a)
		}
	})
	sortAreasByHeightAt(v.Stacked, pos)

	v.SurroundHeight, v.Footing = footingAt(m, pos, v.Under)

	if v.Under == nil {
		return v
	}

	v.OutDegree = len(v.Under.Neighbours())
	v.InDegree = len(m.Incoming(v.Under.ID))
	v.NarrowestSide = minf(v.Under.SizeX(), v.Under.SizeY())
	v.Exits = m.Falls(v.Under.ID)
	v.DeadEnd = v.OutDegree == 0
	routed := 0
	for _, f := range v.Exits {
		if f.Connected {
			routed++
		}
	}
	v.AllExitsFall = v.OutDegree > 0 && routed == v.OutDegree

	return v
}

// footingAt reads the coordinate against the ground round it rather than
// against the footprint under it. The order of the tests is the whole point of
// mvm-z83.32: a coordinate standing over its surround is raised whether or not
// an area's footprint happens to reach under it, and only a coordinate level
// with its surround and with nothing under it is in a hole.
func footingAt(m *Mesh, pos Vec3, under *Area) (float32, Footing) {
	height, ok := m.SurroundHeight(pos, BuildStandSearch)
	switch {
	case !ok:
		return 0, FootingOffMesh
	case height > RaisedStep:
		return height, FootingRaised
	case under == nil:
		return height, FootingPocket
	default:
		return height, FootingGround
	}
}

// nearestSurfaceDistance is the fallback for a position with nothing within a
// snap of it, which is either a hole in the mesh or somewhere off the map. It
// scans every area, because by then the grid has already said the neighbourhood
// is empty.
func nearestSurfaceDistance(m *Mesh, pos Vec3) float32 {
	best := float32(-1)
	for _, a := range m.Areas {
		d := a.Distance(pos)
		if best < 0 || d < best {
			best = d
		}
	}
	return best
}
