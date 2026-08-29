// Package navmesh reads a Source engine navigation mesh and the mod's per-map
// config, and answers questions about spots and areas without a game server.
//
// It is a model of geometry, not of movement. It will say that a spot snaps to
// the wrong area, or that a named spot sits beside a fall the mesh itself
// describes. It will not say that a bot gets there, because nothing here
// simulates a body.
package navmesh

import "math"

// Vec3 is a Source world position, in Hammer units. The engine stores these as
// 32-bit floats and the plugin arithmetic runs at that width, so the model
// keeps it rather than widening and disagreeing at the last digit.
type Vec3 struct {
	X, Y, Z float32
}

// Sub returns v - w.
func (v Vec3) Sub(w Vec3) Vec3 { return Vec3{v.X - w.X, v.Y - w.Y, v.Z - w.Z} }

// Length is the 3D magnitude.
func (v Vec3) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

// Length2D is the magnitude ignoring height.
func (v Vec3) Length2D() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}

// Distance is |v - w|.
func (v Vec3) Distance(w Vec3) float32 { return v.Sub(w).Length() }

// Distance2D is |v - w| ignoring height.
func (v Vec3) Distance2D(w Vec3) float32 { return v.Sub(w).Length2D() }

// AreaID is an area's identifier as written in the file. Ids are not dense and
// not ordered, so nothing indexes by them without going through Mesh.Area.
type AreaID uint32

// Direction is one of the four sides an area connects along, in the order the
// file writes them.
type Direction uint8

// The four connection directions, in file order.
const (
	North Direction = iota
	East
	South
	West
)

// NumDirections is how many connection lists every area carries.
const NumDirections = 4

// String names the direction for a report line.
func (d Direction) String() string {
	switch d {
	case North:
		return "north"
	case East:
		return "east"
	case South:
		return "south"
	case West:
		return "west"
	default:
		return "?"
	}
}

// Attributes is the engine's own nav attribute bitfield, CNavArea::m_attributeFlags.
type Attributes uint32

// The engine nav attributes, from nav.h NavAttributeType. Only the bits this
// package names are interpreted; Attributes keeps the whole word so an
// unrecognised bit is visible rather than dropped.
const (
	AttrCrouch      Attributes = 1 << 0
	AttrJump        Attributes = 1 << 1
	AttrPrecise     Attributes = 1 << 2
	AttrNoJump      Attributes = 1 << 3
	AttrStop        Attributes = 1 << 4
	AttrRun         Attributes = 1 << 5
	AttrWalk        Attributes = 1 << 6
	AttrAvoid       Attributes = 1 << 7
	AttrTransient   Attributes = 1 << 8
	AttrDontHide    Attributes = 1 << 9
	AttrStand       Attributes = 1 << 10
	AttrNoHostages  Attributes = 1 << 11
	AttrStairs      Attributes = 1 << 12
	AttrNoMerge     Attributes = 1 << 13
	AttrObstacleTop Attributes = 1 << 14
	AttrCliff       Attributes = 1 << 15
)

// Has reports whether every bit in mask is set.
func (a Attributes) Has(mask Attributes) bool { return a&mask == mask }

// TFAttributes is CTFNavArea's own bitfield, written after the engine's as
// per-area custom data. Only the map-authored bits survive a save; the rest are
// computed when the mission starts and are absent from the file.
type TFAttributes uint32

// The TF attributes this package names, from CTFNavArea::TFNavAttributeType.
const (
	TFAttrBlocked       TFAttributes = 1 << 0
	TFAttrSpawnRoomRed  TFAttributes = 1 << 1
	TFAttrSpawnRoomBlu  TFAttributes = 1 << 2
	TFAttrSpawnRoomExit TFAttributes = 1 << 3
	TFAttrHasAmmo       TFAttributes = 1 << 4
	TFAttrHasHealth     TFAttributes = 1 << 5
	TFAttrControlPoint  TFAttributes = 1 << 6
	TFAttrSniperSpot    TFAttributes = 1 << 21
	TFAttrSentrySpot    TFAttributes = 1 << 22
	TFAttrNoSpawning    TFAttributes = 1 << 25
	TFAttrRescueCloset  TFAttributes = 1 << 26
	TFAttrBombDrop      TFAttributes = 1 << 27
)

// Has reports whether every bit in mask is set.
func (a TFAttributes) Has(mask TFAttributes) bool { return a&mask == mask }

// Visibility is the per-pair visibility class the mesh stores, from
// NavVisibilityType.
type Visibility uint8

// The two visibility classes an entry in an area's visible set can carry.
const (
	PotentiallyVisible Visibility = 1 << 0
	CompletelyVisible  Visibility = 1 << 1
)

// VisibleArea is one entry of an area's precomputed visible set.
type VisibleArea struct {
	ID         AreaID
	Visibility Visibility
}

// Area is one nav area: a horizontal rectangle in x and y with a height at each
// of its four corners, plus the areas reachable from it along each side.
//
// The file stores two opposite corners and the heights of the other two, so
// NorthWest.Z is the height at (NorthWest.X, NorthWest.Y) and SouthEast.Z the
// height at (SouthEast.X, SouthEast.Y); NorthEastZ and SouthWestZ fill in the
// rest. Source's y axis grows southward, so NorthWest is the minimum corner.
type Area struct {
	ID           AreaID
	Attributes   Attributes
	TFAttributes TFAttributes
	NorthWest    Vec3
	SouthEast    Vec3
	NorthEastZ   float32
	SouthWestZ   float32
	Place        uint16
	Connections  [NumDirections][]AreaID

	// Visible is the area's own precomputed visible set. When InheritVisibility
	// is non-zero the real set is that area's, with this one layered over it.
	Visible           []VisibleArea
	InheritVisibility AreaID
}

// SizeX is the area's extent along x.
func (a *Area) SizeX() float32 { return a.SouthEast.X - a.NorthWest.X }

// SizeY is the area's extent along y.
func (a *Area) SizeY() float32 { return a.SouthEast.Y - a.NorthWest.Y }

// Center is the middle of the area, at the height the surface has there.
func (a *Area) Center() Vec3 {
	x := (a.NorthWest.X + a.SouthEast.X) * 0.5
	y := (a.NorthWest.Y + a.SouthEast.Y) * 0.5
	return Vec3{x, y, a.ZAt(x, y)}
}

// ZAt is the height of the area's surface above (x, y), bilinear over the four
// corner heights and clamped to the rectangle. This is CNavArea::GetZ.
func (a *Area) ZAt(x, y float32) float32 {
	dx := a.SizeX()
	dy := a.SizeY()

	// A degenerate area has one height and no slope to interpolate over.
	if dx == 0 || dy == 0 {
		return a.NorthEastZ
	}

	u := clamp01((x - a.NorthWest.X) / dx)
	v := clamp01((y - a.NorthWest.Y) / dy)

	northZ := a.NorthWest.Z + u*(a.NorthEastZ-a.NorthWest.Z)
	southZ := a.SouthWestZ + u*(a.SouthEast.Z-a.SouthWestZ)

	return northZ + v*(southZ-northZ)
}

// Contains2D reports whether (x, y) is inside the area's footprint, height
// ignored.
func (a *Area) Contains2D(x, y float32) bool {
	return x >= a.NorthWest.X && x <= a.SouthEast.X &&
		y >= a.NorthWest.Y && y <= a.SouthEast.Y
}

// ClosestPoint is the point of the area's surface nearest p, which is p's x and
// y clamped into the footprint at the height the surface has there. This is
// CNavArea::GetClosestPointOnArea.
func (a *Area) ClosestPoint(p Vec3) Vec3 {
	x := clampf(p.X, a.NorthWest.X, a.SouthEast.X)
	y := clampf(p.Y, a.NorthWest.Y, a.SouthEast.Y)
	return Vec3{x, y, a.ZAt(x, y)}
}

// Distance is how far p is from the nearest point of the area's surface.
func (a *Area) Distance(p Vec3) float32 { return a.ClosestPoint(p).Distance(p) }

// Neighbours lists every area this one connects out to, in direction order and
// with no duplicates removed: a connection appearing twice is a fact about the
// file.
func (a *Area) Neighbours() []AreaID {
	out := make([]AreaID, 0, len(a.Connections[0])+len(a.Connections[1])+
		len(a.Connections[2])+len(a.Connections[3]))
	for d := range a.Connections {
		out = append(out, a.Connections[d]...)
	}
	return out
}

// Mesh is a whole map's nav file: the areas and the header the file opened with.
type Mesh struct {
	Version    uint32
	SubVersion uint32
	BSPSize    uint32
	IsAnalyzed bool
	Places     []string
	Areas      []*Area

	byID     map[AreaID]*Area
	incoming map[AreaID][]AreaID
	grid     *grid
}

// Area returns the area with this id, or nil.
func (m *Mesh) Area(id AreaID) *Area { return m.byID[id] }

// Incoming lists every area that connects into this one. It is the reverse of
// Connections and is what tells a step down from a fall: a link with no return
// is ground you cannot get back up from.
func (m *Mesh) Incoming(id AreaID) []AreaID { return m.incoming[id] }

// ConnectsTo reports whether from has an outgoing connection to to.
func (m *Mesh) ConnectsTo(from, to AreaID) bool {
	a := m.byID[from]
	if a == nil {
		return false
	}
	for d := range a.Connections {
		for _, id := range a.Connections[d] {
			if id == to {
				return true
			}
		}
	}
	return false
}

// IsCompletelyVisible reports whether every corner of other can be seen from
// area, using the visibility the mesh precomputed. This is what
// CNavArea::IsCompletelyVisible answers for NestSightScore.
//
// An area with InheritVisibility set carries only its own difference from that
// area's set, so both are consulted, nearest first.
func (m *Mesh) IsCompletelyVisible(area, other AreaID) bool {
	a := m.byID[area]
	if a == nil {
		return false
	}
	if area == other {
		return true
	}
	if v, ok := lookupVisible(a.Visible, other); ok {
		return v.Has(CompletelyVisible)
	}
	if a.InheritVisibility != 0 {
		if base := m.byID[a.InheritVisibility]; base != nil {
			if v, ok := lookupVisible(base.Visible, other); ok {
				return v.Has(CompletelyVisible)
			}
		}
	}
	return false
}

// Has reports whether every bit in mask is set.
func (v Visibility) Has(mask Visibility) bool { return v&mask == mask }

func lookupVisible(set []VisibleArea, id AreaID) (Visibility, bool) {
	for _, e := range set {
		if e.ID == id {
			return e.Visibility, true
		}
	}
	return 0, false
}

func clamp01(f float32) float32 { return clampf(f, 0, 1) }

func clampf(f, lo, hi float32) float32 {
	if f < lo {
		return lo
	}
	if f > hi {
		return hi
	}
	return f
}
