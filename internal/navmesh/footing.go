package navmesh

// Footing is what the mesh offers a body at one coordinate, and it is the
// distinction mvm-z83.32 was filed for.
//
// "No nav area under it" is not one thing. A spot on top of a rock and a spot
// in a ground-level pit both have no area over them and both read as a gap in
// the mesh, but they are different failures and one of them is not a failure at
// all. What separates them is not in the footprint, it is in how far the ground
// around the coordinate sits below it, so that height decides the verdict here
// rather than being a column somebody reads afterwards.
type Footing uint8

// The four ways a coordinate can sit on the mesh.
const (
	// FootingGround is ordinary: an area is under the coordinate and the
	// ground round it is at the same level.
	FootingGround Footing = iota

	// FootingRaised is a coordinate standing more than a step above every
	// surface near it. A spot on a rock, on a roof or on a container reads
	// here, and so does a sniper spot written at eye height. It is normally
	// deliberate: what makes it a fault is what the caller wanted, not the
	// shape.
	FootingRaised

	// FootingPocket is a hole in the mesh at ground level: no area under the
	// coordinate, meshed ground all round it within a snap's reach, and that
	// ground level with it so there is nothing lower to snap down to. A path
	// to it ends at the edge of the hole, the arrival test never comes true,
	// and a bot sent there re-picks forever.
	FootingPocket

	// FootingOffMesh is no surface within a snap of the coordinate at all.
	FootingOffMesh
)

// String names the footing for a report line.
func (f Footing) String() string {
	switch f {
	case FootingGround:
		return "on ground"
	case FootingRaised:
		return "raised"
	case FootingPocket:
		return "in a ground-level hole"
	case FootingOffMesh:
		return "off the mesh"
	default:
		return "?"
	}
}

// RaisedStep is how far a coordinate must stand over the ground round it before
// the two stop being the same level.
//
// It is StepHeight because that is the rise a body walks up without jumping: a
// surround within a step is ground a bot is already standing on, and a surround
// further down is a drop it has to come off. The measurement leaves the reading
// unambiguous rather than borderline. Over the shipped configs the raised spots
// sit 58 to 79 above their surround and the ground-level holes sit between 4
// below and 4 above, so nothing lands within thirty units of this line.
const RaisedStep = StepHeight

// SurroundHeight is how far pos stands above the ground round it: the highest
// surface within reach subtracted from pos, so a positive number is a coordinate
// over its surround and a negative one is a coordinate under it.
//
// ok is false when there is no surface within reach at all, which is a
// coordinate off the mesh rather than one at any height over it.
func (m *Mesh) SurroundHeight(pos Vec3, reach float32) (height float32, ok bool) {
	best := float32(0)
	for _, a := range m.AreasWithin(pos, reach) {
		if z := a.ClosestPoint(pos).Z; !ok || z > best {
			best, ok = z, true
		}
	}
	if !ok {
		return 0, false
	}

	return pos.Z - best, true
}

// height is one of these measurements as a report writes it. It exists to keep
// a height of minus a hundredth from printing as "-0", which reads as a
// direction the number does not have.
func height(f float32) float32 {
	if f > -0.5 && f < 0.5 {
		return 0
	}
	return f
}
