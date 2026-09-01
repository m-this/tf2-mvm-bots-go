package engine

/*
The pathing seam: computing a route with the length cap, the clocks the budget
runs on, and the faults injector's hook.
*/

// PathingCalls are the answers.
type PathingCalls struct {
	ComputeToTargetBuilt func(p Path, bot Bot, target int32, maxDistance float32) bool
	ComputeToPosBuilt    func(p Path, bot Bot, goal [3]float32, maxDistance float32) bool
	PathLength           func(p Path) float32
	EngineTime           func() float32
	GameTickCount        func() int32
	UnreachableGoal      func(actor int32) (bool, [3]float32)
	Pathing              func(b PluginBot) bool
	MoveWedgedDefender   func(actor int32) bool
	AdjacentCount        func(a Area, dir Direction) int32
	AdjacentArea         func(a Area, dir Direction, index int32) Area
	OldWedgeRecovery     func() bool
	HasPathGoalVector    func(b PluginBot) bool
	HasPathGoalEntity    func(b PluginBot) bool
	PathGoalVector       func(b PluginBot) [3]float32
	PathGoalEntity       func(b PluginBot) int32
}

var pathings PathingCalls

// InstallPathings puts a set of answers behind them.
func InstallPathings(c PathingCalls) func() {
	previous := pathings
	pathings = c
	return func() { pathings = previous }
}

// ComputeToTargetBuilt computes a route to an entity and says whether one was
// built.
//
//sp:method ComputeToTarget
func (p Path) ComputeToTargetBuilt(bot Bot, target int32, maxDistance float32) bool {
	if pathings.ComputeToTargetBuilt == nil {
		missing("PathFollower.ComputeToTarget")
	}
	return pathings.ComputeToTargetBuilt(p, bot, target, maxDistance)
}

// ComputeToPosBuilt computes a route to a point and says whether one was built.
//
//sp:method ComputeToPos
func (p Path) ComputeToPosBuilt(bot Bot, goal [3]float32, maxDistance float32) bool {
	if pathings.ComputeToPosBuilt == nil {
		missing("PathFollower.ComputeToPos")
	}
	return pathings.ComputeToPosBuilt(p, bot, goal, maxDistance)
}

// Length is how long the computed route is.
//
//sp:method GetLength
func (p Path) Length() float32 {
	if pathings.PathLength == nil {
		missing("PathFollower.GetLength")
	}
	return pathings.PathLength(p)
}

// WallClock is GetEngineTime, for measuring rather than scheduling.
//
//sp:native GetEngineTime
func WallClock() float32 {
	if pathings.EngineTime == nil {
		missing("GetEngineTime")
	}
	return pathings.EngineTime()
}

// GameTickCount is the server's frame number.
//
//sp:native GetGameTickCount
func GameTickCount() int32 {
	if pathings.GameTickCount == nil {
		missing("GetGameTickCount")
	}
	return pathings.GameTickCount()
}

// FeaturePathLengthCap is FEATURE_PATH_LENGTH_CAP.
//
//sp:global FEATURE_PATH_LENGTH_CAP
func FeaturePathLengthCap() int32 { return 15 }

// UnreachableGoal is the faults injector's hook: a goal with no path to it, on
// demand. Ported, faults.
//
//sp:body DebugFaults_UnreachableGoal fills
func UnreachableGoal(actor int32) (injected bool, goal [3]float32) {
	if pathings.UnreachableGoal == nil {
		missing("DebugFaults_UnreachableGoal")
	}
	return pathings.UnreachableGoal(actor)
}

// Pathing is the read half of bPathing: whether the plugin's own walking is on.
//
//sp:property bPathing
func (b PluginBot) Pathing() bool {
	if pathings.Pathing == nil {
		missing("PluginBot.bPathing")
	}
	return pathings.Pathing(b)
}

// MoveWedgedDefender teleports a bot off ground it cannot leave on its own.
// Ported, stuckwatch.
//
//sp:body MoveWedgedDefender
func MoveWedgedDefender(actor int32) bool {
	if pathings.MoveWedgedDefender == nil {
		missing("MoveWedgedDefender")
	}
	return pathings.MoveWedgedDefender(actor)
}

// FeatureWatchIdleBots is FEATURE_WATCH_IDLE_BOTS.
//
//sp:global FEATURE_WATCH_IDLE_BOTS
func FeatureWatchIdleBots() int32 { return 17 }

// FeatureWatchLurkingSnipers is FEATURE_WATCH_LURKING_SNIPERS.
//
//sp:global FEATURE_WATCH_LURKING_SNIPERS
func FeatureWatchLurkingSnipers() int32 { return 18 }

// SpawnNavRecovery is redbots_manager_spawn_nav_recovery, the whole watch's
// switch.
//
//sp:global redbots_manager_spawn_nav_recovery
func SpawnNavRecovery() ConVar { return 0 }

// SpawnNavRecoveryRadius is how near spawn still counts as in it.
//
//sp:global redbots_manager_spawn_nav_recovery_radius
func SpawnNavRecoveryRadius() ConVar { return 0 }

// SpawnNavRecoveryTime is how long a bot gets to leave before it is moved.
//
//sp:global redbots_manager_spawn_nav_recovery_time
func SpawnNavRecoveryTime() ConVar { return 0 }

// Direction is cbasenpc's NavDirType: the four sides of a nav area.
//
//sp:tag NavDirType
type Direction int32

// DirectionNorth is NORTH, the first side.
//
//sp:global NORTH
func DirectionNorth() Direction { return 0 }

// DirectionCount is NUM_DIRECTIONS.
//
//sp:global NUM_DIRECTIONS
func DirectionCount() Direction { return 4 }

// AdjacentCount is how many areas touch this one on that side.
//
//sp:method GetAdjacentCount
func (a Area) AdjacentCount(dir Direction) int32 {
	if pathings.AdjacentCount == nil {
		missing("CNavArea.GetAdjacentCount")
	}
	return pathings.AdjacentCount(a, dir)
}

// AdjacentArea is one of them, by index.
//
//sp:method GetAdjacentArea
func (a Area) AdjacentArea(dir Direction, index int32) Area {
	if pathings.AdjacentArea == nil {
		missing("CNavArea.GetAdjacentArea")
	}
	return pathings.AdjacentArea(a, dir, index)
}

// OldWedgeRecovery is the faults injector asking for the pre-2.21.3 behaviour,
// kept only so a run can measure what replacing it was worth. Ported, faults.
//
//sp:body DebugFaults_OldWedgeRecovery
func OldWedgeRecovery() bool {
	if pathings.OldWedgeRecovery == nil {
		missing("DebugFaults_OldWedgeRecovery")
	}
	return pathings.OldWedgeRecovery()
}

// HasPathGoalVector says a behaviour set a point to walk to.
//
//sp:method HasPathGoalVector
func (b PluginBot) HasPathGoalVector() bool {
	if pathings.HasPathGoalVector == nil {
		missing("PluginBot.HasPathGoalVector")
	}
	return pathings.HasPathGoalVector(b)
}

// HasPathGoalEntity says a behaviour set a thing to walk to.
//
//sp:method HasPathGoalEntity
func (b PluginBot) HasPathGoalEntity() bool {
	if pathings.HasPathGoalEntity == nil {
		missing("PluginBot.HasPathGoalEntity")
	}
	return pathings.HasPathGoalEntity(b)
}

// PathGoalVector is the point.
//
//sp:property vecPathGoal
func (b PluginBot) PathGoalVector() [3]float32 {
	if pathings.PathGoalVector == nil {
		missing("PluginBot.vecPathGoal")
	}
	return pathings.PathGoalVector(b)
}

// PathGoalEntity is the thing.
//
//sp:property iPathGoalEntity
func (b PluginBot) PathGoalEntity() int32 {
	if pathings.PathGoalEntity == nil {
		missing("PluginBot.iPathGoalEntity")
	}
	return pathings.PathGoalEntity(b)
}
