package engine

// TankCalls are the answers.
type TankCalls struct {
	ScaleVector      func(v [3]float32, scale float32) [3]float32
	AddVectors       func(a [3]float32, b [3]float32) [3]float32
	ComputeToPos     func(p Path, bot Bot, goal [3]float32, maxDistance float32, includeGoal bool)
	EntPropVector    func(entity int32, propType PropType, prop string) [3]float32
	LogError         func(format string, args ...any)
	CountExcept      func(name string, ignore int32) int32
	ApproachWeighted func(l Locomotion, goal [3]float32, weight float32)
}

var tanks TankCalls

// InstallTanks puts a set of answers behind them.
func InstallTanks(c TankCalls) func() {
	previous := tanks
	Fill(&c)
	tanks = c
	return func() { tanks = previous }
}

// FlamethrowerReachRange is FLAMETHROWER_REACH_RANGE, near enough that anything
// inside it is a threat rather than scenery.
//
//sp:global FLAMETHROWER_REACH_RANGE
func FlamethrowerReachRange() float32 { return 350.0 }

// MaxTankAttackers is how many bots the manager lets attack a tank at once.
//
//sp:global redbots_manager_bot_max_tank_attackers
func MaxTankAttackers() ConVar { return 0 }

// CountOfBotsWithNamedActionExcept is the same count with one bot left out,
// which is how a bot asks how many others are already on the job.
//
//sp:body GetCountOfBotsWithNamedAction
func CountOfBotsWithNamedActionExcept(name string, ignore int32) int32 {
	return tanks.CountExcept(name, ignore)
}

// FeatureDemoTankPipes is the switch between the pipes and the stickies for a
// demoman standing at a tank.
//
//sp:global FEATURE_DEMO_TANK_PIPES
func FeatureDemoTankPipes() int32 { return 11 }

// ClassHeavyweapons is TFClass_Heavy.
//
//sp:global TFClass_Heavy
func ClassHeavyweapons() Class { return 6 }

// ApproachWeighted is one step towards a point with a weight on it, which is
// what the game's own declaration takes and this caller passes.
//
//sp:method Approach
func (l Locomotion) ApproachWeighted(goal [3]float32, weight float32) {
	tanks.ApproachWeighted(l, goal, weight)
}

// ScaleVector makes it that long, in place: SourcePawn's own takes no
// destination, so the Go assigns the answer back to the same vector.
//
//sp:native ScaleVector inplace
func ScaleVector(v [3]float32, scale float32) [3]float32 { return tanks.ScaleVector(v, scale) }

// AddVectors adds them.
//
//sp:native AddVectors
func AddVectors(a [3]float32, b [3]float32) (sum [3]float32) { return tanks.AddVectors(a, b) }

// ComputeToPos builds a path to a position, with its own arguments: a tank is a
// moving hull, and the goal is wanted even when the path fails.
//
//sp:method ComputeToPos
func (p Path) ComputeToPos(bot Bot, goal [3]float32, maxDistance float32, includeGoal bool) {
	tanks.ComputeToPos(p, bot, goal, maxDistance, includeGoal)
}

// EntPropVector is a vector property off an entity.
//
//sp:native GetEntPropVector
func EntPropVector(entity int32, propType PropType, prop string) (out [3]float32) {
	return tanks.EntPropVector(entity, propType, prop)
}

// LogError writes a line to the error log.
//
//sp:native LogError
func LogError(format string, args ...any) { tanks.LogError(format, args...) }
