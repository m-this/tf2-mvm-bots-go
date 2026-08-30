/*
Package nestspot is the part of source/redbots3/util.sp that turns a nav area
into the ground an engineer builds on, and the ring he stands on to build there.

Every engineer behaviour asks these three, and the build ones asked them through an
extern until now.
*/
package nestspot

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// How near an authored spot has to be to a nest area before it is that nest.
//
//sp:name NEST_SPOT_MATCH_RANGE
const spotMatchRange = 400.0

// How far off the ring a stand point may find its ground, and how much of a rise
// still counts as the same floor.
const (
	//sp:name BUILD_STAND_SEARCH
	standSearch = 120.0
	//sp:name BUILD_STAND_STOREY
	standStorey = 100.0
)

/*
NestBuildPosition is where a nest area's building actually goes: the coordinate
somebody walked the map to find when there is one, and the area's own centre when
there is not.
*/
//
//sp:name NestBuildPosition
func NestBuildPosition(area engine.Area) (out [3]float32) {
	// Before the GetCenter, not after it: reading the centre of a null area reads through a null
	if area == engine.NullArea() {
		out[0] = 0.0
		out[1] = 0.0
		out[2] = 0.0

		return out
	}

	out = area.Center()

	best := float32(spotMatchRange)

	/* A fresh destination each time, because the same array cannot be both

	SourcePawn passes an array by reference and a generated function zeroes its
	out-parameters at entry, so passing one variable as both the candidate and
	the answer zeroes the candidate before it is read. The emitter refuses that
	shape now; this is what it looks like written safely. */
	nearest, closest := FromList(engine.EngineerNestSpots(), out, best)

	out = nearest
	best = closest

	nearest, closest = FromList(engine.NestTankOnlySpots(), out, best)

	out = nearest
	best = closest

	/* The third distance is written and never read again

	The shipped code threads one best through all three lists and the last one
	goes out of scope with the function. SourcePawn writes it through a
	parameter, so it needs a name here whether or not anything reads it. */
	//nolint:staticcheck,ineffassign,wastedassign // the name exists because SourcePawn writes the distance through it
	nearest, closest = FromList(engine.NestNoTankSpots(), out, best)

	out = nearest

	return out
}

// FromList takes the nearest authored spot in one list, when it beats what the
// caller already had.
//
//sp:name NestSpotFromList
func FromList(spots engine.List, inout [3]float32, best float32) (out [3]float32, closest float32) {
	centre := inout

	out = inout
	closest = best

	for i := int32(0); i < spots.Length(); i++ {
		spot := spots.GetArray(i)

		distance := engine.VectorDistance(centre, spot)

		if distance < closest {
			closest = distance
			out = spot
		}
	}

	return out, closest
}

/*
NestZoneOf is the zone a nest area belongs to, and empty when the map names none.

Coaltown is why zones exist at all: the ground behind the wall on the right is eight
hundred units from the nest it serves and two hundred from a different one, so
nearest is the wrong answer and no distance rule fixes that.
*/
//
//sp:name NestZoneOf
//sp:length zone maxlength
//
//nolint:revive,ineffassign,staticcheck,wastedassign // the writes are the point: SourcePawn passes the buffer by reference and //sp:length carries its size
func NestZoneOf(area engine.Area, zone engine.Text, maxlength int32) {
	zone[0] = 0

	if area == engine.NullArea() {
		return
	}

	spots := engine.EngineerNestSpots()
	zones := engine.EngineerNestZones()

	centre := area.Center()

	best := float32(spotMatchRange)

	for i := int32(0); i < spots.Length() && i < zones.Length(); i++ {
		spot := spots.GetArray(i)

		distance := engine.VectorDistance(centre, spot)

		if distance < best {
			best = distance
			zone = zones.GetString(i)
		}
	}
}

/*
BuildStandPoint is where the man stands to put a building on a spot: a build's reach
away, on the side this attempt asks for, and on ground the nav mesh admits.

False when there is nothing walkable there, which is the caller's cue to go round to
the next side rather than to walk at thin air.
*/
//
//sp:name BuildStandPoint
func BuildStandPoint(spot [3]float32, from [3]float32, attempt int32, attempts int32, reach float32) (ok bool, stand [3]float32) {
	away := engine.SubtractVectors(from, spot)

	away[2] = 0.0

	// He is standing on it, so any side will do to start from
	length, away := engine.NormalizeVector(away)

	if length < 1.0 {
		away[0] = 1.0
		away[1] = 0.0
	}

	yaw := engine.ArcTangent2(away[1], away[0]) + engine.DegToRad(360.0/float32(attempts)*float32(attempt))

	stand[0] = spot[0] + engine.Cosine(yaw)*reach
	stand[1] = spot[1] + engine.Sine(yaw)*reach
	stand[2] = spot[2]

	area := engine.NearestNavArea(stand, false, standSearch, false, true, engine.TeamAny())

	if area == engine.NullArea() {
		return false, stand
	}

	ground := area.ClosestPointOnArea(stand)

	if engine.FloatAbs(ground[2]-spot[2]) > standStorey {
		return false, stand
	}

	stand = ground

	return true, stand
}

// RandomPointIn is somewhere inside the area, on its own surface rather than
// inside the box that bounds it.
//
//sp:name CNavArea_GetRandomPoint
func RandomPointIn(area engine.Area) (buffer [3]float32) {
	eLo, eHi := area.Extent()

	var spot [3]float32

	spot[0] = engine.RandomFloat(eLo[0], eHi[0])
	spot[1] = engine.RandomFloat(eLo[1], eHi[1])
	spot[2] = area.Z(spot[0], spot[1])

	buffer = spot

	return buffer
}
