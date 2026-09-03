/*
Package pathing is the path computation layer of
source/redbots3/nextbot_behavior.sp: every route request goes through here, so
one place owns the length cap, the per-frame budget, the failure count and the
slow-search log line.
*/
package pathing

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

/*
PathNudgeStep is how far off refusing ground a nudge steps.

The mesh usually refuses from one particular piece of ground rather than for the
whole journey, so the answer is to get off that ground: a step in the goal's
direction, guarded the same way the attack strafe guards one, and the next
computation is made from somewhere else.
*/
//
//sp:name PATH_NUDGE_STEP
const PathNudgeStep = 120.0

// PathSlowMs is the search cost worth a log line.
//
//sp:name PATH_SLOW_MS
const PathSlowMs = 20.0

// PathLengthCapDistance is the longest route the cap allows.
//
//sp:name PATH_LENGTH_CAP
const PathLengthCapDistance = 6000.0

/*
PathsPerFrame is how many paths the whole team may compute in one frame.

NavAreaBuildPath is a search over the map's nav areas and it is what the
watchdog has caught the server inside three times now: an unreachable goal makes
that search walk the whole mesh, and six bots asking in the same frame
multiplies it. Two a frame at 66 ticks is a hundred and thirty a second, far
more than the refresh ever wants, so nobody waits for a path in practice. What
it removes is the frame where everybody asks at once.
*/
//
//sp:name PATHS_PER_FRAME
const PathsPerFrame = 2

//sp:name m_iPathBudgetTick
var pathBudgetTick int32

//sp:name m_iPathsThisTick
var pathsThisTick int32

// TakePathBudget says there is room to compute a path this frame. Only the
// per-frame refresh asks: a behaviour that computes once when it starts has
// nothing to retry with, so it is never refused.
//
//sp:name TakePathBudget
func TakePathBudget() bool {
	tick := engine.GameTickCount()

	if tick != pathBudgetTick {
		pathBudgetTick = tick
		pathsThisTick = 0
	}

	if pathsThisTick >= PathsPerFrame {
		return false
	}

	pathsThisTick++

	return true
}

//sp:name m_bPathFailed
var pathFailed [slots.Count]bool

//sp:name m_iPathFailures
var pathFailures [slots.Count]int32

// PathFailuresOf is how many times this bot's route requests have failed.
//
//sp:name PathFailuresOf
func PathFailuresOf(client int32) int32 {
	return pathFailures[client]
}

// PathFailedFor says the last request failed.
//
//sp:name PathFailedFor
func PathFailedFor(client int32) bool {
	return pathFailed[client]
}

// NudgeTowardsGoal steps a grounded bot off ground the mesh refuses from.
//
//sp:name NudgeTowardsGoal
//sp:const goal
func NudgeTowardsGoal(client int32, myBot engine.Bot, goal [3]float32) {
	myLoco := myBot.Locomotion()

	if !myLoco.IsOnGround() {
		return
	}

	myOrigin := engine.AbsOriginOf(client)
	towards := engine.SubtractVectors(goal, myOrigin)

	towards[2] = 0.0

	length, towards := engine.NormalizeVector(towards)

	if length < 1.0 {
		return
	}

	var step [3]float32
	step[0] = myOrigin[0] + towards[0]*PathNudgeStep
	step[1] = myOrigin[1] + towards[1]*PathNudgeStep
	step[2] = myOrigin[2]

	if !myLoco.IsPotentiallyTraversable(myOrigin, step, engine.Immediately()) || myLoco.HasPotentialGap(myOrigin, step) {
		return
	}

	myLoco.Approach(step)
}

// RepathToTarget asks for a route to an entity, measured and counted.
//
//sp:name RepathToTarget
func RepathToTarget(actor int32, myBot engine.Bot, target int32) {
	began := engine.WallClock()
	built := engine.PathOf(actor).ComputeToTargetBuilt(myBot, target, PathLengthCap())

	SayIfSlow(actor, began, "to a target")
	NotePathResult(actor, built)
}

// SayIfSlow puts a search that cost real frame time in the log.
//
//sp:name SayIfSlow
func SayIfSlow(actor int32, began float32, what string) {
	ms := (engine.WallClock() - began) * 1000.0

	if ms < PathSlowMs {
		return
	}

	engine.LogMessage("Path: %N spent %.0fms searching %s", actor, ms, what)
}

/*
RepathToPos asks for a route to a point, measured and counted.

The faults injector can substitute a goal with no path to it: a wedged bot is
not expensive on its own, and what the cores show is NavAreaBuildPath walking
the whole mesh for an answer it never finds. Pinning a bot reproduces the wedge
and none of the cost, so the injector sends the held bot at a point off the mesh
instead.
*/
//
//sp:name RepathToPos
//sp:const goal
func RepathToPos(actor int32, myBot engine.Bot, goal [3]float32) {
	injected, unreachable := engine.UnreachableGoal(actor)

	if injected {
		began := engine.WallClock()
		built := engine.PathOf(actor).ComputeToPosBuilt(myBot, unreachable, PathLengthCap())

		SayIfSlow(actor, began, "a goal with no path")
		NotePathResult(actor, built)
		return
	}

	began := engine.WallClock()
	built := engine.PathOf(actor).ComputeToPosBuilt(myBot, goal, PathLengthCap())

	SayIfSlow(actor, began, "to a position")
	NotePathResult(actor, built)
}

// PathLengthCap is the cap when the feature asks for one, and no cap otherwise.
//
//sp:name PathLengthCap
func PathLengthCap() float32 {
	if engine.Feature(engine.FeaturePathLengthCap()) {
		return PathLengthCapDistance
	}

	return 0.0
}

// NotePathResult counts a failure once per streak, so the count reads as
// incidents rather than frames.
//
//sp:name NotePathResult
func NotePathResult(actor int32, built bool) {
	failed := !built || engine.PathOf(actor).Length() <= 0.0

	if failed && !pathFailed[actor] {
		pathFailures[actor]++
	}

	pathFailed[actor] = failed
}

// PathRetryInterval is how long a refused path waits before it is asked for
// again: retrying a refusal on the walking interval is most of a frame's path
// work for nothing.
//
//sp:name PATH_RETRY_INTERVAL
const PathRetryInterval = 0.5

/*
PluginBotSimulateFrame is the per-frame walk: whenever a behaviour has set a
goal and bPathing, this is what actually gets the bot there.

An empty path is a failure the same as a refusal, and it is the shape seen in
play. The engineer skips it inside his supply runs so the two pathings do not
fight.
*/
//
//sp:name PluginBot_SimulateFrame
func PluginBotSimulateFrame(client int32) {
	// SimulateFrame > PFContext::RecalculatePath. This is used whenever we
	// want to path somewhere constantly.
	if engine.PluginBotOf(client).Pathing() {
		if engine.PlayerClass(client) == engine.ClassEngineer() {
			// Dumb hack for engineer so pathing does not conflict.
			if engine.LookupEntityActionByName(client, "DefenderGetAmmo") != engine.InvalidAction() || engine.LookupEntityActionByName(client, "DefenderGetHealth") != engine.InvalidAction() {
				return
			}
		}

		shouldPathToVec := engine.PluginBotOf(client).HasPathGoalVector()
		shouldPathToEntity := engine.PluginBotOf(client).HasPathGoalEntity()

		if shouldPathToVec || shouldPathToEntity {
			myBot := engine.NextBotOf(client)

			var goal [3]float32

			if shouldPathToVec {
				goal = engine.PluginBotOf(client).PathGoalVector()
			} else {
				goal = engine.AbsOriginOf(engine.PluginBotOf(client).PathGoalEntity())
			}

			if engine.RepathTime(client) <= engine.GameTime() && TakePathBudget() {
				engine.CombatOf(client).UpdateLastKnownArea()

				var built bool

				if shouldPathToVec {
					vecGoal := engine.PluginBotOf(client).PathGoalVector()
					built = engine.PathOf(client).ComputeToPosBuilt(myBot, vecGoal, PathLengthCap())
				} else {
					built = engine.PathOf(client).ComputeToTargetBuilt(myBot, engine.PluginBotOf(client).PathGoalEntity(), PathLengthCap())
				}

				failed := !built || engine.PathOf(client).Length() <= 0.0

				if failed && !pathFailed[client] {
					pathFailures[client]++
				}

				pathFailed[client] = failed

				engine.SetRepathTime(client, engine.GameTime()+engine.ChooseFloat(failed, PathRetryInterval, 0.2))
			}

			if pathFailed[client] {
				NudgeTowardsGoal(client, myBot, goal)
			} else {
				engine.PathOf(client).Update(myBot)
			}
		}
	}
}
