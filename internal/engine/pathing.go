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
