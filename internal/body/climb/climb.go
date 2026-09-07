/*
Package climb gets the engineer up onto ground the map named, instead of leaving
him building or wrenching at the bottom of it.

Bigrock puts both nests and the teleporter exit on rocks about seventy units
above the floor beside them, with no nav area on top. The path finder can only
take him to the foot, every placement from down there is refused, and a sentry
standing up on the rock cannot be repaired from below. Reported from play as
the exit in the bot lane (mvm-fgs) and the nest at the foot of the rock
(mvm-wxp).

One helper for the three builders and the repair walk, so a spot on a rock is
one case rather than four. A rise of more than a step and no more than a
crouch jump, from close enough to land on it: he looks at the spot, walks into
it and crouch jumps, which is what a person does. Bounded by a count, and when
the count runs out and he is still standing at the foot of it, he is lifted
onto it the way the break already lifts him onto his nest. A building he can
place but never reach again is not a nest, so the lift is the guarantee and the
jump is the ordinary way.
*/
package climb

import (
	"github.com/m-this/tf2-mvm-bots-go/internal/body/slots"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

// Range is how far out a jump still lands on top rather than on the wall, and
// so how near he has to be before the height is what keeps him off.
//
//sp:name CLIMB_RANGE
const Range = 140.0

/*
TopReach is how far short of the spot he stands once he is up on the ground it
sits on.

Less than a build's reach, because the ground up there is about as wide as the
spot's own nav-less rock and a full reach out is its edge. It is not zero: the
break puts him on the spot itself, and a man standing on the spot looking at it
is looking at his feet, which the game refused for a whole 45 second attempt.
Measured on Bigrock's exit, an exit aimed at from 21 to 48 units out landed 55
from its spot.
*/
//
//sp:name CLIMB_TOP_REACH
const TopReach = 40.0

// TopRange is how far from the spot he may stand up there and still be left
// alone. Landings come down anywhere from 60 to 120 out, and a teleporter aimed
// at from 97 was refused from every side, so past this he walks in.
//
//sp:name CLIMB_TOP_RANGE
const TopRange = 70.0

// TopOnSpot is how close to the spot counts as standing on it, which is where a
// build aimed at the spot is aimed at his feet. The break's teleport puts him at
// zero; a landing rarely does.
//
//sp:name CLIMB_TOP_ON_SPOT
const TopOnSpot = 16.0

// stepHold is one press of forward up on the rock: a tenth of a second is about
// thirty units, so a step stops inside the reach it aims for. The jump's hold
// of three tenths walked him off Bigrock's nest.
//
//sp:name CLIMB_STEP_HOLD
const stepHold = 0.1

const (
	// Above a step and no higher than a crouch jump. Anything higher is a wall.
	//
	//sp:name CLIMB_RISE_MIN
	riseMin = 24.0
	//sp:name CLIMB_RISE_MAX
	riseMax = 72.0
	//sp:name CLIMB_INTERVAL
	interval = 0.7
	//sp:name CLIMB_HOLD
	hold = 0.3
	// Jumps before he is lifted instead.
	//
	//sp:name CLIMB_LIMIT
	limit = 6
	// Lifts per Begin before the ground is taken to be unfit and he is left to the
	// caller's own fallback. Thirty eight in one run is what an unbounded lift did.
	//
	//sp:name CLIMB_LIFT_LIMIT
	liftLimit = 2
	// How far above the spot a lift puts him. A declared spot is where somebody's
	// feet were, and Bigrock's exit is 27 units under the rock it names: set down
	// at the spot he is inside the rock, and the engine pushes him out and off it.
	// From a crouch jump's height above it he lands on whatever the surface is.
	//
	//sp:name CLIMB_LIFT_ABOVE
	liftAbove = 90.0
)

// The answers ToSpot gives, in the order a caller tests them.
const (
	// None is nothing to climb: he is level with it, it is a wall, or he is
	// not near it. The caller carries on as if this package did not exist.
	//
	//sp:name CLIMB_NONE
	None = 0
	// Busy is a jump or a lift in flight. The caller stops walking and waits.
	//
	//sp:name CLIMB_BUSY
	Busy = 1
	// Landed is the frame he arrives on top. Where he stands is now the stand
	// point, because asking the mesh again answers with the floor he just left.
	//
	//sp:name CLIMB_LANDED
	Landed = 2
)

var (
	//sp:name m_ctClimbAt
	climbAt [slots.Count]float32
	//sp:name m_iClimbs
	climbs [slots.Count]int32
	//sp:name m_iClimbLifts
	lifts [slots.Count]int32
	//sp:name m_bClimbLifted
	lifted [slots.Count]bool
	// Where he stood when he last jumped or was lifted, or the floor point the
	// caller named. Up on top he steps back towards it, because that is the side
	// the rock is known to extend to: he landed on it coming from there.
	//
	//sp:name m_vClimbFoot
	foot [slots.Count][3]float32
	// On top since the last landing, until Begin. What the repair walk reads
	// so that it does not path him back down while he wrenches.
	//
	//sp:name m_bClimbOnTop
	onTop [slots.Count]bool
)

// ResetClimb forgets a seat's climb, which the next bot in it did not start.
//
//sp:name Go_ResetClimb
func ResetClimb(client int32) {
	Begin(client)
}

// Begin is a fresh count for a fresh spot, or for the same spot after he left it.
//
//sp:name ClimbBegin
func Begin(actor int32) {
	climbAt[actor] = 0.0
	climbs[actor] = 0
	lifts[actor] = 0
	lifted[actor] = false
	onTop[actor] = false
}

// OnTop is whether he landed on something since the last Begin.
//
//sp:name ClimbOnTop
func OnTop(actor int32) bool {
	return onTop[actor]
}

/*
Beside is whether he already stands level with the spot and within a jump of it,
which is where the break's teleport puts him. A stand point taken from the mesh
from there is the floor below, and walking to it is walking off the rock.
*/
//
//sp:name ClimbBeside
//sp:const spot
func Beside(actor int32, spot [3]float32) bool {
	origin := engine.AbsOriginOf(actor)

	rise := spot[2] - origin[2]

	reach := engine.SubtractVectors(spot, origin)

	reach[2] = 0.0

	return rise < riseMin && rise > -riseMin && engine.VectorLength(reach) <= Range
}

/*
AtFoot is whether he is close enough to the spot that the height, not the walk,
is what keeps him off it: within a jump's range flat and no higher than a crouch
jump below it. Nothing should path him from here, because the path finder has
nothing to offer on the rock and asks the whole mesh before saying so.
*/
//
//sp:name ClimbAtFoot
//sp:const spot
func AtFoot(actor int32, spot [3]float32) bool {
	origin := engine.AbsOriginOf(actor)

	rise := spot[2] - origin[2]

	reach := engine.SubtractVectors(spot, origin)

	reach[2] = 0.0

	return rise > -riseMin && rise <= riseMax && engine.VectorLength(reach) <= Range
}

// MarkOnTop says he is level with the spot without having climbed: the break's
// teleport put him there. toward is the floor point beside the rock he would
// otherwise have walked to, which is the side to step back to.
//
//sp:name ClimbMarkOnTop
//sp:const toward
func MarkOnTop(actor int32, toward [3]float32) {
	onTop[actor] = true
	foot[actor] = toward
}

/*
StepBack puts him a short reach from a spot he is up beside, and is true while he
is still moving: off it when he is standing on it, in to it when he landed far
out.

Up here the stand point is wherever he stands, because nothing on the rock is on
the mesh: a path asked for to a point up here is a search for ground the mesh
does not have, and that is the kind of frame the server's watchdog kills. So
nothing is pathed and nothing is steered. He looks where he is going and presses
forward, the way the jump does. Off the spot he goes towards the side he came
up, which is the side the rock is known to reach to; in to it he goes straight,
because the spot itself is on the rock.
*/
//
//sp:name ClimbStepBack
//sp:const spot
func StepBack(actor int32, myBody engine.Body, spot [3]float32) bool {
	here := engine.SubtractVectors(engine.AbsOriginOf(actor), spot)

	here[2] = 0.0

	//nolint:staticcheck,ineffassign,wastedassign // the name exists because SourcePawn writes the unit vector through it
	distance, here := engine.NormalizeVector(here)

	if distance >= TopOnSpot && distance <= TopRange {
		return false
	}

	// Looking at a point past where he stops, so the walk does not stall on the aim
	var goal [3]float32

	if distance > TopRange {
		goal = spot
	} else {
		away := engine.SubtractVectors(foot[actor], spot)

		away[2] = 0.0

		length, unit := engine.NormalizeVector(away)

		if length < 1.0 {
			unit[0] = 1.0
			unit[1] = 0.0
		}

		goal[0] = spot[0] + unit[0]*(TopReach+Range)
		goal[1] = spot[1] + unit[1]*(TopReach+Range)
		goal[2] = spot[2]
	}

	engine.AimHeadTowards(myBody, goal, engine.AimMandatory(), 0.2, engine.NoAddress(), "Stepping on a spot")

	if myBody.IsHeadAimingOnTarget() {
		engine.ExtraButtonsOf(actor).PressButtons(engine.InForward(), stepHold)
	}

	return true
}

// Say writes the reason behind a climb, under redbots_manager_debug_actions.
//
//sp:name ClimbSay
func Say(actor int32, what string, why string, rise float32, flat float32) {
	if !engine.DebugActions().Bool() {
		return
	}

	engine.PrintToServer("[climb] %N %s %s, rise %.0f of %.0f to %.0f, out %.0f of %.0f, climb %d of %d",
		actor, why, what, rise, riseMin, riseMax, flat, Range, climbs[actor], limit)
}

/*
ToSpot crouch jumps onto the ground the spot sits on, and lifts him onto it when
the jumps run out.

The stand point comes off the nav mesh, so for a spot on a rock the mesh does not
cover it is the floor underneath: he arrives, the spot is over his head, and
everything from down there is refused. This puts him on top instead.

Once he is up, where he stands is where he stands: Landed tells the caller to keep
it rather than recompute it. The count resets when he makes it, so falling off and
climbing again costs another six jumps rather than none; the caller's own clocks
bound the pair of them.

The lift happens once per climb: six jumps that did not land earn it, and landing
clears it, so falling off again starts a new six. The between-rounds setup already
sets his origin onto the nest, so a lift mid-wave is the same move at a moment he
has earned.
*/
//
//sp:name ClimbToSpot
func ToSpot(actor int32, myBody engine.Body, spot [3]float32, what string) int32 {
	if !engine.Feature(engine.FeatureEngineerClimbs()) {
		return None
	}

	origin := engine.AbsOriginOf(actor)

	rise := spot[2] - origin[2]

	reach := engine.SubtractVectors(spot, origin)

	reach[2] = 0.0

	out := engine.VectorLength(reach)

	if rise < riseMin {
		if climbs[actor] > 0 || lifted[actor] {
			// Level with it at the top of the jump is not up: Rottenburg's exit read as landed six
			// times in a row and he was on the floor below each time the next frame came round
			if !engine.NextBotOf(actor).Locomotion().IsOnGround() {
				return Busy
			}

			Say(actor, what, "landed at", rise, out)

			climbs[actor] = 0
			climbAt[actor] = 0.0
			lifted[actor] = false
			onTop[actor] = true

			return Landed
		}

		return None
	}

	// Higher than a crouch jump is not a ledge, it is a wall, and no number of jumps will do it
	// Not a ledge but a wall, or not at the foot of it yet. Said nowhere: this is every tick of a walk
	if rise > riseMax || out > Range {
		return None
	}

	if climbs[actor] >= limit {
		if lifted[actor] || lifts[actor] >= liftLimit {
			Say(actor, what, "out of climbs at", rise, out)

			return None
		}

		Say(actor, what, "lifted onto", rise, out)

		lifted[actor] = true
		lifts[actor]++
		foot[actor] = origin

		up := spot
		up[2] += liftAbove

		engine.EntityOf(actor).SetAbsOrigin(up)

		return Busy
	}

	Say(actor, what, "climbing to", rise, out)

	foot[actor] = origin

	engine.AimHeadTowards(myBody, spot, engine.AimMandatory(), 0.2, engine.NoAddress(), "Climbing to a spot")

	if climbAt[actor] > engine.GameTime() {
		return Busy
	}

	climbAt[actor] = engine.GameTime() + interval
	climbs[actor]++

	// Forward is along where he is looking, which is the spot, so the three together are a person
	engine.ExtraButtonsOf(actor).PressButtons(engine.InForward()|engine.InJump()|engine.InDuck(), hold)

	return Busy
}
