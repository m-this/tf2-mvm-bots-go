package engine

// NestSpotCalls are the answers for the nest geometry.
type NestSpotCalls struct {
	ArcTangent2        func(y float32, x float32) float32
	DegToRad           func(degrees float32) float32
	Cosine             func(radians float32) float32
	Sine               func(radians float32) float32
	FloatAbs           func(value float32) float32
	ClosestPointOnArea func(a Area, from [3]float32) [3]float32
	Extent             func(a Area) ([3]float32, [3]float32)
	Z                  func(a Area, x float32, y float32) float32
}

var nestSpots NestSpotCalls

// InstallNestSpots puts a set of answers behind them.
func InstallNestSpots(c NestSpotCalls) func() {
	previous := nestSpots
	nestSpots = c
	return func() { nestSpots = previous }
}

// EngineerNestSpots are the nest coordinates the map configuration names.
//
//sp:global g_arrMapConfig.adtEngineerNestLocation
func EngineerNestSpots() List { return 0 }

// EngineerNestZones are the zones those nests belong to, one per spot.
//
//sp:global g_arrMapConfig.adtEngineerNestZone
func EngineerNestZones() List { return 0 }

// NestTankOnlySpots are the nests that are only worth holding on a tank wave.
//
//sp:global g_arrMapConfig.adtNestTankOnlyLocation
func NestTankOnlySpots() List { return 0 }

// NestNoTankSpots are the nests that are only worth holding when no tank is
// coming.
//
//sp:global g_arrMapConfig.adtNestNoTankLocation
func NestNoTankSpots() List { return 0 }

// ArcTangent2 is the angle of a vector, in radians.
//
//sp:native ArcTangent2
func ArcTangent2(y float32, x float32) float32 {
	if nestSpots.ArcTangent2 == nil {
		missing("ArcTangent2")
	}
	return nestSpots.ArcTangent2(y, x)
}

// DegToRad converts, because the ring is written in degrees and the maths is in
// radians.
//
//sp:native DegToRad
func DegToRad(degrees float32) float32 {
	if nestSpots.DegToRad == nil {
		missing("DegToRad")
	}
	return nestSpots.DegToRad(degrees)
}

// Cosine of an angle in radians.
//
//sp:native Cosine
func Cosine(radians float32) float32 {
	if nestSpots.Cosine == nil {
		missing("Cosine")
	}
	return nestSpots.Cosine(radians)
}

// Sine of an angle in radians.
//
//sp:native Sine
func Sine(radians float32) float32 {
	if nestSpots.Sine == nil {
		missing("Sine")
	}
	return nestSpots.Sine(radians)
}

// FloatAbs drops the sign.
//
//sp:native FloatAbs
func FloatAbs(value float32) float32 {
	if nestSpots.FloatAbs == nil {
		missing("FloatAbs")
	}
	return nestSpots.FloatAbs(value)
}

// ClosestPointOnArea is where on the area a position lands, which is the ground
// under a stand point rather than the point itself.
//
//sp:method GetClosestPointOnArea
func (a Area) ClosestPointOnArea(from [3]float32) (ground [3]float32) {
	if nestSpots.ClosestPointOnArea == nil {
		missing("CNavArea.GetClosestPointOnArea")
	}
	return nestSpots.ClosestPointOnArea(a, from)
}

// Extent is the box an area covers, low corner and high.
//
//sp:method GetExtent
func (a Area) Extent() (low [3]float32, high [3]float32) {
	if nestSpots.Extent == nil {
		missing("CNavArea.GetExtent")
	}
	return nestSpots.Extent(a)
}

// Z is the height of the area's own surface under a point, which is what stops a
// random point inside the box floating above or below the floor.
//
//sp:method GetZ
func (a Area) Z(x float32, y float32) float32 {
	if nestSpots.Z == nil {
		missing("CNavArea.GetZ")
	}
	return nestSpots.Z(a, x, y)
}
