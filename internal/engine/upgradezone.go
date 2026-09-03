package engine

/*
Walking to the upgrade station, and the ray tracing that finds the ground in
front of one.

The station is a brush entity, so its centre is inside a wall as often as not.
The plugin picks a nav area near it, lifts the point fifty units, and traces
back at the station to find somewhere a bot can actually stand.
*/

// StationCalls are the answers.
type StationCalls struct {
	FindClosestUpgradeStation func(actor int32) int32
	SetInUpgradeZone          func(client int32, inZone bool)
	RoundState                func() int32
	NearestNavArea            func(origin [3]float32, anyZ bool, maxDistance float32, checkLOS bool, checkGround bool, team int32) Area
	RandomPointIn             func(area Area) [3]float32
	TraceRay                  func(from [3]float32, to [3]float32, mask int32, rayType int32)
	TraceRayFilter            func(from [3]float32, to [3]float32, mask int32, rayType int32, filter TraceFilter)
	TraceEndPosition          func() [3]float32
	PathFailedFor             func(actor int32) bool
	NudgeTowardsGoal          func(actor int32, bot Bot, goal [3]float32)
	IsUpgradeStationEnabled   func(station int32) bool
	Upgrade                   func() Behaviour
}

var stations StationCalls

// InstallStations puts a set of answers behind them.
func InstallStations(c StationCalls) func() {
	previous := stations
	Fill(&c)
	stations = c
	return func() { stations = previous }
}

// TraceFilter is a trace filter function, which SourcePawn passes by name. The
// subset has no function values; this is a name it reads and hands straight on.
//
//sp:tag TraceEntityFilter
type TraceFilter int32

// IgnoreActors is NextBotTraceFilterIgnoreActors.
//
//sp:global NextBotTraceFilterIgnoreActors
func IgnoreActors() TraceFilter { return 0 }

// NullArea is NULL_AREA.
//
//sp:global NULL_AREA
func NullArea() Area { return 0 }

// TeamAny is TEAM_ANY.
//
//sp:global TEAM_ANY
func TeamAny() int32 { return -2 }

// MaskPlayerSolid is MASK_PLAYERSOLID.
//
//sp:global MASK_PLAYERSOLID
func MaskPlayerSolid() int32 { return 0x201400B }

// RayTypeEndPoint is RayType_EndPoint.
//
//sp:global RayType_EndPoint
func RayTypeEndPoint() int32 { return 1 }

// RoundStateRunning is RoundState_RoundRunning.
//
//sp:global RoundState_RoundRunning
func RoundStateRunning() int32 { return 4 }

// TeamRed is TFTeam_Red.
//
//sp:global TFTeam_Red
func TeamRed() Team { return 2 }

// FindClosestUpgradeStation is the station a bot should walk to, or -1.
//
//sp:body FindClosestUpgradeStation
func FindClosestUpgradeStation(actor int32) int32 { return stations.FindClosestUpgradeStation(actor) }

// SetInUpgradeZone tells the game the bot is at a station, which is what a bot
// that cannot reach one pretends.
//
//sp:library TF2_SetInUpgradeZone
func SetInUpgradeZone(client int32, inZone bool) { stations.SetInUpgradeZone(client, inZone) }

// RoundState is what the game is doing.
//
//sp:native GameRules_GetRoundState
func RoundState() int32 { return stations.RoundState() }

// NearestNavArea is the walkable ground closest to a point.
//
//sp:native TheNavMesh.GetNearestNavArea
func NearestNavArea(origin [3]float32, anyZ bool, maxDistance float32, checkLOS bool, checkGround bool, team int32) Area {
	return stations.NearestNavArea(origin, anyZ, maxDistance, checkLOS, checkGround, team)
}

// RandomPointIn is somewhere inside the area.
//
//sp:body CNavArea_GetRandomPoint
func RandomPointIn(area Area) (point [3]float32) { return stations.RandomPointIn(area) }

// TraceRay fires a ray and leaves the result where TraceEndPosition reads it.
//
//sp:native TR_TraceRay
func TraceRay(from [3]float32, to [3]float32, mask int32, rayType int32) {
	stations.TraceRay(from, to, mask, rayType)
}

// TraceRayFilter is the same with a filter that ignores players.
//
//sp:native TR_TraceRayFilter
func TraceRayFilter(from [3]float32, to [3]float32, mask int32, rayType int32, filter TraceFilter) {
	stations.TraceRayFilter(from, to, mask, rayType, filter)
}

// TraceEndPosition is where the last ray stopped.
//
//sp:native TR_GetEndPosition
func TraceEndPosition() (position [3]float32) { return stations.TraceEndPosition() }

// PathFailedFor says the route computation keeps failing, which is when the bot
// is nudged rather than walked.
//
//sp:body PathFailedFor
func PathFailedFor(actor int32) bool { return stations.PathFailedFor(actor) }

// NudgeTowardsGoal moves the bot a step by hand.
//
//sp:body NudgeTowardsGoal
func NudgeTowardsGoal(actor int32, bot Bot, goal [3]float32) {
	stations.NudgeTowardsGoal(actor, bot, goal)
}

// IsUpgradeStationEnabled says the station is open.
//
//sp:body IsUpgradeStationEnabled
func IsUpgradeStationEnabled(station int32) bool { return stations.IsUpgradeStationEnabled(station) }

// Upgrade is CTFBotUpgrade, the behaviour that spends the money.
//
//sp:body CTFBotUpgrade
func Upgrade() Behaviour { return stations.Upgrade() }
