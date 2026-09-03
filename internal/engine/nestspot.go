package engine

// NestSpotCalls are the answers for the nest geometry.
type NestSpotCalls struct {
	ArcTangent2              func(y float32, x float32) float32
	DegToRad                 func(degrees float32) float32
	Cosine                   func(radians float32) float32
	Sine                     func(radians float32) float32
	FloatAbs                 func(value float32) float32
	ClosestPointOnArea       func(a Area, from [3]float32) [3]float32
	MaxFloat                 func(a float32, b float32) float32
	PickConfiguredNestArea   func(client int32, target [3]float32, sentryRange float32) Area
	PickMapHintNestArea      func(client int32, target [3]float32, sentryRange float32) Area
	PickBuildAreaPreRound    func(client int32, sentryRange float32) Area
	IsNestRangeSane          func(rangeToBomb float32, sentryRange float32) bool
	NestDistanceLimit        func() float32
	PickBuildAreaRanged      func(client int32, sentryRange float32) Area
	ScoreNestArea            func(client int32, area NavArea, target [3]float32, sentryRange float32, approach List) float32
	CollectBombApproachAreas func(target [3]float32, sentryRange float32, out List)
	NavAreaAt                func(l NavAreaList, index int32) NavArea
	BestNestArea             func(client int32, areas List, target [3]float32, sentryRange float32) Area
	Extent                   func(a Area) ([3]float32, [3]float32)
	Z                        func(a Area, x float32, y float32) float32
}

var nestSpots NestSpotCalls

// InstallNestSpots puts a set of answers behind them.
func InstallNestSpots(c NestSpotCalls) func() {
	previous := nestSpots
	Fill(&c)
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
func ArcTangent2(y float32, x float32) float32 { return nestSpots.ArcTangent2(y, x) }

// DegToRad converts, because the ring is written in degrees and the maths is in
// radians.
//
//sp:native DegToRad
func DegToRad(degrees float32) float32 { return nestSpots.DegToRad(degrees) }

// Cosine of an angle in radians.
//
//sp:native Cosine
func Cosine(radians float32) float32 { return nestSpots.Cosine(radians) }

// Sine of an angle in radians.
//
//sp:native Sine
func Sine(radians float32) float32 { return nestSpots.Sine(radians) }

// FloatAbs drops the sign.
//
//sp:native FloatAbs
func FloatAbs(value float32) float32 { return nestSpots.FloatAbs(value) }

// ClosestPointOnArea is where on the area a position lands, which is the ground
// under a stand point rather than the point itself.
//
//sp:method GetClosestPointOnArea
func (a Area) ClosestPointOnArea(from [3]float32) (ground [3]float32) {
	return nestSpots.ClosestPointOnArea(a, from)
}

// Extent is the box an area covers, low corner and high.
//
//sp:method GetExtent
func (a Area) Extent() (low [3]float32, high [3]float32) { return nestSpots.Extent(a) }

// Z is the height of the area's own surface under a point, which is what stops a
// random point inside the box floating above or below the floor.
//
//sp:method GetZ
func (a Area) Z(x float32, y float32) float32 { return nestSpots.Z(a, x, y) }

// NestSpotMatchRange is how near an authored spot has to be to a nest area
// before it is that nest. internal/body/nestspot declares it.
//
//sp:global NEST_SPOT_MATCH_RANGE
func NestSpotMatchRange() float32 { return 400.0 }

// FeatureNestZones is the switch on one engineer per named zone.
//
//sp:global FEATURE_NEST_ZONES
func FeatureNestZones() int32 { return 4 }

// BestNestArea is the highest scoring of the areas offered.
//
//sp:body BestNestArea
func BestNestArea(client int32, areas List, target [3]float32, sentryRange float32) Area {
	return nestSpots.BestNestArea(client, areas, target, sentryRange)
}

// NestDepth is redbots_manager_engineer_nest_depth, the share of the bomb's route
// an engineer may hold ground along.
//
//sp:global redbots_manager_engineer_nest_depth
func NestDepth() ConVar { return 0 }

// MaxFloat is the larger of two. The plugin has its own, and the generator's own
// max would emit a second one beside it.
//
//sp:library MaxFloat
func MaxFloat(a float32, b float32) float32 { return nestSpots.MaxFloat(a, b) }

// NavAreaAt is one area of the whole mesh, by index.
//
//sp:method Get
func (l NavAreaList) NavAreaAt(index int32) NavArea { return nestSpots.NavAreaAt(l, index) }

// NavAreaList is TheNavAreas, the mesh as a list.
//
//sp:tag ArrayList
type NavAreaList int32

// AllNavAreas is TheNavAreas itself.
//
//sp:global TheNavAreas
func AllNavAreas() NavAreaList { return 0 }

// PickConfiguredNestArea is the best of the ground the map configuration names.
//
//sp:body PickConfiguredNestArea
func PickConfiguredNestArea(client int32, target [3]float32, sentryRange float32) Area {
	return nestSpots.PickConfiguredNestArea(client, target, sentryRange)
}

// PickMapHintNestArea is the best of the ground the map's own entities name.
//
//sp:body PickMapHintNestArea
func PickMapHintNestArea(client int32, target [3]float32, sentryRange float32) Area {
	return nestSpots.PickMapHintNestArea(client, target, sentryRange)
}

// PickBuildAreaPreRound is the between-rounds answer, which walks the mesh from
// the hatch rather than from the bomb.
//
//sp:body PickBuildAreaPreRound
func PickBuildAreaPreRound(client int32, sentryRange float32) Area {
	return nestSpots.PickBuildAreaPreRound(client, sentryRange)
}

// IsNestRangeSane says the ground is far enough from the bomb to shoot at it and
// near enough to reach it.
//
//sp:body IsNestRangeSane
func IsNestRangeSane(rangeToBomb float32, sentryRange float32) bool {
	return nestSpots.IsNestRangeSane(rangeToBomb, sentryRange)
}

// NestDistanceLimit is how far along the bomb's route an engineer may hold
// ground.
//
//sp:body NestDistanceLimit
func NestDistanceLimit() float32 { return nestSpots.NestDistanceLimit() }

// NestRelocateScoreGainMin is how much better a spot has to score than the one an
// engineer holds before he moves to it between waves.
//
//sp:global redbots_manager_engineer_nest_relocate_score_gain_min
func NestRelocateScoreGainMin() ConVar { return 0 }

// ScoreNestArea is what one piece of ground is worth to this engineer.
//
//sp:body ScoreNestArea
func ScoreNestArea(client int32, area NavArea, target [3]float32, sentryRange float32, approach List) float32 {
	return nestSpots.ScoreNestArea(client, area, target, sentryRange, approach)
}

// CollectBombApproachAreas samples the ground the robots cross.
//
//sp:body CollectBombApproachAreas
func CollectBombApproachAreas(target [3]float32, sentryRange float32, out List) {
	nestSpots.CollectBombApproachAreas(target, sentryRange, out)
}

// PickBuildAreaRanged is PickBuildArea with the sentry range given rather than
// defaulted, which is what the relocation asks.
//
//sp:body PickBuildArea
func PickBuildAreaRanged(client int32, sentryRange float32) Area {
	return nestSpots.PickBuildAreaRanged(client, sentryRange)
}
